package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileToolStructuredLintReportsJSONYAMLTOML(t *testing.T) {
	root := t.TempDir()
	tool := NewWriteFileTool(FileTaskToolConfig{Root: root})
	cases := []struct {
		name    string
		path    string
		content string
	}{
		{name: "json", path: "cfg/bad.json", content: `{"name":`},
		{name: "yaml", path: "cfg/bad.yaml", content: "name: [unterminated\n"},
		{name: "toml", path: "cfg/bad.toml", content: "name =\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := executeWriteFileTool(t, tool, `{"path":`+quoteJSON(t, tc.path)+`,"content":`+quoteJSON(t, tc.content)+`}`)
			if out["status"] != "ok" {
				t.Fatalf("write result = %#v, want ok with lint evidence", out)
			}
			lint := requireStructuredLint(t, out["lint"])
			if lint["success"] != false {
				t.Fatalf("lint = %#v, want success=false", lint)
			}
			if strings.TrimSpace(asString(lint["output"])) == "" {
				t.Fatalf("lint output missing parser evidence: %#v", lint)
			}
			assertFileContent(t, filepath.Join(root, tc.path), tc.content)
		})
	}
}

func TestPatchToolStructuredLintDeltaPreservesPreExistingError(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "cfg", "bad.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"a":`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"cfg/bad.json"}`)

	out := executePatchTool(t, NewPatchTool(cfg), `{"path":"cfg/bad.json","old_string":"\"a\"","new_string":"\"b\""}`)
	if out["status"] != "ok" {
		t.Fatalf("patch result = %#v, want ok with lint evidence", out)
	}
	lint := requireStructuredLint(t, out["lint"])
	if lint["success"] != false {
		t.Fatalf("lint = %#v, want success=false", lint)
	}
	if !strings.Contains(asString(lint["message"]), "Pre-existing lint errors") {
		t.Fatalf("lint message = %#v, want pre-existing error note", lint)
	}
	assertFileContent(t, path, `{"b":`)
}

func TestPatchToolV4AStructuredLintReportsPerFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "cfg", "settings.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte("name = \"ok\"\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"cfg/settings.toml"}`)

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Update File: cfg/settings.toml",
		"@@",
		"-name = \"ok\"",
		"+name =",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)
	if out["status"] != "ok" {
		t.Fatalf("patch result = %#v, want ok with lint evidence", out)
	}
	lintByPath, _ := out["lint"].(map[string]any)
	lint := requireStructuredLint(t, lintByPath["cfg/settings.toml"])
	if lint["success"] != false {
		t.Fatalf("lint = %#v, want success=false", lint)
	}
	if strings.TrimSpace(asString(lint["output"])) == "" {
		t.Fatalf("lint output missing parser evidence: %#v", lint)
	}
	assertFileContent(t, path, "name =\n")
}

func TestWriteFileToolPythonLintReportsSyntaxError(t *testing.T) {
	root := t.TempDir()
	tool := NewWriteFileTool(FileTaskToolConfig{Root: root})
	content := "def broken(:\n    pass\n"

	out := executeWriteFileTool(t, tool, `{"path":"src/bad.py","content":`+quoteJSON(t, content)+`}`)
	if out["status"] != "ok" {
		t.Fatalf("write result = %#v, want ok with lint evidence", out)
	}
	lint := requireStructuredLint(t, out["lint"])
	if lint["success"] != false {
		t.Fatalf("lint = %#v, want success=false", lint)
	}
	if !strings.Contains(asString(lint["output"]), "SyntaxError") {
		t.Fatalf("lint output = %#v, want Python syntax evidence", lint)
	}
	assertFileContent(t, filepath.Join(root, "src", "bad.py"), content)
}

func TestPatchToolPythonLintDeltaPreservesPreExistingError(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src", "bad.py")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte("x =\nname = 'old'\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/bad.py"}`)

	out := executePatchTool(t, NewPatchTool(cfg), `{"path":"src/bad.py","old_string":"'old'","new_string":"'new'"}`)
	if out["status"] != "ok" {
		t.Fatalf("patch result = %#v, want ok with lint evidence", out)
	}
	lint := requireStructuredLint(t, out["lint"])
	if lint["success"] != false {
		t.Fatalf("lint = %#v, want success=false", lint)
	}
	if !strings.Contains(asString(lint["message"]), "Pre-existing lint errors") {
		t.Fatalf("lint message = %#v, want pre-existing error note", lint)
	}
	assertFileContent(t, path, "x =\nname = 'new'\n")
}

func TestPatchToolV4APythonLintReportsPerFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src", "settings.py")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte("value = 1\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/settings.py"}`)

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Update File: src/settings.py",
		"@@",
		"-value = 1",
		"+value =",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)
	if out["status"] != "ok" {
		t.Fatalf("patch result = %#v, want ok with lint evidence", out)
	}
	lintByPath, _ := out["lint"].(map[string]any)
	lint := requireStructuredLint(t, lintByPath["src/settings.py"])
	if lint["success"] != false {
		t.Fatalf("lint = %#v, want success=false", lint)
	}
	if !strings.Contains(asString(lint["output"]), "SyntaxError") {
		t.Fatalf("lint output = %#v, want Python syntax evidence", lint)
	}
	assertFileContent(t, path, "value =\n")
}

