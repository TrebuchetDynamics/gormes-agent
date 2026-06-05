package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func executeRootCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := executeRootCommand(cmd, args...)
	return stdout.String(), stderr.String(), err
}

func writeRootStatusProgressFixture(t testing.TB, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write progress fixture: %v", err)
	}
	return path
}

func withOpenRouterModelCatalogFetcherForTest(t testing.TB, fetcher gormescli.OpenRouterModelCatalogFetchFunc) {
	t.Helper()
	restore := gormescli.SetOpenRouterModelCatalogFetcherForTest(fetcher)
	t.Cleanup(restore)
}

func openRouterModelCatalogOfflineForTest(context.Context) ([]string, error) {
	return nil, errors.New("openrouter model catalog unavailable")
}
