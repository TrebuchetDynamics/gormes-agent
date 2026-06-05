package maintenance

import (
	"context"
	"strings"
)

type DeltaMaintenanceSourceKind string

const (
	DeltaMaintenanceSourceSkill  DeltaMaintenanceSourceKind = "skill"
	DeltaMaintenanceSourceMemory DeltaMaintenanceSourceKind = "memory"
)

type DeltaMaintenanceStatus string

const (
	DeltaMaintenanceStatusNoop          DeltaMaintenanceStatus = "delta_noop"
	DeltaMaintenanceStatusProcessed     DeltaMaintenanceStatus = "processed"
	DeltaMaintenanceStatusNeedsFullScan DeltaMaintenanceStatus = "needs_full_scan"
	DeltaMaintenanceStatusInvalid       DeltaMaintenanceStatus = "invalid"
)

const (
	DeltaMaintenanceEvidenceNoop                       = "delta_noop"
	DeltaMaintenanceEvidenceSourcesMissing             = "delta_sources_missing"
	DeltaMaintenanceEvidenceSourceAnchorPerSource      = "source_anchor_per_source"
	DeltaMaintenanceEvidenceSourceAnchorMissing        = "source_anchor_missing"
	DeltaMaintenanceEvidenceSourceAnchorGlobalFallback = "source_anchor_global_fallback"
	DeltaMaintenanceEvidenceStaleRefreshEmpty          = "stale_refresh_empty"
	DeltaMaintenanceEvidenceFullScanFallback           = "full_scan_fallback"
)

type SourceAnchorMode string

const (
	SourceAnchorModePerSource      SourceAnchorMode = "per_source"
	SourceAnchorModeMissing        SourceAnchorMode = "missing"
	SourceAnchorModeGlobalFallback SourceAnchorMode = "global_fallback"
)

// DeltaMaintenanceSource identifies one changed skill or memory source that a
// learning-loop maintenance pass may touch without walking the whole corpus.
type DeltaMaintenanceSource struct {
	ID   string                     `json:"id"`
	Kind DeltaMaintenanceSourceKind `json:"kind"`
}

type SourceMaintenanceAnchor struct {
	LastCommit string `json:"last_commit,omitempty"`
}

type SourceAnchorEvidence struct {
	Mode       SourceAnchorMode `json:"mode"`
	LastCommit string           `json:"last_commit,omitempty"`
}

type StaleMaintenanceRecord struct {
	ID       string                     `json:"id"`
	SourceID string                     `json:"source_id,omitempty"`
	Kind     DeltaMaintenanceSourceKind `json:"kind,omitempty"`
	Text     string                     `json:"text,omitempty"`
}

type DeltaMaintenanceRequest struct {
	// ChangedSourcesKnown distinguishes a deliberate empty delta from missing
	// delta evidence. Missing evidence must be visible before any broad scan.
	ChangedSourcesKnown bool
	ChangedSources      []DeltaMaintenanceSource

	SourceAnchors map[string]SourceMaintenanceAnchor
	GlobalAnchor  SourceMaintenanceAnchor

	RefreshStale bool
}

type DeltaMaintenanceHandlers struct {
	Extract func(context.Context, []DeltaMaintenanceSource) error
	Score   func(context.Context, []DeltaMaintenanceSource) error

	CountStale func(context.Context) (int, error)
	LoadStale  func(context.Context) ([]StaleMaintenanceRecord, error)
	EmbedStale func(context.Context, []StaleMaintenanceRecord) error
}

type DeltaMaintenancePlan struct {
	Status             DeltaMaintenanceStatus          `json:"status"`
	Evidence           []string                        `json:"evidence,omitempty"`
	ProcessedSourceIDs []string                        `json:"processed_source_ids,omitempty"`
	SourceAnchors      map[string]SourceAnchorEvidence `json:"source_anchors,omitempty"`
	StaleCount         int                             `json:"stale_count,omitempty"`
	FullScanFallback   bool                            `json:"full_scan_fallback,omitempty"`
}

func (p DeltaMaintenancePlan) HasEvidence(evidence string) bool {
	for _, item := range p.Evidence {
		if item == evidence {
			return true
		}
	}
	return false
}

