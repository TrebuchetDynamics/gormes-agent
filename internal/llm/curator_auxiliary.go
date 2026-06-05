package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing"

type CuratorAuxiliaryEvidenceCode = routing.CuratorAuxiliaryEvidenceCode

const (
	CuratorAuxiliaryAutoMain        CuratorAuxiliaryEvidenceCode = routing.CuratorAuxiliaryAutoMain
	CuratorAuxiliaryPartialFallback CuratorAuxiliaryEvidenceCode = routing.CuratorAuxiliaryPartialFallback
	CuratorAuxiliaryLegacyConfig    CuratorAuxiliaryEvidenceCode = routing.CuratorAuxiliaryLegacyConfig
	CuratorAuxiliarySecretStripped  CuratorAuxiliaryEvidenceCode = routing.CuratorAuxiliarySecretStripped
	CuratorAuxiliarySlotMissing     CuratorAuxiliaryEvidenceCode = routing.CuratorAuxiliarySlotMissing
)

type CuratorAuxiliarySlot = routing.CuratorAuxiliarySlot
type CuratorAuxiliaryRequest = routing.CuratorAuxiliaryRequest
type CuratorAuxiliaryEvidence = routing.CuratorAuxiliaryEvidence
type CuratorAuxiliaryBinding = routing.CuratorAuxiliaryBinding

func ResolveCuratorAuxiliary(req CuratorAuxiliaryRequest) CuratorAuxiliaryBinding {
	return routing.ResolveCuratorAuxiliary(req)
}
