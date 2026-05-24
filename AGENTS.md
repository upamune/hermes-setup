# Repository Guidelines

## Project Structure & Module Organization

This repository builds `hermes-setup`, a Go CLI for bootstrapping Hermes Agent on Ubuntu/Debian x86_64 hosts.

- `cmd/hermes-setup/`: CLI entrypoint and command wiring via Kong.
- `internal/setup/`: setup implementation for summon, Tailscale, Hermes, Bitwarden config, doctor checks, SSH mode, and idempotent file updates.
- `scripts/install.sh`: public `curl | sh` bootstrap entrypoint.
- `scripts/ci/integration.sh`: CI integration flow for build, install, doctor, and idempotency checks.
- `.github/workflows/ci.yml`: GitHub Actions workflow, pinned with `pinact`.

## Build, Test, and Development Commands

- `go test ./...`: run all Go tests.
- `go vet ./...`: run static checks; keep this clean before pushing.
- `go build ./cmd/hermes-setup`: compile the CLI.
- `go run ./cmd/hermes-setup --help`: inspect command and flag behavior locally.
- `bash -n scripts/install.sh scripts/ci/integration.sh`: syntax-check shell scripts.
- `GOTOOLCHAIN=go1.26.3+auto go run github.com/suzuki-shunsuke/pinact/v3/cmd/pinact@v3.10.1 run -check -verify .github/workflows/ci.yml`: verify GitHub Actions remain SHA-pinned.

## Coding Style & Naming Conventions

Use `gofmt` for all Go files. Keep CLI parsing in `cmd/hermes-setup`; keep operational logic in `internal/setup`. Prefer small, idempotent functions named after the operation they perform, such as `installTailscale`, `configureHermes`, or `disableSSHD`. Shell scripts should use `set -euo pipefail` for Bash or `set -eu` for POSIX sh.

## Testing Guidelines

Tests use the standard Go `testing` package and live next to implementation files as `*_test.go`. Focus tests on idempotency, config merging, env-file behavior, and safety guards. CI also runs an Ubuntu integration script that builds `./hermes-setup`, runs install and doctor twice, and diffs generated files.

## Security & Configuration Tips

Do not commit real secrets. `TELEGRAM_BOT_TOKEN` and `TELEGRAM_ALLOWED_USERS` belong in Bitwarden Secrets Manager. The only local secret written by setup is `BWS_ACCESS_TOKEN` in `~/.hermes/.env`. Be careful with flags that affect access, especially `--disable-sshd`, `--force-disable-sshd`, `--lock-down-incoming`, and `--force-lock-down-incoming`.

## Commit & Pull Request Guidelines

There is no established local commit history yet. Use concise imperative commit subjects, for example `Add Tailscale SSH setup` or `Harden Bitwarden config`. Pull requests should describe behavior changes, list verification commands, and call out any security-sensitive setup or migration impact.
