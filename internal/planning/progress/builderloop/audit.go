package builderloop

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/builderloop/records"

func DigestLedger(path string) (string, error) {
	return records.DigestLedger(path)
}

func DigestLedgerCounts(path string) (map[string]int, error) {
	return records.DigestLedgerCounts(path)
}

type AuditReportOptions = records.AuditReportOptions

func WriteAuditReport(opts AuditReportOptions) (string, error) {
	return records.WriteAuditReport(opts)
}
