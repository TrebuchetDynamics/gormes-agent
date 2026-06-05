package skillsslash

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
)

func TestHandleReload(t *testing.T) {
	unavailable := HandleReload(context.Background(), nil)
	if !unavailable.Handled || unavailable.Rebuild || !strings.Contains(unavailable.Status, "unavailable") {
		t.Fatalf("HandleReload unavailable = %+v, want consumed unavailable status", unavailable)
	}

	failed := HandleReload(context.Background(), func(context.Context) (ReloadResult, error) {
		return ReloadResult{}, errors.New("scan failed")
	})
	if failed.Rebuild || !strings.Contains(failed.Status, "scan failed") {
		t.Fatalf("HandleReload failed = %+v, want error status without rebuild", failed)
	}

	commands := []skills.SkillSlashCommand{{Command: "/fresh-skill", Name: "fresh-skill"}}
	loaded := HandleReload(context.Background(), func(context.Context) (ReloadResult, error) {
		return ReloadResult{Commands: commands}, nil
	})
	if !loaded.Rebuild || len(loaded.Commands) != 1 || !strings.Contains(loaded.Status, "1 skill(s) available") {
		t.Fatalf("HandleReload loaded = %+v, want rebuild with default status", loaded)
	}
	commands[0].Name = "mutated"
	if loaded.Commands[0].Name != "fresh-skill" {
		t.Fatalf("HandleReload commands aliased caller slice: %+v", loaded.Commands)
	}
}
