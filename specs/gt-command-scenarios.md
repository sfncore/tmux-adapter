# GT Command → Tmux Activity Map

Which `gt` commands cause tmux activity, and what the adapter would see for each. Organized by what the adapter observes.

---

## Agent Appears (adapter sees: new session + live agent process)

| Command | What it does |
|---------|-------------|
| `gt start` | Creates Mayor and Deacon sessions, starts Claude in each. Also cleans up zombie sessions first. |
| `gt start crew <name>` | Creates a crew worker session, starts agent. |
| `gt crew add <name>` | Creates a new crew workspace + session, starts agent. |
| `gt polecat spawn` / `gt sling` | Creates a polecat session, starts agent with work prompt. |
| `gt deacon` (internal restart) | Recreates Deacon session after crash/recycle. |
| `gt refinery` (internal) | Creates Refinery session for merge processing. |
| `gt witness` (internal) | Creates Witness session for rig monitoring. |

---

## Agent Disappears (adapter sees: session destroyed or agent process dies)

| Command | What it does |
|---------|-------------|
| `gt shutdown` | Kills all gastown sessions in order (workers → monitors → infrastructure). |
| `gt down` | Pauses gastown — kills sessions but preserves crew workspaces for restart. |
| `gt done` | Polecat submits work and exits its own session. |
| `gt polecat nuke` | Destroys polecat entirely (session + worktree). |
| `gt crew lifecycle` (internal) | Recycles a crashed crew member — kills old session, creates new one. |
| `gt witness` (recycling) | Detects dead polecats and kills their zombie sessions. |

---

## Text Sent to Agent (adapter sees: output changes after send-keys)

| Command | What it does |
|---------|-------------|
| `gt nudge <target> <message>` | Sends a message to any agent via NudgeSession. |
| `gt broadcast <message>` | Sends a message to multiple agents. |
| `gt sling <target>` | Sends initial work prompt to a newly spawned polecat. |
| `gt swarm` | Sends commands to multiple polecats in parallel. |
| `gt captain` / `gt dog` | Sends health check / heartbeat messages. |
| `gt deacon` (health checks) | Sends periodic health checks to Mayor. |

---

## Agent Hot-Reloads (adapter sees: agent process dies briefly, new one appears in same session)

| Command | What it does |
|---------|-------------|
| `gt handoff` | Kills agent process, respawns pane with new command. Session stays alive. |
| `gt molecule step` | Kills current process, respawns with next step command. |
| `gt escalate` | Restarts agent with escalation command in same pane. |
| Auto-respawn hook | Deacon auto-restarts after crash (3s delay, then respawn-pane). |

---

## Output Captured (adapter sees: same thing clients would see via subscribe-output)

| Command | What it does |
|---------|-------------|
| `gt peek` | Captures last N lines or full scrollback from any session. |
| `gt captain` / `gt dog` / `gt warrant` | Captures output for health analysis. |
| `gt ready` | Polls output looking for prompt indicator during startup. |

---

## Session Attached/Detached (adapter sees: attached status change)

| Command | What it does |
|---------|-------------|
| `gt crew at <name>` | Attaches to a crew member's session (human starts watching). |
| `gt mayor attach` | Attaches to Mayor session. |
| User detaches (`C-b d`) | Detaches from session (human stops watching). |

---

## Session Configured (adapter sees: theme/status changes, but these are cosmetic)

| Command | What it does |
|---------|-------------|
| `gt start` / all agent startups | Applies full gastown config: theme, status bar, keybindings, hooks, mouse mode, environment variables. |
| `gt theme <name>` | Changes session color theme. |

---

## Testing Checklist

To exercise the adapter through its full range, run these commands and verify the adapter reports the expected activity:

### Basic lifecycle
1. `gt start` → adapter sees Mayor + Deacon appear
2. `gt status` → (no tmux changes, but useful to confirm what's running)
3. `gt shutdown` → adapter sees all agents disappear

### Crew
4. `gt crew add max` → adapter sees new crew agent appear
5. `gt crew at max` → adapter sees attached status change
6. (detach with `C-b d`) → adapter sees detached status change
7. `gt nudge gt-<rig>-crew-max "hello"` → adapter sees output change in crew session

### Polecats
8. `gt sling <target>` → adapter sees new polecat appear with output (work prompt)
9. `gt done` (from within polecat) → adapter sees polecat disappear

### Messages
10. `gt nudge hq-mayor "review this"` → adapter sees Mayor output change
11. `gt broadcast "status update"` → adapter sees output change in multiple sessions

### Hot reload
12. `gt handoff` (from within session) → adapter sees brief agent death + new agent in same session

### Output streaming
13. Subscribe to Mayor output, then `gt nudge hq-mayor "do something"` → adapter streams the agent's response in real time
