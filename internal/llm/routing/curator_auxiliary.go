package routing

import "strings"

type CuratorAuxiliaryEvidenceCode string

const (
	CuratorAuxiliaryAutoMain        CuratorAuxiliaryEvidenceCode = "curator_auxiliary_auto_main"
	CuratorAuxiliaryPartialFallback CuratorAuxiliaryEvidenceCode = "curator_auxiliary_partial_fallback"
	CuratorAuxiliaryLegacyConfig    CuratorAuxiliaryEvidenceCode = "curator_auxiliary_legacy_config"
	CuratorAuxiliarySecretStripped  CuratorAuxiliaryEvidenceCode = "curator_auxiliary_secret_stripped"
	CuratorAuxiliarySlotMissing     CuratorAuxiliaryEvidenceCode = "curator_auxiliary_slot_missing"
)

type CuratorAuxiliarySlot struct {
	Provider  string
	Model     string
	BaseURL   string
	APIKey    string
	Timeout   int
	ExtraBody map[string]any
}

type CuratorAuxiliaryRequest struct {
	Main      ModelRoute
	Canonical CuratorAuxiliarySlot
	Legacy    CuratorAuxiliarySlot
}

type CuratorAuxiliaryEvidence struct {
	Code     CuratorAuxiliaryEvidenceCode
	Source   string
	Message  string
	Redacted bool
}

type CuratorAuxiliaryBinding struct {
	Route           ModelRoute
	ExplicitAPIKey  string
	ExplicitBaseURL string
	Timeout         int
	ExtraBody       map[string]any
	Evidence        []CuratorAuxiliaryEvidence
}

func ResolveCuratorAuxiliary(req CuratorAuxiliaryRequest) CuratorAuxiliaryBinding {
	main := normalizeCuratorMainRoute(req.Main)
	canonical := normalizeCuratorSlot(req.Canonical)
	legacy := normalizeCuratorSlot(req.Legacy)
	var evidence []CuratorAuxiliaryEvidence

	if curatorSlotConfigured(canonical) {
		if curatorSlotComplete(canonical) {
			return CuratorAuxiliaryBinding{
				Route:           ModelRoute{Provider: canonical.Provider, Model: canonical.Model},
				ExplicitAPIKey:  stripCuratorCredential(canonical.APIKey, &evidence, "auxiliary.curator.api_key"),
				ExplicitBaseURL: stripCuratorCredential(canonical.BaseURL, &evidence, "auxiliary.curator.base_url"),
				Timeout:         canonical.Timeout,
				ExtraBody:       cloneCuratorExtraBody(canonical.ExtraBody),
				Evidence:        evidence,
			}
		}
		if curatorSlotPartial(canonical) {
			evidence = append(evidence, curatorAuxiliaryEvidence(CuratorAuxiliaryPartialFallback, "auxiliary.curator", "partial auxiliary.curator provider/model falls back to the main model"))
			appendIgnoredCuratorCredentialEvidence(canonical, &evidence)
		} else {
			evidence = append(evidence, curatorAuxiliaryEvidence(CuratorAuxiliaryAutoMain, "auxiliary.curator", "auxiliary.curator is auto; curator review uses the main model"))
			appendIgnoredCuratorCredentialEvidence(canonical, &evidence)
		}
	} else {
		evidence = append(evidence, curatorAuxiliaryEvidence(CuratorAuxiliarySlotMissing, "auxiliary.curator", "auxiliary.curator slot is missing; curator review uses the main model"))
	}

	if curatorSlotComplete(legacy) {
		evidence = append(evidence, curatorAuxiliaryEvidence(CuratorAuxiliaryLegacyConfig, "curator.auxiliary", "deprecated curator.auxiliary provider/model is honored; migrate to auxiliary.curator"))
		return CuratorAuxiliaryBinding{
			Route:           ModelRoute{Provider: legacy.Provider, Model: legacy.Model},
			ExplicitAPIKey:  stripCuratorCredential(legacy.APIKey, &evidence, "curator.auxiliary.api_key"),
			ExplicitBaseURL: stripCuratorCredential(legacy.BaseURL, &evidence, "curator.auxiliary.base_url"),
			Timeout:         legacy.Timeout,
			ExtraBody:       cloneCuratorExtraBody(legacy.ExtraBody),
			Evidence:        evidence,
		}
	}

	return CuratorAuxiliaryBinding{
		Route:     main,
		Timeout:   canonical.Timeout,
		ExtraBody: cloneCuratorExtraBody(canonical.ExtraBody),
		Evidence:  evidence,
	}
}

