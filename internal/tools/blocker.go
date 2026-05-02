package tools

import (
	"fmt"
	"sort"
	"strings"
)

type BlockerType string

const (
	BlockerTypeAccess     BlockerType = "access"
	BlockerTypeInfra      BlockerType = "infra"
	BlockerTypeDependency BlockerType = "dependency"
	BlockerTypeDecision   BlockerType = "decision"
	BlockerTypeBug        BlockerType = "bug"
	BlockerTypeUnknown    BlockerType = "unknown"
)

type BlockerStatus string

const (
	BlockerStatusActive       BlockerStatus = "blocker_active"
	BlockerStatusUnclassified BlockerStatus = "blocker_unclassified"
)

type BlockerRecord struct {
	Title         string        `json:"title,omitempty"`
	Type          BlockerType   `json:"type,omitempty"`
	Status        BlockerStatus `json:"status,omitempty"`
	RecordedAt    string        `json:"recorded_at,omitempty"`
	Blocker       string        `json:"blocker,omitempty"`
	Evidence      string        `json:"evidence,omitempty"`
	UnblocksWhen  string        `json:"unblocks_when,omitempty"`
	Owner         string        `json:"owner,omitempty"`
	Pivot         string        `json:"pivot,omitempty"`
	NextCheck     string        `json:"next_check,omitempty"`
	Degraded      bool          `json:"degraded,omitempty"`
	MissingFields []string      `json:"missing_fields,omitempty"`
}

func NormalizeBlockerRecord(record BlockerRecord) BlockerRecord {
	record.Title = strings.TrimSpace(record.Title)
	record.RecordedAt = strings.TrimSpace(record.RecordedAt)
	record.Blocker = strings.TrimSpace(record.Blocker)
	record.Evidence = strings.TrimSpace(record.Evidence)
	record.UnblocksWhen = strings.TrimSpace(record.UnblocksWhen)
	record.Owner = strings.TrimSpace(record.Owner)
	record.Pivot = strings.TrimSpace(record.Pivot)
	record.NextCheck = strings.TrimSpace(record.NextCheck)
	record.Type = normalizeBlockerType(record.Type)

	missing := missingBlockerFields(record)
	if record.Type == BlockerTypeUnknown || len(missing) > 0 {
		record.Status = BlockerStatusUnclassified
		record.Degraded = true
		record.MissingFields = missing
		return record
	}
	record.Status = BlockerStatusActive
	record.Degraded = false
	record.MissingFields = nil
	return record
}

func FormatBlockerRecord(record BlockerRecord) string {
	record = NormalizeBlockerRecord(record)
	var b strings.Builder
	title := record.Title
	if title == "" {
		title = "untitled"
	}
	fmt.Fprintf(&b, "[BLOCKED] %s \u2014 %s\n", title, record.RecordedAt)
	fmt.Fprintf(&b, "  status: %s\n", record.Status)
	fmt.Fprintf(&b, "  type: %s\n", record.Type)
	fmt.Fprintf(&b, "  blocker: %s\n", record.Blocker)
	fmt.Fprintf(&b, "  evidence: %s\n", record.Evidence)
	fmt.Fprintf(&b, "  unblocks when: %s\n", record.UnblocksWhen)
	fmt.Fprintf(&b, "  owner: %s\n", record.Owner)
	fmt.Fprintf(&b, "  workaround/pivot: %s\n", record.Pivot)
	fmt.Fprintf(&b, "  next check: %s\n", record.NextCheck)
	if len(record.MissingFields) > 0 {
		fmt.Fprintf(&b, "  missing fields: %s\n", strings.Join(record.MissingFields, ","))
	}
	return b.String()
}

func SelectBlockerPivot(records []BlockerRecord) string {
	for _, record := range records {
		if pivot := strings.TrimSpace(record.Pivot); pivot != "" {
			return pivot
		}
	}
	return ""
}

func normalizeBlockerType(kind BlockerType) BlockerType {
	switch BlockerType(strings.ToLower(strings.TrimSpace(string(kind)))) {
	case BlockerTypeAccess:
		return BlockerTypeAccess
	case BlockerTypeInfra:
		return BlockerTypeInfra
	case BlockerTypeDependency:
		return BlockerTypeDependency
	case BlockerTypeDecision:
		return BlockerTypeDecision
	case BlockerTypeBug:
		return BlockerTypeBug
	case BlockerTypeUnknown:
		return BlockerTypeUnknown
	default:
		return BlockerTypeUnknown
	}
}

func missingBlockerFields(record BlockerRecord) []string {
	required := []struct {
		name  string
		value string
	}{
		{name: "blocker", value: record.Blocker},
		{name: "evidence", value: record.Evidence},
		{name: "unblocks_when", value: record.UnblocksWhen},
		{name: "owner", value: record.Owner},
		{name: "pivot", value: record.Pivot},
		{name: "next_check", value: record.NextCheck},
	}
	var missing []string
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			missing = append(missing, field.name)
		}
	}
	sort.Strings(missing)
	return missing
}
