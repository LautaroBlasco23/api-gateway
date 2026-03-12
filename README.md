# API Gateway

A lightweight, modular API gateway written in Go. Backends register themselves at runtime, and the gateway handles proxying, rate limiting, caching, CORS, injection filtering, and request validation — all configured per service with feature flags.

## Quick Start

```bash
# Build
go build -o gateway ./cmd/server

# Run (default port: 8080)
PORT=8080 ./gateway

# Or with Docker
docker-compose up --build
```

The gateway starts empty. No routes are active until a backend registers itself.

---

## Registering a Service

Send a `POST /register` with your service's name, backend URL, routes, and desired features.

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user-service",
    "url": "http://localhost:3000",
    "routes": ["/api/users", "/api/profiles"],
    "features": {
      "ratelimiter": true,
      "injection": true,
      "cors": true,
      "cache": false
    }
  }'
```

**Response:**
```json
{ "status": "registered", "service": "user-service" }
```

After this, requests to `/api/users` and `/api/profiles` (and any parameterized variants you register) are proxied to `http://localhost:3000`.

### Registration fields

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Unique service identifier. Re-registering by name replaces the existing entry. |
| `url` | string | yes | Backend base URL (e.g. `http://localhost:3000`). |
| `routes` | string[] | yes | Route patterns to match. Use `{param}` for dynamic segments (e.g. `/api/users/{id}`). More specific patterns (more literal segments) take priority. |
| `features` | object | no | Feature flags — all default to `false`. |

---

## Feature Flags

Each service independently enables or disables features via the `features` object.

### `cors`

Adds CORS headers to every response and handles `OPTIONS` preflight requests automatically (returns `204 No Content`).

Headers added:
```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
```

### `ratelimiter`

Limits traffic per `service:client-IP` pair using a token bucket:

- **Rate:** 10 requests/second sustained
- **Burst:** up to 20 requests
- **Exceeded response:** `429 Too Many Requests`
- Limiter state resets every 5 minutes to prevent unbounded memory growth.

### `injection`

Scans the URL path, query string, all headers, and the request body for dangerous patterns before proxying. Returns `400 Bad Request` if a match is found.

Detected patterns: SQL injection (`union select`, `or 1=1`, `drop table`), XSS (`<script>`), command injection (`; rm`, `&& rm`), path traversal (`../`), and code execution (`eval(`, `exec(`).

### `cache`

Caches `GET` and `HEAD` responses for 30 seconds. Cache key is `METHOD:pattern?query`, where `pattern` is the matched route pattern (e.g. `/api/users/{id}`), not the literal path. This means `/api/users/123` and `/api/users/456` share the same cache entry. On a cache hit the backend is bypassed entirely.

---

## Request Pipeline

Every proxied request passes through these steps in order (each step only runs if the feature is enabled for that service):

```
CORS → Rate Limiter → Injection Filter → Endpoint Validation → Cache → Reverse Proxy
```

Any step can reject the request early with a `4xx` response. If all pass, the request is forwarded to your backend.

---

## Endpoint Validation

Optionally define validation rules for a specific route and method. The gateway rejects non-conforming requests before they reach your backend.

```bash
curl -X POST http://localhost:8080/register/endpoint \
  -H "Content-Type: application/json" \
  -d '{
    "route": "/api/users",
    "method": "POST",
    "validation": {
      "email": "email",
      "username": "username",
      "password": "password"
    }
  }'
```

**Response:**
```json
{ "status": "registered", "route": "/api/users" }
```

From now on, any `POST /api/users` that is missing a field or has the wrong type gets a `400 Bad Request` with an error message — without touching your backend.

Re-registering the same `route` + `method` pair replaces the existing rules.

### Supported validation types

| Type | Rule |
|---|---|
| `string` | Any string value |
| `integer` | JSON number or numeric string |
| `email` | Standard email format (`user@domain.com`) |
| `username` | 3–32 characters: letters, digits, `_`, `-` |
| `password` | Minimum 8 characters |
| `uuid` | Standard UUID format (`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`) |
| `file_png` | Multipart file with MIME type `image/png` |
| `file_jpg` | Multipart file with MIME type `image/jpeg` |
| `file_pdf` | Multipart file with MIME type `application/pdf` |

