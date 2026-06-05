package gormescli

import "testing"

func TestMemoryCommandConstructorReturnsIndependentInstances(t *testing.T) {
	a := NewMemoryCommand(MemoryCommandOptions{})
	b := NewMemoryCommand(MemoryCommandOptions{})
	if a == b {
		t.Fatal("NewMemoryCommand must return distinct instances; got same pointer")
	}
	if a.Commands()[0] == b.Commands()[0] {
		t.Fatal("subcommand instances must also be distinct between constructor calls")
	}
}

func TestMemoryCommandDefaultsRowBackedSubcommands(t *testing.T) {
	cmd := NewMemoryCommand(MemoryCommandOptions{})
	for _, name := range []string{"status", "setup", "off", "reset"} {
		child, _, err := cmd.Find([]string{name})
		if err != nil || child == nil || child.Name() != name {
			t.Fatalf("memory subcommand %q missing: child=%v err=%v", name, child, err)
		}
	}
}
