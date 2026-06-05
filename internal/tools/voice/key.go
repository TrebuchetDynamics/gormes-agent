package voice

import (
	"fmt"
	"runtime"
	"strings"
)

const DefaultVoiceRecordKey = "ctrl+b"

type VoiceRecordKeyEvidence string

const (
	VoiceRecordKeyEvidenceOK       VoiceRecordKeyEvidence = "voice_record_key_ok"
	VoiceRecordKeyEvidenceInvalid  VoiceRecordKeyEvidence = "voice_record_key_invalid"
	VoiceRecordKeyEvidenceReserved VoiceRecordKeyEvidence = "voice_record_key_reserved"
)

type VoiceRecordKeyOptions struct {
	GOOS string
}

type VoiceRecordKeyBinding struct {
	Raw           string
	Modifier      string
	Key           string
	Named         bool
	PromptToolkit string
	Display       string
	Evidence      VoiceRecordKeyEvidence
	Defaulted     bool
}

type VoiceRecordKeyEvent struct {
	Key    string
	Ctrl   bool
	Alt    bool
	Meta   bool
	Super  bool
	Shift  bool
	Escape bool
}

var voiceRecordKeyModAliases = map[string]string{
	"ctrl":    "ctrl",
	"control": "ctrl",
	"alt":     "alt",
	"option":  "alt",
	"opt":     "alt",
}

var voiceRecordNamedKeyAliases = map[string]string{
	"space":     "space",
	"enter":     "enter",
	"return":    "enter",
	"tab":       "tab",
	"escape":    "escape",
	"esc":       "escape",
	"backspace": "backspace",
	"bs":        "backspace",
	"delete":    "delete",
	"del":       "delete",
}

func ResolveVoiceRecordKey(raw any, opts VoiceRecordKeyOptions) VoiceRecordKeyBinding {
	if opts.GOOS == "" {
		opts.GOOS = runtime.GOOS
	}
	text, ok := raw.(string)
	if !ok {
		return defaultVoiceRecordKey(VoiceRecordKeyEvidenceInvalid)
	}
	lowered := strings.ToLower(strings.TrimSpace(text))
	if lowered == "" {
		return defaultVoiceRecordKey(VoiceRecordKeyEvidenceInvalid)
	}
	parts := strings.Split(lowered, "+")
	clean := parts[:0]
	for _, part := range parts {
		if p := strings.TrimSpace(part); p != "" {
			clean = append(clean, p)
		}
	}
	if len(clean) != 2 {
		return defaultVoiceRecordKey(VoiceRecordKeyEvidenceInvalid)
	}
	mod, ok := voiceRecordKeyModAliases[clean[0]]
	if !ok {
		return defaultVoiceRecordKey(VoiceRecordKeyEvidenceInvalid)
	}
	key := clean[1]
	named := false
	if len(key) == 1 {
		if isReservedVoiceRecordKey(mod, key, opts.GOOS) {
			return defaultVoiceRecordKey(VoiceRecordKeyEvidenceReserved)
		}
	} else {
		canonical, ok := voiceRecordNamedKeyAliases[key]
		if !ok {
			return defaultVoiceRecordKey(VoiceRecordKeyEvidenceInvalid)
		}
		key = canonical
		named = true
	}
	return VoiceRecordKeyBinding{
		Raw:           mod + "+" + key,
		Modifier:      mod,
		Key:           key,
		Named:         named,
		PromptToolkit: promptToolkitVoiceRecordKey(mod, key),
		Display:       displayVoiceRecordKey(mod, key),
		Evidence:      VoiceRecordKeyEvidenceOK,
	}
}

func MatchesVoiceRecordKey(raw any, ev VoiceRecordKeyEvent, opts VoiceRecordKeyOptions) bool {
	binding := ResolveVoiceRecordKey(raw, opts)
	key := strings.ToLower(strings.TrimSpace(ev.Key))
	if key == "" || key != binding.Key || ev.Shift {
		return false
	}
	switch binding.Modifier {
	case "ctrl":
		if ev.Ctrl {
			return !ev.Alt && !ev.Meta && !ev.Super
		}
		return binding.Raw == DefaultVoiceRecordKey && goos(opts) == "darwin" && ev.Super && !ev.Alt && !ev.Meta
	case "alt":
		return (ev.Alt || (ev.Meta && !ev.Escape)) && !ev.Ctrl && !ev.Super
	default:
		return false
	}
}

func FormatVoiceRecordKeyForStatus(raw any) string {
	return ResolveVoiceRecordKey(raw, VoiceRecordKeyOptions{}).Display
}

func defaultVoiceRecordKey(evidence VoiceRecordKeyEvidence) VoiceRecordKeyBinding {
	return VoiceRecordKeyBinding{
		Raw:           DefaultVoiceRecordKey,
		Modifier:      "ctrl",
		Key:           "b",
		PromptToolkit: "c-b",
		Display:       "Ctrl+B",
		Evidence:      evidence,
		Defaulted:     true,
	}
}

func isReservedVoiceRecordKey(mod, key, goosName string) bool {
	switch mod {
	case "ctrl":
		return key == "c" || key == "d" || key == "l"
	case "alt":
		return goosName == "darwin" && (key == "c" || key == "d" || key == "l")
	default:
		return false
	}
}

func promptToolkitVoiceRecordKey(mod, key string) string {
	switch mod {
	case "ctrl":
		return "c-" + key
	case "alt":
		return "a-" + key
	default:
		return key
	}
}

func displayVoiceRecordKey(mod, key string) string {
	prefix := map[string]string{"ctrl": "Ctrl", "alt": "Alt"}[mod]
	if prefix == "" {
		prefix = strings.Title(mod)
	}
	if len(key) == 1 {
		return fmt.Sprintf("%s+%s", prefix, strings.ToUpper(key))
	}
	return fmt.Sprintf("%s+%s", prefix, strings.Title(key))
}

func goos(opts VoiceRecordKeyOptions) string {
	if opts.GOOS != "" {
		return opts.GOOS
	}
	return runtime.GOOS
}
