#!/usr/bin/env bash
set -e

GW="http://localhost:8080"
MAX_RETRIES=60

# Wait for gateway HTTP server to be reachable (with timeout).
# We just care that it accepts connections and returns *any* response.
echo "Waiting for gateway..."
attempt=1
until curl -s "$GW/" -o /dev/null --max-time 1 2>/dev/null; do
  if [ "$attempt" -ge "$MAX_RETRIES" ]; then
    echo "Gateway did not become ready after ${MAX_RETRIES}s."
    echo "You can inspect logs with: docker-compose logs gateway"
    exit 1
  fi
  echo "  Gateway not ready yet (attempt ${attempt}/${MAX_RETRIES})..."
  attempt=$((attempt + 1))
  sleep 1
done

# Register echo backend service with all features enabled
curl -sf -X POST "$GW/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "echo",
    "url": "http://echo-backend:80",
    "routes": ["/echo"],
    "features": {
      "ratelimiter": true,
      "injection": true,
      "cors": true,
      "auth": false,
      "cache": true
    }
  }'

echo ""

# Register endpoint validation rule for POST /echo/users
curl -sf -X POST "$GW/register/endpoint" \
  -H "Content-Type: application/json" \
  -d '{
    "route": "/echo/users",
    "method": "POST",
    "validation": {
      "email": {"type": "email"},
      "username": {"type": "username"},
      "password": {"type": "password"}
    }
  }'

echo ""
echo "Setup complete."
