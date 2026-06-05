package process

import "testing"

func TestCompatibilityFacadePreservesInterruptConstructor(t *testing.T) {
	called := false
	tool := NewInterruptTool(func() { called = true })

	if tool.Name() != "interrupt" {
		t.Fatalf("Name() = %q, want interrupt", tool.Name())
	}
	if _, err := tool.Execute(nil, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !called {
		t.Fatal("interrupt callback was not called")
	}
}

func TestCompatibilityFacadePreservesNotificationNormalize(t *testing.T) {
	plan := NormalizeProcessNotificationRequest(ProcessNotificationRequest{
		Background:       true,
		NotifyOnComplete: true,
		WatchPatterns:    []string{"READY"},
	})

	if !plan.NotifyOnComplete {
		t.Fatal("NotifyOnComplete = false, want true")
	}
	if len(plan.WatchPatterns) != 0 {
		t.Fatalf("WatchPatterns = %v, want disabled", plan.WatchPatterns)
	}
	if len(plan.Evidence) != 1 || plan.Evidence[0].Status != "watch_patterns_disabled" {
		t.Fatalf("Evidence = %#v, want watch_patterns_disabled", plan.Evidence)
	}
}
