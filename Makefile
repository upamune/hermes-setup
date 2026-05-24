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

.PHONY: install-hooks
install-hooks: ## Install local Git hooks.
	git config core.hooksPath .githooks

.PHONY: lint
lint: ## Run static checks and shell syntax checks.
	go vet ./...
	bash -n scripts/install.sh scripts/ci/integration.sh
	$(MAKE) secret-scan

.PHONY: pinact
pinact: ## Verify GitHub Actions are pinned.
	go tool pinact run -check -verify .github/workflows/ci.yml

.PHONY: secret-scan
secret-scan: ## Scan git history and current files for secrets.
	go tool gitleaks git --no-banner --redact --log-level warn
	go tool gitleaks dir . --no-banner --redact --log-level warn

.PHONY: secret-scan-staged
secret-scan-staged: ## Scan staged changes for secrets.
	go tool gitleaks git --pre-commit --staged --no-banner --redact --log-level warn

.PHONY: test
test: ## Run Go tests.
	go test ./...
