.PHONY: build run test lint docker-build up down setup smoke k6-smoke k6-load test-integration

# ── Local ────────────────────────────────────────────────────────────────────

build:
	go build -o gateway ./cmd/server

run: build
	PORT=8080 ./gateway

test:
	go test ./...

lint:
	golangci-lint run

# ── Docker ───────────────────────────────────────────────────────────────────

docker-build:
	docker build -t api-gateway .

up:
	docker-compose up --build -d

down:
	docker-compose down

# ── Integration ──────────────────────────────────────────────────────────────

setup:
	bash tests/setup.sh

smoke:
	bash tests/smoke.sh

k6-smoke:
	k6 run tests/k6/smoke.js

k6-load:
	k6 run tests/k6/load.js

# Spins up the stack, registers services, runs all checks, then tears down.
test-integration: up setup smoke k6-smoke
	@echo "All integration checks passed."
	$(MAKE) down
