package hermes

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"
)

type ProviderTimeoutConfigLoader func() (map[string]any, error)

type ProviderTimeoutEvidence string

const (
	ProviderTimeoutConfigured        ProviderTimeoutEvidence = "timeout_configured"
	ProviderTimeoutConfigUnavailable ProviderTimeoutEvidence = "timeout_config_unavailable"
	ProviderTimeoutConfigInvalid     ProviderTimeoutEvidence = "timeout_config_invalid"
	ProviderTimeoutUnset             ProviderTimeoutEvidence = "timeout_unset"
)

type ProviderTimeoutResolution struct {
	Timeout    time.Duration
	Evidence   ProviderTimeoutEvidence
	ProviderID string
	Model      string
	Field      string
	Source     string
	Reason     string
}

func ResolveProviderRequestTimeout(load ProviderTimeoutConfigLoader, providerID, model string) ProviderTimeoutResolution {
	return resolveProviderTimeout(load, providerID, model, "request_timeout_seconds", "timeout_seconds")
}

func ResolveProviderStaleTimeout(load ProviderTimeoutConfigLoader, providerID, model string) ProviderTimeoutResolution {
	return resolveProviderTimeout(load, providerID, model, "stale_timeout_seconds", "stale_timeout_seconds")
}

func resolveProviderTimeout(load ProviderTimeoutConfigLoader, providerID, model, providerField, modelField string) ProviderTimeoutResolution {
	res := ProviderTimeoutResolution{
		Evidence:   ProviderTimeoutUnset,
		ProviderID: strings.TrimSpace(providerID),
		Model:      strings.TrimSpace(model),
		Field:      providerField,
	}
	if res.ProviderID == "" {
		res.Reason = "provider id unset"
		return res
	}
	if load == nil {
		res.Evidence = ProviderTimeoutConfigUnavailable
		res.Reason = "timeout config loader unavailable"
		return res
	}

	cfg, err := load()
	if err != nil {
		res.Evidence = ProviderTimeoutConfigUnavailable
		res.Reason = "timeout config unavailable"
		return res
	}
	providers, ok := mapStringAny(cfg["providers"])
	if !ok {
		res.Reason = "providers unset"
		return res
	}
	providerCfg, ok := mapStringAny(providers[res.ProviderID])
	if !ok {
		res.Reason = "provider timeout config unset"
		return res
	}

	if res.Model != "" {
		if modelCfg, ok := modelTimeoutConfig(providerCfg, res.Model); ok {
			if raw, exists := modelCfg[modelField]; exists && raw != nil {
				res.Field = modelField
				res.Source = "model"
				return resolveTimeoutValue(res, raw)
			}
		}
	}

	raw, exists := providerCfg[providerField]
	if !exists || raw == nil {
		res.Reason = "timeout unset"
		return res
	}
	res.Source = "provider"
	return resolveTimeoutValue(res, raw)
}

func modelTimeoutConfig(providerCfg map[string]any, model string) (map[string]any, bool) {
	models, ok := mapStringAny(providerCfg["models"])
	if !ok {
		return nil, false
	}
	return mapStringAny(models[model])
}

func resolveTimeoutValue(res ProviderTimeoutResolution, raw any) ProviderTimeoutResolution {
	timeout, ok := parseProviderTimeout(raw)
	if !ok {
		res.Timeout = 0
		res.Evidence = ProviderTimeoutConfigInvalid
		res.Reason = "timeout value invalid"
		return res
	}
	res.Timeout = timeout
	res.Evidence = ProviderTimeoutConfigured
	res.Reason = "timeout configured"
	return res
}

func mapStringAny(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	if !ok || m == nil {
		return nil, false
	}
	return m, true
}

func parseProviderTimeout(raw any) (time.Duration, bool) {
	switch v := raw.(type) {
	case time.Duration:
		return positiveDuration(v)
	case string:
		return parseProviderTimeoutString(v)
	case json.Number:
		seconds, err := strconv.ParseFloat(v.String(), 64)
		if err != nil {
			return 0, false
		}
		return secondsDuration(seconds)
	case float64:
		return secondsDuration(v)
	case float32:
		return secondsDuration(float64(v))
	case int:
		return secondsDuration(float64(v))
	case int8:
		return secondsDuration(float64(v))
	case int16:
		return secondsDuration(float64(v))
	case int32:
		return secondsDuration(float64(v))
	case int64:
		return secondsDuration(float64(v))
	case uint:
		return secondsDuration(float64(v))
	case uint8:
		return secondsDuration(float64(v))
	case uint16:
		return secondsDuration(float64(v))
	case uint32:
		return secondsDuration(float64(v))
	case uint64:
		return secondsDuration(float64(v))
	default:
		return 0, false
	}
}

func parseProviderTimeoutString(raw string) (time.Duration, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false
	}
	if d, err := time.ParseDuration(value); err == nil {
		return positiveDuration(d)
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return secondsDuration(seconds)
}

func secondsDuration(seconds float64) (time.Duration, bool) {
	if seconds <= 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return 0, false
	}
	if seconds > float64(math.MaxInt64)/float64(time.Second) {
		return 0, false
	}
	return positiveDuration(time.Duration(seconds * float64(time.Second)))
}

func positiveDuration(d time.Duration) (time.Duration, bool) {
	if d <= 0 {
		return 0, false
	}
	return d, true
}
