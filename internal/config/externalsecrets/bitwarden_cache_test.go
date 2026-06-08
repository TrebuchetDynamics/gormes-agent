package externalsecrets

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyBitwardenWritesAndReadsDiskCacheWithoutTokenLeak(t *testing.T) {
	ResetSecretSourcesForTests()
	home := t.TempDir()
	env := map[string]string{"BWS_ACCESS_TOKEN": "0.bootstrap"}
	calls := 0

	report := ApplyBitwarden(context.Background(), BitwardenConfig{
		Enabled:          true,
		ProjectID:        "project-123",
		CacheTTLSeconds:  300,
		OverrideExisting: true,
		ServerURL:        "https://vault.bitwarden.eu",
	}, BitwardenOptions{
		HomeDir:   home,
		LookupEnv: mapLookup(env),
		SetEnv:    mapSet(env),
		LookPath:  fakeBWSPath,
		Run: func(_ context.Context, _ string, _ []string, cmdEnv []string) ([]byte, []byte, error) {
			calls++
			if !stringSliceContains(cmdEnv, "BWS_SERVER_URL=https://vault.bitwarden.eu") {
				t.Fatalf("env missing server url: %#v", cmdEnv)
			}
			return []byte(`[{"key":"GORMES_API_KEY","value":"fresh"},{"key":"bad-name","value":"skip"}]`), nil, nil
		},
	})
	if !report.OK() {
		t.Fatalf("first report error = %q", report.Error)
	}
	if calls != 1 {
		t.Fatalf("first fetch calls = %d, want 1", calls)
	}
	cachePath := filepath.Join(home, "cache", "bws_cache.json")
	body, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	if strings.Contains(string(body), "0.bootstrap") || strings.Contains(string(body), "BWS_ACCESS_TOKEN") || strings.Contains(string(body), "bad-name") {
		t.Fatalf("cache leaked token/env/invalid key: %s", body)
	}
	info, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("stat cache: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("cache mode = %o, want 0600", got)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("cache json: %v", err)
	}
	key, _ := payload["key"].(string)
	if key == "" || strings.Contains(key, "0.bootstrap") || !strings.Contains(key, "project-123") || !strings.Contains(key, "https://vault.bitwarden.eu") {
		t.Fatalf("cache key = %q", key)
	}

	ResetSecretSourcesForTests() // simulate a new process: disk cache must still serve the fetch.
	env2 := map[string]string{"BWS_ACCESS_TOKEN": "0.bootstrap"}
	report = ApplyBitwarden(context.Background(), BitwardenConfig{
		Enabled:          true,
		ProjectID:        "project-123",
		CacheTTLSeconds:  300,
		OverrideExisting: true,
		ServerURL:        "https://vault.bitwarden.eu",
	}, BitwardenOptions{
		HomeDir:   home,
		LookupEnv: mapLookup(env2),
		SetEnv:    mapSet(env2),
		LookPath:  fakeBWSPath,
		Run: func(context.Context, string, []string, []string) ([]byte, []byte, error) {
			t.Fatalf("bws should not be invoked on fresh disk-cache hit")
			return nil, nil, nil
		},
	})
	if !report.OK() {
		t.Fatalf("second report error = %q", report.Error)
	}
	if env2["GORMES_API_KEY"] != "fresh" {
		t.Fatalf("disk cache did not apply secret: %#v", env2)
	}
	if got := GetSecretSource("GORMES_API_KEY"); got != BitwardenSourceLabel {
		t.Fatalf("source = %q, want %q", got, BitwardenSourceLabel)
	}
}

func TestApplyBitwardenInProcessCacheAvoidsRepeatedBWS(t *testing.T) {
	ResetSecretSourcesForTests()
	home := t.TempDir()
	env := map[string]string{"BWS_ACCESS_TOKEN": "0.bootstrap"}
	calls := 0
	opts := BitwardenOptions{
		HomeDir:   home,
		LookupEnv: mapLookup(env),
		SetEnv:    mapSet(env),
		LookPath:  fakeBWSPath,
		Run: func(context.Context, string, []string, []string) ([]byte, []byte, error) {
			calls++
			return []byte(`[{"key":"GORMES_API_KEY","value":"fresh"}]`), nil, nil
		},
	}
	cfg := BitwardenConfig{Enabled: true, ProjectID: "project-123", CacheTTLSeconds: 300, OverrideExisting: true}
	if report := ApplyBitwarden(context.Background(), cfg, opts); !report.OK() {
		t.Fatalf("first report = %q", report.Error)
	}
	env["GORMES_API_KEY"] = ""
	if report := ApplyBitwarden(context.Background(), cfg, opts); !report.OK() {
		t.Fatalf("second report = %q", report.Error)
	}
	if calls != 1 {
		t.Fatalf("bws calls = %d, want 1", calls)
	}
}

