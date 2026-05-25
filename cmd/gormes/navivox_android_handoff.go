package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

const navivoxAndroidPackage = "com.trebuchetdynamics.navivox"

var (
	navivoxDescriptorSecretParamPattern = regexp.MustCompile(`(?i)(^|[?&\s;])(?:rest_token|token|pairing_token)=[^\s&;]+`)
	navivoxDescriptorSecretJSONPattern  = regexp.MustCompile(`(?i)("(?:rest_token|token|pairing_token)"\s*:\s*")[^"]*(")`)
	navivoxTokenValuePattern            = regexp.MustCompile(`nvbx_[A-Za-z0-9._~+\-/%=]*`)
)

func defaultOpenNavivoxAndroid() bool {
	if !navivoxAndroidEnvironment() {
		return false
	}
	_, err := exec.LookPath("am")
	return err == nil
}

func navivoxAndroidEnvironment() bool {
	if runtime.GOOS == "android" {
		return true
	}
	if strings.TrimSpace(os.Getenv("TERMUX_VERSION")) != "" {
		return true
	}
	if strings.Contains(strings.ToLower(os.Getenv("PREFIX")), "com.termux") {
		return true
	}
	return strings.TrimSpace(os.Getenv("ANDROID_ROOT")) != "" && strings.TrimSpace(os.Getenv("ANDROID_DATA")) != ""
}

func openNavivoxAndroid(ctx context.Context, descriptor, pkg string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(descriptor) == "" {
		return fmt.Errorf("navivox open: descriptor is empty")
	}
	amPath, err := exec.LookPath("am")
	if err != nil {
		return fmt.Errorf("navivox open: Android activity manager not found")
	}
	if strings.TrimSpace(pkg) == "" {
		pkg = navivoxAndroidPackage
	}
	viewStderr, viewErr := runNavivoxAndroidActivity(ctx, amPath,
		"start",
		"-a", "android.intent.action.VIEW",
		"-c", "android.intent.category.BROWSABLE",
		"-d", descriptor,
		"-p", pkg,
	)
	if viewErr == nil {
		return nil
	}
	shareStderr, shareErr := runNavivoxAndroidActivity(ctx, amPath,
		"start",
		"-a", "android.intent.action.SEND",
		"-t", "text/plain",
		"--es", "android.intent.extra.TEXT", navivoxDescriptorSharePayload(descriptor),
		"-p", pkg,
	)
	if shareErr == nil {
		return nil
	}
	return fmt.Errorf(
		"navivox open: %s; %s",
		formatNavivoxAndroidStartFailure("deep link failed", viewErr, viewStderr),
		formatNavivoxAndroidStartFailure("share fallback failed", shareErr, shareStderr),
	)
}

func runNavivoxAndroidActivity(ctx context.Context, amPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, amPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	return stderr.String(), cmd.Run()
}

func formatNavivoxAndroidStartFailure(label string, err error, stderr string) string {
	redacted := strings.TrimSpace(redactNavivoxDescriptor(stderr))
	if redacted == "" {
		return fmt.Sprintf("%s: %v", label, err)
	}
	return fmt.Sprintf("%s: %v: %s", label, err, redacted)
}

func navivoxDescriptorSharePayload(descriptor string) string {
	parsed, err := url.Parse(descriptor)
	if err != nil {
		return descriptor
	}
	values := parsed.Query()
	if len(values) == 0 {
		return descriptor
	}
	payload := make(map[string]any, len(values))
	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}
		value := vals[0]
		if navivoxDescriptorShareBoolKey(key) {
			payload[key] = strings.EqualFold(value, "true")
			continue
		}
		payload[key] = value
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return descriptor
	}
	return string(raw)
}

func navivoxDescriptorShareBoolKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "bridge_keepalive_required", "setup_handoff", "token_required":
		return true
	default:
		return false
	}
}

func redactNavivoxDescriptor(text string) string {
	if text == "" {
		return ""
	}
	redacted := navivoxDescriptorSecretParamPattern.ReplaceAllString(text, "${1}[redacted]")
	redacted = navivoxDescriptorSecretJSONPattern.ReplaceAllString(redacted, "${1}[redacted]${2}")
	redacted = navivoxTokenValuePattern.ReplaceAllString(redacted, "[redacted]")
	return redacted
}

func shouldOpenNavivoxAndroid(open, noOpen bool) bool {
	return open && !noOpen
}
