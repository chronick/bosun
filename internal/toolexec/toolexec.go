// Package toolexec wraps os/exec calls to external tools (br, hoist, etc.).
// It provides a thin layer for testability and consistent error handling.
package toolexec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes external commands. The default implementation calls os/exec.
// Tests can replace this with a mock.
type Runner interface {
	// Run executes a command and returns combined stdout. Stderr goes to the
	// returned error on non-zero exit.
	Run(ctx context.Context, name string, args ...string) (string, error)

	// Start starts a command and returns the exec.Cmd for lifecycle management.
	Start(ctx context.Context, name string, args ...string) (*exec.Cmd, error)
}

// DefaultRunner is the standard Runner using os/exec.
type DefaultRunner struct{}

// Run executes the command, captures stdout, and returns it trimmed.
func (r *DefaultRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), errMsg)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Start starts the command without waiting for it to finish.
func (r *DefaultRunner) Start(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", name, err)
	}
	return cmd, nil
}

// BR runs a beads_rust command.
func BR(ctx context.Context, r Runner, args ...string) (string, error) {
	return r.Run(ctx, "br", args...)
}

// Hoist runs a hoist command.
func Hoist(ctx context.Context, r Runner, args ...string) (string, error) {
	return r.Run(ctx, "hoist", args...)
}
