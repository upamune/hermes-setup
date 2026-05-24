.DEFAULT_GOAL := help

BIN := hermes-setup
GO_PACKAGE := ./cmd/hermes-setup

.PHONY: help
help: ## Show available targets.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the hermes-setup binary.
	go build -o $(BIN) $(GO_PACKAGE)

.PHONY: clean
clean: ## Remove build artifacts.
	rm -f $(BIN)
	rm -rf dist/

.PHONY: format
format: ## Format Go files.
	gofmt -w cmd internal

.PHONY: install
install: ## Install the CLI into GOPATH/bin or GOBIN.
	go install $(GO_PACKAGE)

.PHONY: lint
lint: ## Run static checks and shell syntax checks.
	go vet ./...
	bash -n scripts/install.sh scripts/ci/integration.sh

.PHONY: test
test: ## Run Go tests.
	go test ./...
