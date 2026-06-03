package profile

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func TestNewCommandWithSeamsIncludesSeedSubcommand(t *testing.T) {
	cmd := NewCommandWithSeams(testSeams(), Options{BuildProvenance: func() BuildProvenance {
		return BuildProvenance{Version: "test", GitCommit: "abc"}
	}})
	if cmd.Use != "profile" {
		t.Fatalf("Use = %q, want profile", cmd.Use)
	}
	seed, _, err := cmd.Find([]string{"seed"})
	if err != nil {
		t.Fatalf("Find seed: %v", err)
	}
	if seed == nil || seed.Use == "" {
		t.Fatalf("seed command not registered")
	}
}

func TestProfileSeedSeamsUsesInjectedBuildProvenance(t *testing.T) {
	seams := ProfileSeedSeamsFromProfileSeams(testSeams(), func() BuildProvenance {
		return BuildProvenance{Version: "test", GitCommit: "abc"}
	})
	got := seams.BuildProvenance()
	if got.Version != "test" || got.GitCommit != "abc" {
		t.Fatalf("BuildProvenance = %+v", got)
	}
}

func testSeams() Seams {
	return Seams{
		ReadActiveProfileName: func() (string, error) { return "main", nil },
		ValidateProfileName:   cli.ValidateProfileName,
		ResolveProfileRoot:    func(name string) (string, error) { return "/tmp/gormes-test/profiles/" + name, nil },
		WriteActiveProfile:    func(name string) error { return nil },
		CreateProfile: func(name string, cloneAll bool) (cli.ProfileCreateResult, error) {
			return cli.ProfileCreateResult{Name: name, Root: "/tmp/gormes-test/profiles/" + name, CloneAll: cloneAll}, nil
		},
		ListKnownProfiles: func() ([]string, error) { return []string{"main"}, nil },
	}
}
