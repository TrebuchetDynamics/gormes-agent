package profile

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"
)

// MigrationV2Options configures the legacy profile-state migration planner.
type MigrationV2Options struct {
	Home       string
	ConfigPath string
	Now        func() time.Time
}

type MigrationV2Plan struct {
	Home                string
	ConfigPath          string
	NoOp                bool
	ProfileAdditions    []MigrationV2ProfileAddition
	CredentialAdditions []MigrationV2CredentialAddition
	ProviderLinks       []MigrationV2ProviderLink
	ChannelLinks        []MigrationV2ChannelLink
	FallbackReads       []MigrationV2FallbackRead
	SecretMovements     []MigrationV2SecretMovement
	ManualActions       []MigrationV2ManualAction
	Conflicts           []MigrationV2Conflict
	ActiveProfile       string
	PreviewLines        []string
}

type MigrationV2ProfileAddition struct {
	ID          string
	Enabled     bool
	DisplayName string
	Workspaces  []string
	Providers   []string
	Channels    []string
	SourcePath  string
}

type MigrationV2CredentialAddition struct {
	ID           string
	Kind         string
	Provider     string
	Channel      string
	OwnerProfile string
	SecretRef    *credentials.SecretRef
}

type MigrationV2ProviderLink struct {
	ProfileID    string
	Provider     string
	CredentialID string
	DefaultModel string
	Endpoint     string
}

type MigrationV2ChannelLink struct {
	ProfileID        string
	Channel          string
	CredentialID     string
	AllowedChats     []string
	AllowedUsers     []string
	RequireMention   bool
	ToolProgress     string
	SlackAppTokenEnv string
}

type MigrationV2FallbackRead struct {
	Code string
	Path string
}

type MigrationV2SecretMovement struct {
	Source       string
	TargetEnv    string
	CredentialID string
	Redacted     bool
}

type MigrationV2ManualAction struct {
	Code    string
	Message string
}

type MigrationV2Conflict struct {
	Kind       string
	ID         string
	SourcePath string
	Resolution string
}

type MigrationV2ApplyResult struct {
	Path       string
	BackupPath string
	NoOp       bool
	Wrote      bool
	Plan       MigrationV2Plan
}
