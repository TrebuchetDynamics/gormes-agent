package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/ttsconfig"
)

// TTSEngine is the configured TTS engine identifier.
type TTSEngine = ttsconfig.Engine

const (
	TTSEngineOpenAI     = ttsconfig.EngineOpenAI
	TTSEngineElevenLabs = ttsconfig.EngineElevenLabs
	TTSEngineEdge       = ttsconfig.EngineEdge
	TTSEngineLocal      = ttsconfig.EngineLocal
	TTSEngineDisabled   = ttsconfig.EngineDisabled
)

// TTSSpeed is the normalized speech speed preset.
type TTSSpeed = ttsconfig.Speed

const (
	TTSSpeedSlow     = ttsconfig.SpeedSlow
	TTSSpeedNormal   = ttsconfig.SpeedNormal
	TTSSpeedFast     = ttsconfig.SpeedFast
	TTSSpeedVeryFast = ttsconfig.SpeedVeryFast
)

// TTSConfig holds per-session TTS settings.
type TTSConfig = ttsconfig.Config

// TTSConfigStore owns per-session TTS settings.
type TTSConfigStore = ttsconfig.Store

// NewTTSConfigStore returns a session-keyed TTS config store.
func NewTTSConfigStore() *TTSConfigStore {
	return ttsconfig.NewStore()
}

var defaultTTSConfig = ttsconfig.DefaultConfig

var knownTTSEngines = ttsconfig.KnownEngines

var ttsSupportedSpeeds = ttsconfig.SupportedSpeeds

var ttsVoices = ttsconfig.Voices

func resolveTTSSpeed(raw string) (TTSSpeed, bool) {
	return ttsconfig.ResolveSpeed(raw)
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
			voices := ttsconfig.VoicesForEngineSorted(cfg.Engine)
			selected := cfg.Voice
			if len(voices) == 0 {
				_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("TTS voice: %s\nEngine %s does not support voice listing.", selected, cfg.Engine))
			} else {
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
		if len(args) < 3 {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("TTS language: %s\nUsage: /tts language [auto|en|es|pt|fr|de|it|ja|ko|zh]", cfg.Language))
			return
		}
		language := normalizeTTSCommandLanguage(strings.Join(args[2:], " "))
		if language == "" {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Unknown TTS language. Supported: auto, en, es, pt, fr, de, it, ja, ko, zh")
			return
		}
		cfg.Language = language
		m.setTTSConfig(ev.ChatKey(), cfg)
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("TTS language set to: %s", language))
	default:
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Usage: /tts [on|off|speed|voice|engine|language]")
	}
}

func (m *Manager) getTTSConfig(sessionKey string) TTSConfig {
	if m.ttsConfigStore == nil {
		m.ttsConfigStore = NewTTSConfigStore()
	}
	return m.ttsConfigStore.Get(sessionKey)
}

func (m *Manager) setTTSConfig(sessionKey string, cfg TTSConfig) {
	if m.ttsConfigStore == nil {
		m.ttsConfigStore = NewTTSConfigStore()
	}
	m.ttsConfigStore.Set(sessionKey, cfg)
}

func engineNames() []string {
	return ttsconfig.EngineNames()
}

func defaultVoiceForEngine(e TTSEngine) string {
	return ttsconfig.DefaultVoiceForEngine(e)
}

func normalizeTTSCommandLanguage(raw string) string {
	lang := strings.ToLower(strings.TrimSpace(raw))
	lang = strings.ReplaceAll(lang, "_", "-")
	switch lang {
	case "auto", "en", "en-us", "es", "es-es", "pt", "pt-br", "fr", "fr-fr", "de", "de-de", "it", "it-it", "ja", "ja-jp", "ko", "ko-kr", "zh", "zh-cn":
		return lang
	default:
		return ""
	}
}
