#!/usr/bin/env bash
# Expose local services over the internet via ngrok free tier.
# Usage: expose.sh <service-port> <sample-port> [ngrok-config-path]
#
# Sets up two tunnels through a single ngrok agent (free tier workaround),
# then prints the public URLs.

set -euo pipefail

SERVICE_PORT="${1:?Usage: expose.sh <service-port> <sample-port> [ngrok-config-path]}"
SAMPLE_PORT="${2:?Usage: expose.sh <service-port> <sample-port> [ngrok-config-path]}"
NGROK_CONFIG="${3:-$HOME/Library/Application Support/ngrok/ngrok.yml}"

# Ensure ngrok is installed
if ! command -v ngrok &>/dev/null; then
  echo "ERROR: ngrok not found. Install: brew install ngrok" >&2
  exit 1
fi

# Read existing authtoken from config
if [[ -f "$NGROK_CONFIG" ]]; then
  AUTH_TOKEN=$(grep 'authtoken:' "$NGROK_CONFIG" | awk '{print $2}' | head -1)
else
  echo "ERROR: ngrok config not found at $NGROK_CONFIG" >&2
  echo "Run 'ngrok config add-authtoken <TOKEN>' first." >&2
  exit 1
fi

if [[ -z "${AUTH_TOKEN:-}" ]]; then
  echo "ERROR: no authtoken in $NGROK_CONFIG" >&2
  exit 1
fi

# Kill any existing ngrok processes
pkill -f ngrok 2>/dev/null || true
sleep 1

# Write config with both tunnels (free tier: one agent, multiple tunnels)
cat > "$NGROK_CONFIG" <<EOF
version: "3"
agent:
    authtoken: ${AUTH_TOKEN}
tunnels:
  service:
    addr: ${SERVICE_PORT}
    proto: http
  sample:
    addr: ${SAMPLE_PORT}
    proto: http
EOF

# Start ngrok with both tunnels
ngrok start --all --log=stdout &>/dev/null &
NGROK_PID=$!

# Wait for tunnels to come up
echo "Starting ngrok tunnels..."
for i in $(seq 1 15); do
  TUNNELS=$(curl -s http://localhost:4040/api/tunnels 2>/dev/null || echo "")
  COUNT=$(echo "$TUNNELS" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('tunnels',[])))" 2>/dev/null || echo "0")
  if [[ "$COUNT" -ge 2 ]]; then
    break
  fi
  sleep 1
done

if [[ "$COUNT" -lt 2 ]]; then
  echo "ERROR: ngrok tunnels failed to start" >&2
  kill $NGROK_PID 2>/dev/null || true
  exit 1
fi

# Extract URLs
SERVICE_URL=$(echo "$TUNNELS" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for t in d['tunnels']:
    if str(t['config']['addr']).endswith(':${SERVICE_PORT}') or t['config']['addr'] == 'http://localhost:${SERVICE_PORT}':
        print(t['public_url'])
        break
")

SAMPLE_URL=$(echo "$TUNNELS" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for t in d['tunnels']:
    if str(t['config']['addr']).endswith(':${SAMPLE_PORT}') or t['config']['addr'] == 'http://localhost:${SAMPLE_PORT}':
        print(t['public_url'])
        break
")

# Extract just the host from the service URL (strip https://)
SERVICE_HOST=$(echo "$SERVICE_URL" | sed 's|https://||')

echo ""
echo "=== ngrok tunnels active ==="
echo "Service:  $SERVICE_URL -> localhost:$SERVICE_PORT"
echo "Sample:   $SAMPLE_URL -> localhost:$SAMPLE_PORT"
echo ""
echo "=== Open this URL ==="
echo "${SAMPLE_URL}/?adapter=${SERVICE_HOST}"
echo ""
echo "ngrok PID: $NGROK_PID"
