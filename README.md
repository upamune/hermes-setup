# hermes-setup

Ubuntu / Debian x86_64 の新規マシンに Hermes Agent をセットアップするためのリポジトリです。

流れは次の通りです。

1. `summon` でマシンの基本セットアップをする
2. Tailscale と Tailscale SSH をセットアップする
3. Hermes Agent をインストールする
4. Bitwarden Secrets Manager 経由で secrets を読む設定にする
5. Hermes gateway を起動する

## 必要な Bitwarden Secrets

Hermes Agent の secrets は Bitwarden Secrets Manager に置きます。ローカルに保存する secret は、Bitwarden を読むための `BWS_ACCESS_TOKEN` だけです。

Bitwarden Project に少なくとも次の secret を作ってください。

```text
TELEGRAM_BOT_TOKEN
TELEGRAM_ALLOWED_USERS
```

Telegram の polling mode では、基本的にこの 2 つで Telegram gateway を起動できます。`TELEGRAM_BOT_TOKEN` は BotFather で発行した token、`TELEGRAM_ALLOWED_USERS` は許可する Telegram user ID のカンマ区切りです。

provider/API key 類が必要な場合も、Hermes が読む環境変数名と同じ secret 名で Bitwarden に置きます。

## インストール

```sh
export HERMES_BWS_PROJECT_ID='xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'
curl -fsSL https://raw.githubusercontent.com/upamune/hermes-setup/main/scripts/install.sh | sh
```

installer は `BWS_ACCESS_TOKEN` を対話的に入力できます。入力された値は `~/.hermes/.env` に保存されます。空 Enter でスキップできます。

非対話環境では先に環境変数で渡してください。

```sh
export BWS_ACCESS_TOKEN='0.xxxxxxxxxxxxxxxxx'
export HERMES_BWS_PROJECT_ID='xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'
curl -fsSL https://raw.githubusercontent.com/upamune/hermes-setup/main/scripts/install.sh | sh
```

既定値:

- provider: `openai-codex`
- model: `gpt-5.5-low`
- Hermes home: `~/.hermes`

## Tailscale / SSH

Hermes Agent より先に Tailscale をセットアップします。

`TS_AUTHKEY` がある場合:

```sh
export TS_AUTHKEY='tskey-auth-...'
```

`tailscale up --ssh --authkey=...` を実行します。

`TS_AUTHKEY` がない場合は、`tailscale up --ssh` を対話的に実行します。`--no-prompt` の場合は Tailscale のログインをスキップします。

Tailscale SSH が使える状態になった後、system OpenSSH server はデフォルトで無効化します。現在の SSH 接続元が Tailscale IP に見えない場合はロックアウト防止のためスキップします。

OpenSSH server を残す場合:

```sh
hermes-setup install --no-disable-sshd
```

public incoming も閉じる場合:

```sh
hermes-setup install --lock-down-incoming
```

これは UFW で default deny incoming、allow outgoing、`tailscale0` inbound allow を設定します。Tailscale 経由に見えない接続では拒否します。

## リモート実行

手元の `hermes-setup` から SSH 先に installer を流し込めます。

```sh
hermes-setup ssh user@example.com -- --bitwarden-project xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

`--` 以降はリモートの `install.sh` に渡されます。`BWS_ACCESS_TOKEN` はリモート側のプロンプトで入力してください。

## Import

Hermes backup zip がある場合はセットアップ中に import できます。

```sh
curl -fsSL https://raw.githubusercontent.com/upamune/hermes-setup/main/scripts/install.sh | sh -s -- \
  --import /path/to/hermes-backup.zip
```

インストール後に実行する場合:

```sh
hermes-setup import /path/to/hermes-backup.zip
```

`hermes import` の後、管理対象の provider/model/Bitwarden 設定を再適用します。

## Doctor

```sh
hermes-setup doctor
hermes-setup doctor --fix
hermes-setup doctor --allow-missing-bws-token
```

`doctor` は OS/arch、必要コマンド、Tailscale、Hermes、Bitwarden 設定、`BWS_ACCESS_TOKEN`、gateway 状態を確認します。`--fix` は管理対象 config を再適用します。

## 冪等性

セットアップは再実行できます。

管理対象として上書きするもの:

- `~/.hermes/config.yaml` の `model.*`
- `~/.hermes/config.yaml` の `toolsets`
- `~/.hermes/config.yaml` の `secrets.bitwarden.*`
- `~/.hermes/.env` の `BWS_ACCESS_TOKEN`

`TELEGRAM_BOT_TOKEN` と `TELEGRAM_ALLOWED_USERS` はローカル `.env` には書きません。Bitwarden Project の secret として管理します。
