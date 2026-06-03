package hermescliparity

import "testing"

func TestManifestContainsHermesSourceBackedTopLevelCommands(t *testing.T) {
	entries := Manifest()
	if len(entries) < 120 {
		t.Fatalf("manifest entries = %d, want broad Hermes CLI coverage", len(entries))
	}
	want := map[string]Status{
		"chat":     StatusImplemented,
		"auth":     StatusRowBacked,
		"sessions": StatusImplemented,
	}
	for path, status := range want {
		found := false
		for _, entry := range entries {
			if len(entry.Path) == 1 && entry.Path[0] == path {
				found = true
				if entry.Status != status || entry.SourceRef == "" {
					t.Fatalf("entry %s = %+v, want status %s with source ref", path, entry, status)
				}
			}
		}
		if !found {
			t.Fatalf("missing manifest entry %s", path)
		}
	}
}
