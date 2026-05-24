package main

import (
	"context"
	"database/sql"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type registryOptions struct {
	searchDB       *sql.DB
	searchSessions gormescli.SessionSearchDirectory
}

type registryOpt func(*registryOptions)

func withSessionSearch(db *sql.DB, sessions gormescli.SessionSearchDirectory) registryOpt {
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
	workspaceScope := profileWorkspaceScopeFromConfig(cfg)
	reg.MustRegister(&tools.EchoTool{})
	reg.MustRegister(&tools.NowTool{})
	reg.MustRegister(&tools.RandIntTool{})
	outputCompaction := tools.AutoOutputCompaction()
	reg.MustRegister(tools.NewExecuteCodeTool(tools.ExecuteCodeToolConfig{
		ConfigSet:        cfg.CodeExecution.Mode != "",
		ConfigValue:      cfg.CodeExecution.Mode,
		DefaultMode:      tools.DefaultExecuteCodeMode,
		SubprocessHome:   config.SubprocessHome,
		WorkspaceScope:   workspaceScope,
		OutputCompaction: outputCompaction,
	}))
	fileTools := tools.FileTaskToolConfig{
		Root:           workspaceScope.DefaultRoot(),
		WorkspaceScope: workspaceScope,
		CWDResolver: func() string {
			return cfg.Terminal.CWD
		},
	}
	reg.MustRegister(tools.NewReadFileTool(fileTools))
	reg.MustRegister(tools.NewSearchFilesTool(fileTools))
	reg.MustRegister(tools.NewWriteFileTool(fileTools))
	reg.MustRegister(tools.NewPatchTool(fileTools))
	reg.MustRegister(tools.NewTerminalTool(tools.TerminalToolConfig{
		Workdir:          cfg.Terminal.CWD,
		SubprocessHome:   config.SubprocessHome,
		WorkspaceScope:   workspaceScope,
		OutputCompaction: outputCompaction,
	}))
	reg.MustRegister(tools.NewClarifyTool(nil))
	for _, tool := range tools.NewWebTools(tools.WebToolsConfig{
		Browser: tools.BrowserHarnessToolsConfig{
			Env: browserCDPEnv(cfg),
			Budget: tools.ToolResultBudgetConfig{
				OutputDir:       defaultBrowserArtifactDir(),
				TextBudgetBytes: 8 * 1024,
				PreviewBytes:    1024,
			},
		},
		Backend: tools.WebBackendConfig{
			Backend:             cfg.Web.Backend,
			UseGateway:          cfg.Web.UseGateway,
			ManagedToolsEnabled: true,
			AuthStorePath:       defaultWebAuthStorePath(),
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
	tools.RegisterXSearchTools(reg, tools.XSearchConfig{
		Fake: true,
	})
	registerAudioTools(reg, cfg)
	registerImageGenerationTool(reg, cfg)
	registerVideoAnalyzeTool(reg, cfg)
	reg.MustRegister(tools.NewVisionAnalyzeTool())
	reg.MustRegister(tools.NewMemoryTool(tools.MemoryToolConfig{
		MemoryDir: defaultMemoryToolDir(),
	}))
	gormescli.RegisterSessionSearchTool(reg, o.searchDB, o.searchSessions)
	for _, tool := range tools.NewSkillsTools(tools.SkillsToolsConfig{
		Root:        cfg.SkillsRoot(),
		BundledRoot: skills.BundledRoot(),
	}) {
		reg.MustRegister(tool)
	}
	reg.MustRegister(tools.NewSkillManagerTool(tools.SkillManagerToolConfig{
		Root: cfg.SkillsRoot(),
	}))
	gormescli.RegisterKanbanTools(reg)
	for _, tool := range tools.NewBrowserHarnessTools(tools.BrowserHarnessToolsConfig{
		Env: browserCDPEnv(cfg),
		Budget: tools.ToolResultBudgetConfig{
			OutputDir:       defaultBrowserArtifactDir(),
			TextBudgetBytes: 8 * 1024,
			PreviewBytes:    1024,
		},
	}) {
		reg.MustRegister(tool)
	}
	gormescli.RegisterDelegationTool(gormescli.DelegationToolOptions{
		ParentCtx:   parentCtx,
		Config:      cfg,
		Registry:    reg,
		ChildClient: childClient,
		ChildModel:  childModel,
	})
	return reg
}

func profileWorkspaceScopeFromConfig(cfg config.Config) *tools.ProfileWorkspaceScope {
	operatorHome, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(operatorHome) == "" {
		operatorHome = config.GormesHome()
	}
	roots := append([]string(nil), cfg.Agents.Defaults.Workspaces...)
	if len(roots) == 0 && strings.TrimSpace(cfg.Agents.Defaults.Workspace) != "" {
		roots = []string{cfg.Agents.Defaults.Workspace}
	}
	scope, err := tools.NewProfileWorkspaceScope(tools.ProfileWorkspaceScopeOptions{
		ProjectRoots: roots,
		ProfileRoot:  config.GormesHome(),
		OperatorHome: operatorHome,
	})
	if err != nil {
		return tools.NewFailClosedProfileWorkspaceScope(err)
	}
	return scope
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
