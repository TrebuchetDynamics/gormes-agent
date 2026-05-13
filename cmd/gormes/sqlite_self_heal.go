package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func isSQLiteCorruptionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "file is not a database") ||
		strings.Contains(msg, "database disk image is malformed") ||
		strings.Contains(msg, "file is encrypted or is not a database")
}

func selfHealCorruptGonchoSQLite(path string) (string, error) {
	backup, err := quarantineCorruptStateFile(path, []string{"-wal", "-shm"})
	if err != nil {
		return "", err
	}
	store, err := memory.OpenSqlite(path, 0, nil)
	if err != nil {
		return backup, fmt.Errorf("recreate memory db: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := store.Close(ctx); err != nil {
		return backup, fmt.Errorf("close recreated memory db: %w", err)
	}
	return backup, nil
}

func quarantineCorruptStateFile(path string, sidecarSuffixes []string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("empty state path")
	}
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	backup := uniqueCorruptBackupPath(path)
	if err := os.Rename(path, backup); err != nil {
		return "", fmt.Errorf("quarantine corrupt state file: %w", err)
	}
	for _, suffix := range sidecarSuffixes {
		sidecar := path + suffix
		if _, err := os.Stat(sidecar); err != nil {
			continue
		}
		_ = os.Rename(sidecar, backup+suffix)
	}
	return backup, nil
}

func uniqueCorruptBackupPath(path string) string {
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	base := path + ".corrupt-" + stamp
	candidate := base
	for i := 1; ; i++ {
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}