func TestWriteFileToolShellLintReportsJavaScriptErrors(t *testing.T) {
	root := t.TempDir()
	var calledName string
	var calledArgs []string
	withShellLintStubs(t,
		func(name string) (string, error) {
			if name != "node" {
				t.Fatalf("looked up %q, want node", name)
			}
			return "/usr/bin/node", nil
		},
		func(ctx context.Context, name string, args ...string) fileLintCommandResult {
			calledName = name
			calledArgs = append([]string(nil), args...)
			return fileLintCommandResult{Output: "SyntaxError: Unexpected token", ExitCode: 1}
		},
	)

	content := "function broken( {\n"
	out := executeWriteFileTool(t, NewWriteFileTool(FileTaskToolConfig{Root: root}), `{"path":"src/bad.js","content":`+quoteJSON(t, content)+`}`)
	if out["status"] != "ok" {
		t.Fatalf("write result = %#v, want ok with lint evidence", out)
	}
	lint := requireStructuredLint(t, out["lint"])
	if lint["success"] != false {
		t.Fatalf("lint = %#v, want success=false", lint)
	}
	if !strings.Contains(asString(lint["output"]), "SyntaxError") {
		t.Fatalf("lint output = %#v, want shell linter evidence", lint)
	}
	if calledName != "/usr/bin/node" || len(calledArgs) != 2 || calledArgs[0] != "--check" || !strings.HasSuffix(calledArgs[1], filepath.Join("src", "bad.js")) {
		t.Fatalf("shell lint command = %q %#v, want node --check <resolved bad.js>", calledName, calledArgs)
	}
	assertFileContent(t, filepath.Join(root, "src", "bad.js"), content)
}

func TestPatchToolShellLintSkipsMissingExternalLinter(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src", "app.js")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte("const value = 1;\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	withShellLintStubs(t,
		func(name string) (string, error) {
			if name != "node" {
				t.Fatalf("looked up %q, want node", name)
			}
			return "", errors.New("missing")
		},
		func(ctx context.Context, name string, args ...string) fileLintCommandResult {
			t.Fatalf("shell linter should not run when lookup fails")
			return fileLintCommandResult{}
		},
	)

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/app.js"}`)

	out := executePatchTool(t, NewPatchTool(cfg), `{"path":"src/app.js","old_string":"1","new_string":"2"}`)
	if out["status"] != "ok" {
		t.Fatalf("patch result = %#v, want ok with skipped lint evidence", out)
	}
	lint := requireStructuredLint(t, out["lint"])
	if lint["status"] != "skipped" || !strings.Contains(asString(lint["message"]), "node not available") {
		t.Fatalf("lint = %#v, want skipped node evidence", lint)
	}
	assertFileContent(t, path, "const value = 2;\n")
}

func TestPatchToolV4AShellLintReportsPerFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src", "app.js")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte("const value = 1;\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/app.js"}`)
	withShellLintStubs(t,
		func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		},
		func(ctx context.Context, name string, args ...string) fileLintCommandResult {
			return fileLintCommandResult{Output: "SyntaxError: Unexpected end of input", ExitCode: 1}
		},
	)

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Update File: src/app.js",
		"@@",
		"-const value = 1;",
		"+const value = ;",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)
	if out["status"] != "ok" {
		t.Fatalf("patch result = %#v, want ok with lint evidence", out)
	}
	lintByPath, _ := out["lint"].(map[string]any)
	lint := requireStructuredLint(t, lintByPath["src/app.js"])
	if lint["success"] != false || !strings.Contains(asString(lint["output"]), "SyntaxError") {
		t.Fatalf("lint = %#v, want per-file shell lint error", lint)
	}
	assertFileContent(t, path, "const value = ;\n")
}

func TestShellLintCommandUsesResolvedPathWithBraces(t *testing.T) {
	root := t.TempDir()
	var lintedPath string
	withShellLintStubs(t,
		func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		},
		func(ctx context.Context, name string, args ...string) fileLintCommandResult {
			if len(args) == 2 {
				lintedPath = args[1]
			}
			return fileLintCommandResult{ExitCode: 0}
		},
	)

	out := executeWriteFileTool(t, NewWriteFileTool(FileTaskToolConfig{Root: root}), `{"path":"src/{test}_file.js","content":"const ok = true;\n"}`)
	if out["status"] != "ok" {
		t.Fatalf("write result = %#v, want ok", out)
	}
	if !strings.Contains(lintedPath, "{test}_file.js") {
		t.Fatalf("linted path = %q, want literal braces preserved", lintedPath)
	}
	if !strings.HasPrefix(lintedPath, root) {
		t.Fatalf("linted path = %q, want resolved path under %q", lintedPath, root)
	}
}

func TestShellLintCommandCoversHermesExtensions(t *testing.T) {
	cases := map[string]string{
		"file.js": "node",
		"file.ts": "npx",
		"file.go": "go",
		"file.rs": "rustfmt",
	}
	for path, want := range cases {
		spec, ok := shellLintSpecForPath(path)
		if !ok {
			t.Fatalf("shellLintSpecForPath(%q) missing", path)
		}
		if spec.command != want {
			t.Fatalf("shellLintSpecForPath(%q).command = %q, want %q", path, spec.command, want)
		}
	}
}

func requireStructuredLint(t *testing.T, raw any) map[string]any {
	t.Helper()
	lint, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("lint = %#v, want object", raw)
	}
	return lint
}

func withShellLintStubs(t *testing.T, lookup func(string) (string, error), run func(context.Context, string, ...string) fileLintCommandResult) {
	t.Helper()
	origLookup := fileLintLookPath
	origRun := fileLintRunCommand
	fileLintLookPath = lookup
	fileLintRunCommand = run
	t.Cleanup(func() {
		fileLintLookPath = origLookup
		fileLintRunCommand = origRun
	})
}
