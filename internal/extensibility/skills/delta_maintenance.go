package skills

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/maintenance"
)

type DeltaMaintenanceSourceKind = maintenance.DeltaMaintenanceSourceKind

const (
	DeltaMaintenanceSourceSkill  DeltaMaintenanceSourceKind = maintenance.DeltaMaintenanceSourceSkill
	DeltaMaintenanceSourceMemory DeltaMaintenanceSourceKind = maintenance.DeltaMaintenanceSourceMemory
)

type DeltaMaintenanceStatus = maintenance.DeltaMaintenanceStatus

const (
	DeltaMaintenanceStatusNoop          DeltaMaintenanceStatus = maintenance.DeltaMaintenanceStatusNoop
	DeltaMaintenanceStatusProcessed     DeltaMaintenanceStatus = maintenance.DeltaMaintenanceStatusProcessed
	DeltaMaintenanceStatusNeedsFullScan DeltaMaintenanceStatus = maintenance.DeltaMaintenanceStatusNeedsFullScan
	DeltaMaintenanceStatusInvalid       DeltaMaintenanceStatus = maintenance.DeltaMaintenanceStatusInvalid
)

const (
	DeltaMaintenanceEvidenceNoop                       = maintenance.DeltaMaintenanceEvidenceNoop
	DeltaMaintenanceEvidenceSourcesMissing             = maintenance.DeltaMaintenanceEvidenceSourcesMissing
	DeltaMaintenanceEvidenceSourceAnchorPerSource      = maintenance.DeltaMaintenanceEvidenceSourceAnchorPerSource
	DeltaMaintenanceEvidenceSourceAnchorMissing        = maintenance.DeltaMaintenanceEvidenceSourceAnchorMissing
	DeltaMaintenanceEvidenceSourceAnchorGlobalFallback = maintenance.DeltaMaintenanceEvidenceSourceAnchorGlobalFallback
	DeltaMaintenanceEvidenceStaleRefreshEmpty          = maintenance.DeltaMaintenanceEvidenceStaleRefreshEmpty
	DeltaMaintenanceEvidenceFullScanFallback           = maintenance.DeltaMaintenanceEvidenceFullScanFallback
)

type SourceAnchorMode = maintenance.SourceAnchorMode

const (
	SourceAnchorModePerSource      SourceAnchorMode = maintenance.SourceAnchorModePerSource
	SourceAnchorModeMissing        SourceAnchorMode = maintenance.SourceAnchorModeMissing
	SourceAnchorModeGlobalFallback SourceAnchorMode = maintenance.SourceAnchorModeGlobalFallback
)

type DeltaMaintenanceSource = maintenance.DeltaMaintenanceSource

type SourceMaintenanceAnchor = maintenance.SourceMaintenanceAnchor

type SourceAnchorEvidence = maintenance.SourceAnchorEvidence

type StaleMaintenanceRecord = maintenance.StaleMaintenanceRecord

type DeltaMaintenanceRequest = maintenance.DeltaMaintenanceRequest

type DeltaMaintenanceHandlers = maintenance.DeltaMaintenanceHandlers

type DeltaMaintenancePlan = maintenance.DeltaMaintenancePlan

func RunDeltaMaintenance(ctx context.Context, req DeltaMaintenanceRequest, handlers DeltaMaintenanceHandlers) (DeltaMaintenancePlan, error) {
	return maintenance.RunDeltaMaintenance(ctx, req, handlers)
}
