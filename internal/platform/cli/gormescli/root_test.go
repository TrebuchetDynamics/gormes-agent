package gormescli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewRootCommandOwnsRootFlagsAndFactories(t *testing.T) {
	calledFinalizer := false
	cmd := NewRootCommand(RootOptions{
		Version: "test-version",
		Finalizers: []func(*cobra.Command){
			func(*cobra.Command) { calledFinalizer = true },
		},
	}, stubRootFactories())

	if cmd.Use != "gormes" || cmd.Version != "test-version" {
		t.Fatalf("root identity = use %q version %q", cmd.Use, cmd.Version)
	}
	for _, name := range []string{"profile", "skills", "model", "provider", "endpoint", "api-key", "offline"} {
		if cmd.PersistentFlags().Lookup(name) == nil {
			t.Fatalf("persistent flag %q missing", name)
		}
	}
	if flag := cmd.Flags().Lookup("continue"); flag == nil || flag.NoOptDefVal != "last" {
		t.Fatalf("continue flag = %#v, want no-opt default last", flag)
	}
	for _, name := range []string{"gateway", "channels", "agent", "dashboard"} {
		if child, _, err := cmd.Find([]string{name}); err != nil || child == nil || child.Name() != name {
			t.Fatalf("root command %q missing: child=%v err=%v", name, child, err)
		}
	}
	if !calledFinalizer {
		t.Fatal("root finalizer was not called")
	}
}

func stubRootFactories() CommandFactories {
	factories := CommandFactories{}
	for _, name := range RootCommandOrder {
		name := name
		factories[name] = func() *cobra.Command {
			if name == "login" {
				return NewDeprecatedLoginCommand()
			}
			return &cobra.Command{Use: name}
		}
	}
	return factories
}
