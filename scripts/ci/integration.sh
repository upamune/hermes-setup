#!/usr/bin/env bash
set -euo pipefail

export PATH="$HOME/.local/bin:$HOME/.bun/bin:$PATH"

export HERMES_HOME="${HERMES_HOME:-${RUNNER_TEMP:-/tmp}/hermes-setup-home}"
export HERMES_BWS_PROJECT_ID="${HERMES_BWS_PROJECT_ID:-00000000-0000-4000-8000-000000000000}"
unset BWS_ACCESS_TOKEN

rm -rf "$HERMES_HOME"
mkdir -p "$HERMES_HOME"

make build

common_args=(
  --skip-summon
  --skip-tailscale
  --skip-self-install
  --no-disable-sshd
  --no-gateway-start
  --no-prompt
  --allow-missing-bws-token
  --home "$HERMES_HOME"
  --bitwarden-project "$HERMES_BWS_PROJECT_ID"
)

./hermes-setup install "${common_args[@]}"
./hermes-setup doctor --home "$HERMES_HOME" --bitwarden-project "$HERMES_BWS_PROJECT_ID" --allow-missing-bws-token --allow-missing-tailscale

if grep -q '^BWS_ACCESS_TOKEN=' "$HERMES_HOME/.env"; then
  echo "BWS_ACCESS_TOKEN should not be written when it is missing and prompts are disabled" >&2
  exit 1
fi

cp "$HERMES_HOME/config.yaml" "$RUNNER_TEMP/config.first.yaml"
cp "$HERMES_HOME/.env" "$RUNNER_TEMP/env.first"

./hermes-setup install "${common_args[@]}"
./hermes-setup doctor --home "$HERMES_HOME" --bitwarden-project "$HERMES_BWS_PROJECT_ID" --allow-missing-bws-token --allow-missing-tailscale

diff -u "$RUNNER_TEMP/config.first.yaml" "$HERMES_HOME/config.yaml"
diff -u "$RUNNER_TEMP/env.first" "$HERMES_HOME/.env"

BWS_ACCESS_TOKEN='0.ci-placeholder-not-a-real-token' ./hermes-setup install "${common_args[@]}"
grep -q "^BWS_ACCESS_TOKEN='0.ci-placeholder-not-a-real-token'$" "$HERMES_HOME/.env"
