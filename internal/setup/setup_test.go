package setup

import (
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
