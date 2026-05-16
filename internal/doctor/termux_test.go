package doctor

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestCheckTermuxRuntimeReportsPCLikePathAndAndroidLifecycle(t *testing.T) {
	got := CheckTermuxRuntime(TermuxRuntimeOptions{
		Env: map[string]string{
			"TERMUX_VERSION": "0.119.0",
			"PREFIX":         "/data/data/com.termux/files/usr",
			"HOME":           "/data/data/com.termux/files/home",
			"PATH":           "/data/data/com.termux/files/usr/bin:/system/bin",
		},
		LookPath: func(name string) (string, error) {
			switch name {
			case "tmux", "termux-wake-lock", "termux-notification":
				return "/data/data/com.termux/files/usr/bin/" + name, nil
			default:
				return "", exec.ErrNotFound
			}
		},
	})

	if got.Name != "Termux runtime" {
		t.Fatalf("Name = %q, want Termux runtime", got.Name)
	}
	if got.Status != StatusWarn {
		t.Fatalf("Status = %v, want WARN for Android lifecycle caveat", got.Status)
	}
	out := got.Format()
	for _, want := range []string{
		"Termux detected",
		"desktop-like command path ready",
		"install_dir=/data/data/com.termux/files/usr/bin",
		"tmux available",
		"termux-api commands available",
		"run long gateway sessions inside tmux",
		"termux-wake-lock",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("Termux runtime output missing %q:\n%s", want, out)
		}
	}
}

func TestCheckTermuxRuntimeWarnsWhenTmuxIsMissingButKeepsForegroundGatewayPossible(t *testing.T) {
	got := CheckTermuxRuntime(TermuxRuntimeOptions{
		Env: map[string]string{
			"TERMUX_VERSION": "0.119.0",
			"PREFIX":         "/data/data/com.termux/files/usr",
			"HOME":           "/data/data/com.termux/files/home",
			"PATH":           "/data/data/com.termux/files/usr/bin:/system/bin",
		},
		LookPath: func(name string) (string, error) {
			switch name {
			case "termux-wake-lock", "termux-notification":
				return "/data/data/com.termux/files/usr/bin/" + name, nil
			default:
				return "", exec.ErrNotFound
			}
		},
	})

	if got.Status != StatusWarn {
		t.Fatalf("Status = %v, want WARN when tmux is missing", got.Status)
	}
	out := got.Format()
	for _, want := range []string{
		"tmux missing",
		"gateway foreground mode still works",
		"pkg install tmux",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("Termux runtime output missing %q:\n%s", want, out)
		}
	}
}

func TestCheckTermuxRuntimeSkipsNonTermux(t *testing.T) {
	got := CheckTermuxRuntime(TermuxRuntimeOptions{
		Env: map[string]string{
			"HOME": "/home/alice",
			"PATH": "/usr/bin",
		},
		LookPath: func(string) (string, error) {
			return "", errors.New("unexpected lookup")
		},
	})

	if got.Status != StatusSkip {
		t.Fatalf("Status = %v, want SKIP", got.Status)
	}
	if len(got.Items) != 0 {
		t.Fatalf("Items = %+v, want none for non-Termux", got.Items)
	}
}
