# hermes-setup

Ubuntu / Debian x86_64 用の Hermes Agent セットアップ。

- machine setup: [upamune/summon](https://github.com/upamune/summon)
- access: Tailscale SSH
- secrets: Bitwarden Secrets Manager
- model: `openai-codex/gpt-5.5-low`
- gateway: Telegram

## Bitwarden

Project: `Hermes keys`

必要な secret:

```text
TELEGRAM_BOT_TOKEN
TELEGRAM_ALLOWED_USERS
```

provider key も必要なら Hermes が読む env 名で追加する。

`BWS_ACCESS_TOKEN` だけ `~/.hermes/.env` に保存する。Telegram 系はローカルに書かない。

## Install

```sh
export HERMES_BWS_PROJECT_ID='xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'
curl -fsSL https://raw.githubusercontent.com/upamune/hermes-setup/main/scripts/install.sh | sh
```

`BWS_ACCESS_TOKEN` は prompt で入れる。非対話なら先に export。

```sh
export BWS_ACCESS_TOKEN='0.xxxxxxxxxxxxxxxxx'
export HERMES_BWS_PROJECT_ID='xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'
curl -fsSL https://raw.githubusercontent.com/upamune/hermes-setup/main/scripts/install.sh | sh
```

## Tailscale

`TS_AUTHKEY` があれば使う。

```sh
export TS_AUTHKEY='tskey-auth-...'
```

なければ `tailscale up --ssh` を対話実行。

デフォルトで Tailscale SSH セットアップ後に system sshd を無効化する。残す場合:

```sh
hermes-setup install --no-disable-sshd
```

incoming も閉じる場合:

```sh
hermes-setup install --lock-down-incoming
```

## Remote

```sh
hermes-setup ssh user@example.com -- --bitwarden-project xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

## Import

```sh
hermes-setup import /path/to/hermes-backup.zip
```

install と同時にやる場合:

```sh
curl -fsSL https://raw.githubusercontent.com/upamune/hermes-setup/main/scripts/install.sh | sh -s -- \
  --import /path/to/hermes-backup.zip
```

## Doctor

```sh
hermes-setup doctor
hermes-setup doctor --fix
```

CI などで `BWS_ACCESS_TOKEN` なしを許す場合:

```sh
hermes-setup doctor --allow-missing-bws-token
```

## Dev

```sh
go test ./...
go vet ./...
go build ./cmd/hermes-setup
bash -n scripts/install.sh scripts/ci/integration.sh
```
