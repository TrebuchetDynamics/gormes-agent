package builderloop

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/builderloop/records"

type LedgerEvent = records.LedgerEvent

func AppendLedgerEvent(path string, event LedgerEvent) error {
	return records.AppendLedgerEvent(path, event)
}
