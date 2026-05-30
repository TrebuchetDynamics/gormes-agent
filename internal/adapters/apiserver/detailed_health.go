package apiserver

import "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/health"

// DetailedHealthSnapshotInput is a value-only health read model. Callers fill
// it from already-available status reads; this package does not bind routes,
// start schedulers, contact providers, or mutate response/run/cron stores.
type DetailedHealthSnapshotInput = health.DetailedHealthSnapshotInput

type DetailedHealthProviderInput = health.DetailedHealthProviderInput

type DetailedHealthResponseStoreInput = health.DetailedHealthResponseStoreInput

type DetailedHealthRunEventsInput = health.DetailedHealthRunEventsInput

type DetailedHealthGatewayInput = health.DetailedHealthGatewayInput

type DetailedHealthCronInput = health.DetailedHealthCronInput

type DetailedHealthSnapshotModel struct {
	Build         BuildInfo                          `json:"build"`
	Provider      DetailedHealthProviderSection      `json:"provider"`
	ResponseStore DetailedHealthResponseStoreSection `json:"response_store"`
	RunEvents     DetailedHealthRunEventsSection     `json:"run_events"`
	Gateway       DetailedHealthGatewaySection       `json:"gateway"`
	Cron          DetailedHealthCronSection          `json:"cron"`
}

type DetailedHealthEvidence = health.DetailedHealthEvidence

type DetailedHealthProviderSection = health.DetailedHealthProviderSection

type DetailedHealthResponseStoreSection = health.DetailedHealthResponseStoreSection

type DetailedHealthRunEventsSection = health.DetailedHealthRunEventsSection

type DetailedHealthGatewaySection = health.DetailedHealthGatewaySection

type DetailedHealthCronSection = health.DetailedHealthCronSection

func DetailedHealthSnapshot(input DetailedHealthSnapshotInput) DetailedHealthSnapshotModel {
	snapshot := health.DetailedHealthSnapshot(input)
	return DetailedHealthSnapshotModel{
		Provider:      snapshot.Provider,
		ResponseStore: snapshot.ResponseStore,
		RunEvents:     snapshot.RunEvents,
		Gateway:       snapshot.Gateway,
		Cron:          snapshot.Cron,
	}
}
