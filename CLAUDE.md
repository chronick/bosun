# CLAUDE.md

## Project

**bosun** -- Agent entrypoint and lifecycle coordinator.

Go 1.22+, minimal dependencies. Calls external tools (br, hoist, agent-mail)
rather than embedding their logic.

## Commands

```bash
go build -o bosun .          # build
go test ./...                # test all
```

## Layout

```
main.go                      # cobra CLI entry point
internal/
  lifecycle/lifecycle.go     # main run loop
  claim/claim.go             # beads task claiming
  lease/lease.go             # agent-mail file lease client
  heartbeat/heartbeat.go     # periodic heartbeat sender
```

## Key Design

- Thin coordinator: sequences calls to br, hoist, agent-mail, $AGENT_COMMAND
- Does NOT implement agent logic, task tracking, or messaging
- Configuration via environment variables only
- Graceful shutdown: SIGTERM → release leases → update task → exit
- Idempotent operations: safe to restart at any point in the lifecycle
- Shell script spirit, Go binary for reliability

## Beads Workflow Integration

<!-- br-agent-instructions-v1 -->
See skiff CLAUDE.md for beads workflow details.
<!-- end-br-agent-instructions -->
