package telegram

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestTelegramCommandUsesInjectedRun(t *testing.T) {
	called := false
	cmd := NewTelegramCommandWithSeams(TelegramCommandSeams{
		Run: func(_ *cobra.Command, _ []string) error {
			called = true
			return nil
		},
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("telegram: %v", err)
	}
	if !called {
		t.Fatal("telegram run seam was not called")
	}
}
