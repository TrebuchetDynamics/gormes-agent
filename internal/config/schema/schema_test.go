package schema

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config/profilestorage"
	"reflect"
	"strings"
	"testing"
)

func TestConfigSchemaAllowedSectionsAndList(t *testing.T) {
	for _, section := range []string{"hermes", "router", "profiles", "credentials", "telegram", "navivox", "updates"} {
		if !AllowsSection(section) {
			t.Fatalf("AllowsSection(%q) = false, want true", section)
		}
	}
	if AllowsSection("typo") {
		t.Fatal("AllowsSection(typo) = true, want false")
	}
	list := AllowedSectionsList()
	if !strings.Contains(list, "hermes") || !strings.Contains(list, "updates") || strings.Contains(list, "typo") {
		t.Fatalf("AllowedSectionsList() = %q, want known sections only", list)
	}
	if strings.Index(list, "agents") > strings.Index(list, "updates") {
		t.Fatalf("AllowedSectionsList() = %q, want stable sorted order", list)
	}
}

func TestConfigSchemaDefaultDocumentV2(t *testing.T) {
	got := DefaultDocumentV2()
	want := map[string]any{
		"config_version": int64(CurrentConfigVersion),
		"profiles": map[string]any{
			profilestorage.DefaultProfileID: map[string]any{
				"enabled": true,
				"name":    "",
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultDocumentV2() = %#v, want %#v", got, want)
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
			if got := ReadVersion(tc.raw); got != tc.want {
				t.Fatalf("ReadVersion(%#v) = %d, want %d", tc.raw, got, tc.want)
			}
		})
	}

	raw := map[string]any{}
	if HasMainProfile(raw) {
		t.Fatal("HasMainProfile(empty) = true, want false")
	}
	EnsureMainProfile(raw)
	if !HasMainProfile(raw) {
		t.Fatalf("EnsureMainProfile did not create main profile: %#v", raw)
	}
	profiles := raw["profiles"].(map[string]any)
	main := profiles[profilestorage.DefaultProfileID].(map[string]any)
	if main["enabled"] != true || main["name"] != "" {
		t.Fatalf("profiles.main = %#v, want enabled true and empty name", main)
	}
}
