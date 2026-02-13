---
name: ngrok-stop
description: Stop ngrok tunnels and restore the ngrok config to its clean state (authtoken only, no tunnel definitions). Use when the user asks to stop ngrok, tear down tunnels, or clean up after testing.
---

# ngrok-stop

Kill ngrok and restore the config.

## Steps

1. Kill ngrok:
```bash
pkill -f ngrok
```

2. Restore the ngrok config to authtoken-only (remove tunnel definitions left by ngrok-start):
```bash
NGROK_CONFIG="$HOME/Library/Application Support/ngrok/ngrok.yml"
AUTH_TOKEN=$(grep 'authtoken:' "$NGROK_CONFIG" | awk '{print $2}' | head -1)
cat > "$NGROK_CONFIG" <<EOF
version: "3"
agent:
    authtoken: ${AUTH_TOKEN}
EOF
```

3. Confirm tunnels are gone:
```bash
curl -s http://localhost:4040/api/tunnels 2>/dev/null && echo "WARNING: ngrok still running" || echo "ngrok stopped"
```
