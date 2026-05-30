package queue

import "testing"

func TestHandleSlashReportsDepthOrQueuedText(t *testing.T) {
	if got := HandleSlash("/queue", 2); got.Enqueue || got.Status != "2 queued message(s)" {
		t.Fatalf("HandleSlash empty = %+v, want depth status", got)
	}
	if got := HandleSlash("/queue follow up after tools", 0); !got.Enqueue || got.Text != "follow up after tools" || got.Status != "" {
		t.Fatalf("HandleSlash text = %+v, want enqueue stripped text", got)
	}
}

func TestMessagesRemoveAtBounds(t *testing.T) {
	q := Messages{}
	q.Enqueue("first")
	q.Enqueue("second")
	q.Enqueue("third")

	if q.RemoveAt(-1) || q.RemoveAt(3) {
		t.Fatalf("RemoveAt out-of-range mutated queue: %v", q.Items())
	}
	if got := q.Items(); len(got) != 3 || got[0] != "first" || got[1] != "second" || got[2] != "third" {
		t.Fatalf("queue after out-of-range RemoveAt = %v", got)
	}
	if !q.RemoveAt(1) {
		t.Fatal("RemoveAt(1) = false, want true")
	}
	if got := q.Items(); len(got) != 2 || got[0] != "first" || got[1] != "third" {
		t.Fatalf("queue after RemoveAt(1) = %v, want [first third]", got)
	}
}

func TestMessagesCancelEditPreservesQueue(t *testing.T) {
	q := Messages{}
	q.Enqueue("keep me")
	if !q.SelectEdit(0) {
		t.Fatal("SelectEdit(0) = false, want true")
	}
	q.CancelEdit()
	if _, ok := q.EditIndex(); ok {
		t.Fatal("EditIndex() ok = true after CancelEdit")
	}
	if got := q.Items(); len(got) != 1 || got[0] != "keep me" {
		t.Fatalf("CancelEdit changed queue: %v", got)
	}
}

func TestMessagesDeleteEditingItemRemovesOnlySelected(t *testing.T) {
	q := Messages{}
	q.Enqueue("first")
	q.Enqueue("second")
	q.Enqueue("third")
	if !q.SelectEdit(1) {
		t.Fatal("SelectEdit(1) = false, want true")
	}
	deleted, ok := q.DeleteEditing()
	if !ok || deleted != "second" {
		t.Fatalf("DeleteEditing() = (%q, %v), want (second, true)", deleted, ok)
	}
	if got := q.Items(); len(got) != 2 || got[0] != "first" || got[1] != "third" {
		t.Fatalf("queue after DeleteEditing = %v, want [first third]", got)
	}
}

func TestMessagesReplaceEditingItem(t *testing.T) {
	q := Messages{}
	q.Enqueue("first")
	q.Enqueue("second")
	if q.ReplaceEditing("nope") {
		t.Fatal("ReplaceEditing without edit = true, want false")
	}
	if !q.SelectEdit(1) {
		t.Fatal("SelectEdit(1) = false, want true")
	}
	if !q.ReplaceEditing("updated") {
		t.Fatal("ReplaceEditing with edit = false, want true")
	}
	if got := q.Items(); len(got) != 2 || got[1] != "updated" {
		t.Fatalf("queue after ReplaceEditing = %v, want second updated", got)
	}
	if idx, ok := q.EditIndex(); !ok || idx != 1 {
		t.Fatalf("EditIndex after ReplaceEditing = (%d, %v), want (1, true)", idx, ok)
	}
}

func TestWindowCentersEditedItem(t *testing.T) {
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
				got = ComputeWindow(tc.length, &tc.editIdx)
			} else {
				got = ComputeWindow(tc.length, nil)
			}
			if got != tc.wantWin {
				t.Fatalf("ComputeWindow(%d, editing=%v idx=%d) = %+v, want %+v", tc.length, tc.editing, tc.editIdx, got, tc.wantWin)
			}
		})
	}
}
