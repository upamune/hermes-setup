package setup

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"charm.land/huh/v2"
	"gopkg.in/yaml.v3"
)

const (
	defaultProvider = "openai-codex"
	defaultModel    = "gpt-5.5-low"
)

type Options struct {
	Provider          string
	Model             string
	Home              string
	ImportPath        string
	BitwardenProject  string
	TailscaleHostname string
	ForceImport       bool
	SkipSummon        bool
	SkipTailscale     bool
	SkipHermesInstall bool
	SkipSelfInstall   bool
	NoPrompt          bool
	AllowMissingBWS   bool
	NoGatewayStart    bool
	LockDownIncoming  bool
	ForceLockDown     bool
	DisableSSHD       bool
	ForceDisableSSHD  bool
	Fix               bool
}

type SSHOptions struct {
	Host       string
	Repo       string
	Ref        string
	RequestTTY bool
	Args       []string
}

func DefaultOptions() Options {
	return Options{
		Provider:         defaultProvider,
		Model:            defaultModel,
		Home:             filepath.Join(homeDir(), ".hermes"),
		BitwardenProject: firstEnv("HERMES_BWS_PROJECT_ID", "BWS_PROJECT_ID"),
		TailscaleHostname: firstEnv("HERMES_SETUP_TAILSCALE_HOSTNAME",
			"TS_HOSTNAME"),
	}
}

func Install(ctx context.Context, opts Options) error {
	if err := checkTarget(); err != nil {
		return err
	}
	if !opts.SkipSummon {
		if err := run(ctx, nil, "sh", "-c", "curl -fsSL https://raw.githubusercontent.com/upamune/summon/main/summon.sh | sh"); err != nil {
			return err
		}
	}
	if !opts.SkipTailscale {
		if err := installTailscale(ctx, opts); err != nil {
			return err
		}
	}
	if !opts.SkipSelfInstall && hasCommand("go") {
		ref := os.Getenv("HERMES_SETUP_REF")
		if ref == "" {
			ref = "latest"
		}
		binDir := filepath.Join(homeDir(), ".local", "bin")
		_ = os.MkdirAll(binDir, 0o755)
		_ = run(ctx, map[string]string{"GOBIN": binDir}, "go", "install", "github.com/upamune/hermes-setup/cmd/hermes-setup@"+ref)
	}
	if !opts.SkipHermesInstall && !hasCommand("hermes") {
		if err := run(ctx, nil, "bash", "-lc", "curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash"); err != nil {
			return err
		}
	}
	if opts.ImportPath != "" {
		if err := Import(ctx, opts); err != nil {
			return err
		}
	}
	if err := configureHermes(opts); err != nil {
		return err
	}
	if !opts.NoGatewayStart {
		_ = run(ctx, mapHome(opts), "hermes", "gateway", "install")
		_ = run(ctx, mapHome(opts), "hermes", "gateway", "restart")
	}
	if opts.LockDownIncoming {
		if err := lockDownIncoming(ctx, opts); err != nil {
			return err
		}
	}
	if opts.DisableSSHD && opts.SkipTailscale {
		fmt.Println("[skip] sshd disable skipped because Tailscale setup was skipped")
	} else if opts.DisableSSHD && !currentSSHLooksTailscale() && !opts.ForceDisableSSHD {
		fmt.Println("[skip] sshd disable skipped because the current SSH client is not a Tailscale IP")
	} else if opts.DisableSSHD {
		if err := disableSSHD(ctx, opts); err != nil {
			return err
		}
	}
	doctorOpts := DefaultOptions()
	doctorOpts.Home = opts.Home
	doctorOpts.AllowMissingBWS = opts.AllowMissingBWS
	return Doctor(ctx, doctorOpts)
}

func Import(ctx context.Context, opts Options) error {
	if opts.ImportPath == "" {
		return errors.New("backup path is required")
	}
	if _, err := os.Stat(opts.ImportPath); err != nil {
		return fmt.Errorf("backup not readable: %w", err)
	}
	_ = run(ctx, mapHome(opts), "hermes", "gateway", "stop")
	args := []string{"import", opts.ImportPath}
	if opts.ForceImport {
		args = append(args, "--force")
	}
	if err := run(ctx, mapHome(opts), "hermes", args...); err != nil {
		return err
	}
	return configureHermes(opts)
}

