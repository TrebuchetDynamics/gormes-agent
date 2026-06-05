package shared

import (
	"strings"
	"testing"
)

func TestReadDotenvKeysReturnsUniqueSortedKeysWithoutValues(t *testing.T) {
	keys, err := ReadDotenvKeys(strings.NewReader(`
# comment
B_TOKEN=secret-b
A_API_KEY=secret-a
invalid
B_TOKEN=other-secret
 EMPTY_VALUE =
`))
	if err != nil {
		t.Fatalf("ReadDotenvKeys: %v", err)
	}
	want := []string{"A_API_KEY", "B_TOKEN", "EMPTY_VALUE"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %#v, want %#v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys = %#v, want %#v", keys, want)
		}
	}
}

func TestSortedStringAnyKeys(t *testing.T) {
	keys := SortedStringAnyKeys(map[string]any{"z": 1, "a": 2})
	if got, want := strings.Join(keys, ","), "a,z"; got != want {
		t.Fatalf("SortedStringAnyKeys = %q, want %q", got, want)
	}
}
