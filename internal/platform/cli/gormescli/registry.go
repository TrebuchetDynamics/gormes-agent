package gormescli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/guard"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type registryOptions struct {
	searchDB       *sql.DB
	searchSessions SessionSearchDirectory
}

type RegistryOpt func(*registryOptions)

func WithSessionSearch(db *sql.DB, sessions SessionSearchDirectory) RegistryOpt {
	return func(o *registryOptions) {
		o.searchDB = db
		o.searchSessions = sessions
	}
}

// BuildDefaultRegistry returns a Registry populated with Gormes's built-in
// Go-native tools. Context-specific toolsets such as Kanban stay hidden unless
// their runtime gate is active. Consumer forks that want to add domain-specific
// tools call reg.Register on the returned *Registry before passing it into the
// kernel Config.
func BuildDefaultRegistry(parentCtx context.Context, cfg config.Config, childClient llm.Client, childModel string, opts ...RegistryOpt) *tools.Registry {
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
				OutputDir:       DefaultBrowserArtifactDir(),
				TextBudgetBytes: 8 * 1024,
				PreviewBytes:    1024,
			},
		},
		Backend: tools.WebBackendConfig{
			Backend:             cfg.Web.Backend,
			UseGateway:          cfg.Web.UseGateway,
			ManagedToolsEnabled: true,
			AuthStorePath:       DefaultWebAuthStorePath(),
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
		ContentProcessor: NewWebContentProcessor(childClient, childModel),
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
		MemoryDir: DefaultMemoryToolDir(),
	}))
	RegisterSessionSearchTool(reg, o.searchDB, o.searchSessions)
	for _, tool := range tools.NewSkillsTools(tools.SkillsToolsConfig{
		Root:        cfg.SkillsRoot(),
		BundledRoot: skills.BundledRoot(),
	}) {
		reg.MustRegister(tool)
	}
	reg.MustRegister(tools.NewSkillManagerTool(tools.SkillManagerToolConfig{
		Root:             cfg.SkillsRoot(),
		GuardAgentCreated: cfg.GuardAgentCreatedSkills(),
		GuardScanner:     guard.ScanSkillToError,
	}))
	RegisterKanbanTools(reg)
	for _, tool := range tools.NewBrowserHarnessTools(tools.BrowserHarnessToolsConfig{
		Env: browserCDPEnv(cfg),
		Budget: tools.ToolResultBudgetConfig{
			OutputDir:       DefaultBrowserArtifactDir(),
			TextBudgetBytes: 8 * 1024,
			PreviewBytes:    1024,
		},
	}) {
		reg.MustRegister(tool)
	}
	RegisterDelegationTool(DelegationToolOptions{
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
		operatorHome = config.GormesBaseHome()
	}
	profileID, profileRoot := activeProfileWorkspaceRoot()
	if err := os.MkdirAll(profileRoot, 0o700); err != nil {
		return tools.NewFailClosedProfileWorkspaceScope(fmt.Errorf("profile workspace: create %s: %w", profileRoot, err))
	}
	profileCfg := cfg.Profiles[profileID]
	workspaceRoot := strings.TrimSpace(profileCfg.Workspace)
	if workspaceRoot == "" {
		workspaceRoot = profileRoot
	} else if isConfiguredProfileWorkspaceRoot(workspaceRoot, profileID, operatorHome) {
		if err := os.MkdirAll(resolveWorkspaceConfigPath(workspaceRoot, operatorHome), 0o700); err != nil {
			return tools.NewFailClosedProfileWorkspaceScope(fmt.Errorf("profile workspace: create %s: %w", workspaceRoot, err))
		}
	}
	roots := profileAllowedPathRoots(profileCfg)
	roots = append(roots, cfg.Agents.Defaults.Workspaces...)
	if strings.TrimSpace(cfg.Agents.Defaults.Workspace) != "" {
		roots = append(roots, cfg.Agents.Defaults.Workspace)
	}
	scope, err := tools.NewProfileWorkspaceScope(tools.ProfileWorkspaceScopeOptions{
		ProfileName:   profileID,
		ProjectRoots:  roots,
		ProfileRoot:   profileRoot,
		WorkspaceRoot: workspaceRoot,
		OperatorHome:  operatorHome,
	})
	if err != nil {
		return tools.NewFailClosedProfileWorkspaceScope(err)
	}
	return scope
}

func activeProfileWorkspaceRoot() (string, string) {
	home := filepath.Clean(strings.TrimSpace(config.GormesHome()))
	if home == "." || home == "" {
		home = filepath.Clean(config.GormesBaseHome())
	}
	if filepath.Base(filepath.Dir(home)) == "profiles" {
		return filepath.Base(home), home
	}
	// Legacy/unscoped processes keep using the configured GORMES_HOME as their
	// local workspace root. Profile-scoped commands set GORMES_HOME to
	// <base>/profiles/<name> before config/tool wiring, which activates the
	// homogeneous profile workspace layout without materializing profiles/main
	// during older base-home tests or operator commands.
	return config.DefaultProfileID, home
}

func isConfiguredProfileWorkspaceRoot(raw, profileID, operatorHome string) bool {
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(profileID) == "" {
		return false
	}
	resolved := resolveWorkspaceConfigPath(raw, operatorHome)
	want := filepath.Join(config.GormesBaseHome(), "profiles", profileID)
	return filepath.Clean(resolved) == filepath.Clean(want)
}

func resolveWorkspaceConfigPath(raw, operatorHome string) string {
	raw = strings.TrimSpace(raw)
	if raw == "~" {
		return operatorHome
	}
	if strings.HasPrefix(raw, "~/") {
		return filepath.Join(operatorHome, strings.TrimPrefix(raw, "~/"))
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw)
	}
	return filepath.Join(operatorHome, raw)
}

func profileAllowedPathRoots(profile config.ProfileCfg) []string {
	roots := make([]string, 0, len(profile.AllowedPaths)+len(profile.Workspaces)+len(profile.AllowedPathRules))
	roots = append(roots, profile.AllowedPaths...)
	roots = append(roots, profile.Workspaces...)
	for _, rule := range profile.AllowedPathRules {
		if strings.TrimSpace(rule.Path) != "" {
			roots = append(roots, rule.Path)
		}
	}
	return roots
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

func RegistryDescriptors(reg *tools.Registry) []llm.ToolDescriptor {
	descs := reg.Descriptors()
	out := make([]llm.ToolDescriptor, len(descs))
	for i, d := range descs {
		out[i] = llm.ToolDescriptor{Name: d.Name, Description: d.Description, Schema: d.Schema}
	}
	return out
}
