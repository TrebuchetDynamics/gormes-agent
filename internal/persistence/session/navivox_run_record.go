package session

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session/navivox"
)

const (
	NavivoxRunStatusInProgress = navivox.RunStatusInProgress
	NavivoxRunStatusCompleted  = navivox.RunStatusCompleted
	NavivoxRunStatusFailed     = navivox.RunStatusFailed
	NavivoxRunStatusStopped    = navivox.RunStatusStopped

	NavivoxEvidenceAvailable   = navivox.EvidenceAvailable
	NavivoxEvidenceUnavailable = navivox.EvidenceUnavailable
	NavivoxEvidenceUnknown     = navivox.EvidenceUnknown
)

type NavivoxRunRecord = navivox.RunRecord
type NavivoxTranscriptEntry = navivox.TranscriptEntry
type NavivoxVoiceEvidence = navivox.VoiceEvidence
type NavivoxSTTEvidence = navivox.STTEvidence
type NavivoxAudioMeta = navivox.AudioMeta
type NavivoxTTSEvidence = navivox.TTSEvidence
type NavivoxToolEvent = navivox.ToolEvent
type NavivoxProviderUsage = navivox.ProviderUsage
type NavivoxProviderCost = navivox.ProviderCost
type NavivoxArtifactRef = navivox.ArtifactRef

func NewNavivoxRunRecord(runID, sessionID, userText string, metadata map[string]any, now time.Time) NavivoxRunRecord {
	return navivox.NewRunRecord(runID, sessionID, userText, metadata, now)
}
