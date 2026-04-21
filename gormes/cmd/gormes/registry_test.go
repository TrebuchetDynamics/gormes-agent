package main

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
)

func TestBuildDefaultRegistryDelegationDisabled(t *testing.T) {
	reg := buildDefaultRegistry(context.Background(), config.DelegationCfg{})
	if _, ok := reg.Get("delegate_task"); ok {
		t.Fatal("delegate_task unexpectedly registered")
	}
}

func TestBuildDefaultRegistryDelegationEnabled(t *testing.T) {
	reg := buildDefaultRegistry(context.Background(), config.DelegationCfg{
		Enabled:               true,
		MaxDepth:              2,
		MaxConcurrentChildren: 4,
		DefaultMaxIterations:  9,
		DefaultTimeout:        time.Minute,
	})
	if _, ok := reg.Get("delegate_task"); !ok {
		t.Fatal("delegate_task not registered")
	}
}
