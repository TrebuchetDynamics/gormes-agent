package main

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestGatewayKanbanSlashProductionConfigInjectsRunner(t *testing.T) {
	setupNativeTUITestEnv(t)

	mgrCfg := gatewayManagerConfig(
		config.Config{},
		map[string]string{},
		map[string]bool{},
		nil,
		nil,
		nil,
		nil,
		nil,
		gateway.RestartConfig{},
	)
	if mgrCfg.KanbanSlashRunner == nil {
		t.Fatal("KanbanSlashRunner is nil; gateway /kanban would not share the CLI command runner")
	}

	out, err := mgrCfg.KanbanSlashRunner(context.Background(), "/kanban init")
	if err != nil {
		t.Fatalf("KanbanSlashRunner(/kanban init): %v\nout=%s", err, out)
	}
	if !strings.Contains(out, "kanban initialized at") {
		t.Fatalf("KanbanSlashRunner output = %q, want init output", out)
	}
}
