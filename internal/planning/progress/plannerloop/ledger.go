package plannerloop

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/plannerloop/ledgerlog"
)

type LedgerEvent = ledgerlog.LedgerEvent
type RowChange = ledgerlog.RowChange
type DriftPromotion = ledgerlog.DriftPromotion
type ProgressStats = ledgerlog.ProgressStats
type AutoloopLedgerEvent = ledgerlog.AutoloopLedgerEvent

func AppendLedgerEvent(path string, event LedgerEvent) error {
	return ledgerlog.AppendLedgerEvent(path, event)
}

func LoadLedger(path string) ([]LedgerEvent, error) {
	return ledgerlog.LoadLedger(path)
}

func LoadLedgerWindow(path string, window time.Duration, now time.Time) ([]LedgerEvent, error) {
	return ledgerlog.LoadLedgerWindow(path, window, now)
}

func appendAutoloopLedgerEvent(path string, event AutoloopLedgerEvent) error {
	return ledgerlog.AppendAutoloopLedgerEvent(path, event)
}
