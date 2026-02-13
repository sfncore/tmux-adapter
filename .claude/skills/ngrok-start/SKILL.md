---
name: ngrok-start
description: Expose two local services (e.g., an API server and a sample/UI) over the internet via ngrok free tier using a single agent with multiple tunnels. Use when the user asks to expose local services via ngrok, test over the internet, share a local server publicly, or set up ngrok tunnels. Handles the free tier limitation of one agent session by configuring both tunnels in the ngrok config file.
---

# ngrok-start

Expose a service + sample (or any two local ports) over the internet via ngrok free tier.

## Key Constraint: Free Tier

Free ngrok allows only **one agent session**. To expose two ports, define both tunnels in the ngrok config and start them with `ngrok start --all`.

## Workflow

### 1. Prerequisites

- ngrok installed (`brew install ngrok`)
- ngrok authtoken configured (`ngrok config add-authtoken <TOKEN>`)
- Both local services already running

### 2. Run the expose script

```bash
bash .claude/skills/ngrok-start/scripts/expose.sh <service-port> <sample-port>
```

Example for tmux-adapter:
```bash
bash .claude/skills/ngrok-start/scripts/expose.sh 8080 8000
```

The script:
- Reads the existing authtoken from `~/Library/Application Support/ngrok/ngrok.yml`
- Kills any existing ngrok processes
- Writes a config with both tunnels
- Starts `ngrok start --all`
- Waits for tunnels to come up
- Queries the ngrok API at `localhost:4040/api/tunnels` for public URLs
- Prints the combined URL with `?adapter=` query param

### 3. If the service needs CORS for the sample's origin

Restart the service with the ngrok domain allowed:

```bash
# Example: tmux-adapter
./tmux-adapter --gt-dir ~/gt --port 8080 --allowed-origins "localhost:*,*.ngrok-free.app"
```

### 4. Share the URL

The script prints a ready-to-use URL like:
```
https://abc123.ngrok-free.app/?adapter=def456.ngrok-free.app
```

The `?adapter=` parameter tells the sample both where to connect the WebSocket and where to import the web component — one param controls everything.

### 5. ngrok interstitial

Free tier shows a "Visit Site" interstitial on first load. Users click through it once per tunnel.

## Troubleshooting

- **ERR_NGROK_108** (multiple agents): Kill all ngrok processes and use `ngrok start --all` with a config file instead of multiple `ngrok http` commands.
- **Tunnels not appearing**: Query `curl -s http://localhost:4040/api/tunnels` to check status.
- **WebSocket rejected**: Ensure the service's `--allowed-origins` includes `*.ngrok-free.app`.
