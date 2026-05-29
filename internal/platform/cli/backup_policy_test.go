package cli

import (
	"reflect"
	"sort"
	"testing"
)

func TestBackupPolicy_DefaultSkipsPreUpdateBackup(t *testing.T) {
	cases := []struct {
		name  string
		flags BackupPolicyFlags
	}{
		{name: "all defaults", flags: BackupPolicyFlags{}},
		{name: "config disabled, no flags", flags: BackupPolicyFlags{ConfigEnabled: false}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveBackupPolicy(tc.flags)
			if got.Requested {
				t.Fatalf("default policy must not request backup; got %+v", got)
			}
			if got.Reason != BackupReasonSkippedDefault {
				t.Fatalf("default policy reason = %q, want %q", got.Reason, BackupReasonSkippedDefault)
			}
		})
	}
}

func TestBackupPolicy_ExplicitBackupEnables(t *testing.T) {
	got := ResolveBackupPolicy(BackupPolicyFlags{Backup: true})
	if !got.Requested {
		t.Fatalf("--backup must request backup; got %+v", got)
	}
	if got.Reason != BackupReasonForced {
		t.Fatalf("--backup reason = %q, want %q", got.Reason, BackupReasonForced)
	}
}

func TestBackupPolicy_NoBackupWins(t *testing.T) {
	cases := []struct {
		name  string
		flags BackupPolicyFlags
	}{
		{name: "both flags", flags: BackupPolicyFlags{Backup: true, NoBackup: true}},
		{name: "no-backup with config enabled", flags: BackupPolicyFlags{NoBackup: true, ConfigEnabled: true}},
		{name: "no-backup alone", flags: BackupPolicyFlags{NoBackup: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveBackupPolicy(tc.flags)
			if got.Requested {
				t.Fatalf("--no-backup must suppress backup; got %+v", got)
			}
			if got.Reason != BackupReasonDisabledByFlag {
				t.Fatalf("--no-backup reason = %q, want %q", got.Reason, BackupReasonDisabledByFlag)
			}
		})
	}
}

func TestBackupPolicy_ConfigEnabledRequestsBackup(t *testing.T) {
	got := ResolveBackupPolicy(BackupPolicyFlags{ConfigEnabled: true})
	if !got.Requested {
		t.Fatalf("config pre_update_backup=true must request backup; got %+v", got)
	}
	if got.Reason != BackupReasonConfigEnabled {
		t.Fatalf("config-enabled reason = %q, want %q", got.Reason, BackupReasonConfigEnabled)
	}
}

func TestBackupManifestExclusions_SkipsCheckpointsAndSQLiteSidecars(t *testing.T) {
	candidates := []string{
		"sessions/abc/messages.db",
		"sessions/abc/messages.db-wal",
		"sessions/abc/messages.db-shm",
		"sessions/abc/messages.db-journal",
		"checkpoints/run-2025-04-29/state.json",
		"checkpoints/cache.bin",
		"profiles/main/config.toml",
		"memory/index.db",
		"memory/index.db-wal",
		"logs/gormes.log",
		"nested/checkpoints/inner.txt",
	}
	wantIncluded := []string{
		"sessions/abc/messages.db",
		"profiles/main/config.toml",
		"memory/index.db",
		"logs/gormes.log",
	}
	wantExcluded := []string{
		"sessions/abc/messages.db-wal",
		"sessions/abc/messages.db-shm",
		"sessions/abc/messages.db-journal",
		"checkpoints/run-2025-04-29/state.json",
		"checkpoints/cache.bin",
		"memory/index.db-wal",
		"nested/checkpoints/inner.txt",
	}

	included, excluded := PartitionBackupCandidates(candidates)

	sort.Strings(included)
	sort.Strings(excluded)
	sort.Strings(wantIncluded)
	sort.Strings(wantExcluded)

	if !reflect.DeepEqual(included, wantIncluded) {
		t.Fatalf("included paths mismatch\n got: %v\nwant: %v", included, wantIncluded)
	}
	if !reflect.DeepEqual(excluded, wantExcluded) {
		t.Fatalf("excluded paths mismatch\n got: %v\nwant: %v", excluded, wantExcluded)
	}

	for _, p := range wantExcluded {
		if !IsExcludedFromBackup(p) {
			t.Errorf("IsExcludedFromBackup(%q) = false, want true", p)
		}
	}
	for _, p := range wantIncluded {
		if IsExcludedFromBackup(p) {
			t.Errorf("IsExcludedFromBackup(%q) = true, want false", p)
		}
	}
}

func TestBackupPolicy_ExcludedReasonExposed(t *testing.T) {
	got := ResolveBackupPolicy(BackupPolicyFlags{Backup: true, Candidates: []string{
		"sessions/a.db",
		"sessions/a.db-wal",
		"checkpoints/x",
	}})
	if !got.Requested {
		t.Fatalf("--backup must request backup; got %+v", got)
	}
	if got.Reason != BackupReasonForced {
		t.Fatalf("primary reason = %q, want %q", got.Reason, BackupReasonForced)
	}
	if len(got.ExcludedPaths) == 0 {
		t.Fatalf("ExcludedPaths must surface manifest exclusions; got %+v", got)
	}
	wantSecondary := BackupReasonManifestExcludedPaths
	found := false
	for _, r := range got.SecondaryReasons {
		if r == wantSecondary {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("SecondaryReasons missing %q; got %v", wantSecondary, got.SecondaryReasons)
	}
}
