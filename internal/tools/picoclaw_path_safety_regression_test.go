package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/sandbox"
)

func TestPicoClawPathSafety_RelativePathsStayUnderWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	outside := filepath.Join(root, "outside")
	prefixSibling := filepath.Join(root, "workspace-sibling")
	processCWD := filepath.Join(root, "process-cwd")
	for _, dir := range []string{workspace, outside, prefixSibling, processCWD} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(workspace, "file.txt"), []byte("workspace"), 0o644); err != nil {
		t.Fatalf("write workspace fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(processCWD, "file.txt"), []byte("process cwd"), 0o644); err != nil {
		t.Fatalf("write process cwd fixture: %v", err)
	}

	oldCWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(processCWD); err != nil {
		t.Fatalf("chdir process fixture: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldCWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	scope := NewFilesystemScope(workspace, nil, nil)
	for _, rel := range []string{"./file.txt", "file.txt"} {
		result := scope.CheckRead(rel, workspace)
		if !result.Allowed {
			t.Fatalf("CheckRead(%q) denied: %+v", rel, result)
		}
		want := filepath.Join(workspace, "file.txt")
		if result.Normalized != want {
			t.Fatalf("CheckRead(%q) normalized = %q, want workspace path %q", rel, result.Normalized, want)
		}
	}

	escaped := scope.CheckRead("../outside/secret.txt", workspace)
	if escaped.Allowed {
		t.Fatalf("../outside path allowed with normalized path %q", escaped.Normalized)
	}
	if escaped.Evidence != "filesystem_read_scope_violation" {
		t.Fatalf("../outside evidence = %q, want filesystem_read_scope_violation", escaped.Evidence)
	}

	sibling := scope.CheckRead(filepath.Join(prefixSibling, "secret.txt"), workspace)
	if sibling.Allowed {
		t.Fatalf("prefix sibling path allowed with normalized path %q", sibling.Normalized)
	}
}

func TestPicoClawPathSafety_FindRootDenied(t *testing.T) {
	workspace := t.TempDir()
	classifier := NewCommandClassifier()
	for _, command := range []string{"find /", "find / -maxdepth 1", "find -- / -maxdepth 1"} {
		decision := classifier.ClassifyDetailed(command)
		if decision.Class != CommandUnsafe || !decision.Blocked {
			t.Fatalf("ClassifyDetailed(%q) = %+v, want unsafe blocked", command, decision)
		}

		guard := GuardCommand(command, ApprovalModeOff)
		if guard.Approved || !guard.Hardline {
			t.Fatalf("GuardCommand(%q) = %+v, want hardline block even with approvals off", command, guard)
		}
	}

	tool := NewTerminalTool(TerminalToolConfig{Workdir: workspace, ApprovalMode: ApprovalModeOff})
	out := executeTerminalTool(t, tool, `{"command":"find / -maxdepth 1","timeout":1}`)
	if out["status"] != "blocked" {
		t.Fatalf("terminal status = %v, want blocked: %#v", out["status"], out)
	}
	if !strings.Contains(asString(out["description"]), "root") && !strings.Contains(asString(out["error"]), "root") {
		t.Fatalf("terminal block = %#v, want root-enumeration evidence", out)
	}

	outside := filepath.Join(filepath.Dir(workspace), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("mkdir outside: %v", err)
	}
	link := filepath.Join(workspace, "outside-link")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink outside fixture: %v", err)
	}
	scope := NewFilesystemScope(workspace, nil, nil)
	result := scope.CheckWrite(filepath.Join("outside-link", "created.txt"), workspace)
	if result.Allowed {
		t.Fatalf("symlink escape allowed with normalized path %q", result.Normalized)
	}
}

func TestPicoClawPathSafety_CommandGuardSeesParsedBody(t *testing.T) {
	classifier := NewCommandClassifier()
	cases := []struct {
		name    string
		tool    string
		key     string
		command string
	}{
		{name: "terminal command", tool: "terminal", key: "command", command: "find / -maxdepth 1"},
		{name: "command field", tool: "safe_tool_name", key: "command", command: "find -- / -maxdepth 1"},
		{name: "cmd field", tool: "safe_tool_name", key: "cmd", command: "find / -type f"},
		{name: "execute_code code field", tool: "execute_code", key: "code", command: "find / -maxdepth 1"},
		{name: "script field", tool: "safe_tool_name", key: "script", command: "find / -maxdepth 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input, err := json.Marshal(map[string]string{tc.key: tc.command})
			if err != nil {
				t.Fatalf("marshal input: %v", err)
			}
			decision := classifier.ClassifyToolRequest(ToolRequest{ToolName: tc.tool, Input: input})
			if decision.Command != tc.command {
				t.Fatalf("decision command = %q, want parsed body %q", decision.Command, tc.command)
			}
			if decision.Class != CommandUnsafe || !decision.Blocked {
				t.Fatalf("ClassifyToolRequest(%s) = %+v, want unsafe blocked", tc.name, decision)
			}
		})
	}
}

func TestPicoClawPathSafety_VirtualPathMaskingPreserved(t *testing.T) {
	root := t.TempDir()
	hostRoot := filepath.Join(root, "host-data")
	hostSibling := filepath.Join(root, "host-data-sibling")
	if err := os.MkdirAll(filepath.Join(hostRoot, "workspace"), 0o755); err != nil {
		t.Fatalf("mkdir host root: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(hostSibling, "workspace"), 0o755); err != nil {
		t.Fatalf("mkdir host sibling: %v", err)
	}
	resolver := sandbox.NewVirtualPathResolver("/mnt/user-data", hostRoot)

	hostOutput := "saved " + filepath.Join(hostRoot, "workspace", "result.txt")
	got := resolver.MaskOutput(hostOutput)
	want := "saved /mnt/user-data/workspace/result.txt"
	if got != want {
		t.Fatalf("MaskOutput host path = %q, want %q", got, want)
	}
	if strings.Contains(got, filepath.ToSlash(hostRoot)) {
		t.Fatalf("MaskOutput leaked host root in %q", got)
	}

	siblingOutput := "read " + filepath.Join(hostSibling, "workspace", "secret.txt")
	if masked := resolver.MaskOutput(siblingOutput); masked != siblingOutput {
		t.Fatalf("MaskOutput prefix sibling = %q, want unchanged %q", masked, siblingOutput)
	}
	if _, err := resolver.HostToVirtual(filepath.Join(hostSibling, "workspace", "secret.txt")); err == nil {
		t.Fatalf("HostToVirtual accepted prefix sibling host path")
	}
	if _, err := resolver.Resolve("/mnt/user-data-sibling/workspace/secret.txt"); err == nil {
		t.Fatalf("Resolve accepted prefix sibling virtual path")
	}
	if _, err := resolver.PathFamily("/mnt/user-data-sibling/workspace/secret.txt"); err == nil {
		t.Fatalf("PathFamily accepted prefix sibling virtual path")
	}
}
