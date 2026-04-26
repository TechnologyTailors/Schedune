.PHONY: help build build-agent build-control-plane test test-agent test-control-plane fmt lint doctor dev-preflight dev-up dev-down demo dev-db-reset example-intake example-schedule example-launch-validate example-launch-dry-run example-launch-execute example-readiness example-orphans

# Configuration
BIN_DIR=bin
AGENT_BIN=$(BIN_DIR)/schedune-agent
CONTROL_PLANE_BIN=$(BIN_DIR)/schedune
SQLITE_DB=var/schedune.db

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: build-agent build-control-plane ## Build all binaries

build-agent: ## Build the Rust agent
	@echo "Building Schedune Agent..."
	@mkdir -p $(BIN_DIR)
	@cd schedune-agent && . $$HOME/.cargo/env && cargo build --release
	@cp schedune-agent/target/release/schedune-agent $(AGENT_BIN)

build-control-plane: ## Build the Go control plane
	@echo "Building Schedune Control Plane..."
	@mkdir -p $(BIN_DIR)
	@cd schedune-control-plane && export CGO_ENABLED=1 && export PATH=$$PATH:$$HOME/.local/go/bin && go build -o ../$(CONTROL_PLANE_BIN) ./cmd/schedune

test: test-agent test-control-plane ## Run all tests

test-agent: ## Run Rust tests
	@cd schedune-agent && . $$HOME/.cargo/env && cargo test

test-control-plane: ## Run Go tests
	@cd schedune-control-plane && export CGO_ENABLED=1 && export PATH=$$PATH:$$HOME/.local/go/bin && go test ./... -v

fmt: ## Format all code
	@cd schedune-agent && . $$HOME/.cargo/env && cargo fmt
	@cd schedune-control-plane && export PATH=$$PATH:$$HOME/.local/go/bin && gofmt -w .

lint: ## Lint all code
	@cd schedune-agent && . $$HOME/.cargo/env && cargo clippy -- -D warnings
	@cd schedune-control-plane && export CGO_ENABLED=1 && export PATH=$$PATH:$$HOME/.local/go/bin && go vet ./...

doctor: build-control-plane ## Run preflight checks to verify local readiness for evaluation
	@./$(CONTROL_PLANE_BIN) doctor

dev-preflight: doctor ## Alias for doctor

dev-db-reset: ## Reset the local SQLite database
	@echo "Resetting local database..."
	@rm -f $(SQLITE_DB)
	@echo "Done."

dev-up: build ## Start the control plane in the background (port 9090)
	@echo "Starting Schedune Control Plane..."
	@./$(CONTROL_PLANE_BIN) server & echo $$! > .schedune.pid
	@echo "Control plane running on http://127.0.0.1:9090"
	@echo "To stop, run 'make dev-down'"

dev-down: ## Stop the background control plane
	@if [ -f .schedune.pid ]; then kill $$(cat .schedune.pid) && rm .schedune.pid && echo "Control plane stopped."; else echo "Not running."; fi

smoke-test: build ## Run headless E2E automated API smoke test
	@bash scripts/smoke-test.sh

demo: doctor build dev-db-reset ## Run the automated end-to-end evaluator demo
	@bash scripts/demo.sh

demo-fixture: build-control-plane dev-db-reset ## Run the fixture-backed evaluator demo (macOS/non-Linux friendly)
	@bash scripts/demo-fixture.sh

demo-fixture-once: build-control-plane dev-db-reset ## Run the fixture-backed evaluator demo and exit cleanly
	@bash scripts/demo-fixture.sh --once

example-nodes: ## List normalized node summaries
	@curl -s http://127.0.0.1:9090/api/v1alpha1/nodes

example-node-explain: ## Explain scheduling eligibility for a specific node (Requires NODE_ID)
	@if [ -z "$(NODE_ID)" ]; then \
		echo "Usage: make example-node-explain NODE_ID=<id>"; \
	else \
		curl -s http://127.0.0.1:9090/api/v1alpha1/nodes/$(NODE_ID)/explain; \
	fi

example-intake: ## Ingest the local node capability payload
	@bash examples/curls/intake.sh

example-schedule: ## Run a workload eligibility evaluation explanation
	@bash examples/curls/schedule-explain.sh

example-launch-validate: ## Validate a launch without executing it
	@bash examples/curls/launch-validate.sh

example-launch-dry-run: ## Dry-run a launch without executing it
	@bash examples/curls/launch-dry-run.sh

example-launch-execute: ## Execute a launch on a local runtime
	@bash examples/curls/launch-execute.sh

example-readiness: ## Check readiness for an execution (Requires EXECUTION_ID)
	@if [ -z "$(EXECUTION_ID)" ]; then \
		echo "Usage: make example-readiness EXECUTION_ID=<id>"; \
		echo "Run 'make example-launch-execute' first to get an execution ID."; \
	else \
		curl -s http://127.0.0.1:9090/api/v1alpha1/launch/$(EXECUTION_ID)/readiness; \
		echo ""; \
	fi

example-orphans: ## List orphaned runtimes on the node
	@curl -s http://127.0.0.1:9090/api/v1alpha1/recovery/orphans; echo ""
