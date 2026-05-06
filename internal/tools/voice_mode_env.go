//go:build !slim

package tools

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

// VoiceModeAudioProbeStatus classifies the optional audio backend probe
// without importing platform audio libraries at package initialization.
type VoiceModeAudioProbeStatus string

const (
	VoiceModeAudioProbeAvailable            VoiceModeAudioProbeStatus = "available"
	VoiceModeAudioProbeNoDevices            VoiceModeAudioProbeStatus = "no_devices"
	VoiceModeAudioProbeQueryFailed          VoiceModeAudioProbeStatus = "query_failed"
	VoiceModeAudioProbeLibraryMissing       VoiceModeAudioProbeStatus = "library_missing"
	VoiceModeAudioProbeSystemLibraryMissing VoiceModeAudioProbeStatus = "system_library_missing"
)

// VoiceModeAudioProbeResult is the redacted result of checking the optional
// audio stack. Production probes can fill it from sounddevice/PortAudio
// equivalents; tests use static fixtures.
type VoiceModeAudioProbeResult struct {
	Status      VoiceModeAudioProbeStatus
	DeviceCount int
}

// VoiceModeEnvironmentDetector mirrors Hermes' detect_audio_environment with
// injectable probes so tests never need real SSH, WSL, Termux, PortAudio, or
// audio hardware.
type VoiceModeEnvironmentDetector struct {
	Env map[string]string

	ProcVersion string
	IsContainer bool
	IsTermux    bool

	TermuxMicrophoneCommand    string
	TermuxAPIAppInstalled      bool
	TermuxAPIAppInstalledProbe func(context.Context) bool

	AudioProbe func(context.Context) VoiceModeAudioProbeResult
}

func (d VoiceModeEnvironmentDetector) Detect(ctx context.Context) VoiceModeEnvironment {
	var warnings []string
	var notices []string

	termuxMic := d.termuxMicrophoneCommand()
	termuxAppInstalled := d.termuxAPIAppInstalled(ctx)
	termuxCapture := termuxMic != "" && termuxAppInstalled

	if d.env("SSH_CLIENT") != "" || d.env("SSH_TTY") != "" || d.env("SSH_CONNECTION") != "" {
		warnings = append(warnings, "Running over SSH -- no audio devices available")
	}
	if d.container() {
		warnings = append(warnings, "Running inside Docker container -- no audio devices")
	}

	wsl := strings.Contains(strings.ToLower(d.procVersion()), "microsoft")
	pulseServer := d.env("PULSE_SERVER") != ""
	if wsl {
		if pulseServer {
			notices = append(notices, "Running in WSL with PulseAudio bridge")
		} else {
			warnings = append(warnings,
				"Running in WSL -- audio requires PulseAudio bridge.\n"+
					"  1. Set PULSE_SERVER=unix:/mnt/wslg/PulseServer\n"+
					"  2. Create ~/.asoundrc pointing ALSA at PulseAudio\n"+
					"  3. Verify with: arecord -d 3 /tmp/test.wav && aplay /tmp/test.wav")
		}
	}

	probe := d.audioProbe(ctx)
	switch probe.Status {
	case VoiceModeAudioProbeAvailable:
	case VoiceModeAudioProbeNoDevices:
		if termuxCapture {
			notices = append(notices, "No PortAudio devices detected, but Termux:API microphone capture is available")
		} else {
			warnings = append(warnings, "No audio input/output devices detected")
		}
	case VoiceModeAudioProbeQueryFailed:
		if pulseServer {
			notices = append(notices, "Audio device query failed but PULSE_SERVER is set -- continuing")
		} else if termuxCapture {
			notices = append(notices, "PortAudio device query failed, but Termux:API microphone capture is available")
		} else {
			warnings = append(warnings, "Audio subsystem error (PortAudio cannot query devices)")
		}
	case VoiceModeAudioProbeSystemLibraryMissing:
		if termuxCapture {
			notices = append(notices, "Termux:API microphone recording available (PortAudio not required)")
		} else if termuxMic != "" && !termuxAppInstalled {
			warnings = append(warnings, termuxAPIAppMissingWarning())
		} else if d.termux() {
			warnings = append(warnings,
				"PortAudio system library not found -- install it first:\n"+
					"  Termux: pkg install portaudio\n"+
					"Then retry /voice on.")
		} else {
			warnings = append(warnings,
				"PortAudio system library not found -- install it first:\n"+
					"  Linux:  sudo apt-get install libportaudio2\n"+
					"  macOS:  brew install portaudio\n"+
					"Then retry /voice on.")
		}
	case VoiceModeAudioProbeLibraryMissing, "":
		if termuxCapture {
			notices = append(notices, "Termux:API microphone recording available (sounddevice not required)")
		} else if termuxMic != "" && !termuxAppInstalled {
			warnings = append(warnings, termuxAPIAppMissingWarning())
		} else {
			warnings = append(warnings, "Audio libraries not installed ("+d.voiceCaptureInstallHint()+")")
		}
	default:
		warnings = append(warnings, "Audio subsystem error (PortAudio cannot query devices)")
	}

	return VoiceModeEnvironment{
		Available: len(warnings) == 0,
		Warnings:  warnings,
		Notices:   notices,
	}
}

func (d VoiceModeEnvironmentDetector) env(key string) string {
	if d.Env != nil {
		return d.Env[key]
	}
	return os.Getenv(key)
}

func (d VoiceModeEnvironmentDetector) procVersion() string {
	if d.ProcVersion != "" {
		return d.ProcVersion
	}
	raw, err := os.ReadFile("/proc/version")
	if err != nil {
		return ""
	}
	return string(raw)
}

func (d VoiceModeEnvironmentDetector) container() bool {
	if d.IsContainer {
		return true
	}
	if d.env("container") != "" {
		return true
	}
	if d.Env != nil || d.ProcVersion != "" {
		return false
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}
	return false
}

func (d VoiceModeEnvironmentDetector) termux() bool {
	if d.IsTermux {
		return true
	}
	prefix := d.env("PREFIX")
	home := d.env("HOME")
	return strings.Contains(prefix, "com.termux") || strings.Contains(home, "com.termux")
}

func (d VoiceModeEnvironmentDetector) termuxMicrophoneCommand() string {
	if !d.termux() {
		return ""
	}
	if d.TermuxMicrophoneCommand != "" {
		return d.TermuxMicrophoneCommand
	}
	path, err := exec.LookPath("termux-microphone-record")
	if err != nil {
		return ""
	}
	return path
}

func (d VoiceModeEnvironmentDetector) termuxAPIAppInstalled(ctx context.Context) bool {
	if !d.termux() {
		return false
	}
	if d.TermuxAPIAppInstalled {
		return true
	}
	if d.TermuxAPIAppInstalledProbe != nil {
		return d.TermuxAPIAppInstalledProbe(ctx)
	}
	return false
}

func (d VoiceModeEnvironmentDetector) audioProbe(ctx context.Context) VoiceModeAudioProbeResult {
	if d.AudioProbe == nil {
		return VoiceModeAudioProbeResult{Status: VoiceModeAudioProbeLibraryMissing}
	}
	return d.AudioProbe(ctx)
}

func (d VoiceModeEnvironmentDetector) voiceCaptureInstallHint() string {
	if d.termux() {
		return "pkg install python-numpy portaudio && python -m pip install sounddevice"
	}
	return "pip install sounddevice numpy"
}

func termuxAPIAppMissingWarning() string {
	return "Termux:API Android app is not installed. Install/update the Termux:API app to use termux-microphone-record."
}
