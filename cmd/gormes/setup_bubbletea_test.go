package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupInteractiveMenusUseBubbleTeaPicker(t *testing.T) {
	for _, path := range []string{"setup.go", "setup_first_run.go"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		if strings.Contains(text, "cli.NewInteractiveMenu") {
			t.Fatalf("%s still uses the legacy raw-mode menu instead of the Bubble Tea setup picker", path)
		}
		if !strings.Contains(text, "runBubbleTeaPick") {
			t.Fatalf("%s does not route interactive setup selection through the Bubble Tea setup picker", path)
		}
	}
}

func TestNoLegacyRawInteractiveMenuImplementation(t *testing.T) {
	root := filepath.Join("..", "..")
	forbidden := []string{
		"New" + "InteractiveMenu",
		"type " + "InteractiveMenu",
		"cli." + "MenuOption",
		"type " + "MenuOption",
		"Prompt" + "YesNo",
		"term." + "MakeRaw",
	}
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(raw)
			for _, needle := range forbidden {
				if strings.Contains(text, needle) {
					t.Fatalf("%s contains legacy raw interactive menu marker %q; interactive selectors must use Bubble Tea", path, needle)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", dir, err)
		}
	}
}
