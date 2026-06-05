package setup

import (
	"strconv"
	"strings"
)

var toolProgressModes = []string{"off", "new", "all", "verbose"}

// TTSProviderOptions returns supported TTS provider choices in setup display order.
func TTSProviderOptions() []Choice {
	return []Choice{
		{Value: "edge", Label: "Edge TTS (free, cloud-based, no setup needed)"},
		{Value: "elevenlabs", Label: "ElevenLabs (premium quality, needs API key)"},
		{Value: "openai", Label: "OpenAI TTS (good quality, needs API key)"},
		{Value: "xai", Label: "xAI TTS (Grok voices, needs API key)"},
		{Value: "minimax", Label: "MiniMax TTS (high quality with voice cloning, needs API key)"},
		{Value: "mistral", Label: "Mistral Voxtral TTS (multilingual, native Opus, needs API key)"},
		{Value: "gemini", Label: "Google Gemini TTS (30 prebuilt voices, prompt-controllable, needs API key)"},
		{Value: "neutts", Label: "NeuTTS (local on-device, free, model download)"},
		{Value: "keep", Label: "Keep current"},
	}
}

// TerminalBackendOptions returns supported terminal backend choices in setup display order.
func TerminalBackendOptions() []Choice {
	return []Choice{
		{Value: "local", Label: "Local - run directly on this machine (default)"},
		{Value: "docker", Label: "Docker - isolated container with configurable resources"},
		{Value: "modal", Label: "Modal - serverless cloud sandbox"},
		{Value: "ssh", Label: "SSH - run on a remote machine"},
		{Value: "daytona", Label: "Daytona - persistent cloud development environment"},
		{Value: "singularity", Label: "Singularity/Apptainer - HPC-friendly container"},
		{Value: "keep", Label: "Keep current"},
	}
}

// TerminalBackendLabel returns the operator-facing label for a terminal backend value.
func TerminalBackendLabel(value string) string {
	switch normalizeChoice(value) {
	case "local":
		return "Local"
	case "docker":
		return "Docker"
	case "modal":
		return "Modal"
	case "ssh":
		return "SSH"
	case "daytona":
		return "Daytona"
	case "singularity", "apptainer":
		return "Singularity/Apptainer"
	default:
		return value
	}
}

// TTSProviderLabel returns the operator-facing label for a TTS provider value.
func TTSProviderLabel(value string) string {
	switch normalizeChoice(value) {
	case "edge":
		return "Edge TTS"
	case "elevenlabs":
		return "ElevenLabs"
	case "openai":
		return "OpenAI TTS"
	case "xai":
		return "xAI TTS"
	case "minimax":
		return "MiniMax TTS"
	case "mistral":
		return "Mistral Voxtral TTS"
	case "gemini":
		return "Google Gemini TTS"
	case "neutts":
		return "NeuTTS"
	case "keep":
		return "Keep current"
	default:
		return value
	}
}

// ParsePositiveInt accepts positive base-10 integers for setup prompts.
func ParsePositiveInt(value string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

// ParseCompressionThreshold accepts setup compression thresholds in the supported range.
func ParseCompressionThreshold(value string) (float64, bool) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || parsed < 0.5 || parsed > 0.95 {
		return 0, false
	}
	return parsed, true
}

// IsKnownToolProgressMode reports whether value is a supported tool-progress display mode.
func IsKnownToolProgressMode(value string) bool {
	value = normalizeChoice(value)
	for _, mode := range toolProgressModes {
		if value == mode {
			return true
		}
	}
	return false
}

// ToolProgressModeIndex returns the zero-based mode index, or -1 for unknown values.
func ToolProgressModeIndex(current string) int {
	current = normalizeChoice(current)
	for i, mode := range toolProgressModes {
		if mode == current {
			return i
		}
	}
	return -1
}

// IsKnownSessionResetPolicy reports whether value is a supported session-reset policy.
func IsKnownSessionResetPolicy(value string) bool {
	switch normalizeChoice(value) {
	case "inactivity", "daily", "manual", "off", "none":
		return true
	default:
		return false
	}
}
