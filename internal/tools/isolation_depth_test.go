package tools

import (
	"testing"
)

func TestIsolationLevel_String(t *testing.T) {
	if IsolationProcess.String() != "process" {
		t.Fatal("process string mismatch")
	}
	if IsolationContainer.String() != "container" {
		t.Fatal("container string mismatch")
	}
	if IsolationVM.String() != "vm" {
		t.Fatal("vm string mismatch")
	}
}

func TestDefaultIsolationConfig(t *testing.T) {
	cfg := DefaultIsolationConfig()
	if cfg.Level != IsolationProcess {
		t.Fatal("default should be process")
	}
	if !cfg.IsAvailable() {
		t.Fatal("process isolation should always be available")
	}
}

func TestIsolationConfig_Availability(t *testing.T) {
	if !(IsolationConfig{Level: IsolationProcess}).IsAvailable() {
		t.Fatal("process should be available")
	}
	if (IsolationConfig{Level: IsolationContainer}).IsAvailable() {
		t.Fatal("container without image should not be available")
	}
	if !(IsolationConfig{Level: IsolationContainer, ContainerImage: "ubuntu"}).IsAvailable() {
		t.Fatal("container with image should be available")
	}
	if (IsolationConfig{Level: IsolationVM}).IsAvailable() {
		t.Fatal("vm without socket should not be available")
	}
}

func TestParseIsolationLevel(t *testing.T) {
	level, ok := ParseIsolationLevel("process")
	if !ok || level != IsolationProcess {
		t.Fatal("parse process failed")
	}
	level, ok = ParseIsolationLevel("container")
	if !ok || level != IsolationContainer {
		t.Fatal("parse container failed")
	}
	_, ok = ParseIsolationLevel("invalid")
	if ok {
		t.Fatal("invalid parse should fail")
	}
}
