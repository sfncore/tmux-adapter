# tmux-adapter

A WebSocket service that exposes [gastown](https://github.com/steveyegge/gastown) agents as a programmatic interface. Clients interact with agents — tmux is an internal implementation detail.

## Quick Start

```bash
go build -o tmux-adapter .
gt start
./tmux-adapter --gt-dir ~/gt --port 8080
```

Connect with any WebSocket client:

```bash
websocat ws://localhost:8080/ws
```

## API

All communication is JSON over a single WebSocket connection at `/ws`. Requests include an `id` for correlation; responses echo it back. Events are pushed without an `id`.

### List Agents

```json
→ {"id":"1", "type":"list-agents"}
← {"id":"1", "type":"list-agents", "agents":[
    {"name":"hq-mayor", "role":"mayor", "runtime":"claude", "rig":null, "workDir":"/Users/me/gt", "attached":false},
    {"name":"gt-myrig-crew-bob", "role":"crew", "runtime":"claude", "rig":"myrig", "workDir":"/Users/me/gt/myrig/crew/bob", "attached":false}
  ]}
```

### Send a Prompt

```json
→ {"id":"2", "type":"send-prompt", "agent":"hq-mayor", "prompt":"please review the PR"}
← {"id":"2", "type":"send-prompt", "ok":true}
```

The adapter handles the full NudgeSession delivery sequence internally (literal mode, 500ms debounce, Escape, Enter with retry, SIGWINCH wake for detached sessions).

### Subscribe to Agent Output

Get full scrollback history and optionally stream new output in one atomic call:

```json
→ {"id":"3", "type":"subscribe-output", "agent":"hq-mayor"}
← {"id":"3", "type":"subscribe-output", "ok":true, "history":"... full scrollback ..."}
← {"type":"output", "name":"hq-mayor", "data":"new output bytes"}
← {"type":"output", "name":"hq-mayor", "data":"more bytes..."}
```

History-only (no streaming):

```json
→ {"id":"4", "type":"subscribe-output", "agent":"hq-mayor", "stream":false}
← {"id":"4", "type":"subscribe-output", "ok":true, "history":"..."}
```

Unsubscribe:

```json
→ {"id":"5", "type":"unsubscribe-output", "agent":"hq-mayor"}
← {"id":"5", "type":"unsubscribe-output", "ok":true}
```

### Subscribe to Agent Lifecycle

```json
→ {"id":"6", "type":"subscribe-agents"}
← {"id":"6", "type":"subscribe-agents", "ok":true, "agents":[...]}
← {"type":"agent-added", "agent":{...}}
← {"type":"agent-removed", "name":"gt-myrig-SomeTask"}
← {"type":"agent-updated", "agent":{...}}
```

`agent-updated` fires when a human attaches to or detaches from a session. Hot-reloads (same session, process restarts) emit `agent-removed` then `agent-added` in quick succession.

Unsubscribe:

```json
→ {"id":"7", "type":"unsubscribe-agents"}
← {"id":"7", "type":"unsubscribe-agents", "ok":true}
```

## Agent Model

```json
{
  "name": "hq-mayor",
  "role": "mayor",
  "runtime": "claude",
  "rig": null,
  "workDir": "/Users/me/gt",
  "attached": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Session identifier (`hq-mayor`, `gt-myrig-crew-bob`) |
| `role` | string | `mayor`, `deacon`, `overseer`, `witness`, `refinery`, `crew`, `polecat`, `boot` |
| `runtime` | string | `claude`, `gemini`, `codex`, `cursor`, `auggie`, `amp`, `opencode` |
| `rig` | string? | Rig name for rig-level agents, `null` for town-level |
| `workDir` | string | Agent's working directory |
| `attached` | bool | Whether a human is viewing the session |

Only agents with a live process are exposed — zombie sessions are filtered out.

## Architecture

```
Clients ◄──ws──► tmux-adapter ◄──control mode──► tmux server
                      │
                      └──pipe-pane (per agent)──► output files
```

- **Control mode**: one `tmux -C` connection handles all commands and receives `%sessions-changed` events for lifecycle tracking
- **Agent detection**: reads `GT_ROLE`/`GT_RIG` env vars, checks `pane_current_command` against known runtimes, walks process descendants for shell-wrapped agents, handles version-as-argv[0] (e.g., Claude showing `2.1.38`)
- **Output streaming**: `pipe-pane -o` activated per-agent on first subscriber, deactivated on last unsubscribe
- **Send prompt**: full NudgeSession sequence with per-agent mutex to prevent interleaving

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--gt-dir` | `~/gt` | Gastown town directory |
| `--port` | `8080` | WebSocket server port |