func SSH(ctx context.Context, opts SSHOptions) error {
	if opts.Host == "" {
		return errors.New("ssh host is required")
	}
	if opts.Repo == "" {
		opts.Repo = "github.com/upamune/hermes-setup"
	}
	if opts.Ref == "" {
		opts.Ref = "latest"
	}
	remoteArgs := append([]string{}, opts.Args...)
	remote := "curl -fsSL https://raw.githubusercontent.com/upamune/hermes-setup/main/scripts/install.sh | sh"
	if len(remoteArgs) > 0 {
		remote += " -s -- " + shellJoin(remoteArgs)
	}

	args := []string{}
	if opts.RequestTTY {
		args = append(args, "-t")
	}
	args = append(args, opts.Host, "env", "HERMES_SETUP_REPO="+shellQuote(opts.Repo), "HERMES_SETUP_REF="+shellQuote(opts.Ref), "sh", "-lc", shellQuote(remote))
	return run(ctx, nil, "ssh", args...)
}

func installTailscale(ctx context.Context, opts Options) error {
	if !hasCommand("tailscale") {
		if err := run(ctx, nil, "sh", "-c", "curl -fsSL https://tailscale.com/install.sh | sh"); err != nil {
			return err
		}
	}
	if hasCommand("systemctl") {
		if err := runPrivileged(ctx, nil, "systemctl", "enable", "--now", "tailscaled"); err != nil {
			return err
		}
	}

	authKey := os.Getenv("TS_AUTHKEY")
	if authKey != "" {
		args := []string{"up", "--ssh", "--authkey=" + authKey}
		if opts.TailscaleHostname != "" {
			args = append(args, "--hostname="+opts.TailscaleHostname)
		}
		if extra := strings.TrimSpace(os.Getenv("TS_EXTRA_ARGS")); extra != "" {
			args = append(args, strings.Fields(extra)...)
		}
		return runPrivileged(ctx, nil, "tailscale", args...)
	}
	if tailscaleIsUp(ctx) {
		return runPrivileged(ctx, nil, "tailscale", "set", "--ssh=true")
	}
	if opts.NoPrompt {
		fmt.Println("[skip] TS_AUTHKEY is not set and --no-prompt is enabled; Tailscale is installed but not logged in")
		return nil
	}
	args := []string{"up", "--ssh"}
	if opts.TailscaleHostname != "" {
		args = append(args, "--hostname="+opts.TailscaleHostname)
	}
	if extra := strings.TrimSpace(os.Getenv("TS_EXTRA_ARGS")); extra != "" {
		args = append(args, strings.Fields(extra)...)
	}
	return runPrivileged(ctx, nil, "tailscale", args...)
}

func tailscaleIsUp(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "tailscale", "status", "--peers=false")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func lockDownIncoming(ctx context.Context, opts Options) error {
	if !opts.ForceLockDown && !currentSSHLooksTailscale() {
		return errors.New("refusing to lock down incoming traffic: current SSH connection is not from a Tailscale IP; reconnect over Tailscale SSH or pass --force-lock-down-incoming")
	}
	if !hasCommand("ufw") {
		if err := runPrivileged(ctx, nil, "apt-get", "update"); err != nil {
			return err
		}
		if err := runPrivileged(ctx, nil, "apt-get", "install", "-y", "ufw"); err != nil {
			return err
		}
	}
	if err := runPrivileged(ctx, nil, "ufw", "--force", "reset"); err != nil {
		return err
	}
	for _, args := range [][]string{
		{"default", "deny", "incoming"},
		{"default", "allow", "outgoing"},
		{"allow", "in", "on", "tailscale0"},
		{"--force", "enable"},
	} {
		if err := runPrivileged(ctx, nil, "ufw", args...); err != nil {
			return err
		}
	}
	return nil
}

func disableSSHD(ctx context.Context, opts Options) error {
	if !opts.ForceDisableSSHD && !currentSSHLooksTailscale() {
		return errors.New("refusing to disable sshd: current SSH connection is not from a Tailscale IP; reconnect over Tailscale SSH or pass --force-disable-sshd")
	}
	if !hasCommand("systemctl") {
		return errors.New("systemctl is required to disable sshd")
	}
	disabled := false
	for _, service := range []string{"ssh.service", "sshd.service"} {
		if !runQuiet(ctx, "systemctl", "cat", service) {
			continue
		}
		if err := runPrivileged(ctx, nil, "systemctl", "disable", "--now", service); err != nil {
			return err
		}
		disabled = true
	}
	if !disabled {
		return errors.New("no OpenSSH systemd service found: tried ssh.service and sshd.service")
	}
	return nil
}

