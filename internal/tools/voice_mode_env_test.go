//go:build !slim

package tools

import (
	"context"
	"strings"
	"testing"
)

func TestVoiceModeDetectSSHAndContainerWarnings(t *testing.T) {
	env := VoiceModeEnvironmentDetector{
		Env:         map[string]string{"SSH_TTY": "/dev/pts/3"},
		IsContainer: true,
		AudioProbe: func(context.Context) VoiceModeAudioProbeResult {
			return VoiceModeAudioProbeResult{Status: VoiceModeAudioProbeAvailable, DeviceCount: 2}
		},
	}.Detect(context.Background())

	if env.Available {
		t.Fatalf("Available = true, want false for SSH/container")
	}
	assertVoiceModeContains(t, env.Warnings, "Running over SSH -- no audio devices available")
	assertVoiceModeContains(t, env.Warnings, "Running inside Docker container -- no audio devices")
}

func TestVoiceModeDetectWSLPulseAudioNotice(t *testing.T) {
	t.Run("pulse bridge keeps WSL available when device query fails", func(t *testing.T) {
		env := VoiceModeEnvironmentDetector{
			Env:         map[string]string{"PULSE_SERVER": "unix:/mnt/wslg/PulseServer"},
			ProcVersion: "Linux version 6.6.87.2-microsoft-standard-WSL2",
			AudioProbe: func(context.Context) VoiceModeAudioProbeResult {
				return VoiceModeAudioProbeResult{Status: VoiceModeAudioProbeQueryFailed}
			},
		}.Detect(context.Background())

		if !env.Available {
			t.Fatalf("Available = false, want true with WSL PulseAudio bridge; warnings=%v", env.Warnings)
		}
		assertVoiceModeContains(t, env.Notices, "Running in WSL with PulseAudio bridge")
		assertVoiceModeContains(t, env.Notices, "Audio device query failed but PULSE_SERVER is set -- continuing")
	})

	t.Run("missing pulse bridge blocks WSL audio", func(t *testing.T) {
		env := VoiceModeEnvironmentDetector{
			ProcVersion: "Linux version 6.6.87.2-microsoft-standard-WSL2",
			AudioProbe: func(context.Context) VoiceModeAudioProbeResult {
				return VoiceModeAudioProbeResult{Status: VoiceModeAudioProbeAvailable, DeviceCount: 2}
			},
		}.Detect(context.Background())

		if env.Available {
			t.Fatalf("Available = true, want false without WSL PulseAudio bridge")
		}
		assertVoiceModeContains(t, env.Warnings, "Running in WSL -- audio requires PulseAudio bridge.")
	})
}

func TestVoiceModeDetectTermuxFallback(t *testing.T) {
	t.Run("termux api microphone satisfies capture without PortAudio", func(t *testing.T) {
		env := VoiceModeEnvironmentDetector{
			Env:                     map[string]string{},
			IsTermux:                true,
			TermuxMicrophoneCommand: "/data/data/com.termux/files/usr/bin/termux-microphone-record",
			TermuxAPIAppInstalled:   true,
			AudioProbe: func(context.Context) VoiceModeAudioProbeResult {
				return VoiceModeAudioProbeResult{Status: VoiceModeAudioProbeLibraryMissing}
			},
		}.Detect(context.Background())

		if !env.Available {
			t.Fatalf("Available = false, want true with Termux:API microphone fallback; warnings=%v", env.Warnings)
		}
		assertVoiceModeContains(t, env.Notices, "Termux:API microphone recording available (sounddevice not required)")
	})

	t.Run("missing termux api app is explicit setup guidance", func(t *testing.T) {
		env := VoiceModeEnvironmentDetector{
			Env:                     map[string]string{},
			IsTermux:                true,
			TermuxMicrophoneCommand: "/data/data/com.termux/files/usr/bin/termux-microphone-record",
			AudioProbe: func(context.Context) VoiceModeAudioProbeResult {
				return VoiceModeAudioProbeResult{Status: VoiceModeAudioProbeLibraryMissing}
			},
		}.Detect(context.Background())

		if env.Available {
			t.Fatalf("Available = true, want false when Termux:API app is missing")
		}
		assertVoiceModeContains(t, env.Warnings, "Termux:API Android app is not installed")
	})
}

func TestVoiceModeDetectorNoOptionalImportsAtInit(t *testing.T) {
	probeCalls := 0
	detector := VoiceModeEnvironmentDetector{
		Env: map[string]string{},
		AudioProbe: func(context.Context) VoiceModeAudioProbeResult {
			probeCalls++
			return VoiceModeAudioProbeResult{Status: VoiceModeAudioProbeNoDevices}
		},
	}

	if probeCalls != 0 {
		t.Fatalf("AudioProbe called during detector construction = %d, want 0", probeCalls)
	}

	env := detector.Detect(context.Background())
	if probeCalls != 1 {
		t.Fatalf("AudioProbe calls after Detect = %d, want 1", probeCalls)
	}
	if env.Available {
		t.Fatalf("Available = true, want false with no detected devices")
	}
	assertVoiceModeContains(t, env.Warnings, "No audio input/output devices detected")
}

func assertVoiceModeContains(t *testing.T, got []string, want string) {
	t.Helper()
	for _, s := range got {
		if strings.Contains(s, want) {
			return
		}
	}
	t.Fatalf("missing %q in %v", want, got)
}
