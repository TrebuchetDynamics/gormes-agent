package skills

import (
	"context"
	"reflect"
	"testing"
)

func TestDeltaMaintenance_NoChangedSourcesReturnsNoop(t *testing.T) {
	var extracted, scored, embedded bool
	plan, err := RunDeltaMaintenance(context.Background(), DeltaMaintenanceRequest{
		ChangedSourcesKnown: true,
		ChangedSources:      nil,
	}, DeltaMaintenanceHandlers{
		Extract: func(context.Context, []DeltaMaintenanceSource) error {
			extracted = true
			return nil
		},
		Score: func(context.Context, []DeltaMaintenanceSource) error {
			scored = true
			return nil
		},
		EmbedStale: func(context.Context, []StaleMaintenanceRecord) error {
			embedded = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("RunDeltaMaintenance: %v", err)
	}
	if plan.Status != DeltaMaintenanceStatusNoop || !plan.HasEvidence(DeltaMaintenanceEvidenceNoop) {
		t.Fatalf("plan = %#v, want delta_noop status/evidence", plan)
	}
	if extracted || scored || embedded {
		t.Fatalf("callbacks invoked extracted=%v scored=%v embedded=%v, want none", extracted, scored, embedded)
	}
}

func TestDeltaMaintenance_ProcessesOnlyChangedSources(t *testing.T) {
	var extracted []DeltaMaintenanceSource
	var scored []DeltaMaintenanceSource
	changed := []DeltaMaintenanceSource{
		{ID: "skill:review", Kind: DeltaMaintenanceSourceSkill},
		{ID: "memory:acme", Kind: DeltaMaintenanceSourceMemory},
	}
	plan, err := RunDeltaMaintenance(context.Background(), DeltaMaintenanceRequest{
		ChangedSourcesKnown: true,
		ChangedSources:      changed,
		SourceAnchors: map[string]SourceMaintenanceAnchor{
			"skill:review": {LastCommit: "abc123"},
			"memory:acme":  {LastCommit: "def456"},
		},
	}, DeltaMaintenanceHandlers{
		Extract: func(_ context.Context, sources []DeltaMaintenanceSource) error {
			extracted = append([]DeltaMaintenanceSource(nil), sources...)
			return nil
		},
		Score: func(_ context.Context, sources []DeltaMaintenanceSource) error {
			scored = append([]DeltaMaintenanceSource(nil), sources...)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("RunDeltaMaintenance: %v", err)
	}
	if plan.Status != DeltaMaintenanceStatusProcessed {
		t.Fatalf("status = %q, want processed", plan.Status)
	}
	if !reflect.DeepEqual(extracted, changed) {
		t.Fatalf("extracted = %#v, want changed only %#v", extracted, changed)
	}
	if !reflect.DeepEqual(scored, changed) {
		t.Fatalf("scored = %#v, want changed only %#v", scored, changed)
	}
	if got := plan.ProcessedSourceIDs; !reflect.DeepEqual(got, []string{"skill:review", "memory:acme"}) {
		t.Fatalf("ProcessedSourceIDs = %#v, want changed IDs", got)
	}
}

func TestDeltaMaintenance_SourceAnchorEvidence(t *testing.T) {
	plan, err := RunDeltaMaintenance(context.Background(), DeltaMaintenanceRequest{
		ChangedSourcesKnown: true,
		ChangedSources: []DeltaMaintenanceSource{
			{ID: "memory:acme", Kind: DeltaMaintenanceSourceMemory},
		},
		SourceAnchors: map[string]SourceMaintenanceAnchor{
			"memory:acme": {LastCommit: "source-commit"},
		},
	}, DeltaMaintenanceHandlers{})
	if err != nil {
		t.Fatalf("RunDeltaMaintenance: %v", err)
	}
	anchor := plan.SourceAnchors["memory:acme"]
	if anchor.Mode != SourceAnchorModePerSource || anchor.LastCommit != "source-commit" {
		t.Fatalf("anchor = %#v, want per-source source-commit", anchor)
	}
	if !plan.HasEvidence(DeltaMaintenanceEvidenceSourceAnchorPerSource) {
		t.Fatalf("evidence = %#v, want per-source evidence", plan.Evidence)
	}
}

func TestDeltaMaintenance_GlobalAnchorFallbackIsExplicit(t *testing.T) {
	plan, err := RunDeltaMaintenance(context.Background(), DeltaMaintenanceRequest{
		ChangedSourcesKnown: true,
		ChangedSources: []DeltaMaintenanceSource{
			{ID: "memory:acme", Kind: DeltaMaintenanceSourceMemory},
		},
		GlobalAnchor: SourceMaintenanceAnchor{LastCommit: "global-commit"},
	}, DeltaMaintenanceHandlers{})
	if err != nil {
		t.Fatalf("RunDeltaMaintenance: %v", err)
	}
	anchor := plan.SourceAnchors["memory:acme"]
	if anchor.Mode != SourceAnchorModeGlobalFallback || anchor.LastCommit != "global-commit" {
		t.Fatalf("anchor = %#v, want global fallback", anchor)
	}
	if !plan.HasEvidence(DeltaMaintenanceEvidenceSourceAnchorGlobalFallback) {
		t.Fatalf("evidence = %#v, want source_anchor_global_fallback", plan.Evidence)
	}
}

func TestDeltaMaintenance_StaleRefreshSkipsWhenCountZero(t *testing.T) {
	var loaded, embedded bool
	plan, err := RunDeltaMaintenance(context.Background(), DeltaMaintenanceRequest{
		ChangedSourcesKnown: true,
		ChangedSources: []DeltaMaintenanceSource{
			{ID: "skill:review", Kind: DeltaMaintenanceSourceSkill},
		},
		SourceAnchors: map[string]SourceMaintenanceAnchor{
			"skill:review": {LastCommit: "abc123"},
		},
		RefreshStale: true,
	}, DeltaMaintenanceHandlers{
		CountStale: func(context.Context) (int, error) {
			return 0, nil
		},
		LoadStale: func(context.Context) ([]StaleMaintenanceRecord, error) {
			loaded = true
			return nil, nil
		},
		EmbedStale: func(context.Context, []StaleMaintenanceRecord) error {
			embedded = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("RunDeltaMaintenance: %v", err)
	}
	if plan.StaleCount != 0 || !plan.HasEvidence(DeltaMaintenanceEvidenceStaleRefreshEmpty) {
		t.Fatalf("plan = %#v, want stale_refresh_empty", plan)
	}
	if loaded || embedded {
		t.Fatalf("loaded=%v embedded=%v, want no stale loader/embedder calls", loaded, embedded)
	}
}

func TestDeltaMaintenance_FullScanFallbackIsExplicit(t *testing.T) {
	var extracted bool
	plan, err := RunDeltaMaintenance(context.Background(), DeltaMaintenanceRequest{}, DeltaMaintenanceHandlers{
		Extract: func(context.Context, []DeltaMaintenanceSource) error {
			extracted = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("RunDeltaMaintenance: %v", err)
	}
	if plan.Status != DeltaMaintenanceStatusNeedsFullScan || !plan.FullScanFallback {
		t.Fatalf("plan = %#v, want full-scan fallback", plan)
	}
	if !plan.HasEvidence(DeltaMaintenanceEvidenceFullScanFallback) {
		t.Fatalf("evidence = %#v, want full_scan_fallback", plan.Evidence)
	}
	if extracted {
		t.Fatal("extract callback invoked during fallback-only plan")
	}
}
