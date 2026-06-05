package providers

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

const Version = "test-version"

type rootRuntime struct{}

func newRootCommandWithRuntime(rootRuntime) *cobra.Command {
	factories := gormescli.CommandFactories{}
	for _, name := range gormescli.RootCommandOrder {
		name := name
		factories[name] = func() *cobra.Command {
			return &cobra.Command{Use: name, RunE: func(*cobra.Command, []string) error { return nil }}
		}
	}
	factories["auth"] = func() *cobra.Command { return NewAuthCommand(testProviderCommandOptions()) }
	factories["logout"] = func() *cobra.Command {
		return gormescli.NewLogoutCommand(gormescli.LogoutSeams{
			NormalizeAuthProvider: NormalizeAuthProvider,
			ConfiguredProvider: func() (string, error) {
				return gormescli.ConfiguredLogoutProvider(NormalizeAuthProvider)
			},
			RunAuthLogout: RunAuthLogoutCommand,
			ResetProviderIfMatching: func(provider string) error {
				return gormescli.ResetLogoutProviderIfMatching(provider, NormalizeAuthProvider)
			},
		}, gormescli.LogoutOptions{BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: Version, GitCommit: "test-git"}
		}})
	}
	return gormescli.NewRootCommand(gormescli.RootOptions{Version: Version}, factories)
}

func testProviderCommandOptions() Options {
	return Options{BuildProvenance: func() gormescli.BuildProvenance {
		return gormescli.BuildProvenance{Version: Version, GitCommit: "test-git"}
	}}
}

func executeOneshotFlagCommand(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func setupOneshotFlagTestEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex"))
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_MODEL", "")
	t.Setenv("GORMES_API_KEY", "")
	t.Setenv("GORMES_INFERENCE_MODEL", "")
	t.Setenv("GORMES_INFERENCE_PROVIDER", "")
}

func writeOneshotFlagConfig(t *testing.T, data []byte) {
	t.Helper()
	path := config.ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}
