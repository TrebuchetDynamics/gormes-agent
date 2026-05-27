package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestNavivoxPairOpenNavivoxInvokesAndroidActivityManager(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_pair_open_token")

	argsPath := filepath.Join(t.TempDir(), "am.args")
	fakeDir := writeFakeNavivoxAM(t, `#!/bin/sh
printf '%s\n' "$@" > "$NAVIVOX_FAKE_AM_ARGS"
printf '%s\n' "$@"
`)
	t.Setenv("NAVIVOX_FAKE_AM_ARGS", argsPath)
	t.Setenv("PATH", fakeDir)

	port := freeLocalTCPPort(t)
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "navivox", "pair", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--no-wait", "--open-navivox")
	if err != nil {
		t.Fatalf("navivox pair --open-navivox --no-wait: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	token := navivoxPairOutputToken(stdout)
	if token == "" || token == "nvbx_pair_open_token" {
		t.Fatalf("navivox pair should print a fresh temporary token instead of reusing env token; token=%q stdout=%s", token, stdout)
	}
	args := readFakeNavivoxAMArgs(t, argsPath)
	if len(args) == 0 || args[0] != "start" {
		t.Fatalf("am args = %#v, want start invocation", args)
	}
	if got := navivoxArgAfter(args, "-a"); got != "android.intent.action.VIEW" {
		t.Fatalf("am action = %q, want android.intent.action.VIEW; args=%#v", got, args)
	}
	if got := navivoxArgAfter(args, "-c"); got != "android.intent.category.BROWSABLE" {
		t.Fatalf("am category = %q, want android.intent.category.BROWSABLE; args=%#v", got, args)
	}
	if got := navivoxArgAfter(args, "-p"); got != navivoxAndroidPackage {
		t.Fatalf("am package = %q, want %q; args=%#v", got, navivoxAndroidPackage, args)
	}
	descriptor := navivoxArgAfter(args, "-d")
	parsed, err := url.Parse(descriptor)
	if err != nil {
		t.Fatalf("parse descriptor %q: %v", descriptor, err)
	}
	if parsed.Scheme != "navivox" || parsed.Host != "connect" {
		t.Fatalf("descriptor target = %s://%s, want navivox://connect", parsed.Scheme, parsed.Host)
	}
	values := parsed.Query()
	for key, want := range map[string]string{
		"base_url":         fmt.Sprintf("http://127.0.0.1:%d", port),
		"websocket_url":    fmt.Sprintf("ws://127.0.0.1:%d/v1/navivox/stream", port),
		"capabilities_url": fmt.Sprintf("http://127.0.0.1:%d/v1/navivox/capabilities", port),
		"rest_token":       token,
	} {
		if got := values.Get(key); got != want {
			t.Fatalf("descriptor %s = %q, want %q in %s", key, got, want, descriptor)
		}
	}
	combined := stdout + stderr
	for _, forbidden := range []string{"rest_token=", descriptor} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("pair output leaked descriptor material %q:\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
	for _, want := range []string{
		"Navivox pairing ready.",
		"Token: " + token,
		"Opened Navivox.",
		"Keep this terminal open.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, removed := range []string{"Pairing QR image:", "scan the QR image", "Next steps"} {
		if strings.Contains(stdout, removed) {
			t.Fatalf("stdout still contains noisy pair output %q:\n%s", removed, stdout)
		}
	}
}

func TestOpenNavivoxAndroidFallsBackToAndroidShareIntent(t *testing.T) {
	argsPath := filepath.Join(t.TempDir(), "am.args")
	countPath := filepath.Join(t.TempDir(), "am.count")
	fakeDir := writeFakeNavivoxAM(t, `#!/bin/sh
count=0
if [ -f "$NAVIVOX_FAKE_AM_COUNT" ]; then
	IFS= read -r count < "$NAVIVOX_FAKE_AM_COUNT"
fi
count=$((count + 1))
printf '%s' "$count" > "$NAVIVOX_FAKE_AM_COUNT"
printf 'CALL:%s\n' "$count" >> "$NAVIVOX_FAKE_AM_ARGS"
printf '%s\n' "$@" >> "$NAVIVOX_FAKE_AM_ARGS"
if [ "$count" = "1" ]; then
	printf '%s\n' "$@" >&2
	exit 1
fi
exit 0
`)
	t.Setenv("NAVIVOX_FAKE_AM_ARGS", argsPath)
	t.Setenv("NAVIVOX_FAKE_AM_COUNT", countPath)
	t.Setenv("PATH", fakeDir)

	descriptor := "navivox://connect?base_url=http%3A%2F%2F127.0.0.1%3A8765&websocket_url=ws%3A%2F%2F127.0.0.1%3A8765%2Fv1%2Fnavivox%2Fstream&capabilities_url=http%3A%2F%2F127.0.0.1%3A8765%2Fv1%2Fnavivox%2Fcapabilities&auth_mode=pairing_token&token_required=true&setup_handoff=true&rest_token=nvbx_share_token"
	if err := openNavivoxAndroid(context.Background(), descriptor, navivoxAndroidPackage); err != nil {
		t.Fatalf("openNavivoxAndroid fallback: %v", err)
	}
	args := readFakeNavivoxAMArgs(t, argsPath)
	first := navivoxArgsAfterCall(t, args, "CALL:1")
	if got := navivoxArgAfter(first, "-a"); got != "android.intent.action.VIEW" {
		t.Fatalf("first am action = %q, want VIEW; args=%#v", got, first)
	}
	second := navivoxArgsAfterCall(t, args, "CALL:2")
	if got := navivoxArgAfter(second, "-a"); got != "android.intent.action.SEND" {
		t.Fatalf("fallback am action = %q, want SEND; args=%#v", got, second)
	}
	if got := navivoxArgAfter(second, "-t"); got != "text/plain" {
		t.Fatalf("fallback MIME = %q, want text/plain; args=%#v", got, second)
	}
	payload := navivoxExtraTextArg(t, second)
	var got map[string]any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("fallback payload is not JSON: %v\npayload=%s", err, payload)
	}
	for key, want := range map[string]any{
		"base_url":         "http://127.0.0.1:8765",
		"websocket_url":    "ws://127.0.0.1:8765/v1/navivox/stream",
		"capabilities_url": "http://127.0.0.1:8765/v1/navivox/capabilities",
		"auth_mode":        "pairing_token",
		"token_required":   true,
		"setup_handoff":    true,
		"rest_token":       "nvbx_share_token",
	} {
		if got[key] != want {
			t.Fatalf("fallback payload %s = %#v, want %#v in %s", key, got[key], want, payload)
		}
	}
}

func TestOpenNavivoxAndroidRedactsActivityManagerErrors(t *testing.T) {
	fakeDir := writeFakeNavivoxAM(t, `#!/bin/sh
printf '%s\n' "$@" >&2
printf '%s\n' 'pairing_token=nvbx_pairing_extra token=raw-token-extra' >&2
exit 23
`)
	t.Setenv("PATH", fakeDir)
	descriptor := "navivox://connect?base_url=http%3A%2F%2F127.0.0.1%3A8765&rest_token=nvbx_secret_token&token=raw-token&pairing_token=nvbx_pairing_token"
	err := openNavivoxAndroid(context.Background(), descriptor, navivoxAndroidPackage)
	if err == nil {
		t.Fatal("openNavivoxAndroid err = nil, want fake am failure")
	}
	message := err.Error()
	for _, forbidden := range []string{"nvbx_secret_token", "nvbx_pairing_token", "nvbx_pairing_extra", "nvbx_", "rest_token=", "token=raw-token", "pairing_token=", "raw-token"} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("openNavivoxAndroid error leaked %q: %s", forbidden, message)
		}
	}
}

