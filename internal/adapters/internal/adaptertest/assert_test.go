package adaptertest

import "testing"

func TestContainsString(t *testing.T) {
	if !ContainsString([]string{"alpha", "beta"}, "beta") {
		t.Fatal("ContainsString returned false for existing value")
	}
	if ContainsString([]string{"alpha", "beta"}, "gamma") {
		t.Fatal("ContainsString returned true for missing value")
	}
}