func (b CuratorAuxiliaryBinding) HasEvidence(code CuratorAuxiliaryEvidenceCode) bool {
	for _, evidence := range b.Evidence {
		if evidence.Code == code {
			return true
		}
	}
	return false
}

func normalizeCuratorMainRoute(route ModelRoute) ModelRoute {
	route.Provider = strings.TrimSpace(route.Provider)
	route.Model = strings.TrimSpace(route.Model)
	if route.Provider == "" {
		route.Provider = "auto"
	}
	return route
}

func normalizeCuratorSlot(slot CuratorAuxiliarySlot) CuratorAuxiliarySlot {
	slot.Provider = strings.TrimSpace(slot.Provider)
	slot.Model = strings.TrimSpace(slot.Model)
	slot.APIKey = strings.TrimSpace(slot.APIKey)
	slot.BaseURL = strings.TrimSpace(slot.BaseURL)
	if slot.ExtraBody == nil {
		slot.ExtraBody = map[string]any{}
	}
	return slot
}

func curatorSlotConfigured(slot CuratorAuxiliarySlot) bool {
	return slot.Provider != "" || slot.Model != "" || slot.APIKey != "" || slot.BaseURL != "" || slot.Timeout != 0 || len(slot.ExtraBody) > 0
}

func curatorSlotComplete(slot CuratorAuxiliarySlot) bool {
	return slot.Provider != "" && slot.Provider != "auto" && slot.Model != ""
}

func curatorSlotPartial(slot CuratorAuxiliarySlot) bool {
	if slot.Provider != "" && slot.Provider != "auto" && slot.Model == "" {
		return true
	}
	if (slot.Provider == "" || slot.Provider == "auto") && slot.Model != "" {
		return true
	}
	return false
}

func stripCuratorCredential(raw string, evidence *[]CuratorAuxiliaryEvidence, source string) string {
	trimmed := strings.TrimSpace(raw)
	if raw != "" && trimmed == "" {
		*evidence = append(*evidence, curatorAuxiliaryEvidence(CuratorAuxiliarySecretStripped, source, "blank auxiliary credential stripped"))
	}
	return trimmed
}

func appendIgnoredCuratorCredentialEvidence(slot CuratorAuxiliarySlot, evidence *[]CuratorAuxiliaryEvidence) {
	if strings.TrimSpace(slot.APIKey) != "" {
		*evidence = append(*evidence, curatorAuxiliaryEvidence(CuratorAuxiliarySecretStripped, "auxiliary.curator.api_key", "ignored auxiliary api_key while falling back to the main model"))
	}
	if strings.TrimSpace(slot.BaseURL) != "" {
		*evidence = append(*evidence, curatorAuxiliaryEvidence(CuratorAuxiliarySecretStripped, "auxiliary.curator.base_url", "ignored auxiliary base_url while falling back to the main model"))
	}
}

func curatorAuxiliaryEvidence(code CuratorAuxiliaryEvidenceCode, source string, message string) CuratorAuxiliaryEvidence {
	return CuratorAuxiliaryEvidence{
		Code:     code,
		Source:   source,
		Message:  message,
		Redacted: true,
	}
}

func cloneCuratorExtraBody(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
