#!/usr/bin/env bash
set -e

GW="http://localhost:8080"

# Wait for gateway to be healthy
echo "Waiting for gateway..."
until curl -sf -X POST "$GW/register" -o /dev/null --max-time 1 2>/dev/null; do
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
      "email": "email",
      "username": "username",
      "password": "password"
    }
  }'

echo ""
echo "Setup complete."
