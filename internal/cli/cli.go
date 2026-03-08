// Package cli implements a simple command dispatcher using stdlib only.
// No cobra dependency -- just os.Args routing and flag parsing.
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/chronick/bosun/internal/claim"
	"github.com/chronick/bosun/internal/complete"
	"github.com/chronick/bosun/internal/config"
	"github.com/chronick/bosun/internal/heartbeat"
	"github.com/chronick/bosun/internal/lease"
	"github.com/chronick/bosun/internal/lifecycle"
	"github.com/chronick/bosun/internal/register"
)

// Command represents a subcommand with a name, description, and run function.
type Command struct {
	Name  string
	Desc  string
	Usage string
	Run   func(cfg *config.Config, args []string) error
}

// Commands returns the full list of registered commands.
func Commands() []Command {
	return []Command{
		{
			Name:  "run",
			Desc:  "Full lifecycle loop (boot, claim, work, report)",
			Usage: "bosun run",
			Run:   cmdRun,
		},
		{
			Name:  "register",
			Desc:  "Announce agent identity to agent-mail",
			Usage: "bosun register",
			Run:   cmdRegister,
		},
		{
			Name:  "claim",
			Desc:  "Pick highest-priority ready task from beads",
			Usage: "bosun claim",
			Run:   cmdClaim,
		},
		{
			Name:  "lease",
			Desc:  "Acquire file lease via agent-mail",
			Usage: "bosun lease <task-id>",
			Run:   cmdLease,
		},
		{
			Name:  "release",
			Desc:  "Release file lease via agent-mail",
			Usage: "bosun release <task-id>",
			Run:   cmdRelease,
		},
		{
			Name:  "heartbeat",
			Desc:  "Send liveness ping to agent-mail",
			Usage: "bosun heartbeat",
			Run:   cmdHeartbeat,
		},
		{
			Name:  "complete",
			Desc:  "Close task, create PR, release lease",
			Usage: "bosun complete <task-id> [--title <pr-title>]",
			Run:   cmdComplete,
		},
	}
}

// Run dispatches the appropriate subcommand based on args.
func Run(args []string, version string) error {
	if len(args) == 0 {
		printUsage(version)
		return nil
	}

	subcmd := args[0]

	// Handle top-level flags.
	switch subcmd {
	case "--help", "-h", "help":
		printUsage(version)
		return nil
	case "--version", "-v", "version":
		fmt.Printf("bosun %s\n", version)
		return nil
	}

	commands := Commands()
	for _, cmd := range commands {
		if cmd.Name == subcmd {
			cfg := config.Load()
			return cmd.Run(cfg, args[1:])
		}
	}

	return fmt.Errorf("unknown command: %s\nRun 'bosun help' for usage.", subcmd)
}

func printUsage(version string) {
	w := os.Stdout
	fmt.Fprintf(w, "bosun %s -- agent lifecycle coordinator\n\n", version)
	fmt.Fprintln(w, "Usage: bosun <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")

	commands := Commands()
	maxLen := 0
	for _, cmd := range commands {
		if len(cmd.Name) > maxLen {
			maxLen = len(cmd.Name)
		}
	}

	for _, cmd := range commands {
		padding := strings.Repeat(" ", maxLen-len(cmd.Name)+2)
		fmt.Fprintf(w, "  %s%s%s\n", cmd.Name, padding, cmd.Desc)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Environment Variables:")
	fmt.Fprintln(w, "  AGENT_NAME           Agent identity (required for run)")
	fmt.Fprintln(w, "  AGENT_COMMAND        Agent runtime to invoke (required for run)")
	fmt.Fprintln(w, "  AGENT_MAIL_URL       Agent mail server URL (optional)")
	fmt.Fprintln(w, "  BEADS_DB             Path to beads database (optional)")
	fmt.Fprintln(w, "  TASK_FILTER          Beads query filter (optional)")
	fmt.Fprintln(w, "  HEARTBEAT_INTERVAL   Seconds between heartbeats (default: 30)")
	fmt.Fprintln(w, "  IDLE_SLEEP           Seconds when no tasks available (default: 30)")
}

// --- Command implementations ---

func cmdRun(cfg *config.Config, args []string) error {
	if cfg.AgentName == "" {
		return fmt.Errorf("AGENT_NAME is required")
	}
	if cfg.AgentCommand == "" {
		return fmt.Errorf("AGENT_COMMAND is required")
	}
	return lifecycle.Run(cfg)
}

func cmdRegister(cfg *config.Config, args []string) error {
	if cfg.AgentName == "" {
		return fmt.Errorf("AGENT_NAME is required")
	}
	return register.Register(cfg)
}

func cmdClaim(cfg *config.Config, args []string) error {
	taskID, err := claim.Claim(cfg)
	if err != nil {
		return err
	}
	if taskID == "" {
		// No tasks available -- exit cleanly with no output.
		return nil
	}
	fmt.Println(taskID)
	return nil
}

func cmdLease(cfg *config.Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: bosun lease <task-id>")
	}
	return lease.Acquire(cfg, args[0])
}

func cmdRelease(cfg *config.Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: bosun release <task-id>")
	}
	return lease.Release(cfg, args[0])
}

func cmdHeartbeat(cfg *config.Config, args []string) error {
	if cfg.AgentName == "" {
		return fmt.Errorf("AGENT_NAME is required")
	}
	return heartbeat.Send(cfg, "", "idle")
}

func cmdComplete(cfg *config.Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: bosun complete <task-id> [--title <pr-title>]")
	}
	taskID := args[0]
	title := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--title" && i+1 < len(args) {
			title = args[i+1]
			break
		}
	}
	return complete.Complete(cfg, taskID, title)
}
