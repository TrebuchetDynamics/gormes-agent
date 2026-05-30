package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/voice"

const DefaultVoiceRecordKey = voice.DefaultVoiceRecordKey

type VoiceRecordKeyEvidence = voice.VoiceRecordKeyEvidence

const (
	VoiceRecordKeyEvidenceOK       VoiceRecordKeyEvidence = voice.VoiceRecordKeyEvidenceOK
	VoiceRecordKeyEvidenceInvalid  VoiceRecordKeyEvidence = voice.VoiceRecordKeyEvidenceInvalid
	VoiceRecordKeyEvidenceReserved VoiceRecordKeyEvidence = voice.VoiceRecordKeyEvidenceReserved
)

type VoiceRecordKeyOptions = voice.VoiceRecordKeyOptions
type VoiceRecordKeyBinding = voice.VoiceRecordKeyBinding
type VoiceRecordKeyEvent = voice.VoiceRecordKeyEvent

func ResolveVoiceRecordKey(raw any, opts VoiceRecordKeyOptions) VoiceRecordKeyBinding {
	return voice.ResolveVoiceRecordKey(raw, opts)
}

func MatchesVoiceRecordKey(raw any, ev VoiceRecordKeyEvent, opts VoiceRecordKeyOptions) bool {
	return voice.MatchesVoiceRecordKey(raw, ev, opts)
}

func FormatVoiceRecordKeyForStatus(raw any) string {
	return voice.FormatVoiceRecordKeyForStatus(raw)
}
