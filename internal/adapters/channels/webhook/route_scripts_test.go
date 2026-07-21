package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

const webhookScriptHelperPrefix = "webhook-script-helper"

func TestMain(m *testing.M) {
	if strings.HasPrefix(filepath.Base(os.Args[0]), webhookScriptHelperPrefix) {
		runWebhookScriptHelper()
		return
	}
	os.Exit(m.Run())
}

func TestWebhookRuntime_ScriptBeforePrompt(t *testing.T) {
	home := t.TempDir()
	script := installWebhookScriptHelper(t, home)
	var callback AcceptedWebhook
	rt := NewRuntime(RuntimeConfig{
		Routes: NewDynamicRouteSet(home, map[string]RouteConfig{
			"todoist": {
				Secret: InsecureNoAuth,
				Script: filepath.Base(script),
				Prompt: "Task: {body}",
			},
		}),
		ProfileHome:  home,
		MaxBodyBytes: 1024,
		OnAccepted:   func(accepted AcceptedWebhook) { callback = accepted },
	})
	body := []byte(`{"task":{"content":"pay bills"}}`)

	resp := rt.Handle("todoist", InboundRequest{
		Headers:       map[string]string{"X-GitHub-Delivery": "script-transform-1"},
		Body:          body,
		ContentLength: int64(len(body)),
	})
	if resp.StatusCode != 202 || resp.Status != "accepted" {
		t.Fatalf("script response = %+v, want 202 accepted", resp)
	}
	accepted := rt.Accepted()
	if len(accepted) != 1 {
		t.Fatalf("Accepted count = %d, want 1", len(accepted))
	}
	if accepted[0].Prompt != "Task: PAY BILLS" {
		t.Fatalf("Prompt = %q, want transformed prompt", accepted[0].Prompt)
	}
	if got := accepted[0].Parsed.Payload["body"]; got != "PAY BILLS" {
		t.Fatalf("Parsed.Payload[body] = %#v, want transformed value", got)
	}
	if callback.Prompt != "Task: PAY BILLS" || callback.Parsed.Payload["body"] != "PAY BILLS" {
		t.Fatalf("OnAccepted payload = %#v prompt=%q, want transformed evidence", callback.Parsed.Payload, callback.Prompt)
	}
}

func TestWebhookRuntime_RouteScriptDirectExecutionPolicy(t *testing.T) {
	home := t.TempDir()
	script := installWebhookScriptHelper(t, home)
	t.Setenv("WEBHOOK_PARENT_SECRET", "must-not-leak")
	rt := newScriptRuntime(home, filepath.Base(script), 0)

	resp := rt.Handle("script", scriptRequest("script-policy-1", "evidence"))
	if resp.StatusCode != 202 {
		t.Fatalf("policy response = %+v, want 202 accepted", resp)
	}
	payload := rt.Accepted()[0].Parsed.Payload
	if got := payload["argv_count"]; got != float64(0) {
		t.Fatalf("argv_count = %#v, want 0", got)
	}
	if got := payload["cwd"]; got != filepath.Dir(script) {
		t.Fatalf("cwd = %#v, want %q", got, filepath.Dir(script))
	}
	for _, key := range []string{"home", "gormes_home", "hermes_home"} {
		if got := payload[key]; got != home {
			t.Fatalf("%s = %#v, want profile home", key, got)
		}
	}
	if got := payload["parent_secret"]; got != "" {
		t.Fatalf("parent_secret = %#v, want scrubbed", got)
	}
	if got := payload["stdin"]; got != `{"script_test_mode":"evidence","task":{"content":"pay bills"}}` {
		t.Fatalf("stdin = %#v, want compact payload JSON", got)
	}
	gotKeys := interfaceStrings(payload["env_keys"])
	wantKeys := []string{"GORMES_HOME", "HERMES_HOME", "HOME"}
	if os.Getenv("PATH") != "" {
		wantKeys = append(wantKeys, "PATH")
	}
	if runtime.GOOS == "windows" {
		wantKeys = append(wantKeys, "USERPROFILE")
		for _, key := range []string{"SystemRoot", "ComSpec", "PATHEXT"} {
			if os.Getenv(key) != "" {
				wantKeys = append(wantKeys, key)
			}
		}
	}
	sort.Strings(wantKeys)
	if strings.Join(gotKeys, ",") != strings.Join(wantKeys, ",") {
		t.Fatalf("environment keys = %v, want %v", gotKeys, wantKeys)
	}
}

