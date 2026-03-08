# CLAUDE.md

## Project

**bosun** -- Agent entrypoint and lifecycle coordinator.

Go 1.25+, zero external dependencies (stdlib only). Calls external tools
(br, hoist, agent-mail) rather than embedding their logic.

## Commands

```bash
go build -o bosun .          # build
go test ./...                # test all
```

## Layout

```
main.go                           # CLI entry point (dispatches to internal/cli)
internal/
  cli/cli.go                     # Command dispatcher (stdlib only, no cobra)
  cli/cli_test.go                # Dispatcher tests
  config/config.go               # Environment variable loading
  config/config_test.go          # Config tests
  lifecycle/lifecycle.go         # Main run loop with signal handling
  claim/claim.go                 # Beads task claiming and output parsing
  claim/claim_test.go            # Claim parsing tests
  complete/complete.go           # Post-task cleanup (close, PR, release)
  register/register.go           # Agent-mail registration
  lease/lease.go                 # Agent-mail file lease client
  heartbeat/heartbeat.go         # Periodic heartbeat sender
  toolexec/toolexec.go           # os/exec wrapper for external tools
```

## Key Design

- Zero external dependencies: stdlib only (no cobra, no third-party packages)
- Thin coordinator: sequences calls to br, hoist, agent-mail, $AGENT_COMMAND
- Does NOT implement agent logic, task tracking, or messaging
- Configuration via environment variables only
- Graceful shutdown: SIGTERM -> release leases -> update task -> exit
- Idempotent operations: safe to restart at any point in the lifecycle
- Shell script spirit, Go binary for reliability

## Beads Workflow Integration

<!-- br-agent-instructions-v1 -->
See skiff CLAUDE.md for beads workflow details.
<!-- end-br-agent-instructions -->