func TestApplyBitwardenCacheDisabledAlwaysFetchesAndDoesNotWriteDisk(t *testing.T) {
	ResetSecretSourcesForTests()
	home := t.TempDir()
	env := map[string]string{"BWS_ACCESS_TOKEN": "0.bootstrap"}
	calls := 0
	cfg := BitwardenConfig{Enabled: true, ProjectID: "project-123", CacheTTLSeconds: 0, OverrideExisting: true}
	opts := BitwardenOptions{
		HomeDir:   home,
		LookupEnv: mapLookup(env),
		SetEnv:    mapSet(env),
		LookPath:  fakeBWSPath,
		Run: func(context.Context, string, []string, []string) ([]byte, []byte, error) {
			calls++
			return []byte(`[{"key":"GORMES_API_KEY","value":"fresh"}]`), nil, nil
		},
	}
	if report := ApplyBitwarden(context.Background(), cfg, opts); !report.OK() {
		t.Fatalf("first report = %q", report.Error)
	}
	if report := ApplyBitwarden(context.Background(), cfg, opts); !report.OK() {
		t.Fatalf("second report = %q", report.Error)
	}
	if calls != 2 {
		t.Fatalf("bws calls = %d, want 2", calls)
	}
	if _, err := os.Stat(filepath.Join(home, "cache", "bws_cache.json")); !os.IsNotExist(err) {
		t.Fatalf("cache should not be written when disabled: %v", err)
	}
}

func TestApplyBitwardenIgnoresStaleWrongKeyMalformedAndServerMismatchCaches(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{not-json`},
		{name: "wrong-key", body: `{"key":"wrong|project-123|","fetched_at":4102444800,"secrets":{"GORMES_API_KEY":"cached"}}`},
		{name: "stale", body: `{"key":"` + bitwardenCacheKeyStringForTest("0.bootstrap", "project-123", "") + `","fetched_at":1,"secrets":{"GORMES_API_KEY":"cached"}}`},
		{name: "server-mismatch", body: `{"key":"` + bitwardenCacheKeyStringForTest("0.bootstrap", "project-123", "https://vault.bitwarden.eu") + `","fetched_at":4102444800,"secrets":{"GORMES_API_KEY":"cached"}}`},
		{name: "non-object-secrets", body: `{"key":"` + bitwardenCacheKeyStringForTest("0.bootstrap", "project-123", "") + `","fetched_at":4102444800,"secrets":["bad"]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ResetSecretSourcesForTests()
			home := t.TempDir()
			cacheDir := filepath.Join(home, "cache")
			if err := os.MkdirAll(cacheDir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(cacheDir, "bws_cache.json"), []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			env := map[string]string{"BWS_ACCESS_TOKEN": "0.bootstrap"}
			calls := 0
			report := ApplyBitwarden(context.Background(), BitwardenConfig{
				Enabled:          true,
				ProjectID:        "project-123",
				CacheTTLSeconds:  300,
				OverrideExisting: true,
			}, BitwardenOptions{
				HomeDir:   home,
				LookupEnv: mapLookup(env),
				SetEnv:    mapSet(env),
				LookPath:  fakeBWSPath,
				Run: func(context.Context, string, []string, []string) ([]byte, []byte, error) {
					calls++
					return []byte(`[{"key":"GORMES_API_KEY","value":"live"}]`), nil, nil
				},
			})
			if !report.OK() {
				t.Fatalf("report error = %q", report.Error)
			}
			if calls != 1 {
				t.Fatalf("bws calls = %d, want fallback fetch", calls)
			}
			if env["GORMES_API_KEY"] != "live" {
				t.Fatalf("env = %#v", env)
			}
		})
	}
}

func TestApplyBitwardenCacheWriteFailureDoesNotBlockFetch(t *testing.T) {
	ResetSecretSourcesForTests()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "cache"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"BWS_ACCESS_TOKEN": "0.bootstrap"}
	calls := 0
	report := ApplyBitwarden(context.Background(), BitwardenConfig{
		Enabled:          true,
		ProjectID:        "project-123",
		CacheTTLSeconds:  300,
		OverrideExisting: true,
	}, BitwardenOptions{
		HomeDir:   home,
		LookupEnv: mapLookup(env),
		SetEnv:    mapSet(env),
		LookPath:  fakeBWSPath,
		Run: func(context.Context, string, []string, []string) ([]byte, []byte, error) {
			calls++
			return []byte(`[{"key":"GORMES_API_KEY","value":"fresh"}]`), nil, nil
		},
	})
	if !report.OK() {
		t.Fatalf("report error = %q", report.Error)
	}
	if calls != 1 || env["GORMES_API_KEY"] != "fresh" {
		t.Fatalf("fetch/apply after write failure: calls=%d env=%#v", calls, env)
	}
}

func mapLookup(env map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) { v, ok := env[key]; return v, ok }
}

func mapSet(env map[string]string) func(string, string) error {
	return func(key, value string) error { env[key] = value; return nil }
}

func fakeBWSPath(string) (string, error) { return "/tmp/fake-bws", nil }

func bitwardenCacheKeyStringForTest(token, projectID, serverURL string) string {
	return bitwardenCacheKeyString(bitwardenCacheKey{tokenFingerprint: bitwardenTokenFingerprint(token), projectID: projectID, serverURL: serverURL})
}
