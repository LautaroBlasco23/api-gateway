.PHONY: help build run test lint docker-build up down setup smoke k6-smoke k6-load test-integration

help:
	@echo "Available commands:"
	@echo "  make build            Build the gateway binary"
	@echo "  make run              Build and run the gateway on PORT=8080"
	@echo "  make test             Run all Go tests"
	@echo "  make lint             Run golangci-lint"
	@echo "  make docker-build     Build the Docker image"
	@echo "  make up               Start the stack with docker-compose"
	@echo "  make down             Stop the stack with docker-compose"
	@echo "  make setup            Run test setup script"
	@echo "  make smoke            Run smoke tests"
	@echo "  make k6-smoke         Run k6 smoke tests"
	@echo "  make k6-load          Run k6 load tests"
	@echo "  make test-integration Full integration test flow (up → setup → smoke → k6-smoke → down)"

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