func TestWebhookRuntime_RouteScriptAliases(t *testing.T) {
	home := t.TempDir()
	script := installWebhookScriptHelper(t, home)
	for _, configured := range []string{
		filepath.Base(script),
		"~/.hermes/scripts/" + filepath.Base(script),
		"~/.gormes/scripts/" + filepath.Base(script),
	} {
		t.Run(configured, func(t *testing.T) {
			rt := newScriptRuntime(home, configured, 0)
			resp := rt.Handle("script", scriptRequest("alias-1", "transform"))
			if resp.StatusCode != 202 || resp.Status != "accepted" {
				t.Fatalf("Handle() = %+v, want accepted", resp)
			}
		})
	}
}

func TestWebhookRuntime_RouteScriptInvalidPathsFailClosed(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "scripts", "directory"), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	nonExecutable := filepath.Join(home, "scripts", "not-executable")
	if err := os.WriteFile(nonExecutable, []byte("not a program"), 0o600); err != nil {
		t.Fatalf("WriteFile(non-executable) error = %v", err)
	}
	badExecutable := filepath.Join(home, "scripts", "bad-executable")
	if runtime.GOOS == "windows" {
		badExecutable += ".exe"
	}
	if err := os.WriteFile(badExecutable, []byte("not a program"), 0o700); err != nil {
		t.Fatalf("WriteFile(bad-executable) error = %v", err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0o700); err != nil {
		t.Fatalf("WriteFile(outside) error = %v", err)
	}
	cases := map[string]string{
		"absolute":            outside,
		"traversal":           "../outside",
		"unsupported_tilde":   "~/outside",
		"missing":             "missing",
		"directory":           "directory",
		"environment_dollar":  "$WEBHOOK_SCRIPT",
		"environment_percent": "%WEBHOOK_SCRIPT%",
		"launch_failure":      filepath.Base(badExecutable),
	}
	if runtime.GOOS != "windows" {
		cases["non_executable"] = filepath.Base(nonExecutable)
		symlink := filepath.Join(home, "scripts", "escape-link")
		if err := os.Symlink(outside, symlink); err != nil {
			t.Fatalf("Symlink() error = %v", err)
		}
		cases["symlink_escape"] = filepath.Base(symlink)
	}
	for name, configured := range cases {
		t.Run(name, func(t *testing.T) {
			rt := newScriptRuntime(home, configured, 0)
			resp := rt.Handle("script", scriptRequest("invalid-path-1", "transform"))
			if resp.StatusCode != 200 || resp.Status != "ignored" || resp.Reason != "script" {
				t.Fatalf("Handle() = %+v, want 200 ignored reason=script", resp)
			}
			if len(rt.Accepted()) != 0 {
				t.Fatalf("Accepted count = %d, want 0", len(rt.Accepted()))
			}
		})
	}
}

func TestWebhookRuntime_RouteScriptRequiresProfileHome(t *testing.T) {
	home := t.TempDir()
	script := installWebhookScriptHelper(t, home)
	rt := newScriptRuntime("", filepath.Base(script), 0)
	resp := rt.Handle("script", scriptRequest("blank-home-1", "transform"))
	if resp.StatusCode != 200 || resp.Reason != "script" {
		t.Fatalf("Handle() = %+v, want ignored script", resp)
	}
}

func TestWebhookRuntime_RouteScriptRunsAfterFilters(t *testing.T) {
	home := t.TempDir()
	rt := NewRuntime(RuntimeConfig{
		Routes: NewDynamicRouteSet(home, map[string]RouteConfig{
			"script": {
				Secret:  InsecureNoAuth,
				Filters: map[string]any{"field": "payload.label", "equals": "urgent"},
				Script:  "missing",
			},
		}),
		ProfileHome:  home,
		MaxBodyBytes: 1024,
	})
	body := []byte(`{"payload":{"label":"later"}}`)
	resp := rt.Handle("script", InboundRequest{Headers: map[string]string{"X-GitHub-Delivery": "filter-order-1"}, Body: body})
	if resp.StatusCode != 200 || resp.Reason != "filter" {
		t.Fatalf("Handle() = %+v, want filter rejection before missing script", resp)
	}
}

