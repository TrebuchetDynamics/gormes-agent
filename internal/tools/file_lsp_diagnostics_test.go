package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/lsp"
)

func TestWriteFileToolIncludesLSPDiagnosticDelta(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "src/main.go", "package main\nfunc main() { existing() }\n")
	service := &fakeLSPService{
		baseline: []lsp.Diagnostic{{
			Message:  "existing error",
			Severity: 1,
			Source:   "fake-lsp",
			Code:     "E_EXISTING",
			Range:    lsp.Range{Start: lsp.Position{Line: 0}, End: lsp.Position{Line: 0}},
		}},
		post: []lsp.Diagnostic{
			{
				Message:  "existing error",
				Severity: 1,
				Source:   "fake-lsp",
				Code:     "E_EXISTING",
				Range:    lsp.Range{Start: lsp.Position{Line: 1}, End: lsp.Position{Line: 1}},
			},
			{
				Message:  "new semantic error",
				Severity: 1,
				Source:   "fake-lsp",
				Code:     "E_NEW",
				Range:    lsp.Range{Start: lsp.Position{Line: 3}, End: lsp.Position{Line: 3}},
			},
		},
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a", LSPDiagnostics: service}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/main.go"}`)
	content := "inserted\npackage main\nfunc main() { existing(); missing() }\n"
	out := executeWriteFileTool(t, NewWriteFileTool(cfg), `{"path":"src/main.go","content":`+quoteJSON(t, content)+`}`)

	if out["status"] != "ok" {
		t.Fatalf("write result = %#v, want ok", out)
	}
	payload := requireLSPDiagnostics(t, out["lsp"])
	if payload["success"] != false || payload["status"] != "diagnostics" {
		t.Fatalf("lsp = %#v, want diagnostic evidence", payload)
	}
	diagnostics, ok := payload["diagnostics"].([]any)
	if !ok || len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one new diagnostic", payload["diagnostics"])
	}
	diag, ok := diagnostics[0].(map[string]any)
	if !ok || diag["message"] != "new semantic error" || diag["code"] != "E_NEW" {
		t.Fatalf("diagnostic = %#v, want new semantic error only", diagnostics[0])
	}
	if !service.called {
		t.Fatalf("fake LSP service was not called")
	}
}

func TestPatchToolLSPDiagnosticsDegradeWithoutFailingPatch(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "src/app.go", "package main\nfunc main() {}\n")
	service := &fakeLSPService{unsupported: true}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a", LSPDiagnostics: service}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/app.go"}`)
	out := executePatchTool(t, NewPatchTool(cfg), `{"path":"src/app.go","old_string":"main() {}","new_string":"main() { missing() }"}`)

	if out["status"] != "ok" {
		t.Fatalf("patch result = %#v, want ok despite unsupported LSP", out)
	}
	payload := requireLSPDiagnostics(t, out["lsp"])
	if payload["status"] != "skipped" || payload["success"] != true {
		t.Fatalf("lsp = %#v, want skipped success evidence", payload)
	}
	if payload["message"] != "LSP diagnostics unsupported for src/app.go" {
		t.Fatalf("message = %#v, want unsupported evidence", payload["message"])
	}
}

type fakeLSPService struct {
	baseline    []lsp.Diagnostic
	post        []lsp.Diagnostic
	unsupported bool
	called      bool
}

func (f *fakeLSPService) PostEditDiagnostics(ctx context.Context, req lsp.PostEditRequest) lsp.PostEditResult {
	_ = ctx
	f.called = true
	if f.unsupported {
		return lsp.PostEditResult{Status: lsp.StatusSkipped, Success: true, Message: "LSP diagnostics unsupported for " + req.RelativePath}
	}
	return lsp.NewPostEditResult(req, f.baseline, f.post)
}

func requireLSPDiagnostics(t *testing.T, raw any) map[string]any {
	t.Helper()
	payload, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("lsp = %#v, want object", raw)
	}
	return payload
}

func writeFixtureFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
