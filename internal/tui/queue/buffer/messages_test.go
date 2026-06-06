package buffer

import "testing"

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
