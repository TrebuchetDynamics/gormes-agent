package tuiadapter

import (
	"context"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestRuntimeBundleAppliesSessionAdaptersOnly(t *testing.T) {
	branch := func(context.Context, tui.BranchRequest) (tui.BranchResult, error) { return tui.BranchResult{}, nil }
	export := func(context.Context, string) (string, error) { return "", nil }
	title := func(string, string) (tui.SessionTitleResult, error) { return tui.SessionTitleResult{}, nil }
	directory := func(int) ([]tui.SessionDirectoryEntry, error) { return nil, nil }
	resume := func(context.Context, string) (tui.SessionResumeResult, error) { return tui.SessionResumeResult{}, nil }
	tree := func(context.Context, tui.SessionTreeRequest) (tui.SessionTreeResult, error) {
		return tui.SessionTreeResult{}, nil
	}
	label := func(context.Context, tui.SessionTreeLabelRequest) (tui.SessionTreeLabelResult, error) {
		return tui.SessionTreeLabelResult{}, nil
	}
	restore := func(context.Context, tui.SessionTreeRestoreRequest) (tui.SessionTreeRestoreResult, error) {
		return tui.SessionTreeRestoreResult{}, nil
	}
	reset := func() error { return nil }

	skin := func(tui.SkinConfigRequest) (tui.SkinConfigResult, error) {
		return tui.SkinConfigResult{Name: "ares"}, nil
	}
	opts := tui.Options{SkinConfig: skin, ModelName: "gpt-test"}
	bundle := RuntimeBundle{
		Session: SessionBundle{
			Export:      export,
			Branch:      branch,
			Title:       title,
			Directory:   directory,
			Resume:      resume,
			Tree:        tree,
			TreeLabel:   label,
			TreeRestore: restore,
			Reset:       reset,
		},
	}
	bundle.Apply(&opts)

	if opts.SessionExport == nil || opts.SessionBranch == nil || opts.SessionTitle == nil || opts.SessionDirectory == nil || opts.SessionResume == nil || opts.SessionTree == nil || opts.SessionTreeLabel == nil || opts.SessionTreeRestore == nil || opts.SessionReset == nil {
		t.Fatalf("session adapters not fully applied: %+v", opts)
	}
	if opts.SkinConfig == nil || opts.ModelName != "gpt-test" {
		t.Fatalf("non-session options were clobbered: %+v", opts)
	}
}

func TestRuntimeBundleAppliesModelAdaptersWithoutClobberingSession(t *testing.T) {
	setModel := func(string, string) error { return nil }
	catalog := func() ([]tui.ModelPickerCatalogProvider, error) { return nil, nil }
	export := func(context.Context, string) (string, error) { return "", nil }

	opts := tui.Options{SessionExport: export, SkinName: "ares"}
	bundle := RuntimeBundle{
		Model: ModelBundle{
			SetSessionModel: setModel,
			Catalog:         catalog,
			Provider:        "openai-codex",
			Name:            "gpt-5.5",
		},
	}
	bundle.Apply(&opts)

	if opts.SetSessionModelFunc == nil || opts.ModelPickerCatalog == nil || opts.ModelProvider != "openai-codex" || opts.ModelName != "gpt-5.5" {
		t.Fatalf("model adapters not fully applied: %+v", opts)
	}
	if opts.SessionExport == nil || opts.SkinName != "ares" {
		t.Fatalf("non-model options were clobbered: %+v", opts)
	}
}

func TestRuntimeBundleAppliesPresentationAdaptersWithoutClobberingTools(t *testing.T) {
	voice := func(tui.VoiceToggleRequest) (tui.VoiceToggleResult, error) { return tui.VoiceToggleResult{}, nil }
	skin := func(tui.SkinConfigRequest) (tui.SkinConfigResult, error) { return tui.SkinConfigResult{}, nil }
	configure := func(tui.ToolsConfigureRequest) (tui.ToolsConfigureResult, error) {
		return tui.ToolsConfigureResult{}, nil
	}

	opts := tui.Options{ToolsConfigure: configure, ModelName: "gpt-test"}
	bundle := RuntimeBundle{
		Presentation: PresentationBundle{
			VoiceRecordKey: "ctrl+v",
			VoiceToggle:    voice,
			SkinName:       "mono",
			SkinConfig:     skin,
		},
	}
	bundle.Apply(&opts)

	if opts.VoiceRecordKey != "ctrl+v" || opts.VoiceToggle == nil || opts.SkinName != "mono" || opts.SkinConfig == nil {
		t.Fatalf("presentation adapters not fully applied: %+v", opts)
	}
	if opts.ToolsConfigure == nil || opts.ModelName != "gpt-test" {
		t.Fatalf("non-presentation options were clobbered: %+v", opts)
	}
}

func TestRuntimeBundleAppliesToolSkillAdaptersWithoutClobberingModel(t *testing.T) {
	configure := func(tui.ToolsConfigureRequest) (tui.ToolsConfigureResult, error) {
		return tui.ToolsConfigureResult{}, nil
	}
	skillsCommand := func(string) string { return "skills ok" }
	reload := func(context.Context) (tui.SkillSlashReloadResult, error) { return tui.SkillSlashReloadResult{}, nil }
	catalog := func() ([]tui.ModelPickerCatalogProvider, error) { return nil, nil }

	opts := tui.Options{ModelPickerCatalog: catalog, ModelName: "gpt-test"}
	bundle := RuntimeBundle{
		ToolSkill: ToolSkillBundle{
			ToolsConfigure:     configure,
			SkillsCommand:      skillsCommand,
			SkillSlashCommands: []skills.SkillSlashCommand{{Name: "plan"}},
			SkillSlashReload:   reload,
		},
	}
	bundle.Apply(&opts)

	if opts.ToolsConfigure == nil || opts.SkillsCommand == nil || len(opts.SkillSlashCommands) != 1 || opts.SkillSlashCommands[0].Name != "plan" || opts.SkillSlashReload == nil {
		t.Fatalf("tool/skill adapters not fully applied: %+v", opts)
	}
	if opts.ModelPickerCatalog == nil || opts.ModelName != "gpt-test" {
		t.Fatalf("non-tool/skill options were clobbered: %+v", opts)
	}
}
