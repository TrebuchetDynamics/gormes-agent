package ttsconfig

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
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
	redacted := redaction.RedactSecrets(value)
	redacted = strings.NewReplacer(
		"api_key=[redacted]", "[redacted]",
		"api-key=[redacted]", "[redacted]",
		"apikey=[redacted]", "[redacted]",
		"authorization=[redacted]", "[redacted]",
		"bearer=[redacted]", "[redacted]",
		"token=[redacted]", "[redacted]",
		"secret=[redacted]", "[redacted]",
		"password=[redacted]", "[redacted]",
	).Replace(redacted)
	redacted = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return ' '
		}
		if unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, redacted)
	fields := strings.Fields(redacted)
	out := make([]string, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		lower := strings.ToLower(field)
		nextRedacted := i+1 < len(fields) && strings.Contains(strings.ToLower(fields[i+1]), "[redacted]")
		if configSecretField(lower) && (strings.Contains(lower, "[redacted]") || nextRedacted) {
			out = append(out, "[redacted]")
			if nextRedacted {
				i++
			}
			continue
		}
		out = append(out, field)
	}
	return strings.Join(out, " ")
}

func configSecretField(value string) bool {
	return strings.Contains(value, "api_key") || strings.Contains(value, "api-key") || strings.Contains(value, "apikey") || strings.Contains(value, "authorization") || strings.Contains(value, "bearer") || strings.Contains(value, "token") || strings.Contains(value, "secret") || strings.Contains(value, "password")
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
