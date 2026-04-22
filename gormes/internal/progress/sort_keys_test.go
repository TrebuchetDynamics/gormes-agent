package progress

import (
	"slices"
	"testing"
)

func TestSortedMapKeys_UsesNaturalRoadmapOrder(t *testing.T) {
	got := sortedMapKeys(map[string]int{
		"2.B.10": 1,
		"2.B.2":  1,
		"2.B.9":  1,
		"2.E.1":  1,
		"2.E.0":  1,
		"10":     1,
		"2":      1,
		"1":      1,
	})

	want := []string{
		"1",
		"2",
		"2.B.2",
		"2.B.9",
		"2.B.10",
		"2.E.0",
		"2.E.1",
		"10",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("sortedMapKeys() = %#v, want %#v", got, want)
	}
}
