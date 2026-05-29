package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	gormesgoncho "github.com/TrebuchetDynamics/goncho/integration/gormes"
	"github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/goncho"
)

// newChannelGonchoService constructs a goncho.Service wired for channel runtime
// use. It attaches a Hermes-backed DialecticCaller when a provider client is
// available, enabling in-process honcho_reasoning without an external Honcho
// process.
func newChannelGonchoService(db *sql.DB, cfg goncho.Config, log *slog.Logger, client llm.Client, model string) *goncho.Service {
	svc := goncho.NewService(db, cfg, log)
	if client != nil {
		svc.SetDialecticCaller(NewHermesDialecticCaller(client, model))
	}
	return svc
}

// registerChannelGonchoTools wires honcho_* memory tools onto the agent registry
// backed by the given goncho service. This is the shared entry point all
// channels (Telegram, WhatsApp, Slack, Discord) call to enable memory.
func registerChannelGonchoTools(reg *tools.Registry, svc *goncho.Service) {
	gonchotools.RegisterHonchoTools(reg, svc)
}

// registerGormesGonchoTools wires the public Goncho v0.1.x Gormes adapter
// tools onto the registry. These are the stable goncho_* tools exposed by the
// released github.com/TrebuchetDynamics/goncho/integration/gormes package.
func registerGormesGonchoTools(reg *tools.Registry, mem *gormesgoncho.Runtime) {
	if reg == nil || mem == nil {
		return
	}
	reg.MustRegister(mem.ContextTool)
	reg.MustRegister(mem.SearchTool)
	reg.MustRegister(mem.RememberTool)
	reg.MustRegister(mem.ReviewTool)
	reg.MustRegister(mem.HandoffTool)
}

func formatGormesGonchoStatus(status gormesgoncho.Status) string {
	ready := "unavailable"
	if status.Ready {
		ready = "ready"
	}
	tools := append([]string(nil), status.ToolNames...)
	sort.Strings(tools)
	return fmt.Sprintf("goncho: %s workspace_id=%s observer_id=%s database=%s tools=%s", ready, status.WorkspaceID, status.ObserverID, status.DatabasePath, strings.Join(tools, ","))
}
