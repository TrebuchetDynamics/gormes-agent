package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/doctor"
)

func TestDoctorCustomEndpointAllSet(t *testing.T) {
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Endpoint: "https://example.invalid",
			APIKey:   "secret",
			Model:    "m",
		},
	}

	got := doctorCustomEndpointReadiness(cfg)

	if got.Name != "Custom endpoint" {
		t.Fatalf("Name = %q, want %q", got.Name, "Custom endpoint")
	}
	if got.Status != doctor.StatusPass {
		t.Fatalf("Status = %v, want %v", got.Status, doctor.StatusPass)
	}
	if !strings.Contains(got.Summary, "configured") {
		t.Fatalf("Summary = %q, want it to contain %q", got.Summary, "configured")
	}
	for _, item := range got.Items {
		if item.Status == doctor.StatusWarn {
			t.Fatalf("item %q has Status=Warn but expected none flagged: %+v", item.Name, item)
		}
	}
}

func TestDoctorCustomEndpointMissingAPIKey(t *testing.T) {
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Endpoint: "https://example.invalid",
			APIKey:   "",
			Model:    "m",
		},
	}

	got := doctorCustomEndpointReadiness(cfg)

	if got.Status != doctor.StatusWarn {
		t.Fatalf("Status = %v, want %v", got.Status, doctor.StatusWarn)
	}
	apiKey, ok := findItem(got.Items, "api_key")
	if !ok {
		t.Fatalf("missing api_key item in: %+v", got.Items)
	}
	if apiKey.Status != doctor.StatusWarn {
		t.Fatalf("api_key item Status = %v, want %v", apiKey.Status, doctor.StatusWarn)
	}
	if apiKey.Note != "missing" {
		t.Fatalf("api_key item Note = %q, want %q", apiKey.Note, "missing")
	}
}

func TestDoctorCustomEndpointMissingModel(t *testing.T) {
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Endpoint: "https://example.invalid",
			APIKey:   "secret",
			Model:    "",
		},
	}

	got := doctorCustomEndpointReadiness(cfg)

	if got.Status != doctor.StatusFail {
		t.Fatalf("Status = %v, want %v", got.Status, doctor.StatusFail)
	}
	model, ok := findItem(got.Items, "model")
	if !ok {
		t.Fatalf("missing model item in: %+v", got.Items)
	}
	if model.Status != doctor.StatusFail {
		t.Fatalf("model item Status = %v, want %v", model.Status, doctor.StatusFail)
	}
	if model.Note != "missing" {
		t.Fatalf("model item Note = %q, want %q", model.Note, "missing")
	}
}

func TestDoctorCustomEndpointAllEmpty(t *testing.T) {
	cfg := config.Config{Hermes: config.HermesCfg{}}

	got := doctorCustomEndpointReadiness(cfg)

	if got.Status != doctor.StatusWarn {
		t.Fatalf("Status = %v, want %v", got.Status, doctor.StatusWarn)
	}
	if got.Summary != "disabled" {
		t.Fatalf("Summary = %q, want %q", got.Summary, "disabled")
	}
}

func TestDoctorCmdInvokesCustomEndpointReadiness(t *testing.T) {
	setupCustomEndpointDoctorEnv(t)
	t.Setenv("GORMES_ENDPOINT", "https://example.invalid")
	t.Setenv("GORMES_API_KEY", "secret")
	t.Setenv("GORMES_MODEL", "m")

	stdout, err := captureDoctorStdout(t, func() error {
		cmd := newRootCommand()
		cmd.SetArgs([]string{"doctor", "--offline"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s", err, stdout)
	}

	if !strings.Contains(stdout, "[PASS] Custom endpoint:") {
		t.Fatalf("stdout missing [PASS] Custom endpoint: line:\n%s", stdout)
	}
	if !strings.Contains(stdout, "configured") {
		t.Fatalf("stdout missing 'configured' summary:\n%s", stdout)
	}
}

func findItem(items []doctor.ItemInfo, name string) (doctor.ItemInfo, bool) {
	for _, it := range items {
		if it.Name == name {
			return it, true
		}
	}
	return doctor.ItemInfo{}, false
}

func setupCustomEndpointDoctorEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
}

func captureDoctorStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	orig := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(&buf, r)
	}()

	runErr := fn()
	_ = w.Close()
	<-done
	os.Stdout = orig
	return buf.String(), runErr
}
