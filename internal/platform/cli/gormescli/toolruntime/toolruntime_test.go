package toolruntime

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestRegisterSessionSearchToolAddsSessionSearchDescriptor(t *testing.T) {
	reg := tools.NewRegistry()

	RegisterSessionSearchTool(reg, nil, nil)

	if _, ok := reg.Get("session_search"); !ok {
		t.Fatal("RegisterSessionSearchTool did not register session_search")
	}
}

func TestRegisterKanbanToolsAddsTaskScopedTools(t *testing.T) {
	t.Setenv("GORMES_KANBAN_TASK", "task-123")
	reg := tools.NewRegistry()

	RegisterKanbanTools(reg)

	for _, name := range []string{"kanban_show", "kanban_complete", "kanban_block", "kanban_heartbeat", "kanban_comment", "kanban_create", "kanban_link"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("RegisterKanbanTools did not register %s", name)
		}
	}
	if _, ok := reg.Get("kanban_list"); ok {
		t.Fatal("task-scoped kanban registration unexpectedly registered kanban_list")
	}
}

func TestRegisterDelegationToolDisabledLeavesRegistryUnchanged(t *testing.T) {
	reg := tools.NewRegistry()

	RegisterDelegationTool(DelegationToolOptions{Config: config.Config{}, Registry: reg})

	if _, ok := reg.Get("delegate_task"); ok {
		t.Fatal("disabled delegation registered delegate_task")
	}
}
