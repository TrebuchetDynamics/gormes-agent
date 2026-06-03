package migrate

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCollectMigrationEnvSnapshotIncludesProviderKeys(t *testing.T) {
	t.Setenv("GORMES_TELEGRAM_TOKEN", "tg-secret")
	t.Setenv("OPENROUTER_API_KEY", "or-secret")
	t.Setenv("RANDOM_USER_VAR", "ignored")
	got := CollectMigrationEnvSnapshot()
	if got["GORMES_TELEGRAM_TOKEN"] != "tg-secret" || got["OPENROUTER_API_KEY"] != "or-secret" {
		t.Fatalf("snapshot missing expected keys: %+v", got)
	}
	if _, ok := got["RANDOM_USER_VAR"]; ok {
		t.Fatalf("snapshot included unrelated env: %+v", got)
	}
}

func TestRunOpenClawApplyRequiresSource(t *testing.T) {
	var out strings.Builder
	err := RunOpenClawApply(&out, "", "", false, false, BuildProvenance{})
	if err == nil || ExitCode(err) != 2 {
		t.Fatalf("err=%v code=%d, want exit 2", err, ExitCode(err))
	}
	if !strings.Contains(err.Error(), "--source is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunHermesDryRunWrapsBuildProvenance(t *testing.T) {
	t.Setenv("HERMES_HOME", "")
	t.Setenv("HOME", t.TempDir())
	var out strings.Builder
	if err := RunHermesDryRun(&out, "", BuildProvenance{Version: "test", GitCommit: "abc"}); err != nil {
		t.Fatalf("RunHermesDryRun: %v", err)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
	}
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, out.String())
	}
	if got.Build.Version != "test" {
		t.Fatalf("build.version = %q, want test", got.Build.Version)
	}
}

func TestCleanupReportShapeIncludesBuild(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var out strings.Builder
	if err := RunOpenClawCleanup(&out, "gormes migrate openclaw cleanup", true, BuildProvenance{Version: "test", GitCommit: "abc"}); err != nil {
		t.Fatalf("RunOpenClawCleanup: %v", err)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		DryRun bool `json:"dry_run"`
	}
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, out.String())
	}
	if got.Build.Version != "test" || !got.DryRun {
		t.Fatalf("report = %+v", got)
	}
}
