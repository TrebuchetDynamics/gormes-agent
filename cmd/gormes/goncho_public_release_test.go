package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	gormesgoncho "github.com/TrebuchetDynamics/goncho/integration/gormes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestGormesGonchoDependencyUsesLatestPublicRelease(t *testing.T) {
	cmd := exec.Command("go", "list", "-m", "-json", "github.com/TrebuchetDynamics/goncho")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list goncho module: %v\n%s", err, out)
	}
	var mod struct {
		Path    string
		Version string
		Replace *struct {
			Path    string
			Version string
		}
	}
	if err := json.Unmarshal(out, &mod); err != nil {
		t.Fatalf("parse go list module JSON: %v\n%s", err, out)
	}
	if mod.Path != "github.com/TrebuchetDynamics/goncho" {
		t.Fatalf("module path = %q, want github.com/TrebuchetDynamics/goncho", mod.Path)
	}
	if mod.Replace != nil {
		t.Fatalf("goncho dependency is replaced by %s %s, want latest public release", mod.Replace.Path, mod.Replace.Version)
	}
	if mod.Version != "v0.1.0" {
		t.Fatalf("goncho version = %q, want latest public release v0.1.0", mod.Version)
	}
}

func TestGormesGonchoPublicRuntimeRegistersToolsAndReportsStatus(t *testing.T) {
	ctx := context.Background()
	mem, err := gormesgoncho.Open(ctx, gormesgoncho.Config{
		DatabasePath: filepath.Join(t.TempDir(), "goncho.db"),
		WorkspaceID:  "gormes",
		ObserverID:   "gormes",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = mem.Close(context.Background()) })

	reg := tools.NewRegistry()
	registerGormesGonchoTools(reg, mem)

	for _, name := range []string{"goncho_context", "goncho_search", "goncho_remember", "goncho_review", "goncho_handoff"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("missing public Goncho tool %q", name)
		}
	}

	status := mem.Status()
	line := formatGormesGonchoStatus(status)
	for _, want := range []string{"goncho: ready", "workspace_id=gormes", "observer_id=gormes", "goncho_context", "goncho_search", "goncho_remember", "goncho_review", "goncho_handoff"} {
		if !strings.Contains(line, want) {
			t.Fatalf("status line %q missing %q", line, want)
		}
	}
}
