package tuiapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
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

func newPlainRemoteTUIModel() tui.Model {
	return tui.NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, tui.Options{})
}

func loadNativeTUITestConfig(t *testing.T) config.Config {
	t.Helper()
	setupNativeTUITestEnv(t)
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	return cfg
}

func captureOfflineTUIProgramOptions(t *testing.T, cfg config.Config) []tea.ProgramOption {
	t.Helper()
	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}
	var captured []tea.ProgramOption
	if err := RunResolved(cmd, Invocation{Config: cfg}, Runtime{
		ProgramFactory: func(_ tea.Model, opts ...tea.ProgramOption) Program {
			captured = append(captured, opts...)
			return fakeTUIProgram{}
		},
	}); err != nil {
		t.Fatalf("RunResolved: %v", err)
	}
	return captured
}

func runOfflineTUIForTest(t *testing.T, cfg config.Config, inspect func(tea.Model)) {
	t.Helper()
	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}
	if err := RunResolved(cmd, Invocation{Config: cfg}, Runtime{
		ProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) Program {
			return fakeTUIProgram{run: func() {
				if inspect != nil {
					inspect(model)
				}
			}}
		},
	}); err != nil {
		t.Fatalf("RunResolved: %v", err)
	}
}

func runRemoteTUIForTest(t *testing.T, modelName string) tea.Model {
	t.Helper()
	setupNativeTUITestEnv(t)
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}

	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)
		frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1, Model: modelName}
		data, _ := json.Marshal(frame)
		fmt.Fprintf(w, "event: frame\ndata: %s\n\n", data)
		flusher.Flush()
		<-r.Context().Done()
	}))
	defer gateway.Close()

	var captured tea.Model
	err = RunResolved(newRootCommand(), Invocation{
		Config:    cfg,
		RemoteURL: gateway.URL,
	}, Runtime{
		ProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) Program {
			captured = model
			return fakeTUIProgram{}
		},
	})
	if err != nil {
		t.Fatalf("RunResolved(remote): %v", err)
	}
	if captured == nil {
		t.Fatal("did not capture remote TUI model")
	}
	return captured
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
