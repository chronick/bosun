# bosun

Agent entrypoint and lifecycle coordinator for agentic coding swarms.

Bosun runs inside each agent container, sequencing the boot-claim-work-report cycle by calling other tools. It doesn't implement any agent logic itself.

## Install

```bash
# Copy to container image, or:
go install github.com/chronick/bosun@latest
```

## Usage

Bosun is typically the container entrypoint, configured via environment variables:

```bash
AGENT_NAME=coder-1 \
AGENT_COMMAND="claude-code --dangerously-skip-permissions" \
AGENT_MAIL_URL=http://agent-mail.skiff.local:8765 \
bosun run
```

### Commands

```bash
bosun run                    # full lifecycle loop (boot, claim, work, report)
bosun register               # announce identity to agent-mail
bosun claim                  # pick highest-priority ready task from beads
bosun lease <task-id>        # acquire file lease via agent-mail
bosun release <task-id>      # release file lease
bosun heartbeat              # ping agent-mail with status
bosun complete <task-id>     # close task, create PR, release lease
```

## Lifecycle

```
bosun run
  |
  +-- register with agent-mail
  |
  +-- loop:
  |     +-- br ready | claim highest priority task
  |     +-- acquire file lease
  |     +-- br update <task> --status=in_progress
  |     +-- run $AGENT_COMMAND /workspace
  |     +-- on success:
  |     |     +-- br close <task>
  |     |     +-- hoist pr agent/$AGENT_NAME
  |     +-- on failure:
  |     |     +-- br update <task> --status=open
  |     +-- release file lease
  |     +-- hoist reset agent/$AGENT_NAME
  |
  +-- on SIGTERM: release leases, update task status, exit
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `AGENT_NAME` | yes | Agent identity (e.g., `coder-1`) |
| `AGENT_COMMAND` | yes | Agent runtime to invoke |
| `AGENT_MAIL_URL` | no | Agent mail server URL |
| `BEADS_DB` | no | Path to beads database |
| `TASK_FILTER` | no | Beads query filter |
| `HEARTBEAT_INTERVAL` | no | Seconds between heartbeats (default: 30) |
| `IDLE_SLEEP` | no | Seconds to wait when no tasks (default: 30) |

## Dependencies

Bosun calls these tools -- they must be available in the container:

- `br` (beads_rust) -- task tracking
- `hoist` -- git worktree management
- `agent-mail` CLI or HTTP API -- messaging and file leases
- Agent runtime (`$AGENT_COMMAND`) -- the actual coding agent

## Part of the Agentic Coding Stack

| Tool | Role |
|------|------|
| **skiff** | Container orchestration (lifecycle, health, DNS) |
| **hoist** | Git worktree management |
| **bosun** | Agent entrypoint/coordinator |
| **beads** | Task tracking and priority |
| **agent-mail** | Inter-agent messaging and file leases |
