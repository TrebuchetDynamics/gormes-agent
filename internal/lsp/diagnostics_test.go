package lsp

import "testing"

func TestDiagnosticDeltaFiltersShiftedBaseline(t *testing.T) {
	pre := "package main\nfunc main() {\n\texisting()\n}\n"
	post := "package main\n// inserted\nfunc main() {\n\texisting()\n\tmissing()\n}\n"
	baseline := []Diagnostic{diagAt(2, "existing", "E_EXISTING")}
	postDiagnostics := []Diagnostic{
		diagAt(3, "existing", "E_EXISTING"),
		diagAt(4, "missing", "E_NEW"),
	}

	delta := DiagnosticDelta(&pre, post, baseline, postDiagnostics)
	if len(delta) != 1 {
		t.Fatalf("delta = %#v, want one new diagnostic", delta)
	}
	if delta[0].Message != "missing" || delta[0].Code != "E_NEW" {
		t.Fatalf("delta[0] = %#v, want new diagnostic", delta[0])
	}
}

func TestBuildLineShiftCoversInsertDeleteAndReplace(t *testing.T) {
	insert := BuildLineShift("a\nb\n", "x\na\nb\n")
	if got, ok := insert(1); !ok || got != 2 {
		t.Fatalf("insert shift line 1 = %d/%v, want 2/true", got, ok)
	}
	delete := BuildLineShift("a\nb\nc\n", "a\nc\n")
	if _, ok := delete(1); ok {
		t.Fatalf("deleted line mapped, want dropped")
	}
	if got, ok := delete(2); !ok || got != 1 {
		t.Fatalf("delete shift line 2 = %d/%v, want 1/true", got, ok)
	}
	replace := BuildLineShift("a\nb\nc\n", "a\nx\nc\n")
	if _, ok := replace(1); ok {
		t.Fatalf("replaced line mapped, want dropped")
	}
}

func diagAt(line int, message, code string) Diagnostic {
	return Diagnostic{
		Severity: 1,
		Source:   "fake-lsp",
		Code:     code,
		Message:  message,
		Range:    Range{Start: Position{Line: line}, End: Position{Line: line}},
	}
}
