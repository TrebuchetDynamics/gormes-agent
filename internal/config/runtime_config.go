package config

import runtimeconfig "github.com/TrebuchetDynamics/gormes-agent/internal/config/runtime"

// UpdatesCfg controls `gormes update` behavior. PreUpdateBackup is the
// config equivalent of `--backup` and is silent-default off. BackupKeep
// is the retention budget applied after a successful write; <=0 keeps
// the built-in default of 5 (matches Hermes' upstream).
type UpdatesCfg = runtimeconfig.Updates

type CronCfg = runtimeconfig.Cron

// ApprovalsCfg mirrors Hermes' approval policy settings that affect native Go tools.
type ApprovalsCfg = runtimeconfig.Approvals

// WebCfg mirrors Hermes config.yaml's web.backend and web.use_gateway fields.
type WebCfg = runtimeconfig.Web

// BrowserCfg mirrors Hermes browser/CDP connection settings used by browser
// tools and CDP-backed web_extract fallback.
type BrowserCfg = runtimeconfig.Browser

// WorkspaceCfg configures workspace-level file access policy.
type WorkspaceCfg = runtimeconfig.Workspace

const (
	// WorkspaceModeReadonly is the canonical value for readonly mode.
	WorkspaceModeReadonly = runtimeconfig.WorkspaceModeReadonly
	// WorkspaceModeReadWrite is the canonical value for read-write mode.
	WorkspaceModeReadWrite = runtimeconfig.WorkspaceModeReadWrite
)

// SecurityCfg mirrors Hermes config.yaml security controls that affect native Go tools.
type SecurityCfg = runtimeconfig.Security

type WebsiteBlocklistCfg = runtimeconfig.WebsiteBlocklist

type HermesCfg = runtimeconfig.Hermes

type AgentRuntimeCfg = runtimeconfig.Agent

type RuntimeCfg = runtimeconfig.Runtime

type TerminalCfg = runtimeconfig.Terminal

// CodeExecutionCfg controls the native execute_code tool mode. Hermes defaults
// to project mode; Gormes keeps strict as the built-in default until the
// shell-only guard is intentionally relaxed by explicit config.
type CodeExecutionCfg = runtimeconfig.CodeExecution

type TUICfg = runtimeconfig.TUI

type InputCfg = runtimeconfig.Input
