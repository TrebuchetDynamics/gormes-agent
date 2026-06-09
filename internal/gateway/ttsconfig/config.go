package ttsconfig

import (
	"fmt"
	"sort"
	"strings"
)

// Engine is the configured TTS engine identifier.
type Engine string

const (
	EngineOpenAI     Engine = "openai"
	EngineElevenLabs Engine = "elevenlabs"
	EngineEdge       Engine = "edge"
	EngineLocal      Engine = "local"
	EngineDisabled   Engine = "disabled"
)

// Speed is the normalized speech speed preset.
type Speed string

const (
	SpeedSlow     Speed = "slow"
	SpeedNormal   Speed = "normal"
	SpeedFast     Speed = "fast"
	SpeedVeryFast Speed = "very-fast"
)

// Config holds per-session TTS settings.
type Config struct {
	Enabled  bool   `json:"enabled"`
	Engine   Engine `json:"engine"`
	Voice    string `json:"voice"`
	Speed    Speed  `json:"speed"`
	Language string `json:"language"`
}

func (c Config) String() string {
	enabled := "disabled"
	if c.Enabled {
		enabled = "enabled"
	}
	return fmt.Sprintf("TTS: %s\nengine: %s\nvoice: %s\nspeed: %s\nlanguage: %s",
		enabled, configLineValue(string(c.Engine)), configLineValue(c.Voice), configLineValue(string(c.Speed)), configLineValue(c.Language))
}

func configLineValue(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

var DefaultConfig = Config{
	Enabled:  true,
	Engine:   EngineEdge,
	Voice:    "en-US-AriaNeural",
	Speed:    SpeedNormal,
	Language: "auto",
}

var KnownEngines = []Engine{EngineOpenAI, EngineElevenLabs, EngineEdge, EngineLocal, EngineDisabled}

var ValidSpeeds = map[string]Speed{
	"slow": SpeedSlow, "normal": SpeedNormal, "fast": SpeedFast,
	"very-fast": SpeedVeryFast, "veryfast": SpeedVeryFast,
	"very_fast": SpeedVeryFast, "very fast": SpeedVeryFast,
}

var SupportedSpeeds = fmt.Sprintf("%s, %s, %s, %s", SpeedSlow, SpeedNormal, SpeedFast, SpeedVeryFast)

var Voices = map[Engine][]string{
	EngineOpenAI:     {"alloy", "echo", "fable", "onyx", "nova", "shimmer"},
	EngineElevenLabs: {"rachel", "domi", "bella", "elli", "josh", "arnold"},
	EngineEdge:       {"en-US-AriaNeural", "en-US-JennyNeural", "en-US-GuyNeural"},
	EngineLocal:      {},
	EngineDisabled:   {},
}

func ResolveSpeed(raw string) (Speed, bool) {
	s, ok := ValidSpeeds[strings.ToLower(strings.TrimSpace(raw))]
	return s, ok
}

func EngineNames() []string {
	n := make([]string, len(KnownEngines))
	for i, e := range KnownEngines {
		n[i] = string(e)
	}
	return n
}

func VoicesForEngineSorted(e Engine) []string {
	voices := append([]string(nil), Voices[e]...)
	sort.Strings(voices)
	return voices
}

func DefaultVoiceForEngine(e Engine) string {
	if v := Voices[e]; len(v) > 0 {
		return v[0]
	}
	return "default"
}
