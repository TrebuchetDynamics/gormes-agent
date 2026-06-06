package tuiapp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

const testVersion = "test-version"

func newRootCommand() *cobra.Command {
	return newRootCommandWithRuntime(Runtime{})
}

func newRootCommandWithRuntime(runtime Runtime) *cobra.Command {
	if runtime.Version == "" {
		runtime.Version = testVersion
	}
	if runtime.KanbanCommandOptions.BuildProvenance == nil && runtime.KanbanCommandOptions.ExitCodeError == nil {
		runtime.KanbanCommandOptions = defaultKanbanCommandOptions()
	}
	cmd := gormescli.NewRootCommand(gormescli.RootOptions{
		Version: runtime.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunRootCommand(cmd, args, runtime)
		},
	}, stubRootFactories())
	gormescli.InstallRootRPCModeFlags(cmd)
	return cmd
}

func stubRootFactories() gormescli.CommandFactories {
	factories := gormescli.CommandFactories{}
	for _, name := range gormescli.RootCommandOrder {
		name := name
		factories[name] = func() *cobra.Command {
			return &cobra.Command{Use: name, RunE: func(*cobra.Command, []string) error { return nil }}
		}
	}
	return factories
}

func executeRootCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func writeSetupToolsFixtureConfig(t *testing.T, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config home: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func readCLIPlatformToolsets(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse config: %v\n%s", err, string(data))
	}
	platformToolsets, _ := doc["platform_toolsets"].(map[string]any)
	raw, _ := platformToolsets["cli"].([]any)
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		out = append(out, value.(string))
	}
	return out
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
