package inventory

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/fidelity"
)

func TestBuildHermesContractInventoryRejectsStaleLatestBaseline(t *testing.T) {
	root := hermesContractInventoryFixtureRoot(t)
	report, err := BuildHermesContractInventory(HermesContractInventoryOptions{
		Root:          root,
		RequireLatest: true,
		OriginRevision: func(context.Context, string) (string, error) {
			return "def456", nil
		},
		BaselineDelta: func(context.Context, string, string, string) (fidelity.BaselineDelta, error) {
			return fidelity.BaselineDelta{Added: []string{"agent/new_feature.py"}, Modified: []string{"agent/curator.py"}}, nil
		},
	})
	if err != nil {
		t.Fatalf("BuildHermesContractInventory: %v", err)
	}
	if report.OK {
		t.Fatal("stale baseline report OK = true, want false")
	}
	if report.BaselineFreshness == nil {
		t.Fatal("BaselineFreshness missing")
	}
	if report.BaselineFreshness.State != "stale" || report.BaselineFreshness.RecordedSHA != "abc123" || report.BaselineFreshness.LatestSHA != "def456" {
		t.Fatalf("freshness = %+v", report.BaselineFreshness)
	}
	if report.BaselineDelta == nil || len(report.BaselineDelta.Added) != 1 || len(report.BaselineDelta.Modified) != 1 {
		t.Fatalf("delta = %+v", report.BaselineDelta)
	}
}

func TestBuildHermesContractInventoryAcceptsMatchingLatestBaseline(t *testing.T) {
	root := hermesContractInventoryFixtureRoot(t)
	report, err := BuildHermesContractInventory(HermesContractInventoryOptions{
		Root:          root,
		RequireLatest: true,
		OriginRevision: func(context.Context, string) (string, error) {
			return "abc123", nil
		},
	})
	if err != nil {
		t.Fatalf("BuildHermesContractInventory: %v", err)
	}
	if !report.OK {
		t.Fatal("matching baseline report OK = false, want true")
	}
	if report.BaselineFreshness == nil || report.BaselineFreshness.State != "match" {
		t.Fatalf("freshness = %+v", report.BaselineFreshness)
	}
}

func TestBuildHermesContractInventoryFailsClosedWhenOriginUnverified(t *testing.T) {
	root := hermesContractInventoryFixtureRoot(t)
	report, err := BuildHermesContractInventory(HermesContractInventoryOptions{
		Root:          root,
		RequireLatest: true,
		OriginRevision: func(context.Context, string) (string, error) {
			return "", errors.New("origin/main unavailable")
		},
	})
	if err != nil {
		t.Fatalf("BuildHermesContractInventory: %v", err)
	}
	if report.OK {
		t.Fatal("unverified baseline report OK = true, want false")
	}
	if report.BaselineFreshness == nil || report.BaselineFreshness.State != "unverified" {
		t.Fatalf("freshness = %+v", report.BaselineFreshness)
	}
	if !strings.Contains(report.BaselineFreshness.Error, "origin/main unavailable") {
		t.Fatalf("freshness error = %q", report.BaselineFreshness.Error)
	}
}
