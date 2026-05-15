package installtest

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInstallLocalEndToEnd_PublishedBinaryRunsCoreCLI(t *testing.T) {
	root := repoRoot(t)
	sb := t.TempDir()
	installHome := filepath.Join(sb, "install-home")
	binDir := filepath.Join(sb, "bin")
	operatorHome := filepath.Join(sb, "operator-home")

	install := exec.Command("sh", filepath.Join(root, "install.sh"), "--local", "--skip-setup", "--restart-gateway", "never")
	install.Dir = root
	install.Env = append(isolatedInstallEnv(t, operatorHome),
		"GORMES_INSTALL_HOME="+installHome,
		"GORMES_BIN_DIR="+binDir,
		"GORMES_SKIP_SETUP=1",
		"GORMES_RESTART_GATEWAY=never",
	)
	installOut, err := install.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh --local failed: %v\noutput:\n%s", err, installOut)
	}

	bin := filepath.Join(binDir, "gormes")
	if info, err := os.Stat(bin); err != nil {
		t.Fatalf("published gormes binary missing at %s: %v\ninstall output:\n%s", bin, err, installOut)
	} else if info.Mode()&0111 == 0 {
		t.Fatalf("published gormes binary is not executable: mode=%s path=%s", info.Mode(), bin)
	}

	runtimeEnv, runtimeHome := isolatedRuntimeEnv(t, sb)
	versionText := assertTextCommand(t, bin, runtimeEnv, "--version")
	if !strings.Contains(versionText, "gormes version") {
		t.Fatalf("gormes --version output = %q, want gormes version evidence", versionText)
	}
	assertJSONCommand(t, bin, runtimeEnv, "version", "--json")

	setupPlan := assertTextCommand(t, bin, runtimeEnv, "setup", "gateway", "--plan")
	for _, want := range []string{
		"Messaging Platforms",
		"Plan only: no files will be written and no live APIs will be called.",
		"Telegram",
		"Discord",
		"Slack",
		"WhatsApp",
		"Gateway action:",
	} {
		if !strings.Contains(setupPlan, want) {
			t.Fatalf("setup gateway --plan missing %q:\n%s", want, setupPlan)
		}
	}
	for _, path := range []string{filepath.Join(runtimeHome, ".env"), filepath.Join(runtimeHome, "config.toml")} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("setup gateway --plan wrote %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat setup plan output %s: %v", path, err)
		}
	}

	restartEnv := replaceEnvValue(runtimeEnv, "PATH", binDir)
	restartOut, restartErr := runCommandCombinedWithin(t, 3*time.Second, bin, restartEnv, "gateway", "restart", "--json", "--timeout=500ms")
	if restartErr == nil {
		t.Fatalf("unconfigured gateway restart unexpectedly succeeded:\n%s", restartOut)
	}
	for _, want := range []string{
		"gateway restart: start gateway",
		"no channels configured",
	} {
		if !strings.Contains(restartOut, want) {
			t.Fatalf("gateway restart output missing %q:\n%s", want, restartOut)
		}
	}
	for _, reject := range []string{
		"gateway restart: no live gateway runtime (status=missing_state",
		"hermes gateway",
		"~/.hermes",
	} {
		if strings.Contains(restartOut, reject) {
			t.Fatalf("gateway restart output contains stale/unsafe text %q:\n%s", reject, restartOut)
		}
	}
	assertJSONCommand(t, bin, restartEnv, "gateway", "status", "--json")

	configSet := assertJSONCommand(t, bin, runtimeEnv, "config", "set", "hermes.endpoint", "https://example.invalid/v1", "--json")
	if strings.Contains(configSet, "https://example.invalid/v1") {
		t.Fatalf("config set --json echoed raw config value:\n%s", configSet)
	}

	assertJSONCommand(t, bin, runtimeEnv, "gateway", "reload", "--json")
	onboard := assertJSONCommand(t, bin, runtimeEnv, "onboard", "--json")
	var onboardReport struct {
		Home string `json:"home"`
	}
	if err := json.Unmarshal([]byte(onboard), &onboardReport); err != nil {
		t.Fatalf("onboard --json should be valid JSON: %v\noutput:\n%s", err, onboard)
	}
	if onboardReport.Home != runtimeHome {
		t.Fatalf("onboard --json home = %q, want isolated runtime home %q\noutput:\n%s", onboardReport.Home, runtimeHome, onboard)
	}

	doctor := assertJSONCommand(t, bin, runtimeEnv, "doctor", "--offline", "--json")
	var doctorReport struct {
		Failed bool `json:"failed"`
	}
	if err := json.Unmarshal([]byte(doctor), &doctorReport); err != nil {
		t.Fatalf("doctor --offline --json should be valid JSON: %v\noutput:\n%s", err, doctor)
	}
	if doctorReport.Failed {
		t.Fatalf("doctor --offline --json reported failed=true:\n%s", doctor)
	}

	assertJSONCommand(t, bin, runtimeEnv, "gateway", "status", "--json")
}

