package progress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

var itemKnownJSONFields = map[string]bool{
	"name":             true,
	"priority":         true,
	"status":           true,
	"contract":         true,
	"contract_status":  true,
	"slice_size":       true,
	"execution_owner":  true,
	"module":           true,
	"trust_class":      true,
	"degraded_mode":    true,
	"fixture":          true,
	"source_refs":      true,
	"ready_when":       true,
	"not_ready_when":   true,
	"blocked_by":       true,
	"unblocks":         true,
	"acceptance":       true,
	"note":             true,
	"blocker":          true,
	"write_scope":      true,
	"test_commands":    true,
	"no_test_required": true,
	"done_signal":      true,
	"wired":            true,
	"pr":               true,
	"owner":            true,
	"eta":              true,
	"health":           true,
	"planner_verdict":  true,
	"provenance":       true,
}

var itemKnownSliceJSONFields = map[string]bool{
	"trust_class":    true,
	"source_refs":    true,
	"ready_when":     true,
	"not_ready_when": true,
	"blocked_by":     true,
	"unblocks":       true,
	"acceptance":     true,
	"write_scope":    true,
	"test_commands":  true,
	"done_signal":    true,
}

func (it *Item) UnmarshalJSON(data []byte) error {
	type itemAlias Item
	var alias itemAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	extra := map[string]json.RawMessage{}
	for key, value := range raw {
		if !itemKnownJSONFields[key] || itemPreserveKnownZeroValue(key, value) {
			extra[key] = append(json.RawMessage(nil), value...)
		}
	}

	*it = Item(alias)
	if len(extra) > 0 {
		it.Extra = extra
	}
	return nil
}

func (it Item) MarshalJSON() ([]byte, error) {
	type itemAlias Item
	alias := itemAlias(it)
	alias.Extra = nil

	body, err := marshalNoEscape(alias)
	if err != nil {
		return nil, err
	}
	return marshalObjectWithExtra(it.Name, body, it.Extra)
}

func (p *Provenance) UnmarshalJSON(data []byte) error {
	type provenanceAlias Provenance
	var alias provenanceAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	extra, err := unknownObjectFields(data, provenanceKnownJSONFields, nil)
	if err != nil {
		return err
	}
	*p = Provenance(alias)
	if len(extra) > 0 {
		p.Extra = extra
	}
	return nil
}

func (p Provenance) MarshalJSON() ([]byte, error) {
	type provenanceAlias Provenance
	alias := provenanceAlias(p)
	alias.Extra = nil

	body, err := marshalNoEscape(alias)
	if err != nil {
		return nil, err
	}
	return marshalObjectWithExtra("provenance", body, p.Extra)
}

func (v *PlannerVerdict) UnmarshalJSON(data []byte) error {
	type verdictAlias PlannerVerdict
	var alias verdictAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	extra, err := unknownObjectFields(data, plannerVerdictKnownJSONFields, plannerVerdictPreserveKnownZeroValue)
	if err != nil {
		return err
	}
	*v = PlannerVerdict(alias)
	if len(extra) > 0 {
		v.Extra = extra
	}
	return nil
}

func (v PlannerVerdict) MarshalJSON() ([]byte, error) {
	type verdictAlias PlannerVerdict
	alias := verdictAlias(v)
	alias.Extra = nil

	body, err := marshalNoEscape(alias)
	if err != nil {
		return nil, err
	}
	return marshalObjectWithExtra("planner_verdict", body, v.Extra)
}

func (h *RowHealth) UnmarshalJSON(data []byte) error {
	type healthAlias RowHealth
	var alias healthAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	extra, err := unknownObjectFields(data, rowHealthKnownJSONFields, nil)
	if err != nil {
		return err
	}
	*h = RowHealth(alias)
	if len(extra) > 0 {
		h.Extra = extra
	}
	return nil
}

func (h RowHealth) MarshalJSON() ([]byte, error) {
	type healthAlias RowHealth
	alias := healthAlias(h)
	alias.Extra = nil

	body, err := marshalNoEscape(alias)
	if err != nil {
		return nil, err
	}
	return marshalObjectWithExtra("health", body, h.Extra)
}

