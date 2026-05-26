package gateway

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// TTSEngine is the configured TTS engine identifier.
type TTSEngine string

const (
	TTSEngineOpenAI     TTSEngine = "openai"
	TTSEngineElevenLabs TTSEngine = "elevenlabs"
	TTSEngineEdge       TTSEngine = "edge"
	TTSEngineLocal      TTSEngine = "local"
	TTSEngineDisabled   TTSEngine = "disabled"
)

// TTSSpeed is the normalized speech speed preset.
type TTSSpeed string

const (
	TTSSpeedSlow     TTSSpeed = "slow"
	TTSSpeedNormal   TTSSpeed = "normal"
	TTSSpeedFast     TTSSpeed = "fast"
	TTSSpeedVeryFast TTSSpeed = "very-fast"
)

// TTSConfig holds per-session TTS settings.
type TTSConfig struct {
	Enabled  bool      `json:"enabled"`
	Engine   TTSEngine `json:"engine"`
	Voice    string    `json:"voice"`
	Speed    TTSSpeed  `json:"speed"`
	Language string    `json:"language"`
}

func (c TTSConfig) String() string {
	enabled := "disabled"
	if c.Enabled {
		enabled = "enabled"
	}
	return fmt.Sprintf("TTS: %s\nengine: %s\nvoice: %s\nspeed: %s\nlanguage: %s",
		enabled, c.Engine, c.Voice, c.Speed, c.Language)
}

var defaultTTSConfig = TTSConfig{
	Enabled:  true,
	Engine:   TTSEngineEdge,
	Voice:    "en-US-AriaNeural",
	Speed:    TTSSpeedNormal,
	Language: "auto",
}

var knownTTSEngines = []TTSEngine{TTSEngineOpenAI, TTSEngineElevenLabs, TTSEngineEdge, TTSEngineLocal, TTSEngineDisabled}

var validTTSSpeeds = map[string]TTSSpeed{
	"slow": TTSSpeedSlow, "normal": TTSSpeedNormal, "fast": TTSSpeedFast,
	"very-fast": TTSSpeedVeryFast, "veryfast": TTSSpeedVeryFast,
	"very_fast": TTSSpeedVeryFast, "very fast": TTSSpeedVeryFast,
}

var ttsSupportedSpeeds = fmt.Sprintf("%s, %s, %s, %s", TTSSpeedSlow, TTSSpeedNormal, TTSSpeedFast, TTSSpeedVeryFast)

var ttsVoices = map[TTSEngine][]string{
	TTSEngineOpenAI:     {"alloy", "echo", "fable", "onyx", "nova", "shimmer"},
	TTSEngineElevenLabs: {"rachel", "domi", "bella", "elli", "josh", "arnold"},
	TTSEngineEdge:       {"en-US-AriaNeural", "en-US-JennyNeural", "en-US-GuyNeural"},
	TTSEngineLocal:      {},
	TTSEngineDisabled:   {},
}

func resolveTTSSpeed(raw string) (TTSSpeed, bool) {
	s, ok := validTTSSpeeds[strings.ToLower(strings.TrimSpace(raw))]
	return s, ok
}

func (m *Manager) handleTTSCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	args := strings.Fields(ev.Text)
	cfg := m.getTTSConfig(ev.ChatKey())

	if len(args) < 2 {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, cfg.String())
		return
	}

	switch strings.ToLower(args[1]) {
	case "on":
		cfg.Enabled = true
		if cfg.Engine == TTSEngineDisabled {
			cfg.Engine = TTSEngineEdge
			cfg.Voice = defaultVoiceForEngine(cfg.Engine)
		}
		m.setTTSConfig(ev.ChatKey(), cfg)
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "TTS enabled")
	case "off":
		cfg.Enabled = false
		m.setTTSConfig(ev.ChatKey(), cfg)
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "TTS disabled")
	case "speed":
		if len(args) < 3 {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("TTS speed: %s\nUsage: /tts speed [slow|normal|fast|very-fast]", cfg.Speed))
			return
		}
		raw := strings.Join(args[2:], " ")
		s, ok := resolveTTSSpeed(raw)
		if !ok {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("Unknown TTS speed: %s\nSupported speeds: %s", raw, ttsSupportedSpeeds))
			return
		}
		cfg.Speed = s
		m.setTTSConfig(ev.ChatKey(), cfg)
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("TTS speed set to: %s", s))
	case "voice":
		if len(args) < 3 {
			voices := ttsVoices[cfg.Engine]
			selected := cfg.Voice
			if len(voices) == 0 {
				_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("TTS voice: %s\nEngine %s does not support voice listing.", selected, cfg.Engine))
			} else {
				sort.Strings(voices)
				_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("TTS voice: %s\nAvailable voices for %s: %s", selected, cfg.Engine, strings.Join(voices, ", ")))
			}
			return
		}
		name := strings.TrimSpace(strings.Join(args[2:], " "))
		if len(ttsVoices[cfg.Engine]) > 0 {
			canonical := ""
			for _, v := range ttsVoices[cfg.Engine] {
				if strings.EqualFold(v, name) {
					canonical = v
					break
				}
			}
			if canonical == "" {
				_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("Unknown voice: %s\nAvailable: %s", name, strings.Join(ttsVoices[cfg.Engine], ", ")))
				return
			}
			name = canonical
		}
		cfg.Voice = name
		m.setTTSConfig(ev.ChatKey(), cfg)
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("TTS voice set to: %s", name))
	case "engine":
		if len(args) < 3 {
			names := make([]string, len(knownTTSEngines))
			for i, e := range knownTTSEngines {
				names[i] = string(e)
			}
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("TTS engine: %s\nAvailable: %s", cfg.Engine, strings.Join(names, ", ")))
			return
		}
		raw := strings.ToLower(strings.Join(args[2:], " "))
		var found TTSEngine
		for _, e := range knownTTSEngines {
			if string(e) == raw {
				found = e
				break
			}
		}
		if found == "" {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("Unknown TTS engine: %s\nAvailable: %s", raw, strings.Join(engineNames(), ", ")))
			return
		}
		cfg.Engine = found
		cfg.Voice = defaultVoiceForEngine(found)
		cfg.Enabled = found != TTSEngineDisabled
		m.setTTSConfig(ev.ChatKey(), cfg)
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("TTS engine set to: %s", found))
	case "language":
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Manual TTS language selection is not implemented yet. Current mode: auto.")
	default:
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Usage: /tts [on|off|speed|voice|engine|language]")
	}
}

func (m *Manager) getTTSConfig(sessionKey string) TTSConfig {
	m.reasoningMu.Lock()
	defer m.reasoningMu.Unlock()
	if m.ttsConfigs == nil {
		m.ttsConfigs = make(map[string]TTSConfig)
	}
	cfg, ok := m.ttsConfigs[sessionKey]
	if !ok {
		cfg = defaultTTSConfig
		m.ttsConfigs[sessionKey] = cfg
	}
	return cfg
}

func (m *Manager) setTTSConfig(sessionKey string, cfg TTSConfig) {
	m.reasoningMu.Lock()
	defer m.reasoningMu.Unlock()
	if m.ttsConfigs == nil {
		m.ttsConfigs = make(map[string]TTSConfig)
	}
	m.ttsConfigs[sessionKey] = cfg
}

func engineNames() []string {
	n := make([]string, len(knownTTSEngines))
	for i, e := range knownTTSEngines {
		n[i] = string(e)
	}
	return n
}

func defaultVoiceForEngine(e TTSEngine) string {
	if v := ttsVoices[e]; len(v) > 0 {
		return v[0]
	}
	return "default"
}