func currentSSHLooksTailscale() bool {
	fields := strings.Fields(os.Getenv("SSH_CONNECTION"))
	if len(fields) == 0 {
		return false
	}
	ip := fields[0]
	return strings.HasPrefix(ip, "100.") || strings.HasPrefix(strings.ToLower(ip), "fd7a:115c:a1e0:")
}

func Doctor(ctx context.Context, opts Options) error {
	if opts.Fix {
		fixOpts := DefaultOptions()
		fixOpts.Home = opts.Home
		if opts.BitwardenProject != "" {
			fixOpts.BitwardenProject = opts.BitwardenProject
		}
		if err := configureHermes(fixOpts); err != nil {
			return err
		}
		fmt.Println("[ok] repaired managed config")
	}

	failures := 0
	check := func(ok bool, msg string) {
		prefix := "ok"
		if !ok {
			prefix = "fail"
			failures++
		}
		fmt.Printf("[%s] %s\n", prefix, msg)
	}

	check(runtime.GOOS == "linux", "OS is Linux")
	check(runtime.GOARCH == "amd64", "architecture is x86_64/amd64")
	check(isDebianLike(), "distribution looks like Ubuntu/Debian")
	for _, name := range []string{"sh", "curl", "git"} {
		check(hasCommand(name), name+" is available")
	}
	check(hasCommand("tailscale"), "tailscale command is available")
	if hasCommand("systemctl") {
		check(runQuiet(ctx, "systemctl", "is-enabled", "--quiet", "tailscaled"), "tailscaled is enabled in systemd")
		check(runQuiet(ctx, "systemctl", "is-active", "--quiet", "tailscaled"), "tailscaled is active in systemd")
	}
	if hasCommand("tailscale") {
		check(runQuiet(ctx, "tailscale", "status", "--peers=false"), "tailscale is logged in")
	}
	check(hasCommand("hermes"), "hermes command is available")

	cfg, err := readYAML(configPath(opts.Home))
	check(err == nil, "Hermes config.yaml is readable")
	if err == nil {
		check(nestedString(cfg, "model", "provider") == defaultProvider, "model.provider is "+defaultProvider)
		check(nestedString(cfg, "model", "default") == defaultModel, "model.default is "+defaultModel)
		check(nestedBool(cfg, "secrets", "bitwarden", "enabled"), "Bitwarden Secrets Manager is enabled")
		check(nestedString(cfg, "secrets", "bitwarden", "project_id") != "", "Bitwarden project_id is configured")
	}

	env, err := readEnv(envPath(opts.Home))
	check(err == nil, "Hermes .env is readable")
	if err == nil {
		hasBWS := env["BWS_ACCESS_TOKEN"] != "" || os.Getenv("BWS_ACCESS_TOKEN") != ""
		if opts.AllowMissingBWS {
			check(true, "BWS_ACCESS_TOKEN is present or explicitly allowed to be missing")
		} else {
			check(hasBWS, "BWS_ACCESS_TOKEN is present")
		}
		check(true, "Telegram secrets are expected from Bitwarden")
	}

	if hasCommand("hermes") {
		_ = run(ctx, mapHome(opts), "hermes", "--version")
		_ = run(ctx, mapHome(opts), "hermes", "secrets", "bitwarden", "status")
		_ = run(ctx, mapHome(opts), "hermes", "secrets", "bitwarden", "sync")
		_ = run(ctx, mapHome(opts), "hermes", "doctor")
		_ = run(ctx, mapHome(opts), "hermes", "gateway", "status")
	}
	if failures > 0 {
		return fmt.Errorf("doctor found %d issue(s)", failures)
	}
	return nil
}

func configureHermes(opts Options) error {
	if err := os.MkdirAll(opts.Home, 0o700); err != nil {
		return err
	}
	cfgPath := configPath(opts.Home)
	cfg, _ := readYAML(cfgPath)
	if cfg == nil {
		cfg = map[string]any{}
	}
	setNested(cfg, opts.Provider, "model", "provider")
	setNested(cfg, opts.Model, "model", "default")
	setNested(cfg, opts.Model, "model", "model")
	setNested(cfg, []any{"terminal", "web", "file", "skills"}, "toolsets")
	setNested(cfg, true, "secrets", "bitwarden", "enabled")
	setNested(cfg, "BWS_ACCESS_TOKEN", "secrets", "bitwarden", "access_token_env")
	if opts.BitwardenProject != "" {
		setNested(cfg, opts.BitwardenProject, "secrets", "bitwarden", "project_id")
	}
	setNested(cfg, 300, "secrets", "bitwarden", "cache_ttl_seconds")
	setNested(cfg, true, "secrets", "bitwarden", "override_existing")
	setNested(cfg, true, "secrets", "bitwarden", "auto_install")
	if err := writeYAML(cfgPath, cfg); err != nil {
		return err
	}

	envUpdates := map[string]string{}
	if v := os.Getenv("BWS_ACCESS_TOKEN"); v != "" {
		envUpdates["BWS_ACCESS_TOKEN"] = v
	} else if !opts.NoPrompt {
		if v, ok, err := promptSecret("BWS_ACCESS_TOKEN", "Blank to skip."); err != nil {
			return err
		} else if ok && v != "" {
			envUpdates["BWS_ACCESS_TOKEN"] = v
		}
	}
	if len(envUpdates) > 0 {
		if err := upsertEnv(envPath(opts.Home), envUpdates); err != nil {
			return err
		}
	} else if err := ensureEnvFile(envPath(opts.Home)); err != nil {
		return err
	}
	return nil
}

