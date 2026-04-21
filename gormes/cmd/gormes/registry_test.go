package main

import (
	"testing"
)

func TestBuildDefaultRegistryIncludesCoreToolsOnly(t *testing.T) {
	reg := buildDefaultRegistry()
	for _, name := range []string{"echo", "now", "rand_int"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("tool %q not registered", name)
		}
	}
	if _, ok := reg.Get("delegate_task"); ok {
		t.Fatal("delegate_task unexpectedly registered")
	}
}
