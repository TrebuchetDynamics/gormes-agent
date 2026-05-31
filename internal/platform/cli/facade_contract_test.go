package cli

import (
	"context"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/backup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/providerstatus"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/toolsets"
)

func TestRootFacadeDelegatesToResponsibilityPackages(t *testing.T) {
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
}
