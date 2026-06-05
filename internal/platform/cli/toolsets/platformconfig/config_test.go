package platformconfig

import "testing"

func TestFacadeExposesPlatformToolsetConfigBehavior(t *testing.T) {
	cfg, _ := ParsePlatformToolsetConfig(map[string]any{
		"platform_toolsets": map[string]any{
			"cli": []any{"web", "terminal"},
		},
	})

	status, err := cfg.PlatformStatus("cli")
	if err != nil {
		t.Fatalf("PlatformStatus(cli): %v", err)
	}
	if got, want := status.Platform, "cli"; got != want {
		t.Fatalf("status platform = %q, want %q", got, want)
	}
	if len(status.RuntimeToolsets) != 2 || status.RuntimeToolsets[0] != "terminal" || status.RuntimeToolsets[1] != "web" {
		t.Fatalf("runtime toolsets = %v, want [terminal web]", status.RuntimeToolsets)
	}
}
