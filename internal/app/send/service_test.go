package send

import "testing"

func TestSanitizeResultStripsTerminalResponses(t *testing.T) {
	got := SanitizeResult(Result{
		Target:    "telegram\x1b[53;1R",
		Message:   "hello\x1b[53;1Rworld",
		MessageID: "m\x1b[53;1R1",
		Note:      "note\x1b[53;1R",
		Error:     "err\x1b[53;1R",
		Evidence:  "evidence\x1b[53;1R",
		Source:    "source\x1b[53;1R",
	})
	if got.Target != "telegram" || got.Message != "helloworld" || got.MessageID != "m1" || got.Note != "note" || got.Error != "err" || got.Evidence != "evidence" || got.Source != "source" {
		t.Fatalf("SanitizeResult = %+v", got)
	}
}

func TestDefaultBackendReturnsUnavailableEvidence(t *testing.T) {
	got, err := DefaultBackend(nil, "telegram", "hello")
	if err != nil {
		t.Fatalf("DefaultBackend err = %v", err)
	}
	if got.Success || got.Target != "telegram" || got.Evidence != BackendUnavailableEvidence || got.Error == "" {
		t.Fatalf("DefaultBackend = %+v", got)
	}
}
