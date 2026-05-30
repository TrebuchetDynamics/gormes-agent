package tools

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/systemevents"
)

const (
	SystemEventCodeEnqueued    = systemevents.SystemEventCodeEnqueued
	SystemEventCodeHeartbeat   = systemevents.SystemEventCodeHeartbeat
	SystemEventCodePresence    = systemevents.SystemEventCodePresence
	SystemEventCodeUnavailable = systemevents.SystemEventCodeUnavailable

	SystemEventModeNextHeartbeat = systemevents.SystemEventModeNextHeartbeat
	SystemEventModeNow           = systemevents.SystemEventModeNow
)

type SystemEventMode = systemevents.SystemEventMode
type SystemEventsOptions = systemevents.SystemEventsOptions
type SystemEventsManager = systemevents.SystemEventsManager
type SystemEventRequest = systemevents.SystemEventRequest
type SystemEvent = systemevents.SystemEvent
type SystemHeartbeatState = systemevents.SystemHeartbeatState
type SystemPresenceEntry = systemevents.SystemPresenceEntry
type SystemPresenceUpdate = systemevents.SystemPresenceUpdate
type SystemEventsSnapshot = systemevents.SystemEventsSnapshot
type SystemDegradedStatus = systemevents.SystemDegradedStatus
type SystemEventResult = systemevents.SystemEventResult
type SystemHeartbeatResult = systemevents.SystemHeartbeatResult
type SystemPresenceResult = systemevents.SystemPresenceResult

func NewSystemEventsManager(opts SystemEventsOptions) SystemEventsManager {
	return systemevents.NewSystemEventsManager(opts)
}

func FormatSystemStatus(snapshot SystemEventsSnapshot, auditPath string) string {
	return systemevents.FormatSystemStatus(snapshot, auditPath)
}
