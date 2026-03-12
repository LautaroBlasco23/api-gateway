# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Start the gateway (choose: local or Docker, with live logs)
make start

# Run all tests (shell smoke tests + k6 load tests)
make test

# Install required tools (k6, checks Docker)
make install-tools

# Build the binary
make build

# Lint the code (requires golangci-lint)
make lint

# View all available commands
make help
```

See `Makefile` for internal commands (`_docker-up`, `_docker-down`, etc.).

## Architecture

The gateway is a **modular monolith** with a single-pass request pipeline. All state is in-memory and lost on restart — there is no persistence layer.

### Request pipeline (`internal/gateway/handler.go`)

Every proxied request flows through this sequence, each step gated by the service's feature flags:

```
CORS → Rate Limiter → Injection Filter → Endpoint Validation → Cache → Reverse Proxy
```

The body is read once with `io.ReadAll` at the start of `handler.proxy()` and restored via `io.NopCloser(bytes.NewReader(...))` before each feature that may consume it (injection filter, validator, proxy).

### Service registry (`internal/registry/`)

Backends self-register at runtime via `POST /register`. The registry stores services and endpoint validation rules in slices protected by `sync.RWMutex`. Route resolution uses **parameterized pattern matching** — segments like `{id}` in a registered route match any non-empty literal segment in the request path. When multiple patterns match, the one with the most literal (non-parameterized) segments wins (`specificity` score). Re-registering a service by name replaces it in-place.

### Feature flags

Each `Service` carries a `Features` struct. Features are checked directly in `handler.proxy()` — there is no middleware chain. Adding a new feature means adding a field to `Features` and a guarded block in the handler.

### Validation (`internal/validation/`)

`Validate()` dispatches on field type: non-file types parse the body as JSON; `file_*` types use `mime/multipart`. After multipart parsing the body is consumed — the handler restores it from `bodyBytes` before proxying.

### Cache (`internal/features/cache/`)

`ResponseRecorder` wraps `http.ResponseWriter` to capture status, headers, and body while simultaneously writing to the real response. The recorded result is stored under the key `METHOD:pattern?query` (using the matched route pattern, not the literal path) with a 30-second TTL. This means `/api/users/123` and `/api/users/456` share the same cache entry when both match `/api/users/{id}`. Only GET and HEAD are cached.

### Rate limiter (`internal/features/ratelimiter/`)

One `rate.Limiter` (10 r/s, burst 20) per `"service:ip"` key. All limiters are wiped every 5 minutes to prevent unbounded map growth.

## Key Design Constraints

- **Persistent registry, in-memory cache**: service registrations and endpoint validation rules are saved to `registry.json` (via `REGISTRY_FILE` env var) on every write and loaded on startup. Cached responses are still in-memory only.
- **`auth` feature flag exists but is not implemented** in V1 — the field is parsed and stored but no auth logic runs.
- **File validation consumes the body**: `multipart.NewReader` reads from `r.Body`. The handler always restores `r.Body` from `bodyBytes` after validation.
- **Route matching is pattern-based**: segments in `{braces}` match any non-empty literal segment. `/api` does NOT match `/api/users` — routes must be registered explicitly (or with a `{param}` segment). `FindByRoute` and `FindEndpoint` both return a `RouteMatch` / params tuple.
