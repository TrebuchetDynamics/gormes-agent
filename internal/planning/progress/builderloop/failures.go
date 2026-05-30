package builderloop

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/builderloop/records"

type FailureRecord = records.FailureRecord

func ReadFailureRecord(root, slug string) (FailureRecord, error) {
	return records.ReadFailureRecord(root, slug)
}

func WriteFailureRecord(root, slug string, rc int, reason, stderrPath string, finalErrors []string) error {
	return records.WriteFailureRecord(root, slug, rc, reason, stderrPath, finalErrors)
}
