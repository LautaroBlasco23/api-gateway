#!/usr/bin/env bash
set -e

GW="http://localhost:8080"
PASS=0
FAIL=0

assert_status() {
  local description="$1"
  local expected="$2"
  local actual="$3"

  if [ "$actual" = "$expected" ]; then
    echo "  PASS: $description (got $actual)"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $description (expected $expected, got $actual)"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== Smoke Tests ==="

# 1. GET /echo proxied to backend
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$GW/echo")
assert_status "GET /echo proxied to backend" "200" "$STATUS"

# 2. OPTIONS /echo CORS preflight
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X OPTIONS \
  -H "Origin: http://example.com" \
  -H "Access-Control-Request-Method: GET" \
  "$GW/echo")
assert_status "OPTIONS /echo CORS preflight" "204" "$STATUS"

# 3. POST /echo/users with valid JSON body
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/echo/users" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","username":"user1","password":"pass1234"}')
assert_status "POST /echo/users with valid JSON body" "200" "$STATUS"

# 4. POST /echo/users with invalid email
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/echo/users" \
  -H "Content-Type: application/json" \
  -d '{"email":"not-an-email","username":"user1","password":"pass1234"}')
assert_status "POST /echo/users with invalid email" "400" "$STATUS"

# 5. POST /echo/users with missing password
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GW/echo/users" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","username":"user1"}')
assert_status "POST /echo/users with missing password" "400" "$STATUS"

# 6. GET /echo with <script> in query string (injection)
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  "$GW/echo?q=%3Cscript%3Ealert%281%29%3C%2Fscript%3E")
assert_status "GET /echo with <script> in query string" "400" "$STATUS"

# 7. GET /nonexistent (no registered route)
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$GW/nonexistent")
assert_status "GET /nonexistent (no registered route)" "400" "$STATUS"

# 8. Rate limit — fire 30 rapid requests, expect at least one 429
echo ""
echo "  Testing rate limiter (30 rapid requests)..."
GOT_429=false
for i in $(seq 1 30); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$GW/echo")
  if [ "$STATUS" = "429" ]; then
    GOT_429=true
    break
  fi
done
if [ "$GOT_429" = "true" ]; then
  echo "  PASS: Rate limiter triggered 429 under load"
  PASS=$((PASS + 1))
else
  echo "  FAIL: Rate limiter did not trigger 429 (may need more requests or lower burst)"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