func TestRouteScriptTimeoutClamp(t *testing.T) {
	for _, test := range []struct {
		name string
		in   time.Duration
		want time.Duration
	}{
		{name: "default", in: 0, want: 30 * time.Second},
		{name: "negative", in: -time.Second, want: 30 * time.Second},
		{name: "lower", in: 2 * time.Second, want: 2 * time.Second},
		{name: "above cap", in: 31 * time.Second, want: 30 * time.Second},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := clampRouteScriptTimeout(test.in); got != test.want {
				t.Fatalf("clampRouteScriptTimeout(%s) = %s, want %s", test.in, got, test.want)
			}
		})
	}
}

func TestWebhookRuntime_RouteScriptOutputContracts(t *testing.T) {
	home := t.TempDir()
	script := installWebhookScriptHelper(t, home)
	ignoredModes := []string{"empty", "silent", "silent_object", "ignore_object", "array", "nonzero", "huge_stdout", "huge_stderr"}
	for _, mode := range ignoredModes {
		t.Run(mode, func(t *testing.T) {
			rt := newScriptRuntime(home, filepath.Base(script), 0)
			resp := rt.Handle("script", scriptRequest("output-"+mode, mode))
			if resp.StatusCode != 200 || resp.Status != "ignored" || resp.Reason != "script" {
				t.Fatalf("Handle() = %+v, want 200 ignored reason=script", resp)
			}
		})
	}

	rt := newScriptRuntime(home, filepath.Base(script), 0)
	resp := rt.Handle("script", scriptRequest("output-text", "text"))
	if resp.StatusCode != 202 {
		t.Fatalf("text response = %+v, want accepted", resp)
	}
	output, _ := rt.Accepted()[0].Parsed.Payload["script_output"].(string)
	if !strings.Contains(output, "[redacted]") || strings.Contains(output, "must-not-surface") {
		t.Fatalf("script_output = %q, want redacted text", output)
	}
}

func TestWebhookRuntime_RouteScriptTimeoutAndCancellation(t *testing.T) {
	home := t.TempDir()
	script := installWebhookScriptHelper(t, home)
	rt := newScriptRuntime(home, filepath.Base(script), 20*time.Millisecond)
	if resp := rt.Handle("script", scriptRequest("timeout-1", "sleep")); resp.StatusCode != 200 || resp.Reason != "script" {
		t.Fatalf("timeout response = %+v, want ignored script", resp)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if resp := rt.HandleContext(ctx, "script", scriptRequest("cancel-1", "sleep")); resp.StatusCode != 200 || resp.Reason != "script" {
		t.Fatalf("cancel response = %+v, want ignored script", resp)
	}
}

func TestWebhookRuntime_RouteScriptIgnoredDeliveryIDRemainsReusable(t *testing.T) {
	home := t.TempDir()
	script := installWebhookScriptHelper(t, home)
	path := filepath.Join(home, DynamicRoutesFilename)
	writeDynamicRoutes(t, path, `{"script":{"secret":"INSECURE_NO_AUTH","script":"missing","prompt":"Task: {body}"}}`)
	rt := NewRuntime(RuntimeConfig{Routes: NewDynamicRouteSet(home, nil), ProfileHome: home, MaxBodyBytes: 1024})
	req := InboundRequest{
		Headers: map[string]string{"X-GitHub-Delivery": "reusable-script-1"},
		Body:    []byte(`{"task":{"content":"pay bills"},"script_test_mode":"transform"}`),
	}
	if resp := rt.Handle("script", req); resp.StatusCode != 200 || resp.Reason != "script" {
		t.Fatalf("missing-script response = %+v, want ignored script", resp)
	}

	writeDynamicRoutes(t, path, fmt.Sprintf(`{"script":{"secret":"INSECURE_NO_AUTH","script":%q,"prompt":"Task: {body}"}}`, filepath.Base(script)))
	next := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, next, next); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}
	if resp := rt.Handle("script", req); resp.StatusCode != 202 || resp.Status != "accepted" {
		t.Fatalf("reused delivery response = %+v, want accepted", resp)
	}
	if got := rt.Accepted()[0].Prompt; got != "Task: PAY BILLS" {
		t.Fatalf("Prompt = %q, want transformed prompt", got)
	}
}

