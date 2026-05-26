#!/bin/sh
set -eu

repo="${HERMES_SETUP_REPO:-github.com/upamune/hermes-setup}"
ref="${HERMES_SETUP_REF:-main}"
setup_user=""
clean_count=0

while [ "$#" -gt 0 ]; do
  arg="$1"
  shift
  case "$arg" in
    --)
      ;;
    --setup-user)
      if [ "$#" -eq 0 ]; then
        echo "hermes-setup: --setup-user requires a value" >&2
        exit 1
      fi
      setup_user="$1"
      shift
      ;;
    --setup-user=*)
      setup_user="${arg#--setup-user=}"
      ;;
    *)
      clean_count=$((clean_count + 1))
      eval "clean_arg_$clean_count=\$arg"
      ;;
  esac
done

set --
i=1
while [ "$i" -le "$clean_count" ]; do
  eval "clean_arg=\$clean_arg_$i"
  set -- "$@" "$clean_arg"
  i=$((i + 1))
done

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

if [ "$(id -u)" = "0" ] && [ -n "$setup_user" ] && [ -z "${HERMES_SETUP_BOOTSTRAPPED_USER:-}" ]; then
  case "$setup_user" in
    root|""|[!a-z]*|*[!a-z0-9_-]*|*-)
      echo "hermes-setup: invalid --setup-user: $setup_user" >&2
      exit 1
      ;;
  esac

  if ! command -v sudo >/dev/null 2>&1; then
    apt-get update
    apt-get install -y sudo
  fi

  if ! id -u "$setup_user" >/dev/null 2>&1; then
    useradd --create-home --shell /bin/bash --groups sudo "$setup_user"
  elif ! id -nG "$setup_user" | grep -qw sudo; then
    usermod --append --groups sudo "$setup_user"
  fi

  echo "$setup_user ALL=(ALL) NOPASSWD:ALL" >"/etc/sudoers.d/hermes-setup-$setup_user"
  chmod 0440 "/etc/sudoers.d/hermes-setup-$setup_user"

  if [ -f /root/.ssh/authorized_keys ]; then
    ssh_dir="/home/$setup_user/.ssh"
    authorized_keys="$ssh_dir/authorized_keys"
    mkdir -p "$ssh_dir"
    touch "$authorized_keys"
    tmp_keys="$authorized_keys.tmp"
    cat "$authorized_keys" /root/.ssh/authorized_keys | awk 'NF && !seen[$0]++' >"$tmp_keys"
    mv "$tmp_keys" "$authorized_keys"
    chmod 0700 "$ssh_dir"
    chmod 0600 "$authorized_keys"
    chown -R "$setup_user:$(id -gn "$setup_user")" "$ssh_dir"
  fi

  export HERMES_SETUP_BOOTSTRAPPED_USER=1
  exec sudo --preserve-env=HERMES_SETUP_BOOTSTRAPPED_USER,HERMES_SETUP_REPO,HERMES_SETUP_REF,HERMES_SETUP_SKIP_SUMMON,HERMES_SETUP_SKIP_TAILSCALE,HERMES_SETUP_SKIP_HERMES_INSTALL,HERMES_SETUP_SKIP_SELF_INSTALL,HERMES_SETUP_NO_PROMPT,HERMES_SETUP_ALLOW_MISSING_BWS_TOKEN,HERMES_SETUP_NO_GATEWAY_START,HERMES_SETUP_LOCK_DOWN_INCOMING,HERMES_SETUP_FORCE_LOCK_DOWN_INCOMING,HERMES_SETUP_NO_DISABLE_SSHD,HERMES_SETUP_FORCE_DISABLE_SSHD,HERMES_SETUP_TERMINFO_NAME,HERMES_SETUP_TERMINFO_B64,HERMES_BWS_PROJECT_ID,BWS_PROJECT_ID,BWS_ACCESS_TOKEN,TS_AUTHKEY,TS_HOSTNAME,HERMES_SETUP_TAILSCALE_HOSTNAME,TS_EXTRA_ARGS \
    -H -u "$setup_user" sh -c 'cd && curl -fsSL https://raw.githubusercontent.com/upamune/hermes-setup/main/scripts/install.sh | sh -s -- "$@"' hermes-setup "$@"
fi

if [ "${HERMES_SETUP_SKIP_SUMMON:-0}" != "1" ]; then
  curl -fsSL https://raw.githubusercontent.com/upamune/summon/main/summon.sh | sh
fi

cd "${HOME:-.}"

if [ -n "${HERMES_SETUP_TERMINFO_NAME:-}" ] && [ -n "${HERMES_SETUP_TERMINFO_B64:-}" ] && command -v tic >/dev/null 2>&1 && command -v base64 >/dev/null 2>&1; then
  mkdir -p "$HOME/.terminfo"
  if printf '%s' "$HERMES_SETUP_TERMINFO_B64" | base64 -d | TERMINFO="$HOME/.terminfo" tic -x -; then
    echo "hermes-setup: installed terminfo for $HERMES_SETUP_TERMINFO_NAME under $HOME/.terminfo"
  else
    echo "hermes-setup: failed to install terminfo for $HERMES_SETUP_TERMINFO_NAME" >&2
  fi
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
