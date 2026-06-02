package navivox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const AndroidPackage = "com.trebuchetdynamics.navivox"

// DefaultOpenAndroid reports whether this environment can launch Navivox with Android's activity manager.
func DefaultOpenAndroid() bool {
	if !AndroidEnvironment() {
		return false
	}
	_, err := exec.LookPath("am")
	return err == nil
}

// AndroidEnvironment reports whether the process appears to run on Android or Termux.
func AndroidEnvironment() bool {
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

// OpenAndroid hands a Navivox descriptor to the Android app by deep link, with a share fallback.
func OpenAndroid(ctx context.Context, descriptor, pkg string) error {
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
		pkg = AndroidPackage
	}
	viewStderr, viewErr := runAndroidActivity(ctx, amPath,
		"start",
		"-a", "android.intent.action.VIEW",
		"-c", "android.intent.category.BROWSABLE",
		"-d", descriptor,
		"-p", pkg,
	)
	if viewErr == nil {
		return nil
	}
	shareStderr, shareErr := runAndroidActivity(ctx, amPath,
		"start",
		"-a", "android.intent.action.SEND",
		"-t", "text/plain",
		"--es", "android.intent.extra.TEXT", SharePayload(descriptor),
		"-p", pkg,
	)
	if shareErr == nil {
		return nil
	}
	return fmt.Errorf(
		"navivox open: %s; %s",
		formatAndroidStartFailure("deep link failed", viewErr, viewStderr),
		formatAndroidStartFailure("share fallback failed", shareErr, shareStderr),
	)
}

func runAndroidActivity(ctx context.Context, amPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, amPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	return stderr.String(), cmd.Run()
}

func formatAndroidStartFailure(label string, err error, stderr string) string {
	redacted := strings.TrimSpace(Redact(stderr))
	if redacted == "" {
		return fmt.Sprintf("%s: %v", label, err)
	}
	return fmt.Sprintf("%s: %v: %s", label, err, redacted)
}

// ShouldOpenAndroid applies the positive/negative CLI flags for Android handoff.
func ShouldOpenAndroid(open, noOpen bool) bool {
	return open && !noOpen
}
