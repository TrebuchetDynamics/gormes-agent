package gormescli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSlackManifestFullManifestAppHomeAndScopes(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, stderr, err := executeSlackCommandForTest(NewSlackCommand(), "manifest", "--name", "Gormes Test Bot", "--description", "Test Slack manifest")
	if err != nil {
		t.Fatalf("slack manifest: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var manifest map[string]any
	if err := json.Unmarshal([]byte(stdout), &manifest); err != nil {
		t.Fatalf("stdout is not JSON: %v\nstdout=%s", err, stdout)
	}

	features := requireJSONMap(t, manifest, "features")
	appHome := requireJSONMap(t, features, "app_home")
	if got := appHome["home_tab_enabled"]; got != false {
		t.Fatalf("features.app_home.home_tab_enabled = %v, want false", got)
	}
	if got := appHome["messages_tab_enabled"]; got != true {
		t.Fatalf("features.app_home.messages_tab_enabled = %v, want true", got)
	}
	if got := appHome["messages_tab_read_only_enabled"]; got != false {
		t.Fatalf("features.app_home.messages_tab_read_only_enabled = %v, want false", got)
	}
	if _, ok := features["assistant_view"].(map[string]any); !ok {
		t.Fatalf("features.assistant_view missing or wrong type: %#v", features["assistant_view"])
	}
	if _, ok := features["bot_user"].(map[string]any); !ok {
		t.Fatalf("features.bot_user missing or wrong type: %#v", features["bot_user"])
	}

	scopes := requireJSONStringSlice(t, requireJSONMap(t, requireJSONMap(t, manifest, "oauth_config"), "scopes"), "bot")
	for _, want := range []string{"assistant:write", "groups:read", "im:write", "commands"} {
		if !containsString(scopes, want) {
			t.Fatalf("bot scopes missing %q: %v", want, scopes)
		}
	}

	events := requireJSONStringSlice(t, requireJSONMap(t, requireJSONMap(t, manifest, "settings"), "event_subscriptions"), "bot_events")
	for _, want := range []string{"assistant_thread_started", "message.groups", "message.im"} {
		if !containsString(events, want) {
			t.Fatalf("bot events missing %q: %v", want, events)
		}
	}

	slashes := requireJSONArray(t, features, "slash_commands")
	for _, want := range []string{"/hermes", "/stop", "/model", "/kanban"} {
		if !slashManifestHasCommand(slashes, want) {
			t.Fatalf("slash manifest missing %s: %#v", want, slashes)
		}
	}
	if len(slashes) > 50 {
		t.Fatalf("slash command count = %d, want <= 50", len(slashes))
	}
}

func TestSlackManifestSlashesOnly(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, stderr, err := executeSlackCommandForTest(NewSlackCommand(), "manifest", "--slashes-only")
	if err != nil {
		t.Fatalf("slack manifest --slashes-only: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var slashes []map[string]any
	if err := json.Unmarshal([]byte(stdout), &slashes); err != nil {
		t.Fatalf("stdout is not slash array JSON: %v\nstdout=%s", err, stdout)
	}
	for _, want := range []string{"/hermes", "/stop", "/model", "/kanban"} {
		if !slashManifestMapHasCommand(slashes, want) {
			t.Fatalf("slash-only manifest missing %s: %#v", want, slashes)
		}
	}
	if strings.Contains(stdout, "app_home") || strings.Contains(stdout, "oauth_config") {
		t.Fatalf("slashes-only output included full manifest sections:\n%s", stdout)
	}
}

func TestSlackManifestWriteDefaultAndCustomPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	stdout, stderr, err := executeSlackCommandForTest(NewSlackCommand(), "manifest", "--write")
	if err != nil {
		t.Fatalf("slack manifest --write: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("stdout = %q, want empty when writing", stdout)
	}
	defaultPath := filepath.Join(home, "slack-manifest.json")
	requireFileContains(t, defaultPath, `"app_home"`)
	requireFileContains(t, defaultPath, `"groups:read"`)
	if !strings.Contains(stderr, "Slack manifest written to: "+defaultPath) ||
		!strings.Contains(stderr, "Features -> App Manifest") {
		t.Fatalf("stderr missing written path and next steps:\n%s", stderr)
	}

	customPath := filepath.Join(t.TempDir(), "nested", "custom-slack-manifest.json")
	stdout, stderr, err = executeSlackCommandForTest(NewSlackCommand(), "manifest", "--write="+customPath, "--slashes-only")
	if err != nil {
		t.Fatalf("slack manifest --write=custom --slashes-only: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("stdout = %q, want empty when writing custom path", stdout)
	}
	requireFileContains(t, customPath, `"/hermes"`)
	if strings.Contains(readFileString(t, customPath), `"app_home"`) {
		t.Fatalf("custom slashes-only manifest included full app_home section")
	}
	if !strings.Contains(stderr, "Slack manifest written to: "+customPath) {
		t.Fatalf("stderr missing custom path:\n%s", stderr)
	}
}

func requireJSONMap(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()
	got, ok := parent[key].(map[string]any)
	if !ok {
		t.Fatalf("%s missing or wrong type: %#v", key, parent[key])
	}
	return got
}

func requireJSONArray(t *testing.T, parent map[string]any, key string) []any {
	t.Helper()
	got, ok := parent[key].([]any)
	if !ok {
		t.Fatalf("%s missing or wrong type: %#v", key, parent[key])
	}
	return got
}

func requireJSONStringSlice(t *testing.T, parent map[string]any, key string) []string {
	t.Helper()
	raw := requireJSONArray(t, parent, key)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		if !ok {
			t.Fatalf("%s contains non-string value: %#v", key, v)
		}
		out = append(out, s)
	}
	return out
}

func slashManifestHasCommand(slashes []any, want string) bool {
	for _, raw := range slashes {
		row, ok := raw.(map[string]any)
		if ok && row["command"] == want {
			return true
		}
	}
	return false
}

func slashManifestMapHasCommand(slashes []map[string]any, want string) bool {
	for _, row := range slashes {
		if row["command"] == want {
			return true
		}
	}
	return false
}

func requireFileContains(t *testing.T, path string, want string) {
	t.Helper()
	if got := readFileString(t, path); !strings.Contains(got, want) {
		t.Fatalf("%s missing %q:\n%s", path, want, got)
	}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func executeSlackCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	return executeCobraCommandForTest(cmd, cobraCommandExecutionOptions{}, args...)
}
