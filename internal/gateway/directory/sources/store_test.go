package sources

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
)

func TestStoreLoadSanitizesDecodedLedgerUpdatedAt(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, channelDirectorySourcesFileName), []byte(`{"updated_at":"now\nforged=true","platforms":{"telegram":[{"platform":"telegram","id":"42","name":"Ops"}]}}`), 0o600); err != nil {
		t.Fatalf("write ledger: %v", err)
	}

	ledger, evidence := NewStore(root).Load()
	if evidence.Code != "" {
		t.Fatalf("evidence = %+v, want none", evidence)
	}
	if strings.Contains(ledger.UpdatedAt, "\n") || ledger.UpdatedAt != "now forged=true" {
		t.Fatalf("UpdatedAt = %q, want sanitized single-line value", ledger.UpdatedAt)
	}
}

func TestStoreLoadNormalizesDecodedLedger(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, channelDirectorySourcesFileName), []byte(`{"updated_at":" now ","platforms":{" Telegram ":[{"platform":" Telegram ","chat_id":" -100 ","chat_name":" Ops ","thread_id":" 7 ","chat_topic":" Release Room "}]}}`), 0o600); err != nil {
		t.Fatalf("write padded ledger: %v", err)
	}

	ledger, evidence := NewStore(root).Load()
	if evidence.Code != "" {
		t.Fatalf("evidence = %+v, want none", evidence)
	}
	if _, ok := ledger.Platforms[" Telegram "]; ok {
		t.Fatalf("platform keys = %+v, want raw platform key normalized away", ledger.Platforms)
	}
	entries := ledger.Platforms["telegram"]
	if len(entries) != 1 {
		t.Fatalf("telegram entries = %+v, want one normalized entry", entries)
	}
	got := entries[0]
	if got.Platform != "telegram" || got.ID != "-100:7" || got.Name != "Ops / Release Room" || got.ChatID != "-100" || got.ThreadID != "7" || got.ChatTopic != "Release Room" {
		t.Fatalf("entry = %+v, want normalized remembered source", got)
	}
}

func TestRememberSourceHonorsCanceledContextBeforeWriting(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.RememberSource(ctx, model.RememberedSourceEntry{Platform: "telegram", ID: "42", Name: "Ops"})
	if err == nil {
		t.Fatal("RememberSource canceled context err = nil, want context cancellation")
	}
	if _, statErr := os.Stat(filepath.Join(root, channelDirectorySourcesFileName)); !os.IsNotExist(statErr) {
		t.Fatalf("ledger stat after canceled RememberSource = %v, want no file created", statErr)
	}
}

func TestRememberSourcePropagatesCorruptLedger(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, channelDirectorySourcesFileName)
	if err := os.WriteFile(path, []byte(`{"platforms":`), 0o600); err != nil {
		t.Fatalf("write corrupt ledger: %v", err)
	}

	err := NewStore(root).RememberSource(context.Background(), model.RememberedSourceEntry{Platform: "telegram", ID: "42", Name: "Ops"})
	if err == nil {
		t.Fatal("RememberSource err = nil, want corrupt ledger decode error")
	}
	contents, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read ledger after failed remember: %v", readErr)
	}
	if string(contents) != `{"platforms":` {
		t.Fatalf("ledger contents = %q, want corrupt ledger preserved", contents)
	}
}

func TestZeroValueStoreLoadIgnoresProcessWorkingDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(channelDirectorySourcesFileName, []byte(`{"platforms":{"telegram":[{"platform":"telegram","id":"leaked","name":"from cwd"}]}}`), 0o600); err != nil {
		t.Fatalf("write cwd ledger: %v", err)
	}

	ledger, evidence := (Store{}).Load()
	if evidence.Code != "" {
		t.Fatalf("evidence = %+v, want none for unconfigured store", evidence)
	}
	if got := ledger.Platforms["telegram"]; len(got) != 0 {
		t.Fatalf("telegram entries = %+v, want zero-value store to ignore cwd ledger", got)
	}
}
