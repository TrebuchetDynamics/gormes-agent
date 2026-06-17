package dashboard

import "testing"

func TestNewCronReaderOpensInFreshHome(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	reader, closer, ok := newCronReader()
	if !ok {
		t.Skip("cron store unavailable in this environment")
	}
	defer closer()
	if _, err := reader.List(); err != nil {
		t.Fatalf("cron List: %v", err)
	}
}

func TestNewSessionsListerUnavailableWithoutMemoryDB(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	// No memory.db in a fresh home -> directory open fails -> ok=false (graceful).
	if _, _, ok := newSessionsLister(); ok {
		t.Fatal("expected sessions lister unavailable when memory.db is absent")
	}
}
