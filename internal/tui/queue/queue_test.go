package queue

import "testing"

func TestRootFacadePreservesQueueAPI(t *testing.T) {
	if WindowSize != 3 {
		t.Fatalf("WindowSize = %d, want 3", WindowSize)
	}
	q := Messages{}
	q.Enqueue("first")
	if got := HandleSlash("/queue second", q.Len()); !got.Enqueue || got.Text != "second" {
		t.Fatalf("HandleSlash facade = %+v, want queued second", got)
	}
	if got := ComputeWindow(4, nil); got.Start != 0 || got.End != 3 || got.ShowLead || !got.ShowTail {
		t.Fatalf("ComputeWindow facade = %+v, want head window with tail", got)
	}
	if got := RenderWidget("", q, 80, nil); got != "queued (1)\n  1. first" {
		t.Fatalf("RenderWidget facade = %q", got)
	}
}
