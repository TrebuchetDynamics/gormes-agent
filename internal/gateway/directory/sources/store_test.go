package sources

import (
	"os"
	"testing"
)

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
