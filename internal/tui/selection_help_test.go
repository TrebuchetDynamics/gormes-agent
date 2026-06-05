package tui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// forbiddenCopyHotkeys lists user-visible copy/clipboard advertisements that
// the native Bubble Tea TUI must not promise: Hermes Ink's custom selection
// keybindings, OSC 52 clipboard escapes, and any "Ink" runtime mention.
// Matching is case-insensitive at the call site.
var forbiddenCopyHotkeys = []string{
	"Cmd+C",
	"Ctrl+C",
	"Ctrl-Shift-C",
	"Cmd-Shift-C",
	"OSC 52",
	"clipboard hotkey",
	"Ink",
}

func TestTerminalNativeSelectionHelpCompatibilityWrapper(t *testing.T) {
	if got := SelectionHelpLine(); got != TerminalNativeSelectionHelp {
		t.Errorf("SelectionHelpLine() = %q; want %q", got, TerminalNativeSelectionHelp)
	}
}

// TestTUIPackageDoesNotAdvertiseCopyHotkey reads every non-test .go file in
// internal/tui via os.ReadFile and parses it so we walk only string literals
// (not comments or identifiers); none of those literals may advertise a
// copy/clipboard shortcut that this package does not actually implement.
// Test files are skipped so this fixture itself can name the forbidden
// tokens in its allowlist above.
func TestTUIPackageDoesNotAdvertiseCopyHotkey(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read tui package dir: %v", err)
	}

	fset := token.NewFileSet()
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}

		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		file, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		scanned++

		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			val, err := strconv.Unquote(lit.Value)
			if err != nil {
				val = lit.Value
			}
			lower := strings.ToLower(val)
			for _, bad := range forbiddenCopyHotkeys {
				if strings.Contains(lower, strings.ToLower(bad)) {
					t.Errorf("%s contains forbidden copy-hotkey advertisement %q in string literal %q",
						name, bad, val)
				}
			}
			return true
		})
	}

	if scanned == 0 {
		t.Fatal("no production .go files were scanned in internal/tui")
	}
}
