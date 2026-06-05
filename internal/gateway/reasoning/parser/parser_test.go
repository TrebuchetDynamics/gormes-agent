package parser

import (
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/reasoning/model"
)

func TestParse_ShowFormReturnsActionShow(t *testing.T) {
	cmd, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse(nil) err = %v, want nil", err)
	}
	if cmd.Action != model.ReasoningActionShow {
		t.Fatalf("Action = %v, want ReasoningActionShow", cmd.Action)
	}
	if cmd.Effort != model.ReasoningEffort("") {
		t.Fatalf("Effort = %q, want empty", cmd.Effort)
	}
	if cmd.Global {
		t.Fatalf("Global = true, want false")
	}

	cmd, err = Parse([]string{})
	if err != nil {
		t.Fatalf("Parse([]) err = %v, want nil", err)
	}
	if cmd.Action != model.ReasoningActionShow {
		t.Fatalf("Action = %v, want ReasoningActionShow", cmd.Action)
	}
}

func TestParse_SetSessionScoped(t *testing.T) {
	for _, effort := range []string{"high", "low", "medium"} {
		t.Run(effort, func(t *testing.T) {
			cmd, err := Parse([]string{effort})
			if err != nil {
				t.Fatalf("Parse([%q]) err = %v, want nil", effort, err)
			}
			if cmd.Action != model.ReasoningActionSet {
				t.Fatalf("Action = %v, want ReasoningActionSet", cmd.Action)
			}
			if cmd.Effort != model.ReasoningEffort(effort) {
				t.Fatalf("Effort = %q, want %q", cmd.Effort, effort)
			}
			if cmd.Global {
				t.Fatalf("Global = true, want false")
			}
		})
	}
}

func TestParse_SetGlobal(t *testing.T) {
	cmd, err := Parse([]string{"low", "--global"})
	if err != nil {
		t.Fatalf("Parse([low --global]) err = %v, want nil", err)
	}
	if cmd.Action != model.ReasoningActionSet {
		t.Fatalf("Action = %v, want ReasoningActionSet", cmd.Action)
	}
	if cmd.Effort != model.ReasoningEffort("low") {
		t.Fatalf("Effort = %q, want low", cmd.Effort)
	}
	if !cmd.Global {
		t.Fatalf("Global = false, want true")
	}
}

func TestParse_ResetSession(t *testing.T) {
	cmd, err := Parse([]string{"reset"})
	if err != nil {
		t.Fatalf("Parse([reset]) err = %v, want nil", err)
	}
	if cmd.Action != model.ReasoningActionReset {
		t.Fatalf("Action = %v, want ReasoningActionReset", cmd.Action)
	}
	if cmd.Global {
		t.Fatalf("Global = true, want false")
	}
	if cmd.Effort != model.ReasoningEffort("") {
		t.Fatalf("Effort = %q, want empty", cmd.Effort)
	}
}

func TestParse_RejectGlobalReset(t *testing.T) {
	_, err := Parse([]string{"reset", "--global"})
	if err == nil {
		t.Fatalf("Parse([reset --global]) err = nil, want ErrResetGlobalUnsupported")
	}
	if !errors.Is(err, model.ErrResetGlobalUnsupported) {
		t.Fatalf("err = %v, want ErrResetGlobalUnsupported", err)
	}
}

func TestParse_RejectInvalidEffort(t *testing.T) {
	_, err := Parse([]string{"bogus"})
	if err == nil {
		t.Fatalf("Parse([bogus]) err = nil, want ErrInvalidEffort")
	}
	if !errors.Is(err, model.ErrInvalidEffort) {
		t.Fatalf("err = %v, want ErrInvalidEffort", err)
	}
}
