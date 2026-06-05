//go:build !slim

package voice

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/termux"
)

// AudioProbeStatus classifies the optional audio backend probe without
// importing platform audio libraries at package initialization.
type AudioProbeStatus string

const (
	AudioProbeAvailable            AudioProbeStatus = "available"
	AudioProbeNoDevices            AudioProbeStatus = "no_devices"
	AudioProbeQueryFailed          AudioProbeStatus = "query_failed"
	AudioProbeLibraryMissing       AudioProbeStatus = "library_missing"
	AudioProbeSystemLibraryMissing AudioProbeStatus = "system_library_missing"
)

// AudioProbeResult is the redacted result of checking the optional audio stack.
// Production probes can fill it from sounddevice/PortAudio equivalents; tests
// use static fixtures.
type AudioProbeResult struct {
	Status      AudioProbeStatus
	DeviceCount int
}

// Environment describes the audio environment.
type Environment struct {
	Available bool     `json:"available"`
	Warnings  []string `json:"warnings,omitempty"`
	Notices   []string `json:"notices,omitempty"`
}

// EnvironmentDetector mirrors Hermes' detect_audio_environment with injectable
// probes so tests never need real SSH, WSL, Termux, PortAudio, or audio
// hardware.
type EnvironmentDetector struct {
	Env map[string]string

	ProcVersion string
	IsContainer bool
	IsTermux    bool

	TermuxMicrophoneCommand    string
	TermuxAPIAppInstalled      bool
	TermuxAPIAppInstalledProbe func(context.Context) bool

	AudioProbe func(context.Context) AudioProbeResult
}

func (d EnvironmentDetector) Detect(ctx context.Context) Environment {
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
	case AudioProbeAvailable:
	case AudioProbeNoDevices:
		if termuxCapture {
			notices = append(notices, "No PortAudio devices detected, but Termux:API microphone capture is available")
		} else {
			warnings = append(warnings, "No audio input/output devices detected")
		}
	case AudioProbeQueryFailed:
		if pulseServer {
			notices = append(notices, "Audio device query failed but PULSE_SERVER is set -- continuing")
		} else if termuxCapture {
			notices = append(notices, "PortAudio device query failed, but Termux:API microphone capture is available")
		} else {
			warnings = append(warnings, "Audio subsystem error (PortAudio cannot query devices)")
		}
	case AudioProbeSystemLibraryMissing:
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
	case AudioProbeLibraryMissing, "":
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

	return Environment{
		Available: len(warnings) == 0,
		Warnings:  warnings,
		Notices:   notices,
	}
}

func (d EnvironmentDetector) env(key string) string {
	if d.Env != nil {
		return d.Env[key]
	}
	return os.Getenv(key)
}

func (d EnvironmentDetector) procVersion() string {
	if d.ProcVersion != "" {
		return d.ProcVersion
	}
	raw, err := os.ReadFile("/proc/version")
	if err != nil {
		return ""
	}
	return string(raw)
}

func (d EnvironmentDetector) container() bool {
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

func (d EnvironmentDetector) termux() bool {
	return d.IsTermux || termux.IsEnvironment(d.env)
}

func (d EnvironmentDetector) termuxMicrophoneCommand() string {
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

func (d EnvironmentDetector) termuxAPIAppInstalled(ctx context.Context) bool {
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

func (d EnvironmentDetector) audioProbe(ctx context.Context) AudioProbeResult {
	if d.AudioProbe == nil {
		return AudioProbeResult{Status: AudioProbeLibraryMissing}
	}
	return d.AudioProbe(ctx)
}

func (d EnvironmentDetector) voiceCaptureInstallHint() string {
	if d.termux() {
		return "pkg install python-numpy portaudio && python -m pip install sounddevice"
	}
	return "pip install sounddevice numpy"
}

func termuxAPIAppMissingWarning() string {
	return "Termux:API Android app is not installed. Install/update the Termux:API app to use termux-microphone-record."
}
