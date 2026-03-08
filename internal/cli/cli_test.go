package cli

import (
	"strings"
	"testing"
)

func TestRun_Help(t *testing.T) {
	// --help should not error.
	for _, arg := range []string{"--help", "-h", "help"} {
		err := Run([]string{arg}, "test")
		if err != nil {
			t.Errorf("Run(%q) returned error: %v", arg, err)
		}
	}
}

func TestRun_Version(t *testing.T) {
	for _, arg := range []string{"--version", "-v", "version"} {
		err := Run([]string{arg}, "1.2.3")
		if err != nil {
			t.Errorf("Run(%q) returned error: %v", arg, err)
		}
	}
}

func TestRun_NoArgs(t *testing.T) {
	err := Run(nil, "test")
	if err != nil {
		t.Errorf("Run(nil) returned error: %v", err)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	err := Run([]string{"nonexistent"}, "test")
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("error = %q, want to contain 'unknown command'", err.Error())
	}
}

func TestCommands_AllPresent(t *testing.T) {
	expected := []string{"run", "register", "claim", "lease", "release", "heartbeat", "complete"}
	commands := Commands()

	names := make(map[string]bool)
	for _, cmd := range commands {
		names[cmd.Name] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing command: %s", name)
		}
	}
}

func TestCommands_NoDuplicates(t *testing.T) {
	commands := Commands()
	seen := make(map[string]bool)

	for _, cmd := range commands {
		if seen[cmd.Name] {
			t.Errorf("duplicate command: %s", cmd.Name)
		}
		seen[cmd.Name] = true
	}
}

func TestRun_ClaimRequiresNoSpecialEnv(t *testing.T) {
	// claim command should work without AGENT_NAME set
	// (it will fail trying to run br, but the dispatcher itself should work).
	// We can't test the full flow without br, but we verify dispatch works.
	err := Run([]string{"claim"}, "test")
	// This will fail because `br` is not available, which is expected.
	// We just verify it dispatched to the right command (not "unknown command").
	if err != nil && strings.Contains(err.Error(), "unknown command") {
		t.Errorf("claim command not registered properly")
	}
}

func TestRun_LeaseRequiresTaskID(t *testing.T) {
	err := Run([]string{"lease"}, "test")
	if err == nil {
		t.Fatal("expected error for lease without task-id")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestRun_ReleaseRequiresTaskID(t *testing.T) {
	err := Run([]string{"release"}, "test")
	if err == nil {
		t.Fatal("expected error for release without task-id")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestRun_CompleteRequiresTaskID(t *testing.T) {
	err := Run([]string{"complete"}, "test")
	if err == nil {
		t.Fatal("expected error for complete without task-id")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestRun_RunRequiresAgentName(t *testing.T) {
	// Unset env vars to ensure clean test.
	t.Setenv("AGENT_NAME", "")
	t.Setenv("AGENT_COMMAND", "")

	err := Run([]string{"run"}, "test")
	if err == nil {
		t.Fatal("expected error for run without AGENT_NAME")
	}
	if !strings.Contains(err.Error(), "AGENT_NAME") {
		t.Errorf("error = %q, want to mention AGENT_NAME", err.Error())
	}
}

func TestRun_RegisterRequiresAgentName(t *testing.T) {
	t.Setenv("AGENT_NAME", "")

	err := Run([]string{"register"}, "test")
	if err == nil {
		t.Fatal("expected error for register without AGENT_NAME")
	}
	if !strings.Contains(err.Error(), "AGENT_NAME") {
		t.Errorf("error = %q, want to mention AGENT_NAME", err.Error())
	}
}

func TestRun_HeartbeatRequiresAgentName(t *testing.T) {
	t.Setenv("AGENT_NAME", "")

	err := Run([]string{"heartbeat"}, "test")
	if err == nil {
		t.Fatal("expected error for heartbeat without AGENT_NAME")
	}
	if !strings.Contains(err.Error(), "AGENT_NAME") {
		t.Errorf("error = %q, want to mention AGENT_NAME", err.Error())
	}
}
