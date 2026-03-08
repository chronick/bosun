// Package complete handles post-task cleanup: close beads issue, create PR,
// release lease, and reset worktree. Each step is best-effort.
package complete

import (
	"context"
	"fmt"
	"log"

	"github.com/chronick/bosun/internal/config"
	"github.com/chronick/bosun/internal/lease"
	"github.com/chronick/bosun/internal/toolexec"
)

// Complete performs all post-task cleanup for the given task ID.
// Each step is best-effort -- failure in one does not block others.
func Complete(cfg *config.Config, taskID string, prTitle string) error {
	return CompleteWithRunner(context.Background(), cfg, taskID, prTitle, &toolexec.DefaultRunner{})
}

// CompleteWithRunner performs post-task cleanup using the provided runner.
func CompleteWithRunner(ctx context.Context, cfg *config.Config, taskID string, prTitle string, runner toolexec.Runner) error {
	var errs []error

	// Step 1: Close the beads issue.
	if _, err := toolexec.BR(ctx, runner, "close", taskID); err != nil {
		log.Printf("bosun: failed to close task %s: %v", taskID, err)
		errs = append(errs, err)
	} else {
		log.Printf("bosun: closed task %s", taskID)
	}

	// Step 2: Create PR via hoist (if agent name is set).
	if cfg.AgentName != "" {
		branch := fmt.Sprintf("agent/%s", cfg.AgentName)
		args := []string{"pr", branch}
		if prTitle != "" {
			args = append(args, "--title", prTitle)
		}

		output, err := toolexec.Hoist(ctx, runner, args...)
		if err != nil {
			log.Printf("bosun: failed to create PR: %v", err)
			errs = append(errs, err)
		} else {
			if output != "" {
				fmt.Println(output)
			}
			log.Printf("bosun: PR created for %s", branch)
		}
	}

	// Step 3: Release file lease.
	if err := lease.ReleaseWithContext(ctx, cfg, taskID); err != nil {
		log.Printf("bosun: failed to release lease for %s: %v", taskID, err)
		errs = append(errs, err)
	}

	// Step 4: Reset worktree.
	if cfg.AgentName != "" {
		branch := fmt.Sprintf("agent/%s", cfg.AgentName)
		if _, err := toolexec.Hoist(ctx, runner, "reset", branch); err != nil {
			log.Printf("bosun: failed to reset worktree %s: %v", branch, err)
			errs = append(errs, err)
		} else {
			log.Printf("bosun: reset worktree %s", branch)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("complete: %d step(s) failed (see logs)", len(errs))
	}
	return nil
}
