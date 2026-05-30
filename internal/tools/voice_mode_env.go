//go:build !slim

package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/voice"

// VoiceModeAudioProbeStatus classifies the optional audio backend probe
// without importing platform audio libraries at package initialization.
type VoiceModeAudioProbeStatus = voice.AudioProbeStatus

const (
	VoiceModeAudioProbeAvailable            = voice.AudioProbeAvailable
	VoiceModeAudioProbeNoDevices            = voice.AudioProbeNoDevices
	VoiceModeAudioProbeQueryFailed          = voice.AudioProbeQueryFailed
	VoiceModeAudioProbeLibraryMissing       = voice.AudioProbeLibraryMissing
	VoiceModeAudioProbeSystemLibraryMissing = voice.AudioProbeSystemLibraryMissing
)

// VoiceModeAudioProbeResult is the redacted result of checking the optional
// audio stack. Production probes can fill it from sounddevice/PortAudio
// equivalents; tests use static fixtures.
type VoiceModeAudioProbeResult = voice.AudioProbeResult

// VoiceModeEnvironment describes the audio environment.
type VoiceModeEnvironment = voice.Environment

// VoiceModeEnvironmentDetector mirrors Hermes' detect_audio_environment with
// injectable probes so tests never need real SSH, WSL, Termux, PortAudio, or
// audio hardware.
type VoiceModeEnvironmentDetector = voice.EnvironmentDetector