func newScriptRuntime(home, script string, timeout time.Duration) *Runtime {
	return NewRuntime(RuntimeConfig{
		Routes: NewDynamicRouteSet(home, map[string]RouteConfig{
			"script": {Secret: InsecureNoAuth, Script: script, Prompt: "Task: {body}"},
		}),
		ProfileHome:   home,
		ScriptTimeout: timeout,
		MaxBodyBytes:  1024,
	})
}

func scriptRequest(deliveryID, mode string) InboundRequest {
	body, _ := json.Marshal(map[string]any{
		"task":             map[string]any{"content": "pay bills"},
		"script_test_mode": mode,
	})
	return InboundRequest{
		Headers:       map[string]string{"X-GitHub-Delivery": deliveryID},
		Body:          body,
		ContentLength: int64(len(body)),
	}
}

func installWebhookScriptHelper(t *testing.T, home string) string {
	t.Helper()
	dir := filepath.Join(home, "scripts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll(scripts) error = %v", err)
	}
	name := webhookScriptHelperPrefix
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	dst := filepath.Join(dir, name)
	src, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable() error = %v", err)
	}
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("Open(test binary) error = %v", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		t.Fatalf("OpenFile(helper) error = %v", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		t.Fatalf("Copy(helper) error = %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("Close(helper) error = %v", err)
	}
	return dst
}

func runWebhookScriptHelper() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(2)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		os.Exit(2)
	}
	mode, _ := payload["script_test_mode"].(string)
	switch mode {
	case "empty":
		return
	case "silent":
		fmt.Print("[SILENT]")
		return
	case "silent_object":
		payload = map[string]any{"[SILENT]": true}
	case "ignore_object":
		payload = map[string]any{"__hermes_ignore__": true}
	case "array":
		_ = json.NewEncoder(os.Stdout).Encode([]string{"not", "an", "object"})
		return
	case "nonzero":
		fmt.Fprint(os.Stderr, "private failure detail")
		os.Exit(7)
	case "huge_stdout":
		_, _ = io.WriteString(os.Stdout, strings.Repeat("x", maxRouteScriptStdout+1))
		return
	case "huge_stderr":
		_, _ = io.WriteString(os.Stderr, strings.Repeat("x", maxRouteScriptStderr+1))
	case "sleep":
		time.Sleep(2 * time.Second)
	case "text":
		fmt.Print("message API_KEY=must-not-surface")
		return
	case "evidence":
		cwd, _ := os.Getwd()
		envKeys := make([]string, 0, len(os.Environ()))
		for _, entry := range os.Environ() {
			key, _, _ := strings.Cut(entry, "=")
			envKeys = append(envKeys, key)
		}
		sort.Strings(envKeys)
		payload = map[string]any{
			"argv_count":    len(os.Args) - 1,
			"cwd":           cwd,
			"home":          os.Getenv("HOME"),
			"gormes_home":   os.Getenv("GORMES_HOME"),
			"hermes_home":   os.Getenv("HERMES_HOME"),
			"parent_secret": os.Getenv("WEBHOOK_PARENT_SECRET"),
			"env_keys":      envKeys,
			"stdin":         string(raw),
		}
	default:
		task, _ := payload["task"].(map[string]any)
		payload["body"] = strings.ToUpper(strings.TrimSpace(webhookScriptString(task["content"])))
	}
	if err := json.NewEncoder(os.Stdout).Encode(payload); err != nil {
		os.Exit(3)
	}
}

func interfaceStrings(value any) []string {
	values, _ := value.([]any)
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func webhookScriptString(value any) string {
	text, _ := value.(string)
	return text
}
