package guidance

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultSoulMDPortsUpstreamDefaultSoulWithGormPersona(t *testing.T) {
	path, ok := upstreamDefaultSoulPath(t)
	if !ok {
		t.Skip("upstream default_soul.py not available; skipping default SOUL drift check")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read upstream default_soul.py: %v", err)
	}
	upstream, ok := extractPythonStringConstant(string(data), "DEFAULT_SOUL_MD")
	if !ok {
		t.Fatalf("could not extract DEFAULT_SOUL_MD from %s", path)
	}
	want := strings.Replace(upstream,
		"Hermes Agent, an intelligent AI assistant created by Nous Research",
		"Gorm, an AI assistant run by gormes, a Go-native Hermes-compatible agent runtime",
		1,
	)
	if DefaultSoulMD != want {
		t.Fatalf("DefaultSoulMD drifted from Hermes DEFAULT_SOUL_MD with Gorm persona substitution\n--- got ---\n%q\n--- want ---\n%q", DefaultSoulMD, want)
	}
}

func TestDefaultSoulMDOwnsGormPersonaBoundary(t *testing.T) {
	for _, want := range []string{"You are Gorm,", "run by gormes"} {
		if !strings.Contains(DefaultSoulMD, want) {
			t.Fatalf("DefaultSoulMD must contain %q:\n%s", want, DefaultSoulMD)
		}
	}
	for _, forbidden := range []string{"Gormes Agent", "Hermes Agent", "Nous Research"} {
		if strings.Contains(DefaultSoulMD, forbidden) {
			t.Fatalf("DefaultSoulMD contains upstream-only identity marker %q:\n%s", forbidden, DefaultSoulMD)
		}
	}
	if DefaultAgentIdentity != DefaultSoulMD {
		t.Fatalf("DefaultAgentIdentity must reuse DefaultSoulMD")
	}
}
