package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear env vars.
	t.Setenv("AGENT_NAME", "")
	t.Setenv("AGENT_COMMAND", "")
	t.Setenv("AGENT_MAIL_URL", "")
	t.Setenv("BEADS_DB", "")
	t.Setenv("TASK_FILTER", "")
	t.Setenv("HEARTBEAT_INTERVAL", "")
	t.Setenv("IDLE_SLEEP", "")

	cfg := Load()

	if cfg.AgentName != "" {
		t.Errorf("AgentName = %q, want empty", cfg.AgentName)
	}
	if cfg.HeartbeatInterval != 30*time.Second {
		t.Errorf("HeartbeatInterval = %v, want 30s", cfg.HeartbeatInterval)
	}
	if cfg.IdleSleep != 30*time.Second {
		t.Errorf("IdleSleep = %v, want 30s", cfg.IdleSleep)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("AGENT_NAME", "test-agent")
	t.Setenv("AGENT_COMMAND", "echo hello")
	t.Setenv("AGENT_MAIL_URL", "http://localhost:8765")
	t.Setenv("BEADS_DB", "/tmp/beads.db")
	t.Setenv("TASK_FILTER", "repo:skiff")
	t.Setenv("HEARTBEAT_INTERVAL", "60")
	t.Setenv("IDLE_SLEEP", "10")

	cfg := Load()

	if cfg.AgentName != "test-agent" {
		t.Errorf("AgentName = %q, want %q", cfg.AgentName, "test-agent")
	}
	if cfg.AgentCommand != "echo hello" {
		t.Errorf("AgentCommand = %q, want %q", cfg.AgentCommand, "echo hello")
	}
	if cfg.AgentMailURL != "http://localhost:8765" {
		t.Errorf("AgentMailURL = %q, want %q", cfg.AgentMailURL, "http://localhost:8765")
	}
	if cfg.BeadsDB != "/tmp/beads.db" {
		t.Errorf("BeadsDB = %q, want %q", cfg.BeadsDB, "/tmp/beads.db")
	}
	if cfg.TaskFilter != "repo:skiff" {
		t.Errorf("TaskFilter = %q, want %q", cfg.TaskFilter, "repo:skiff")
	}
	if cfg.HeartbeatInterval != 60*time.Second {
		t.Errorf("HeartbeatInterval = %v, want 60s", cfg.HeartbeatInterval)
	}
	if cfg.IdleSleep != 10*time.Second {
		t.Errorf("IdleSleep = %v, want 10s", cfg.IdleSleep)
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	t.Setenv("HEARTBEAT_INTERVAL", "not-a-number")
	t.Setenv("IDLE_SLEEP", "abc")

	cfg := Load()

	// Should fall back to defaults.
	if cfg.HeartbeatInterval != 30*time.Second {
		t.Errorf("HeartbeatInterval = %v, want 30s (default)", cfg.HeartbeatInterval)
	}
	if cfg.IdleSleep != 30*time.Second {
		t.Errorf("IdleSleep = %v, want 30s (default)", cfg.IdleSleep)
	}
}