func TestNavivoxPairOpenNavivoxMissingAMFallsBackToQR(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_missing_am_token")
	t.Setenv("PATH", t.TempDir())

	port := freeLocalTCPPort(t)
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "navivox", "pair", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--no-wait", "--open-navivox")
	if err != nil {
		t.Fatalf("navivox pair with missing am should keep QR fallback: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	token := navivoxPairOutputToken(stdout)
	if token == "" || token == "nvbx_missing_am_token" {
		t.Fatalf("navivox pair should print a fresh temporary token instead of reusing env token; token=%q stdout=%s", token, stdout)
	}
	for _, want := range []string{
		"Open failed; use QR.",
		"QR: " + filepath.Join(home, "navivox", "pairing.png"),
		"Token: " + token,
		"Waiting for Navivox connection skipped (--no-wait).",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stderr, token) || strings.Contains(stdout, "nvbx_missing_am_token") || strings.Contains(stdout+stderr, "rest_token=") {
		t.Fatalf("fallback output leaked descriptor token material:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestNavivoxPairPrintDeeplinkRequiresExplicitFlag(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_print_deeplink_token")

	port := freeLocalTCPPort(t)
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "navivox", "pair", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--no-wait")
	if err != nil {
		t.Fatalf("navivox pair default: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	token := navivoxPairOutputToken(stdout)
	if token == "" || token == "nvbx_print_deeplink_token" {
		t.Fatalf("default pair output should print a fresh temporary token for manual entry; token=%q\nstdout=%s", token, stdout)
	}
	for _, forbidden := range []string{"navivox://connect?", "rest_token="} {
		if strings.Contains(stdout+stderr, forbidden) {
			t.Fatalf("default pair output leaked %q:\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}

	port = freeLocalTCPPort(t)
	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "navivox", "pair", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--no-wait", "--print-deeplink")
	if err != nil {
		t.Fatalf("navivox pair --print-deeplink: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	printedToken := navivoxPairOutputToken(stdout)
	if printedToken == "" || printedToken == "nvbx_print_deeplink_token" {
		t.Fatalf("--print-deeplink should print a fresh temporary token; token=%q\nstdout=%s", printedToken, stdout)
	}
	for _, want := range []string{
		"Warning: navivox://connect descriptor contains a secret; do not share it.",
		"navivox://connect?",
		"rest_token=" + printedToken,
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("--print-deeplink stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestNavivoxConnectOpenNavivoxUsesFirstReachableEntry(t *testing.T) {
	argsPath := filepath.Join(t.TempDir(), "am.args")
	fakeDir := writeFakeNavivoxAM(t, `#!/bin/sh
printf '%s\n' "$@" > "$NAVIVOX_FAKE_AM_ARGS"
`)
	t.Setenv("NAVIVOX_FAKE_AM_ARGS", argsPath)
	t.Setenv("PATH", fakeDir)

	cmd, buf := newConnectInfoTestCommand(t)
	cfg := config.Config{Navivox: config.NavivoxCfg{
		Enabled:      true,
		BindHost:     "127.0.0.1",
		Port:         8765,
		ExposureMode: config.NavivoxExposureLocal,
		AuthMode:     config.NavivoxAuthStaticToken,
		Token:        "nvbx_connect_open_token",
	}}
	if err := runNavivoxConnectInfoForConfigWithOptions(cmd, cfg, navivoxConnectInfoOptions{openNavivox: true, androidPackage: navivoxAndroidPackage}); err != nil {
		t.Fatalf("navivox connect --open-navivox: %v\noutput=%s", err, buf.String())
	}
	args := readFakeNavivoxAMArgs(t, argsPath)
	descriptor := navivoxArgAfter(args, "-d")
	parsed, err := url.Parse(descriptor)
	if err != nil {
		t.Fatalf("parse descriptor %q: %v", descriptor, err)
	}
	values := parsed.Query()
	for key, want := range map[string]string{
		"base_url":         "http://127.0.0.1:8765",
		"websocket_url":    "ws://127.0.0.1:8765/v1/navivox/stream",
		"capabilities_url": "http://127.0.0.1:8765/v1/navivox/capabilities",
		"rest_token":       "nvbx_connect_open_token",
	} {
		if got := values.Get(key); got != want {
			t.Fatalf("descriptor %s = %q, want %q in %s", key, got, want, descriptor)
		}
	}
	out := buf.String()
	for _, want := range []string{
		"Opening Navivox directly...",
		"Navivox connect URLs:",
		"Use `gormes navivox pair` for the one-terminal QR pairing flow.",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("connect output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Scan this QR from Navivox:") || strings.ContainsAny(out, "█▀▄") {
		t.Fatalf("connect output should not duplicate the terminal QR; use navivox pair for QR:\n%s", out)
	}
	if strings.Contains(out, "nvbx_connect_open_token") || strings.Contains(out, "rest_token=") || strings.Contains(out, descriptor) {
		t.Fatalf("connect output leaked descriptor/token material:\n%s", out)
	}
}

func writeFakeNavivoxAM(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "am")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake am: %v", err)
	}
	return dir
}

func readFakeNavivoxAMArgs(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fake am args: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

func navivoxArgAfter(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func navivoxArgsAfterCall(t *testing.T, lines []string, marker string) []string {
	t.Helper()
	for i, line := range lines {
		if line != marker {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(lines[j], "CALL:") {
				end = j
				break
			}
		}
		return lines[i+1 : end]
	}
	t.Fatalf("missing %s in am args %#v", marker, lines)
	return nil
}

func navivoxExtraTextArg(t *testing.T, args []string) string {
	t.Helper()
	for i, arg := range args {
		if arg == "--es" && i+2 < len(args) && args[i+1] == "android.intent.extra.TEXT" {
			return args[i+2]
		}
	}
	t.Fatalf("missing --es android.intent.extra.TEXT in args %#v", args)
	return ""
}
