package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResetDefaultConfigPreservesBreadcrumbOfPriorConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	priorBody := "[hermes]\nprovider = 'openai'\nendpoint = 'https://example.com/v1'\nmodel = 'custom-prior'\n"
	if err := os.WriteFile(configPath, []byte(priorBody), 0o600); err != nil {
		t.Fatal(err)
	}

	breadcrumb, err := ResetDefaultConfigWithClock(configPath, func() time.Time {
		return time.Date(2026, 4, 22, 10, 11, 12, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("ResetDefaultConfigWithClock: %v", err)
	}
	if !strings.HasSuffix(breadcrumb, "config.toml.before-reset.20260422T101112Z") {
		t.Fatalf("breadcrumb = %q, want config.toml.before-reset.<UTC>", breadcrumb)
	}

	got, err := os.ReadFile(breadcrumb)
	if err != nil {
		t.Fatalf("read breadcrumb %s: %v", breadcrumb, err)
	}
	if string(got) != priorBody {
		t.Fatalf("breadcrumb body = %q, want exact prior config %q", got, priorBody)
	}
	info, err := os.Stat(breadcrumb)
	if err != nil {
		t.Fatalf("stat breadcrumb: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("breadcrumb mode = %v, want 0o600 (must not leak secrets to other users)", info.Mode().Perm())
	}
}

func TestResetDefaultConfigNoBreadcrumbWhenNoPriorConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	breadcrumb, err := ResetDefaultConfigWithClock(configPath, func() time.Time {
		return time.Date(2026, 4, 22, 10, 11, 12, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("ResetDefaultConfigWithClock: %v", err)
	}
	if breadcrumb != "" {
		t.Fatalf("fresh-install breadcrumb = %q, want empty", breadcrumb)
	}

	entries, err := os.ReadDir(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "config.toml.before-reset.") {
			t.Fatalf("fresh-install reset must NOT produce an empty breadcrumb; got %s", entry.Name())
		}
	}
}