func isolatedInstallEnv(t *testing.T, home string) []string {
	t.Helper()
	env := []string{
		"HOME=" + home,
		"PATH=" + os.Getenv("PATH"),
		"TMPDIR=" + t.TempDir(),
		"TERM=dumb",
		"XDG_CACHE_HOME=" + filepath.Join(home, ".cache"),
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
	}
	for _, key := range []string{
		"CGO_ENABLED",
		"GOCACHE",
		"GOFLAGS",
		"GOMODCACHE",
		"GONOSUMDB",
		"GOPATH",
		"GOPRIVATE",
		"GOPROXY",
		"GOROOT",
		"GOSUMDB",
		"GOTOOLCHAIN",
	} {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	if _, ok := os.LookupEnv("GOMODCACHE"); !ok {
		if value := goEnv(t, "GOMODCACHE"); value != "" {
			env = append(env, "GOMODCACHE="+value)
		}
	}
	if _, ok := os.LookupEnv("GOPATH"); !ok {
		if value := goEnv(t, "GOPATH"); value != "" {
			env = append(env, "GOPATH="+value)
		}
	}
	if _, ok := os.LookupEnv("GOCACHE"); !ok {
		env = append(env, "GOCACHE="+filepath.Join(home, ".cache", "go-build"))
	}
	return env
}

func isolatedRuntimeEnv(t *testing.T, sb string) ([]string, string) {
	t.Helper()
	runtimeHome := filepath.Join(sb, "runtime-home")
	for _, dir := range []string{
		runtimeHome,
		filepath.Join(sb, "operator-runtime-home"),
		filepath.Join(sb, "runtime-tmp"),
		filepath.Join(sb, "hermes-home"),
		filepath.Join(sb, "codex-home"),
		filepath.Join(sb, "xdg-cache"),
		filepath.Join(sb, "xdg-config"),
		filepath.Join(sb, "xdg-data"),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("create runtime dir %s: %v", dir, err)
		}
	}
	return []string{
		"HOME=" + filepath.Join(sb, "operator-runtime-home"),
		"PATH=" + os.Getenv("PATH"),
		"TMPDIR=" + filepath.Join(sb, "runtime-tmp"),
		"TERM=dumb",
		"GORMES_HOME=" + runtimeHome,
		"HERMES_HOME=" + filepath.Join(sb, "hermes-home"),
		"CODEX_HOME=" + filepath.Join(sb, "codex-home"),
		"XDG_CACHE_HOME=" + filepath.Join(sb, "xdg-cache"),
		"XDG_CONFIG_HOME=" + filepath.Join(sb, "xdg-config"),
		"XDG_DATA_HOME=" + filepath.Join(sb, "xdg-data"),
	}, runtimeHome
}

func assertJSONCommand(t *testing.T, bin string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\noutput:\n%s", bin, strings.Join(args, " "), err, out)
	}
	var decoded any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("%s %s did not emit valid JSON: %v\noutput:\n%s", bin, strings.Join(args, " "), err, out)
	}
	return string(out)
}

func assertTextCommand(t *testing.T, bin string, env []string, args ...string) string {
	t.Helper()
	out, err := runCommandCombined(t, bin, env, args...)
	if err != nil {
		t.Fatalf("%s %s failed: %v\noutput:\n%s", bin, strings.Join(args, " "), err, out)
	}
	return out
}

func runCommandCombined(t *testing.T, bin string, env []string, args ...string) (string, error) {
	t.Helper()
	return runCommandCombinedWithin(t, 30*time.Second, bin, env, args...)
}

func runCommandCombinedWithin(t *testing.T, timeout time.Duration, bin string, env []string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("%s %s timed out after %s\noutput:\n%s", bin, strings.Join(args, " "), timeout, out)
	}
	return string(out), err
}

func replaceEnvValue(env []string, key, value string) []string {
	prefix := key + "="
	next := append([]string(nil), env...)
	for i, entry := range next {
		if strings.HasPrefix(entry, prefix) {
			next[i] = prefix + value
			return next
		}
	}
	return append(next, prefix+value)
}

func goEnv(t *testing.T, key string) string {
	t.Helper()
	cmd := exec.Command("go", "env", key)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go env %s: %v", key, err)
	}
	return strings.TrimSpace(string(out))
}
