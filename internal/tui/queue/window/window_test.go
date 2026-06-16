package window

import "testing"

func TestComputeCentersEditedItem(t *testing.T) {
	cases := []struct {
		name    string
		length  int
		editIdx int
		editing bool
		wantWin Window
	}{
		{name: "empty queue with no edit collapses to zero window", length: 0, editing: false, wantWin: Window{}},
		{name: "no edit selection anchors window at head", length: 5, editing: false, wantWin: Window{Start: 0, End: 3, ShowLead: false, ShowTail: true}},
		{name: "edit at head keeps window at start", length: 5, editIdx: 0, editing: true, wantWin: Window{Start: 0, End: 3, ShowLead: false, ShowTail: true}},
		{name: "edit in middle centres window with lead and tail", length: 5, editIdx: 2, editing: true, wantWin: Window{Start: 1, End: 4, ShowLead: true, ShowTail: true}},
		{name: "edit at tail clamps window to end", length: 5, editIdx: 4, editing: true, wantWin: Window{Start: 2, End: 5, ShowLead: true, ShowTail: false}},
		{name: "queue shorter than window with edit at head shows everything", length: 2, editIdx: 0, editing: true, wantWin: Window{Start: 0, End: 2, ShowLead: false, ShowTail: false}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got Window
			if tc.editing {
				got = Compute(tc.length, &tc.editIdx)
			} else {
				got = Compute(tc.length, nil)
			}
			if got != tc.wantWin {
				t.Fatalf("Compute(%d, editing=%v idx=%d) = %+v, want %+v", tc.length, tc.editing, tc.editIdx, got, tc.wantWin)
			}
		})
	}
}
