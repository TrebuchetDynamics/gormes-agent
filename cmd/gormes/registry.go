package main

import (
	"context"
	"database/sql"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kanbantools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/sessionsearchtool"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/subagent"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type registryOptions struct {
	searchDB       *sql.DB
	searchSessions sessionsearchtool.SessionSearchDirectory
}

type registryOpt func(*registryOptions)

func withSessionSearch(db *sql.DB, sessions sessionsearchtool.SessionSearchDirectory) registryOpt {
	return func(o *registryOptions) {
		o.searchDB = db
		o.searchSessions = sessions
	}
}

// buildDefaultRegistry returns a Registry populated with Gormes's built-in
// Go-native tools. Context-specific toolsets such as Kanban stay hidden unless
// their runtime gate is active. Consumer forks that want to add domain-specific
// tools call reg.Register on the returned *Registry before passing it into the
// kernel Config.
func buildDefaultRegistry(parentCtx context.Context, cfg config.Config, childClient hermes.Client, childModel string, opts ...registryOpt) *tools.Registry {
	var o registryOptions
	for _, opt := range opts {
		opt(&o)
	}
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})
	reg.MustRegister(&tools.NowTool{})
	reg.MustRegister(&tools.RandIntTool{})
	reg.MustRegister(tools.NewExecuteCodeTool(tools.ExecuteCodeToolConfig{
		ConfigSet:   cfg.CodeExecution.Mode != "",
		ConfigValue: cfg.CodeExecution.Mode,
		DefaultMode: tools.DefaultExecuteCodeMode,
	}))
	fileTools := tools.FileTaskToolConfig{}
	reg.MustRegister(tools.NewReadFileTool(fileTools))
	reg.MustRegister(tools.NewSearchFilesTool(fileTools))
	reg.MustRegister(tools.NewWriteFileTool(fileTools))
	reg.MustRegister(tools.NewPatchTool(fileTools))
	reg.MustRegister(tools.NewTerminalTool(tools.TerminalToolConfig{Workdir: cfg.Terminal.CWD}))
	reg.MustRegister(tools.NewClarifyTool(nil))
	for _, tool := range tools.NewWebTools(tools.WebToolsConfig{
		Browser: tools.BrowserHarnessToolsConfig{
			Env: browserCDPEnv(cfg),
			Budget: tools.ToolResultBudgetConfig{
				OutputDir:       filepath.Join(filepath.Dir(config.ToolAuditLogPath()), "browser-artifacts"),
				TextBudgetBytes: 8 * 1024,
				PreviewBytes:    1024,
			},
		},
		Backend: tools.WebBackendConfig{
			Backend:             cfg.Web.Backend,
			UseGateway:          cfg.Web.UseGateway,
			ManagedToolsEnabled: true,
			AuthStorePath:       filepath.Join(config.GormesHome(), "auth.json"),
		},
		Policy: tools.WebWebsitePolicy{
			Enabled:           cfg.Security.WebsiteBlocklist.Enabled,
			Domains:           cfg.Security.WebsiteBlocklist.Domains,
			SharedFiles:       cfg.Security.WebsiteBlocklist.SharedFiles,
			SharedFileBaseDir: cfg.Security.WebsiteBlocklist.BaseDir,
		},
		Processing: tools.WebContentProcessingConfig{
			Enabled: childClient != nil,
		},
		ContentProcessor: newHermesWebContentProcessor(childClient, childModel),
	}) {
		reg.MustRegister(tool)
	}
	tools.RegisterHomeAssistantTools(reg, tools.HomeAssistantConfig{})
	registerAudioTools(reg, cfg)
	registerImageGenerationTool(reg, cfg)
	registerVideoAnalyzeTool(reg, cfg)
	reg.MustRegister(tools.NewVisionAnalyzeTool())
	reg.MustRegister(tools.NewMemoryTool(tools.MemoryToolConfig{
		MemoryDir: filepath.Join(config.GormesHome(), "memory"),
	}))
	reg.MustRegister(sessionsearchtool.NewSessionSearchTool(sessionsearchtool.SessionSearchToolConfig{
		DB:       o.searchDB,
		Sessions: o.searchSessions,
	}))
	for _, tool := range tools.NewSkillsTools(tools.SkillsToolsConfig{
		Root:        cfg.SkillsRoot(),
		BundledRoot: skills.BundledRoot(),
	}) {
		reg.MustRegister(tool)
	}
	reg.MustRegister(tools.NewSkillManagerTool(tools.SkillManagerToolConfig{
		Root: cfg.SkillsRoot(),
	}))
	for _, tool := range kanbantools.NewTools(kanbantools.ConfigFromEnv()) {
		reg.MustRegister(tool)
	}
	for _, tool := range tools.NewBrowserHarnessTools(tools.BrowserHarnessToolsConfig{
		Env: browserCDPEnv(cfg),
		Budget: tools.ToolResultBudgetConfig{
			OutputDir:       filepath.Join(filepath.Dir(config.ToolAuditLogPath()), "browser-artifacts"),
			TextBudgetBytes: 8 * 1024,
			PreviewBytes:    1024,
		},
	}) {
		reg.MustRegister(tool)
	}
	if cfg.Delegation.Enabled {
		var drafter subagent.CandidateDrafter
		skillsRoot := cfg.SkillsRoot()
		if skillsRoot != "" {
			drafter = skillsCandidateDrafter{store: skills.NewStore(skillsRoot, 0)}
		}
		opts := subagent.ManagerOpts{
			ParentCtx:            parentCtx,
			ParentID:             "root",
			Depth:                0,
			Registry:             subagent.NewRegistry(),
			ToolExecutor:         tools.NewInProcessToolExecutor(reg),
			MaxDepth:             cfg.Delegation.MaxDepth,
			DefaultMaxIterations: cfg.Delegation.DefaultMaxIterations,
			DefaultMaxConcurrent: cfg.Delegation.MaxConcurrentChildren,
			DefaultTimeout:       cfg.Delegation.DefaultTimeout,
			RunLogPath:           cfg.Delegation.ResolvedRunLogPath(),
			ToolAudit:            audit.NewJSONLWriter(config.ToolAuditLogPath()),
		}
		if childClient != nil {
			descs := registryDescriptors(reg)
			opts.NewRunner = func() subagent.Runner {
				runner := subagent.NewHermesRunner(childClient, childModel, descs)
				return runner
			}
		}
		reg.MustRegister(subagent.NewDelegateTool(subagent.NewManager(opts), drafter))
	}
	return reg
}