func checkTarget() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("unsupported OS %s: this setup targets Ubuntu/Debian Linux", runtime.GOOS)
	}
	if runtime.GOARCH != "amd64" {
		return fmt.Errorf("unsupported architecture %s: this setup targets x86_64/amd64", runtime.GOARCH)
	}
	if !isDebianLike() {
		return errors.New("unsupported distribution: expected Ubuntu/Debian or compatible /etc/os-release")
	}
	return nil
}

func isDebianLike() bool {
	b, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return false
	}
	s := strings.ToLower(string(b))
	return strings.Contains(s, "id=ubuntu") || strings.Contains(s, "id=debian") || strings.Contains(s, "id_like=debian")
}

func readYAML(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if len(strings.TrimSpace(string(b))) == 0 {
		return out, nil
	}
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func writeYAML(path string, value map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func setNested(root map[string]any, value any, keys ...string) {
	cur := root
	for _, key := range keys[:len(keys)-1] {
		next, ok := cur[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[key] = next
		}
		cur = next
	}
	cur[keys[len(keys)-1]] = value
}

func nestedString(root map[string]any, keys ...string) string {
	var cur any = root
	for _, key := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = m[key]
	}
	v, _ := cur.(string)
	return v
}

func nestedBool(root map[string]any, keys ...string) bool {
	var cur any = root
	for _, key := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		cur = m[key]
	}
	v, _ := cur.(bool)
	return v
}

func readEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		out[parts[0]] = strings.Trim(parts[1], `"'`)
	}
	return out, sc.Err()
}

func upsertEnv(path string, updates map[string]string) error {
	existing := []string{}
	if b, err := os.ReadFile(path); err == nil {
		existing = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	}
	seen := map[string]bool{}
	for i, line := range existing {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") || !strings.Contains(trim, "=") {
			continue
		}
		key := strings.SplitN(trim, "=", 2)[0]
		if value, ok := updates[key]; ok {
			existing[i] = key + "=" + shellQuote(value)
			seen[key] = true
		}
	}
	for key, value := range updates {
		if !seen[key] {
			existing = append(existing, key+"="+shellQuote(value))
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(existing, "\n")+"\n"), 0o600)
}

func ensureEnvFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte{}, 0o600)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func shellJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func run(ctx context.Context, env map[string]string, name string, args ...string) error {
	fmt.Printf("+ %s %s\n", name, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	return cmd.Run()
}

func runPrivileged(ctx context.Context, env map[string]string, name string, args ...string) error {
	if os.Geteuid() == 0 {
		return run(ctx, env, name, args...)
	}
	return run(ctx, env, "sudo", append([]string{name}, args...)...)
}

func runQuiet(ctx context.Context, name string, args ...string) bool {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func EnvBool(name string) bool {
	v := strings.ToLower(os.Getenv(name))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	return ""
}

func promptSecret(title, description string) (string, bool, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", false, err
	}
	if stat.Mode()&os.ModeCharDevice == 0 {
		fmt.Println("[skip] BWS_ACCESS_TOKEN is not set and stdin is not interactive")
		return "", false, nil
	}
	value := ""
	input := huh.NewInput().
		Title(title).
		Password(true).
		Value(&value)
	if description != "" {
		input = input.Description(description)
	}
	if err := input.Run(); err != nil {
		return "", false, err
	}
	return strings.TrimSpace(value), true, nil
}

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "."
}

func configPath(home string) string {
	return filepath.Join(home, "config.yaml")
}

func envPath(home string) string {
	return filepath.Join(home, ".env")
}

func mapHome(opts Options) map[string]string {
	return map[string]string{"HERMES_HOME": opts.Home}
}
