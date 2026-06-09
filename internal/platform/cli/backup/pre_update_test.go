package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWritePreUpdateBackupWritesStampedArchiveAndPrunesBestEffort(t *testing.T) {
	home := t.TempDir()
	backupsDir := filepath.Join(home, "lifecycle", "backups")
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		t.Fatalf("mkdir backups: %v", err)
	}
	mustWriteFile(t, filepath.Join(home, "config.toml"), "[hermes]\nmodel = 'x'\n")
	mustWriteFile(t, filepath.Join(home, "auth.json"), `{"credential_pool":{}}`)
	mustWriteFile(t, filepath.Join(backupsDir, "pre-update-old-a.zip"), "old-a")
	mustWriteFile(t, filepath.Join(backupsDir, "pre-update-old-b.zip"), "old-b")
	mustWriteFile(t, filepath.Join(backupsDir, "operator-notes.txt"), "keep me")

	oldA := filepath.Join(backupsDir, "pre-update-old-a.zip")
	oldB := filepath.Join(backupsDir, "pre-update-old-b.zip")
	base := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(oldA, base.Add(-3*time.Hour), base.Add(-3*time.Hour)); err != nil {
		t.Fatalf("chtimes old-a: %v", err)
	}
	if err := os.Chtimes(oldB, base.Add(-2*time.Hour), base.Add(-2*time.Hour)); err != nil {
		t.Fatalf("chtimes old-b: %v", err)
	}

	res, err := WritePreUpdateBackup(context.Background(), PreUpdateBackupRequest{
		Home: home,
		Now:  base,
		Keep: 2,
	})
	if err != nil {
		t.Fatalf("WritePreUpdateBackup: %v", err)
	}

	wantPath := filepath.Join(backupsDir, "pre-update-20260608T120000Z.zip")
	if res.Path != wantPath {
		t.Fatalf("Path = %q, want %q", res.Path, wantPath)
	}
	if res.FileCount == 0 || res.SizeBytes == 0 {
		t.Fatalf("result did not report written files/bytes: %+v", res)
	}
	if res.PrunedCount != 1 || res.PrunedBytes == 0 {
		t.Fatalf("prune evidence = count %d bytes %d, want one old backup pruned", res.PrunedCount, res.PrunedBytes)
	}
	if _, err := os.Stat(filepath.Join(backupsDir, "operator-notes.txt")); err != nil {
		t.Fatalf("operator file must not be pruned: %v", err)
	}
	if _, err := os.Stat(wantPath + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("tmp file must not remain; stat err = %v", err)
	}
	list, err := ListBackups(backupsDir)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("backup list len = %d, want keep budget 2; list=%+v", len(list), list)
	}
}

func mustWriteFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
