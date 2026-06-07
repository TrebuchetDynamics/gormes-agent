package entry

import "testing"

func TestSortEntriesByNameID(t *testing.T) {
	entries := []Entry{
		{ID: "3", Name: "ops"},
		{ID: "2", Name: "general"},
		{ID: "1", Name: "general"},
	}

	SortEntriesByNameID(entries)

	got := []string{entries[0].ID, entries[1].ID, entries[2].ID}
	want := []string{"1", "2", "3"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted IDs = %v, want %v", got, want)
		}
	}
}
