// Package internal_test runs cross-package invariants that can't live inside
// any single package — specifically, the capacity-must-be-declared rule from
// spec §7.8. This test walks the AST of every non-test .go file under
// internal/ and fails on any make(chan ...) call without a capacity
// argument.
package internal_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNoUnboundedChannelsInInternal(t *testing.T) {
	// Resolve the absolute path to internal/ from this test file's
	// location. The test runs from internal/ package directory.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	internalDir := filepath.Dir(file)
	// Safety: confirm the directory name is "internal" so we don't scan the
	// wrong tree if the test is relocated.
	if filepath.Base(internalDir) != "internal" {
		t.Fatalf("expected to run from internal/, got %s", internalDir)
	}

	fset := token.NewFileSet()
	violations := 0

	walkErr := filepath.Walk(internalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(fset, path, src, parser.AllErrors)
		if err != nil {
			t.Logf("parse %s: %v (skipping)", path, err)
			return nil
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok || id.Name != "make" {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			// Only flag chan makes. Slices/maps are fine with 1-arg make.
			if _, isChan := call.Args[0].(*ast.ChanType); !isChan {
				return true
			}
			if len(call.Args) < 2 {
				pos := fset.Position(call.Pos())
				relPath, _ := filepath.Rel(internalDir, path)
				t.Errorf("%s:%d unbounded make(chan ...) — spec §7.8 requires explicit capacity", relPath, pos.Line)
				violations++
			}
			return true
		})
		return nil
	})

	if walkErr != nil {
		t.Fatalf("walk %s: %v", internalDir, walkErr)
	}
	if violations == 0 {
		t.Logf("internal/ channel-capacity invariant holds")
	}
}