func RunDeltaMaintenance(ctx context.Context, req DeltaMaintenanceRequest, handlers DeltaMaintenanceHandlers) (DeltaMaintenancePlan, error) {
	if err := ctx.Err(); err != nil {
		return DeltaMaintenancePlan{}, err
	}
	if !req.ChangedSourcesKnown {
		return DeltaMaintenancePlan{
			Status:           DeltaMaintenanceStatusNeedsFullScan,
			Evidence:         []string{DeltaMaintenanceEvidenceFullScanFallback},
			FullScanFallback: true,
			SourceAnchors:    map[string]SourceAnchorEvidence{},
		}, nil
	}

	sources := normalizedMaintenanceSources(req.ChangedSources)
	if len(sources) == 0 {
		return DeltaMaintenancePlan{
			Status:        DeltaMaintenanceStatusNoop,
			Evidence:      []string{DeltaMaintenanceEvidenceNoop},
			SourceAnchors: map[string]SourceAnchorEvidence{},
		}, nil
	}

	plan := DeltaMaintenancePlan{
		Status:             DeltaMaintenanceStatusProcessed,
		SourceAnchors:      make(map[string]SourceAnchorEvidence, len(sources)),
		ProcessedSourceIDs: make([]string, 0, len(sources)),
	}
	for _, source := range sources {
		if strings.TrimSpace(source.ID) == "" {
			plan.Status = DeltaMaintenanceStatusInvalid
			plan.addEvidence(DeltaMaintenanceEvidenceSourcesMissing)
			return plan, nil
		}
		plan.ProcessedSourceIDs = append(plan.ProcessedSourceIDs, source.ID)
		plan.SourceAnchors[source.ID] = sourceAnchorEvidence(source.ID, req)
		plan.addEvidence(evidenceForSourceAnchor(plan.SourceAnchors[source.ID].Mode))
	}

	if handlers.Extract != nil {
		if err := handlers.Extract(ctx, sources); err != nil {
			return plan, err
		}
	}
	if handlers.Score != nil {
		if err := handlers.Score(ctx, sources); err != nil {
			return plan, err
		}
	}
	if req.RefreshStale && handlers.CountStale != nil {
		count, err := handlers.CountStale(ctx)
		if err != nil {
			return plan, err
		}
		if count < 0 {
			count = 0
		}
		plan.StaleCount = count
		if count == 0 {
			plan.addEvidence(DeltaMaintenanceEvidenceStaleRefreshEmpty)
			return plan, nil
		}
		if handlers.LoadStale != nil {
			records, err := handlers.LoadStale(ctx)
			if err != nil {
				return plan, err
			}
			if handlers.EmbedStale != nil {
				if err := handlers.EmbedStale(ctx, records); err != nil {
					return plan, err
				}
			}
		}
	}
	return plan, nil
}

func normalizedMaintenanceSources(in []DeltaMaintenanceSource) []DeltaMaintenanceSource {
	if len(in) == 0 {
		return nil
	}
	out := make([]DeltaMaintenanceSource, 0, len(in))
	seen := map[string]bool{}
	for _, source := range in {
		source.ID = strings.TrimSpace(source.ID)
		if source.ID == "" {
			out = append(out, source)
			continue
		}
		if seen[source.ID] {
			continue
		}
		seen[source.ID] = true
		out = append(out, source)
	}
	return out
}

func sourceAnchorEvidence(sourceID string, req DeltaMaintenanceRequest) SourceAnchorEvidence {
	if anchor, ok := req.SourceAnchors[sourceID]; ok && strings.TrimSpace(anchor.LastCommit) != "" {
		return SourceAnchorEvidence{Mode: SourceAnchorModePerSource, LastCommit: strings.TrimSpace(anchor.LastCommit)}
	}
	if strings.TrimSpace(req.GlobalAnchor.LastCommit) != "" {
		return SourceAnchorEvidence{Mode: SourceAnchorModeGlobalFallback, LastCommit: strings.TrimSpace(req.GlobalAnchor.LastCommit)}
	}
	return SourceAnchorEvidence{Mode: SourceAnchorModeMissing}
}

func evidenceForSourceAnchor(mode SourceAnchorMode) string {
	switch mode {
	case SourceAnchorModePerSource:
		return DeltaMaintenanceEvidenceSourceAnchorPerSource
	case SourceAnchorModeGlobalFallback:
		return DeltaMaintenanceEvidenceSourceAnchorGlobalFallback
	default:
		return DeltaMaintenanceEvidenceSourceAnchorMissing
	}
}

func (p *DeltaMaintenancePlan) addEvidence(evidence string) {
	if evidence == "" || p.HasEvidence(evidence) {
		return
	}
	p.Evidence = append(p.Evidence, evidence)
}