Mixed requests (JSON fields + file uploads in the same multipart body) are supported.

---

## Route Matching

Routes use **pattern matching**. Segments wrapped in `{braces}` match any non-empty path segment and capture its value (e.g. `/api/users/{id}` matches `/api/users/123`).

When multiple patterns from different services match the same request, the one with more literal (non-parameterized) segments wins. For example, `/api/users/admin` beats `/api/users/{id}` for a request to `/api/users/admin`.

---

## Full Example

### 1. Start the gateway

```bash
PORT=8080 ./gateway
```

### 2. Register your backend

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "products-service",
    "url": "http://localhost:4000",
    "routes": ["/api/products"],
    "features": {
      "ratelimiter": true,
      "injection": true,
      "cors": true,
      "cache": true
    }
  }'
```

### 3. Add endpoint validation

```bash
curl -X POST http://localhost:8080/register/endpoint \
  -H "Content-Type: application/json" \
  -d '{
    "route": "/api/products",
    "method": "POST",
    "validation": {
      "name": "string",
      "price": "integer"
    }
  }'
```

### 4. Make requests through the gateway

```bash
# Proxied to http://localhost:4000/api/products (cached for 30s)
curl http://localhost:8080/api/products

# Validated, rate-limited, injection-filtered, then proxied
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -d '{"name": "Widget", "price": 42}'
```

---

## Configuration

| Environment Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Port the gateway listens on |
| `REGISTRY_FILE` | `registry.json` | Path to the JSON file used for persisting registered services and endpoint rules |

---

## Important Notes

- **JSON persistence.** Registered services and endpoint rules are saved to `registry.json` (configurable via `REGISTRY_FILE`) on every write. The file is loaded on startup — backends do not need to re-register after a restart. Cached responses are still in-memory only.
- **Re-registering** a service by the same `name` replaces it in-place.
- **Pattern matching, not prefix.** `/api` only matches `/api` exactly. To handle `/api/users` you must register `/api/users` (or `/api/{resource}` for dynamic segments).

---

## Development

```bash
# Build
go build -o gateway ./cmd/server

# Lint (requires golangci-lint)
golangci-lint run
```

---

## Testing

This project contains **unit tests** for core logic (e.g. route pattern matching) as well as **integration and load tests**. Integration and load tests require the gateway to be running.

### Smoke Tests (Shell Script)

Quick functional validation of all major features via curl:

```bash
# Start gateway and test backend
docker-compose up --build

# Run smoke tests in another terminal
cd tests && bash smoke.sh
```

Tests cover:
- **Routing**: Request proxying to backend (200)
- **CORS**: Preflight handling (204)
- **Validation**: Valid and invalid payloads (200 vs 400)
- **Injection filtering**: Malicious patterns blocked (400)
- **Route registration**: Non-existent routes rejected (400)
- **Rate limiting**: Rapid requests trigger 429

### K6 Smoke Tests

Same functional checks using the k6 load testing tool with a single virtual user:

```bash
k6 run tests/k6/smoke.js
```

Benefits: structured assertions, cleaner syntax, easier to extend.

### K6 Load Tests

Performance and reliability testing under sustained load:

```bash
k6 run tests/k6/load.js
```

Characteristics:
- **Ramping scenario**: Gradually scales from 1 to 20 virtual users, sustains for 1 minute, then ramps down
- **Performance thresholds**:
  - 95% of requests must complete in <500ms
  - <5% error rate (accepts both 200 and 429 as valid)
- Detects performance degradation and ensures the gateway handles realistic traffic

### Setup Script

The `tests/setup.sh` script initializes the test environment by registering the test backend and configuring validation rules. It's called automatically by `docker-compose`.

---

## Project Structure

```
.
├── cmd/server/          # Entry point
├── internal/
│   ├── gateway/         # Request pipeline (handler, router, proxy)
│   ├── registry/        # Service registration and storage
│   ├── validation/      # Request body validation
│   └── features/        # Cache, rate limiter, CORS, injection filter
└── CLAUDE.md            # Developer notes
```
