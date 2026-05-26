package setup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigureHermesIsIdempotentAndPreservesOtherEnv(t *testing.T) {
	t.Setenv("BWS_ACCESS_TOKEN", "0.test-token")
	t.Setenv("HERMES_BWS_PROJECT_ID", "project-123")

	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".env"), []byte("KEEP_ME=yes\nBWS_ACCESS_TOKEN='old'\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	opts := DefaultOptions()
	opts.Home = home
	if err := configureHermes(opts); err != nil {
		t.Fatal(err)
	}
	if err := configureHermes(opts); err != nil {
		t.Fatal(err)
	}

	cfg, err := readYAML(filepath.Join(home, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := nestedString(cfg, "model", "provider"); got != defaultProvider {
		t.Fatalf("model.provider = %q", got)
	}
	if got := nestedString(cfg, "model", "default"); got != defaultModel {
		t.Fatalf("model.default = %q", got)
	}
	if got := nestedBool(cfg, "secrets", "bitwarden", "enabled"); !got {
		t.Fatalf("secrets.bitwarden.enabled = %v", got)
	}
	if got := nestedString(cfg, "secrets", "bitwarden", "project_id"); got != "project-123" {
		t.Fatalf("secrets.bitwarden.project_id = %q", got)
	}

	env, err := readEnv(filepath.Join(home, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if got := env["KEEP_ME"]; got != "yes" {
		t.Fatalf("KEEP_ME = %q", got)
	}
	if got := env["BWS_ACCESS_TOKEN"]; got != "0.test-token" {
		t.Fatalf("BWS_ACCESS_TOKEN = %q", got)
	}
	if _, ok := env["TELEGRAM_ALLOWED_USERS"]; ok {
		t.Fatal("TELEGRAM_ALLOWED_USERS should come from Bitwarden, not local .env")
	}
	if _, ok := env["TELEGRAM_BOT_TOKEN"]; ok {
		t.Fatal("TELEGRAM_BOT_TOKEN should come from Bitwarden, not local .env")
	}
}

func TestValidSetupUserName(t *testing.T) {
	valid := []string{"hermes", "hermes-agent", "hermes_agent", "h1"}
	for _, name := range valid {
		if !validSetupUserName(name) {
			t.Fatalf("validSetupUserName(%q) = false", name)
		}
	}

	invalid := []string{"", "Hermes", "1hermes", "hermes.", "hermes-", "hermes agent"}
	for _, name := range invalid {
		if validSetupUserName(name) {
			t.Fatalf("validSetupUserName(%q) = true", name)
		}
	}
}

func TestInstallScriptArgsDropsSetupUser(t *testing.T) {
	got := installScriptArgs([]string{
		"install",
		"--setup-user", "hermes",
		"--bitwarden-project", "project-123",
		"--setup-user=other",
		"--no-disable-sshd",
	})
	want := []string{"--bitwarden-project", "project-123", "--no-disable-sshd"}
	if len(got) != len(want) {
		t.Fatalf("installScriptArgs length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("installScriptArgs[%d] = %q, want %q; got %#v", i, got[i], want[i], got)
		}
	}
}

func TestSSHRemoteArgsDropsSeparator(t *testing.T) {
	got := sshRemoteArgs([]string{"--", "--setup-user", "hermes", "--no-disable-sshd"})
	want := []string{"--setup-user", "hermes", "--no-disable-sshd"}
	if len(got) != len(want) {
		t.Fatalf("sshRemoteArgs length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sshRemoteArgs[%d] = %q, want %q; got %#v", i, got[i], want[i], got)
		}
	}
}

func TestShellJoinQuotesArgs(t *testing.T) {
	got := shellJoin(sshRemoteArgs([]string{"--", "--setup-user", "hermes"}))
	want := "'--setup-user' 'hermes'"
	if got != want {
		t.Fatalf("shellJoin(sshRemoteArgs(...)) = %q, want %q", got, want)
	}
}

func TestDefaultSetupRefHasFallback(t *testing.T) {
	if got := defaultSetupRef(context.Background()); got == "" {
		t.Fatal("defaultSetupRef returned empty string")
	}
}

func TestGhosttyTerminfoEnvDisabledByDefaultForOtherTerms(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("HERMES_SETUP_GHOSTTY_TERMINFO", "")
	if got := ghosttyTerminfoEnv(context.Background()); got != nil {
		t.Fatalf("ghosttyTerminfoEnv() = %#v, want nil", got)
	}
}

func TestMergeAuthorizedKeysPreservesExistingAndDeduplicates(t *testing.T) {
	got := string(mergeAuthorizedKeys(
		[]byte("ssh-ed25519 existing\nssh-ed25519 shared\n"),
		[]byte("ssh-ed25519 shared\nssh-rsa root\n"),
	))
	want := "ssh-ed25519 existing\nssh-ed25519 shared\nssh-rsa root\n"
	if got != want {
		t.Fatalf("mergeAuthorizedKeys() = %q, want %q", got, want)
	}
}
