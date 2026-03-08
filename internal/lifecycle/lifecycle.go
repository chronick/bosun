// Package lifecycle implements the main bosun run loop:
// register -> loop(claim -> lease -> work -> complete/fail -> release -> reset) -> shutdown.
package lifecycle

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/chronick/bosun/internal/claim"
	"github.com/chronick/bosun/internal/complete"
	"github.com/chronick/bosun/internal/config"
	"github.com/chronick/bosun/internal/heartbeat"
	"github.com/chronick/bosun/internal/lease"
	"github.com/chronick/bosun/internal/register"
	"github.com/chronick/bosun/internal/toolexec"
)

// state tracks the current lifecycle state for shutdown cleanup.
type state struct {
	mu        sync.Mutex
	taskID    string
	agentCmd  *exec.Cmd
}

func (s *state) setTask(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taskID = id
}

func (s *state) getTask() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.taskID
}

func (s *state) setCmd(cmd *exec.Cmd) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agentCmd = cmd
}

func (s *state) getCmd() *exec.Cmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.agentCmd
}

// Run executes the full lifecycle loop. It blocks until SIGTERM/SIGINT.
func Run(cfg *config.Config) error {
	return RunWithRunner(cfg, &toolexec.DefaultRunner{})
}

// RunWithRunner executes the lifecycle loop with a custom tool runner.
func RunWithRunner(cfg *config.Config, runner toolexec.Runner) error {
	log.Printf("bosun: starting lifecycle for agent %s", cfg.AgentName)

	// Set up cancellation context for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := &state{}

	// Trap SIGTERM, SIGINT for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigCh
		log.Printf("bosun: received %s, shutting down...", sig)
		cancel()
		shutdown(cfg, st, runner)
	}()

	// Step 1: Register with agent-mail.
	if err := register.RegisterWithContext(ctx, cfg); err != nil {
		log.Printf("bosun: registration failed (continuing): %v", err)
	}

	// Step 2: Start heartbeat loop.
	go heartbeat.StartLoop(ctx, cfg, st.getTask)

	// Step 3: Main work loop.
	for {
		select {
		case <-ctx.Done():
			log.Println("bosun: context cancelled, exiting loop")
			return nil
		default:
		}

		err := iteration(ctx, cfg, st, runner)
		if err != nil {
			if ctx.Err() != nil {
				// Context cancelled -- clean exit.
				return nil
			}
			log.Printf("bosun: iteration error: %v", err)
		}
	}
}

// iteration runs one claim-work-complete cycle. Returns nil to continue looping.
func iteration(ctx context.Context, cfg *config.Config, st *state, runner toolexec.Runner) error {
	// Claim a task.
	taskID, err := claim.ClaimWithRunner(ctx, cfg, runner)
	if err != nil {
		return fmt.Errorf("claim: %w", err)
	}

	if taskID == "" {
		// No tasks -- idle sleep.
		log.Printf("bosun: no tasks, sleeping %s", cfg.IdleSleep)
		return sleepCtx(ctx, cfg.IdleSleep)
	}

	st.setTask(taskID)
	defer st.setTask("")

	log.Printf("bosun: working on task %s", taskID)

	// Acquire file lease.
	if err := lease.AcquireWithContext(ctx, cfg, taskID); err != nil {
		log.Printf("bosun: lease acquire failed (continuing): %v", err)
	}

	// Create worktree.
	branch := fmt.Sprintf("agent/%s", cfg.AgentName)
	if _, err := toolexec.Hoist(ctx, runner, "create", branch); err != nil {
		log.Printf("bosun: hoist create failed (continuing): %v", err)
	}

	// Run the agent command.
	agentErr := runAgent(ctx, cfg, st)

	if agentErr != nil {
		// Agent failed -- revert task to open.
		log.Printf("bosun: agent failed: %v", agentErr)
		if _, err := toolexec.BR(ctx, runner, "update", taskID, "--status=open"); err != nil {
			log.Printf("bosun: failed to revert task %s: %v", taskID, err)
		}
	} else {
		// Agent succeeded -- complete the task.
		log.Printf("bosun: agent succeeded for task %s", taskID)
		if err := complete.CompleteWithRunner(ctx, cfg, taskID, "", runner); err != nil {
			log.Printf("bosun: complete failed: %v", err)
		}
	}

	// Release lease regardless of outcome.
	if err := lease.ReleaseWithContext(ctx, cfg, taskID); err != nil {
		log.Printf("bosun: lease release failed: %v", err)
	}

	// Reset worktree.
	if _, err := toolexec.Hoist(ctx, runner, "reset", branch); err != nil {
		log.Printf("bosun: hoist reset failed: %v", err)
	}

	return nil
}

// runAgent executes the agent command and waits for completion.
func runAgent(ctx context.Context, cfg *config.Config, st *state) error {
	parts := strings.Fields(cfg.AgentCommand)
	if len(parts) == 0 {
		return fmt.Errorf("AGENT_COMMAND is empty")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	st.setCmd(cmd)
	defer st.setCmd(nil)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start agent: %w", err)
	}

	return cmd.Wait()
}

// shutdown performs graceful cleanup when a signal is received.
func shutdown(cfg *config.Config, st *state, runner toolexec.Runner) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Kill agent process if running.
	if cmd := st.getCmd(); cmd != nil && cmd.Process != nil {
		log.Println("bosun: sending SIGTERM to agent process")
		_ = cmd.Process.Signal(syscall.SIGTERM)

		// Give it a moment to exit gracefully.
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			log.Println("bosun: agent did not exit, sending SIGKILL")
			_ = cmd.Process.Kill()
		}
	}

	// Revert in-progress task to open.
	if taskID := st.getTask(); taskID != "" {
		log.Printf("bosun: reverting task %s to open", taskID)
		if _, err := toolexec.BR(shutdownCtx, runner, "update", taskID, "--status=open"); err != nil {
			log.Printf("bosun: failed to revert task: %v", err)
		}

		// Release lease.
		if err := lease.ReleaseWithContext(shutdownCtx, cfg, taskID); err != nil {
			log.Printf("bosun: failed to release lease: %v", err)
		}
	}

	// Send final heartbeat.
	_ = heartbeat.SendWithContext(shutdownCtx, cfg, st.getTask(), "shutdown")

	log.Println("bosun: shutdown complete")
}

// sleepCtx sleeps for the given duration, returning early if ctx is cancelled.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
