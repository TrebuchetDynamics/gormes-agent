package cli

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/backup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/parity"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/providerstatus"
	clipty "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/pty"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/toolsets"
)

func TestRootFacadeDelegatesToResponsibilityPackages(t *testing.T) {
	gotCLI, gotCLIErr := LoadCLISurface("../../../testdata/cli_surface.json")
	wantCLI, wantCLIErr := parity.LoadCLISurface("../../../testdata/cli_surface.json")
	if !reflect.DeepEqual(gotCLI, wantCLI) || (gotCLIErr != nil) != (wantCLIErr != nil) {
		t.Fatalf("LoadCLISurface facade = %+v, %v; want parity package result %+v, %v", gotCLI, gotCLIErr, wantCLI, wantCLIErr)
	}
	gotDocs, gotDocsErr := LoadDocsSurface("../../../testdata/docs_surface.json")
	wantDocs, wantDocsErr := parity.LoadDocsSurface("../../../testdata/docs_surface.json")
	if !reflect.DeepEqual(gotDocs, wantDocs) || (gotDocsErr != nil) != (wantDocsErr != nil) {
		t.Fatalf("LoadDocsSurface facade = %+v, %v; want parity package result %+v, %v", gotDocs, gotDocsErr, wantDocs, wantDocsErr)
	}

	gotAlias := ResolveCommandAlias("/provider openrouter --global")
	wantAlias := commands.ResolveCommandAlias("/provider openrouter --global")
	if !reflect.DeepEqual(gotAlias, wantAlias) {
		t.Fatalf("ResolveCommandAlias facade = %+v, want commands package result %+v", gotAlias, wantAlias)
	}
	quick := map[string]QuickCommandAlias{
		"g":    {Type: "alias", Target: "/goal"},
		"ship": {Type: "alias", Target: "/g now"},
	}
	gotQuick := ResolveQuickCommandAlias("/ship with tests", quick)
	wantQuick := commands.ResolveQuickCommandAlias("/ship with tests", quick)
	if !reflect.DeepEqual(gotQuick, wantQuick) {
		t.Fatalf("ResolveQuickCommandAlias facade = %+v, want commands package result %+v", gotQuick, wantQuick)
	}
	gotTypo, gotTypoOK := TypoSuggestion([]string{"login", "--provider", "plain-secret-provider", "--portal-url", "https://example.invalid"})
	wantTypo, wantTypoOK := commands.TypoSuggestion([]string{"login", "--provider", "plain-secret-provider", "--portal-url", "https://example.invalid"})
	if gotTypo != wantTypo || gotTypoOK != wantTypoOK {
		t.Fatalf("TypoSuggestion facade = %q, %v; want commands package result %q, %v", gotTypo, gotTypoOK, wantTypo, wantTypoOK)
	}

	policyInput := BackupPolicyFlags{Backup: true, NoBackup: false, Candidates: []string{"config.toml", "checkpoints/live.json"}}
	if got, want := ResolveBackupPolicy(policyInput), backup.ResolveBackupPolicy(policyInput); !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveBackupPolicy facade = %+v, want backup package result %+v", got, want)
	}
	if got, want := IsExcludedFromBackup("sessions.db-wal"), backup.IsExcludedFromBackup("sessions.db-wal"); got != want {
		t.Fatalf("IsExcludedFromBackup facade = %v, want backup package result %v", got, want)
	}
	gotIncluded, gotExcluded := PartitionBackupCandidates([]string{"config.toml", "checkpoints/live.json"})
	wantIncluded, wantExcluded := backup.PartitionBackupCandidates([]string{"config.toml", "checkpoints/live.json"})
	if !reflect.DeepEqual(gotIncluded, wantIncluded) || !reflect.DeepEqual(gotExcluded, wantExcluded) {
		t.Fatalf("PartitionBackupCandidates facade = (%v, %v), want backup package result (%v, %v)", gotIncluded, gotExcluded, wantIncluded, wantExcluded)
	}

	authInput := AuthStatusOptions{}
	gotAuth, gotErr := ResolveAuthStatus(context.Background(), "no-such-provider", authInput)
	wantAuth, wantErr := providerstatus.ResolveAuthStatus(context.Background(), "no-such-provider", authInput)
	if !reflect.DeepEqual(gotAuth, wantAuth) || (gotErr != nil) != (wantErr != nil) {
		t.Fatalf("ResolveAuthStatus facade = %+v, %v; want providerstatus package result %+v, %v", gotAuth, gotErr, wantAuth, wantErr)
	}

	foundryInput := AzureFoundryStatusInput{}
	if got, want := RenderAzureFoundryStatus(foundryInput), providerstatus.RenderAzureFoundryStatus(foundryInput); !reflect.DeepEqual(got, want) {
		t.Fatalf("RenderAzureFoundryStatus facade = %+v, want providerstatus package result %+v", got, want)
	}

	pathInput := SystemdPATHOptions{BasePath: "/usr/bin", HostPath: "/mnt/c/Windows/System32", WSLDetected: true}
	if got, want := SystemdPATHEnvironment(pathInput), service.SystemdPATHEnvironment(pathInput); got != want {
		t.Fatalf("SystemdPATHEnvironment facade = %q, want service package result %q", got, want)
	}
	delayInput := ServiceRestartDelaySource{Manager: ServiceManagerUnsupported}
	if got, want := ParseServiceRestartDelay(delayInput), service.ParseServiceRestartDelay(delayInput); !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseServiceRestartDelay facade = %+v, want service package result %+v", got, want)
	}

	toolsetInput := map[string]any{"platform_toolsets": map[string]any{"cli": []any{"no_mcp"}}}
	gotCfg, gotReport := ParsePlatformToolsetConfig(toolsetInput)
	wantCfg, wantReport := toolsets.ParsePlatformToolsetConfig(toolsetInput)
	if !reflect.DeepEqual(gotCfg, wantCfg) || !reflect.DeepEqual(gotReport, wantReport) {
		t.Fatalf("ParsePlatformToolsetConfig facade = (%+v, %+v), want toolsets package result (%+v, %+v)", gotCfg, gotReport, wantCfg, wantReport)
	}

	if got, want := PtyAvailable(), clipty.PtyAvailable(); got != want {
		t.Fatalf("PtyAvailable facade = %v, want pty package result %v", got, want)
	}
	bridge := NewPtyAdapterForSession(&facadeRecordingPtySession{})
	if bridge == nil {
		t.Fatal("NewPtyAdapterForSession facade returned nil")
	}
	if err := bridge.Write(nil); !errors.Is(err, ErrInvalidPtyMessage) {
		t.Fatalf("Write(nil) err = %v, want ErrInvalidPtyMessage", err)
	}
	sidecar := NewPtyChatSidecar(PtyChatSidecarConfig{})
	if sidecar == nil {
		t.Fatal("NewPtyChatSidecar facade returned nil")
	}
	if sidecar.Healthy() {
		t.Fatal("NewPtyChatSidecar nil-sink facade is healthy; want pty package nil-sink behavior")
	}
}

type facadeRecordingPtySession struct{}

func (s *facadeRecordingPtySession) Read(time.Duration, int) ([]byte, error) { return []byte{}, nil }
func (s *facadeRecordingPtySession) Write([]byte) error                      { return nil }
func (s *facadeRecordingPtySession) Resize(int, int) error                   { return nil }
func (s *facadeRecordingPtySession) Close() error                            { return nil }
func (s *facadeRecordingPtySession) IsAlive() bool                           { return true }
func (s *facadeRecordingPtySession) PID() int                                { return 123 }
