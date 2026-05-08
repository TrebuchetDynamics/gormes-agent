package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadProfileDistributionManifestDefaults(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "distribution.yaml"), []byte(`
name: telemetry
description: Compliance monitor
env_requires:
  - name: OPENAI_API_KEY
    description: OpenAI key
  - name: GRAPH_URL
    required: false
    default: http://127.0.0.1:8000
distribution_owned:
  - SOUL.md
  - skills/
`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	got, ok, err := ReadProfileDistributionManifest(root)
	if err != nil {
		t.Fatalf("ReadProfileDistributionManifest: %v", err)
	}
	if !ok {
		t.Fatal("manifest should be present")
	}
	if got.Name != "telemetry" {
		t.Fatalf("Name = %q, want telemetry", got.Name)
	}
	if got.Version != "0.1.0" {
		t.Fatalf("Version = %q, want default 0.1.0", got.Version)
	}
	if len(got.EnvRequires) != 2 {
		t.Fatalf("EnvRequires len = %d, want 2", len(got.EnvRequires))
	}
	if !got.EnvRequires[0].Required {
		t.Fatalf("first env requirement should default to required")
	}
	if got.EnvRequires[1].Required {
		t.Fatalf("second env requirement should honor required: false")
	}
	if got.EnvRequires[1].Default == nil || *got.EnvRequires[1].Default != "http://127.0.0.1:8000" {
		t.Fatalf("second env default = %#v, want URL", got.EnvRequires[1].Default)
	}
	if want := []string{"SOUL.md", "skills"}; len(got.DistributionOwned) != len(want) || got.DistributionOwned[1] != want[1] {
		t.Fatalf("DistributionOwned = %#v, want %#v", got.DistributionOwned, want)
	}
}

func TestReadProfileDistributionManifestMissing(t *testing.T) {
	got, ok, err := ReadProfileDistributionManifest(t.TempDir())
	if err != nil {
		t.Fatalf("missing manifest err = %v, want nil", err)
	}
	if ok {
		t.Fatalf("ok = true, want false with zero manifest %#v", got)
	}
}

func TestReadProfileDistributionManifestRejectsMissingName(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "distribution.yaml"), []byte("version: 1.0.0\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ReadProfileDistributionManifest(root)
	if !errors.Is(err, ErrProfileDistributionInvalid) {
		t.Fatalf("err = %v, want ErrProfileDistributionInvalid", err)
	}
}
