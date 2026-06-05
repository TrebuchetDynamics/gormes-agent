package tuiadapter

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

// RuntimeBundle groups command-layer runtime adapters before applying them to
// tui.Options. Keep construction outside this package so internal/tui stays a
// display/input model and this package remains free of config, kernel, and DB
// ownership.
type RuntimeBundle struct {
	Presentation PresentationBundle
	Session      SessionBundle
	Model        ModelBundle
	ToolSkill    ToolSkillBundle
}

func (b RuntimeBundle) Apply(opts *tui.Options) {
	if opts == nil {
		return
	}
	b.Presentation.Apply(opts)
	b.Session.Apply(opts)
	b.Model.Apply(opts)
	b.ToolSkill.Apply(opts)
}

type PresentationBundle struct {
	VoiceRecordKey string
	VoiceToggle    tui.VoiceToggleFunc
	SkinName       string
	SkinConfig     tui.SkinConfigFunc
}

func (b PresentationBundle) Apply(opts *tui.Options) {
	if opts == nil {
		return
	}
	if b.VoiceRecordKey != "" {
		opts.VoiceRecordKey = b.VoiceRecordKey
	}
	if b.VoiceToggle != nil {
		opts.VoiceToggle = b.VoiceToggle
	}
	if b.SkinName != "" {
		opts.SkinName = b.SkinName
	}
	if b.SkinConfig != nil {
		opts.SkinConfig = b.SkinConfig
	}
}

type SessionBundle struct {
	Export      tui.SessionExportFunc
	Branch      tui.SessionBranchFunc
	Title       tui.SessionTitleFunc
	Directory   tui.SessionDirectoryFunc
	Resume      tui.SessionResumeFunc
	Tree        tui.SessionTreeFunc
	TreeLabel   tui.SessionTreeLabelFunc
	TreeRestore tui.SessionTreeRestoreFunc
	Reset       tui.SessionResetFunc
}

func (b SessionBundle) Apply(opts *tui.Options) {
	if opts == nil {
		return
	}
	if b.Export != nil {
		opts.SessionExport = b.Export
	}
	if b.Branch != nil {
		opts.SessionBranch = b.Branch
	}
	if b.Title != nil {
		opts.SessionTitle = b.Title
	}
	if b.Directory != nil {
		opts.SessionDirectory = b.Directory
	}
	if b.Resume != nil {
		opts.SessionResume = b.Resume
	}
	if b.Tree != nil {
		opts.SessionTree = b.Tree
	}
	if b.TreeLabel != nil {
		opts.SessionTreeLabel = b.TreeLabel
	}
	if b.TreeRestore != nil {
		opts.SessionTreeRestore = b.TreeRestore
	}
	if b.Reset != nil {
		opts.SessionReset = b.Reset
	}
}

type ModelBundle struct {
	SetSessionModel tui.SetSessionModelFunc
	Catalog         tui.ModelPickerCatalogFunc
	Provider        string
	Name            string
}

func (b ModelBundle) Apply(opts *tui.Options) {
	if opts == nil {
		return
	}
	if b.SetSessionModel != nil {
		opts.SetSessionModelFunc = b.SetSessionModel
	}
	if b.Catalog != nil {
		opts.ModelPickerCatalog = b.Catalog
	}
	if b.Provider != "" {
		opts.ModelProvider = b.Provider
	}
	if b.Name != "" {
		opts.ModelName = b.Name
	}
}

type ToolSkillBundle struct {
	ToolsConfigure     tui.ToolsConfigureFunc
	SkillsCommand      func(string) string
	SkillSlashCommands []skills.SkillSlashCommand
	SkillSlashReload   tui.SkillSlashReloadFunc
}

func (b ToolSkillBundle) Apply(opts *tui.Options) {
	if opts == nil {
		return
	}
	if b.ToolsConfigure != nil {
		opts.ToolsConfigure = b.ToolsConfigure
	}
	if b.SkillsCommand != nil {
		opts.SkillsCommand = b.SkillsCommand
	}
	if b.SkillSlashCommands != nil {
		opts.SkillSlashCommands = append([]skills.SkillSlashCommand(nil), b.SkillSlashCommands...)
	}
	if b.SkillSlashReload != nil {
		opts.SkillSlashReload = b.SkillSlashReload
	}
}
