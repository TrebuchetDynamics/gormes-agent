package profiles

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

func TestProfileModuleCommandUsesInjectedSeamsAndBuildProvenance(t *testing.T) {
	var writes []string
	cmd := NewCommandWithSeams(Seams{
		ReadActiveProfileName: func() (string, error) { return "default", nil },
		ValidateProfileName:   cli.ValidateProfileName,
		ResolveProfileRoot: func(name string) (string, error) {
			return "/home/operator-secret/.gormes/profiles/" + name, nil
		},
		WriteActiveProfile: func(name string) error {
			writes = append(writes, name)
			return nil
		},
		ListKnownProfiles: func() ([]string, error) {
			return []string{"default", "work"}, nil
		},
	}, Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
		},
	})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"use", "work", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("profile use --json: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if len(writes) != 1 || writes[0] != "work" {
		t.Fatalf("WriteActiveProfile calls = %v, want [work]", writes)
	}
	if strings.Contains(stdout.String()+stderr.String(), "/home/operator-secret") {
		t.Fatalf("profile module leaked raw root:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Action string `json:"action"`
		Active string `json:"active"`
		Root   string `json:"root"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("profile module stdout must be JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != "test-version" || got.Build.GitCommit != "test-sha" {
		t.Fatalf("build provenance = %+v, want injected test values", got.Build)
	}
	if got.Action != "use" || got.Active != "work" || got.Root != ".../work" {
		t.Fatalf("profile JSON = %+v, want action=use active=work root=.../work", got)
	}
}
