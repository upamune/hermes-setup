package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/upamune/hermes-setup/internal/setup"
)

type CLI struct {
	Install InstallCmd `cmd:"" help:"Install and configure Hermes Agent."`
	Import  ImportCmd  `cmd:"" help:"Import a Hermes backup zip."`
	Doctor  DoctorCmd  `cmd:"" help:"Check local Hermes Agent setup."`
	SSH     SSHCmd     `cmd:"" name:"ssh" help:"Run the installer on a remote host over SSH."`
}

type InstallCmd struct {
	Provider          string `help:"Hermes inference provider." default:"openai-codex"`
	Model             string `help:"Hermes default model." default:"gpt-5.5-low"`
	Home              string `help:"Hermes home directory." type:"path"`
	ImportPath        string `name:"import" help:"Restore a Hermes backup zip before configuration." type:"path"`
	BitwardenProject  string `help:"Bitwarden Secrets Manager project UUID."`
	TailscaleHostname string `help:"Tailscale hostname to pass to tailscale up."`
	ForceImport       bool   `help:"Pass --force to hermes import when supported."`
	SkipSummon        bool   `help:"Skip summon bootstrap."`
	SkipTailscale     bool   `help:"Skip Tailscale installation and Tailscale SSH setup."`
	SkipHermesInstall bool   `help:"Skip Hermes Agent installation."`
	SkipSelfInstall   bool   `help:"Skip installing hermes-setup into ~/.local/bin via go install."`
	NoPrompt          bool   `help:"Do not prompt for missing secrets."`
	NoGatewayStart    bool   `help:"Do not install/restart the gateway service."`
	LockDownIncoming  bool   `help:"Deny inbound traffic with UFW after Tailscale SSH is confirmed."`
	ForceLockDown     bool   `name:"force-lock-down-incoming" help:"Allow inbound lockdown even when the current SSH client is not a Tailscale IP."`
	DisableSSHD       bool   `name:"disable-sshd" help:"Disable system OpenSSH server after Tailscale SSH is confirmed." default:"true" negatable:""`
	ForceDisableSSHD  bool   `name:"force-disable-sshd" help:"Disable system OpenSSH server even when the current SSH client is not a Tailscale IP."`
}

type DoctorCmd struct {
	Home             string `help:"Hermes home directory." type:"path"`
	BitwardenProject string `help:"Bitwarden Secrets Manager project UUID."`
	AllowMissingBWS  bool   `name:"allow-missing-bws-token" help:"Allow doctor to pass without BWS_ACCESS_TOKEN."`
	Fix              bool   `help:"Repair idempotent config managed by hermes-setup."`
}

type ImportCmd struct {
	Home        string `help:"Hermes home directory." type:"path"`
	ForceImport bool   `name:"force" help:"Pass --force to hermes import when supported."`
	Backup      string `arg:"" help:"Hermes backup zip path." type:"path"`
}

type SSHCmd struct {
	Host       string   `arg:"" help:"SSH host, for example user@example.com."`
	Ref        string   `help:"hermes-setup git ref used by the remote installer." default:"latest"`
	Repo       string   `help:"Go module path for hermes-setup." default:"github.com/upamune/hermes-setup"`
	RequestTTY bool     `name:"tty" help:"Request a remote TTY for prompts." default:"true" negatable:""`
	Args       []string `arg:"" optional:"" passthrough:"" help:"Arguments passed to remote install.sh after --."`
}

func (c InstallCmd) Run(ctx context.Context) error {
	opts := setup.DefaultOptions()
	opts.Provider = c.Provider
	opts.Model = c.Model
	if c.Home != "" {
		opts.Home = c.Home
	}
	if c.BitwardenProject != "" {
		opts.BitwardenProject = c.BitwardenProject
	}
	if c.TailscaleHostname != "" {
		opts.TailscaleHostname = c.TailscaleHostname
	}
	opts.ImportPath = c.ImportPath
	opts.ForceImport = c.ForceImport
	opts.SkipSummon = c.SkipSummon || setup.EnvBool("HERMES_SETUP_SKIP_SUMMON")
	opts.SkipTailscale = c.SkipTailscale || setup.EnvBool("HERMES_SETUP_SKIP_TAILSCALE")
	opts.SkipHermesInstall = c.SkipHermesInstall || setup.EnvBool("HERMES_SETUP_SKIP_HERMES_INSTALL")
	opts.SkipSelfInstall = c.SkipSelfInstall || setup.EnvBool("HERMES_SETUP_SKIP_SELF_INSTALL")
	opts.NoPrompt = c.NoPrompt || setup.EnvBool("HERMES_SETUP_NO_PROMPT")
	opts.NoGatewayStart = c.NoGatewayStart || setup.EnvBool("HERMES_SETUP_NO_GATEWAY_START")
	opts.LockDownIncoming = c.LockDownIncoming || setup.EnvBool("HERMES_SETUP_LOCK_DOWN_INCOMING")
	opts.ForceLockDown = c.ForceLockDown || setup.EnvBool("HERMES_SETUP_FORCE_LOCK_DOWN_INCOMING")
	opts.DisableSSHD = c.DisableSSHD && !setup.EnvBool("HERMES_SETUP_NO_DISABLE_SSHD")
	opts.ForceDisableSSHD = c.ForceDisableSSHD || setup.EnvBool("HERMES_SETUP_FORCE_DISABLE_SSHD")
	return setup.Install(ctx, opts)
}

func (c DoctorCmd) Run(ctx context.Context) error {
	opts := setup.DefaultOptions()
	if c.Home != "" {
		opts.Home = c.Home
	}
	if c.BitwardenProject != "" {
		opts.BitwardenProject = c.BitwardenProject
	}
	opts.AllowMissingBWS = c.AllowMissingBWS || setup.EnvBool("HERMES_SETUP_ALLOW_MISSING_BWS_TOKEN")
	opts.Fix = c.Fix
	return setup.Doctor(ctx, opts)
}

func (c ImportCmd) Run(ctx context.Context) error {
	opts := setup.DefaultOptions()
	if c.Home != "" {
		opts.Home = c.Home
	}
	opts.ImportPath = c.Backup
	opts.ForceImport = c.ForceImport
	return setup.Import(ctx, opts)
}

func (c SSHCmd) Run(ctx context.Context) error {
	return setup.SSH(ctx, setup.SSHOptions{
		Host:       c.Host,
		Repo:       c.Repo,
		Ref:        c.Ref,
		RequestTTY: c.RequestTTY,
		Args:       c.Args,
	})
}

func main() {
	var cli CLI
	kctx := kong.Parse(&cli,
		kong.Name("hermes-setup"),
		kong.Description("Install Hermes Agent on Ubuntu/Debian x86_64 machines."),
		kong.UsageOnError(),
	)
	if err := kctx.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "hermes-setup: %v\n", err)
		os.Exit(1)
	}
}
