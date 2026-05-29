package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	gormesgoncho "github.com/TrebuchetDynamics/goncho/integration/gormes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/support/testutil/modassert"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestGormesGonchoDependencyUsesLatestPublicRelease(t *testing.T) {
	modassert.RequirePublicModuleVersion(t, "github.com/TrebuchetDynamics/goncho", "v0.2.0")
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