func browserCDPEnv(cfg config.Config) map[string]string {
	endpoint := cfg.Browser.CDPURL
	if endpoint == "" {
		return nil
	}
	return map[string]string{
		"CHROME_REMOTE_DEBUGGING_URL": endpoint,
		"BROWSER_CDP_URL":             endpoint,
	}
}

func registryDescriptors(reg *tools.Registry) []hermes.ToolDescriptor {
	descs := reg.Descriptors()
	out := make([]hermes.ToolDescriptor, len(descs))
	for i, d := range descs {
		out[i] = hermes.ToolDescriptor{Name: d.Name, Description: d.Description, Schema: d.Schema}
	}
	return out
}

type skillsCandidateDrafter struct {
	store *skills.Store
}

func (d skillsCandidateDrafter) DraftCandidate(_ context.Context, req subagent.CandidateDraftRequest) (string, error) {
	meta, err := d.store.DraftCandidate(skills.CandidateDraft{
		Slug:            req.Slug,
		Goal:            req.Goal,
		Summary:         req.Summary,
		SourceRunID:     req.SourceRunID,
		ParentSessionID: req.ParentSessionID,
		ChildAgentID:    req.ChildAgentID,
		ToolNames:       append([]string(nil), req.ToolNames...),
	})
	if err != nil {
		return "", err
	}
	return meta.CandidateID, nil
}
