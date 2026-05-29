package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestConfigSchemaAllowedSectionsAndList(t *testing.T) {
	for _, section := range []string{"hermes", "router", "profiles", "credentials", "telegram", "navivox", "updates"} {
		if !configSchemaAllowsSection(section) {
			t.Fatalf("configSchemaAllowsSection(%q) = false, want true", section)
		}
	}
	if configSchemaAllowsSection("typo") {
		t.Fatal("configSchemaAllowsSection(typo) = true, want false")
	}
	list := allowedSectionsList()
	if !strings.Contains(list, "hermes") || !strings.Contains(list, "updates") || strings.Contains(list, "typo") {
		t.Fatalf("allowedSectionsList() = %q, want known sections only", list)
	}
	if strings.Index(list, "agents") > strings.Index(list, "updates") {
		t.Fatalf("allowedSectionsList() = %q, want stable sorted order", list)
	}
}

func TestConfigSchemaDefaultDocumentV2(t *testing.T) {
	got := DefaultConfigDocumentV2()
	want := map[string]any{
		"config_version": int64(CurrentConfigVersion),
		"profiles": map[string]any{
			DefaultProfileID: map[string]any{
				"enabled": true,
				"name":    "",
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultConfigDocumentV2() = %#v, want %#v", got, want)
	}
}

func TestConfigSchemaVersionReaderAndMainProfileInvariant(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  map[string]any
		want int
	}{
		{name: "canonical", raw: map[string]any{"config_version": int64(2)}, want: 2},
		{name: "legacy", raw: map[string]any{"_config_version": int64(1)}, want: 1},
		{name: "missing", raw: map[string]any{}, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := readConfigVersion(tc.raw); got != tc.want {
				t.Fatalf("readConfigVersion(%#v) = %d, want %d", tc.raw, got, tc.want)
			}
		})
	}

	raw := map[string]any{}
	if hasMainProfile(raw) {
		t.Fatal("hasMainProfile(empty) = true, want false")
	}
	ensureMainProfile(raw)
	if !hasMainProfile(raw) {
		t.Fatalf("ensureMainProfile did not create main profile: %#v", raw)
	}
	profiles := raw["profiles"].(map[string]any)
	main := profiles[DefaultProfileID].(map[string]any)
	if main["enabled"] != true || main["name"] != "" {
		t.Fatalf("profiles.main = %#v, want enabled true and empty name", main)
	}
}
