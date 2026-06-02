package setup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseToolSelectionResolvesIndexesKeysAndUnknownCustomToolsets(t *testing.T) {
	options := []ToolOption{
		{Key: "web", Label: "Web Search"},
		{Key: "browser", Label: "Browser Automation"},
	}

	got, err := ParseToolSelection("1,browser,custom-mcp-server", options, []string{"terminal"})
	if err != nil {
		t.Fatalf("ParseToolSelection: %v", err)
	}
	want := []string{"web", "browser", "custom-mcp-server"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("selection = %v, want %v", got, want)
	}
}

func TestParseToolSelectionInvalidIndex(t *testing.T) {
	_, err := ParseToolSelection("3", []ToolOption{{Key: "web"}}, nil)
	var invalid InvalidToolSelectionError
	if !errors.As(err, &invalid) || invalid.Token != "3" {
		t.Fatalf("err = %v, want InvalidToolSelectionError token 3", err)
	}
}

func TestWriteToolsConfigPreservesSymlink(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "real-config.toml")
	linkPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(realPath, []byte("platform_toolsets = { cli = [\"terminal\"] }\n"), 0o644); err != nil {
		t.Fatalf("write real config: %v", err)
	}
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	err := WriteToolsConfig(linkPath, map[string]any{
		"platform_toolsets": map[string]any{"cli": []string{"web", "browser"}},
	})
	if err != nil {
		t.Fatalf("WriteToolsConfig: %v", err)
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat config link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config link was replaced with mode %v, want symlink preserved", info.Mode())
	}
	got, err := os.ReadFile(realPath)
	if err != nil {
		t.Fatalf("read real config: %v", err)
	}
	if !strings.Contains(string(got), "web") || !strings.Contains(string(got), "browser") {
		t.Fatalf("real config was not updated through symlink:\n%s", got)
	}
}

func TestToolsProviderRowsAreStableAndFiltered(t *testing.T) {
	rows := ToolsProviderRows([]string{"memory", "web", "terminal"})
	got := make([]string, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.Toolset+":"+row.Kind+":"+row.Label)
	}
	want := []string{
		"web:web:Web search and extraction",
		"memory:honcho:Honcho/Goncho memory provider",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("rows = %v, want %v", got, want)
	}
}
