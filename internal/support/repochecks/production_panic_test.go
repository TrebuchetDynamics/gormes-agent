package repochecks_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestProductionPanicsAreExplicitlyClassified(t *testing.T) {
	allowed := map[string]string{
		"internal/adapters/tuigateway/gateway_mux.go:367":                      "mustRawJSON marshals fixed in-process response maps",
		"internal/automation/cron/safety/prompt_script_safety.go:147":          "static cron threat pattern table must compile at init",
		"internal/app/setup/registry.go:66":                                    "static setup registry metadata invariant",
		"internal/core/subagent/lifecycle/identity/ids.go:18":                  "crypto/rand failure is unrecoverable for ID uniqueness",
		"internal/planning/progress/split.go:164":                              "typed progress split key exhaustiveness invariant",
		"internal/planning/progress/split.go:266":                              "typed progress split key exhaustiveness invariant",
		"internal/platform/cli/gormescli/contractruntime/setup_registry.go:72": "static CLI setup registry metadata invariant",
		"internal/platform/cli/gormescli/plugins.go:23":                        "miswired plugin lifecycle manager programming error",
		"internal/platform/cli/gormescli/rootruntime/root.go:208":              "static root command factory table invariant",
		"internal/tools/browser_harness_backend.go:186":                        "nil transport programming error in constructor",
		"internal/tools/discord/toolsets/toolset.go:226":                       "static Discord toolset registration invariant",
		"internal/tools/goncho/honcho/adapter/tools.go:18":                     "nil registry programming error in tool registration",
		"internal/tools/goncho/honcho/adapter/tools.go:21":                     "nil goncho service programming error in tool registration",
		"internal/tools/goncho/memoryv1/tools.go:22":                           "nil registry programming error in tool registration",
		"internal/tools/goncho/memoryv1/tools.go:25":                           "nil memory store programming error in tool registration",
		"internal/tools/safety/commandpatterns/patterns.go:38":                 "static safety regex table must compile at init",
		"internal/tools/toolkit/core/tool.go:125":                              "MustRegister main-time convenience mirrors Go Must* pattern",
		"internal/tools/whisper/engine/wasi/runtime/emval/values.go:22":        "WASI bridge invariant: invalid memory read cannot continue safely",
		"internal/tools/whisper/engine/wasi/runtime/emval/values.go:28":        "WASI bridge invariant: unsupported module property cannot continue safely",
	}

	root := repoRoot(t)
	found := map[string]struct{}{}
	for _, dir := range []string{"cmd", "internal"} {
		walkGoFiles(t, filepath.Join(root, dir), func(path string) {
			if strings.HasSuffix(path, "_test.go") {
				return
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				ident, ok := call.Fun.(*ast.Ident)
				if !ok || ident.Name != "panic" {
					return true
				}
				rel := filepath.ToSlash(mustRel(t, root, path))
				key := rel + ":" + itoa(fset.Position(call.Pos()).Line)
				found[key] = struct{}{}
				if _, ok := allowed[key]; !ok {
					t.Errorf("unclassified production panic at %s", key)
				}
				return true
			})
		})
	}
	for key := range allowed {
		if _, ok := found[key]; !ok {
			t.Errorf("classified production panic missing or moved: %s (%s)", key, allowed[key])
		}
	}
}

func walkGoFiles(t *testing.T, root string, visit func(string)) {
	t.Helper()
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			visit(path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func mustRel(t *testing.T, base, target string) string {
	t.Helper()
	rel, err := filepath.Rel(base, target)
	if err != nil {
		t.Fatalf("rel %s to %s: %v", base, target, err)
	}
	return rel
}

func itoa(v int) string { return strconv.Itoa(v) }
