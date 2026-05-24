#!/bin/sh
set -eu

repo="${HERMES_SETUP_REPO:-github.com/upamune/hermes-setup}"
ref="${HERMES_SETUP_REF:-latest}"

if [ "$(uname -s)" != "Linux" ]; then
  echo "hermes-setup: Linux is required" >&2
  exit 1
fi

case "$(uname -m)" in
  x86_64|amd64) ;;
  *)
    echo "hermes-setup: x86_64/amd64 is required" >&2
    exit 1
    ;;
esac

if [ "${HERMES_SETUP_SKIP_SUMMON:-0}" != "1" ]; then
  curl -fsSL https://raw.githubusercontent.com/upamune/summon/main/summon.sh | sh
fi

export PATH="$HOME/.local/bin:$HOME/.bun/bin:$PATH"

if command -v mise >/dev/null 2>&1; then
  exec mise exec -- go run "$repo/cmd/hermes-setup@$ref" install --skip-summon "$@"
fi

if command -v go >/dev/null 2>&1; then
  exec go run "$repo/cmd/hermes-setup@$ref" install --skip-summon "$@"
fi

echo "hermes-setup: go was not found after summon" >&2
exit 1
