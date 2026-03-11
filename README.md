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
      "cache": false,
      "auth": false
    }
  }'
```

**Response:**
```json
{ "status": "registered", "service": "user-service" }
```

After this, requests to `/api/users/*` and `/api/profiles/*` are proxied to `http://localhost:3000`.

### Registration fields

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Unique service identifier. Re-registering by name replaces the existing entry. |
| `url` | string | yes | Backend base URL (e.g. `http://localhost:3000`). |
| `routes` | string[] | yes | Path prefixes to match. Longest prefix wins across all services. |
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

Caches `GET` and `HEAD` responses for 30 seconds. Cache key is `METHOD:path?query`. On a cache hit the backend is bypassed entirely.

### `auth`

> **Not implemented in V1.** The field is accepted and stored but no auth logic runs. Reserved for a future release.

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

Routes use **longest-prefix matching** across all registered services. A service registered at `/api/users` takes precedence over one at `/api` for any request starting with `/api/users`.

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

---

## Important Notes

- **No persistence.** All registered services and cached responses live in memory and are lost on restart. Backends must re-register every time the gateway starts.
- **Re-registering** a service by the same `name` replaces it in-place.
- **Auth is a placeholder.** The `auth` flag exists but does nothing in V1.
- **Prefix matching, not exact.** A service at `/api` will match `/api/users`, `/api/products`, etc.

---

## Development

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/validation/...

# Run a single test
go test ./internal/validation/... -run TestValidateEmail

# Lint (requires golangci-lint)
golangci-lint run
```

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
