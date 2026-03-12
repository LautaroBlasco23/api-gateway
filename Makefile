.PHONY: help start test install-tools build lint _docker-up _docker-down _run-local _run-tests _check-tools

help:
	@echo "Available commands:"
	@echo ""
	@echo "  make start             Start the API gateway (choose: local or Docker)"
	@echo "  make test              Run all tests (smoke + load tests)"
	@echo "  make install-tools     Install required tools (k6, etc.)"
	@echo "  make build             Build the gateway binary"
	@echo "  make lint              Run golangci-lint"
	@echo ""

# ────────────────────────────────────────────────────────────────────────────
# MAIN COMMANDS
# ────────────────────────────────────────────────────────────────────────────

start:
	@bash -c '\
		$(MAKE) _check-tools; \
		echo ""; \
		echo "How would you like to start the gateway?"; \
		echo "  1) Local (on this machine)"; \
		echo "  2) Docker (with docker-compose)"; \
		read -p "Enter your choice (1 or 2): " choice; \
		if [ "$$choice" = "1" ]; then \
			$(MAKE) _run-local; \
		elif [ "$$choice" = "2" ]; then \
			$(MAKE) _docker-up; \
			echo ""; \
			echo "Gateway is running in Docker. Tailing logs..."; \
			echo "Press Ctrl+C to stop."; \
			echo ""; \
			docker-compose logs -f gateway; \
		else \
			echo "Invalid choice. Please run make start again."; \
			exit 1; \
		fi \
	'

test:
	@bash -c '$(MAKE) _check-tools && $(MAKE) _run-tests'

install-tools:
	@bash -c '\
		set -e; \
		echo "Checking required tools..."; \
		echo ""; \
		if ! command -v docker >/dev/null 2>&1; then \
			echo "✗ Docker is not installed"; \
			exit 1; \
		fi; \
		echo "✓ Docker is installed"; \
		echo ""; \
		if command -v k6 >/dev/null 2>&1; then \
			echo "✓ k6 is already installed"; \
		else \
			echo "✗ k6 is not installed"; \
			read -p "Do you want to install k6? (y/n): " install_k6; \
			if [ "$$install_k6" = "y" ] || [ "$$install_k6" = "Y" ]; then \
				$(MAKE) _install-k6; \
			fi; \
		fi; \
		echo ""; \
		echo "Tool check complete."; \
	'

build:
	go build -o gateway ./cmd/server

lint:
	golangci-lint run

# ────────────────────────────────────────────────────────────────────────────
# INTERNAL COMMANDS (not meant to be called directly)
# ────────────────────────────────────────────────────────────────────────────

_check-tools:
	@bash -c '\
		if ! command -v docker >/dev/null 2>&1; then \
			echo "✗ Docker is not installed"; \
			read -p "Do you want to install tools now? (y/n): " install; \
			if [ "$$install" = "y" ] || [ "$$install" = "Y" ]; then \
				$(MAKE) install-tools; \
			else \
				echo "Docker is required to run this command."; \
				exit 1; \
			fi; \
		fi; \
		if ! command -v k6 >/dev/null 2>&1; then \
			echo "⚠ k6 is not installed (required for load tests)"; \
			read -p "Do you want to install k6? (y/n): " install_k6; \
			if [ "$$install_k6" = "y" ] || [ "$$install_k6" = "Y" ]; then \
				$(MAKE) _install-k6; \
			fi; \
		fi \
	'

_install-k6:
	@bash -c '\
		set -e; \
		echo "Installing k6..."; \
		ARCH=$$(uname -m); \
		OS=$$(uname -s | tr A-Z a-z); \
		case $$ARCH in \
			x86_64) ARCH="amd64" ;; \
			aarch64) ARCH="arm64" ;; \
		esac; \
		if [ "$$OS" = "darwin" ]; then \
			echo "macOS detected"; \
			brew install k6 || { echo "Failed to install k6"; exit 1; }; \
		else \
			echo "Linux detected"; \
			K6_VERSION=$$(curl -s https://api.github.com/repos/grafana/k6/releases/latest | grep "tag_name" | cut -d"\"" -f4 | cut -d"v" -f2); \
			K6_URL="https://github.com/grafana/k6/releases/download/v$${K6_VERSION}/k6-v$${K6_VERSION}-linux-$${ARCH}.tar.gz"; \
			echo "Downloading k6 from $$K6_URL"; \
			mkdir -p /tmp/k6_install; \
			curl -L "$$K6_URL" | tar xz -C /tmp/k6_install; \
			if sudo -n true 2>/dev/null; then \
				sudo mv /tmp/k6_install/k6 /usr/local/bin/k6; \
				sudo chmod +x /usr/local/bin/k6; \
				rm -rf /tmp/k6_install; \
			else \
				echo ""; \
				echo "✗ Unable to write to /usr/local/bin without password"; \
				echo ""; \
				echo "Options:"; \
				echo "  1) Configure sudo to work without password prompt"; \
				echo "  2) Manually move the binary: sudo mv /tmp/k6_install/k6 /usr/local/bin/k6"; \
				echo "  3) Add $$HOME/.local/bin to your PATH and use:"; \
				echo "     mkdir -p $$HOME/.local/bin && mv /tmp/k6_install/k6 $$HOME/.local/bin/"; \
				exit 1; \
			fi; \
		fi; \
		if command -v k6 >/dev/null 2>&1; then \
			k6 version; \
			echo "✓ k6 installed successfully"; \
		else \
			echo "⚠ k6 not found in PATH. Please check your installation."; \
		fi \
	'

_run-local: build
	@echo "Starting gateway locally..."
	@echo "Listening on http://localhost:8080"
	@echo "Press Ctrl+C to stop."
	@echo ""
	@PORT=8080 ./gateway

_docker-up:
	@echo "Starting gateway with Docker..."
	@docker-compose up --build -d

_run-tests: _docker-up _setup-tests _smoke-tests _k6-smoke-tests _docker-down
	@echo ""
	@echo "✓ All tests passed!"

_setup-tests:
	@bash tests/setup.sh

_smoke-tests:
	@echo ""
	@echo "=== Running Shell Smoke Tests ==="
	@bash tests/smoke.sh

_k6-smoke-tests:
	@bash -c '\
		if command -v k6 >/dev/null 2>&1; then \
			echo ""; \
			echo "=== Running K6 Smoke Tests ==="; \
			k6 run tests/k6/smoke.js; \
		else \
			echo ""; \
			echo "⚠ Skipping K6 tests (k6 not installed)"; \
			echo "  Run: make install-tools to install k6"; \
		fi \
	'

_docker-down:
	@docker-compose down >/dev/null 2>&1