func (s *Subphase) UnmarshalJSON(data []byte) error {
	type subphaseAlias Subphase
	var alias subphaseAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	extra, err := unknownObjectFields(data, subphaseKnownJSONFields, nil)
	if err != nil {
		return err
	}
	*s = Subphase(alias)
	if len(extra) > 0 {
		s.Extra = extra
	}
	return nil
}

func (s Subphase) MarshalJSON() ([]byte, error) {
	type subphaseAlias Subphase
	alias := subphaseAlias(s)
	alias.Extra = nil

	body, err := marshalNoEscape(alias)
	if err != nil {
		return nil, err
	}
	return marshalObjectWithExtra("subphase", body, s.Extra)
}

func marshalObjectWithExtra(label string, body []byte, extra map[string]json.RawMessage) ([]byte, error) {
	if len(extra) == 0 {
		return body, nil
	}
	var known map[string]json.RawMessage
	if err := json.Unmarshal(body, &known); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(extra))
	for key, value := range extra {
		if _, exists := known[key]; exists {
			continue
		}
		value = bytes.TrimSpace(value)
		if len(value) == 0 {
			continue
		}
		if !json.Valid(value) {
			return nil, fmt.Errorf("progress %s extra field %q is not valid JSON", label, key)
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return body, nil
	}
	sort.Strings(keys)

	trimmed := bytes.TrimSpace(body)
	if !bytes.HasPrefix(trimmed, []byte("{")) || !bytes.HasSuffix(trimmed, []byte("}")) {
		return nil, fmt.Errorf("progress %s marshaled to non-object JSON", label)
	}
	var out bytes.Buffer
	out.Write(trimmed[:len(trimmed)-1])
	needsComma := len(trimmed) > 2
	for _, key := range keys {
		if needsComma {
			out.WriteByte(',')
		}
		needsComma = true
		quoted, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		out.Write(quoted)
		out.WriteByte(':')
		out.Write(bytes.TrimSpace(extra[key]))
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

type preserveKnownZeroFunc func(string, json.RawMessage) bool

func unknownObjectFields(data []byte, knownFields map[string]bool, preserveKnownZero preserveKnownZeroFunc) (map[string]json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	extra := map[string]json.RawMessage{}
	for key, value := range raw {
		if !knownFields[key] || (preserveKnownZero != nil && preserveKnownZero(key, value)) {
			extra[key] = append(json.RawMessage(nil), value...)
		}
	}
	if len(extra) == 0 {
		return nil, nil
	}
	return extra, nil
}

func itemPreserveKnownZeroValue(key string, value json.RawMessage) bool {
	if !itemKnownSliceJSONFields[key] {
		return key == "no_test_required" && (bytes.Equal(bytes.TrimSpace(value), []byte(`""`)) || bytes.Equal(bytes.TrimSpace(value), []byte("null")))
	}
	return bytes.Equal(bytes.TrimSpace(value), []byte("[]"))
}

var provenanceKnownJSONFields = map[string]bool{
	"origin_type":   true,
	"upstream_ref":  true,
	"upstream_refs": true,
	"owned_since":   true,
	"note":          true,
}

var plannerVerdictKnownJSONFields = map[string]bool{
	"needs_human":   true,
	"reason":        true,
	"since":         true,
	"reshape_count": true,
	"last_reshape":  true,
	"last_outcome":  true,
}

func plannerVerdictPreserveKnownZeroValue(key string, value json.RawMessage) bool {
	trimmed := bytes.TrimSpace(value)
	switch key {
	case "needs_human":
		return bytes.Equal(trimmed, []byte("false"))
	case "reason", "since", "last_reshape", "last_outcome":
		return bytes.Equal(trimmed, []byte(`""`))
	case "reshape_count":
		return bytes.Equal(trimmed, []byte("0"))
	default:
		return false
	}
}

var rowHealthKnownJSONFields = map[string]bool{
	"attempt_count":        true,
	"consecutive_failures": true,
	"last_attempt":         true,
	"last_success":         true,
	"last_failure":         true,
	"backends_tried":       true,
	"quarantine":           true,
}

var subphaseKnownJSONFields = map[string]bool{
	"name":        true,
	"priority":    true,
	"items":       true,
	"status":      true,
	"drift_state": true,
}
