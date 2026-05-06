package tools

import "testing"

func TestVoiceRecordKeyConfigParsesDefaultAndOverride(t *testing.T) {
	defaulted := ResolveVoiceRecordKey(nil, VoiceRecordKeyOptions{})
	if defaulted.Raw != DefaultVoiceRecordKey || defaulted.PromptToolkit != "c-b" || defaulted.Display != "Ctrl+B" {
		t.Fatalf("default record key = %+v, want ctrl+b / c-b / Ctrl+B", defaulted)
	}
	if defaulted.Evidence != VoiceRecordKeyEvidenceInvalid || !defaulted.Defaulted {
		t.Fatalf("default evidence = %+v, want invalid default fallback", defaulted)
	}

	ctrl := ResolveVoiceRecordKey("control+o", VoiceRecordKeyOptions{})
	if ctrl.Modifier != "ctrl" || ctrl.Key != "o" || ctrl.PromptToolkit != "c-o" || ctrl.Display != "Ctrl+O" || ctrl.Defaulted {
		t.Fatalf("control+o = %+v, want ctrl/o/c-o/Ctrl+O without fallback", ctrl)
	}

	alt := ResolveVoiceRecordKey("option+space", VoiceRecordKeyOptions{})
	if alt.Modifier != "alt" || alt.Key != "space" || !alt.Named || alt.PromptToolkit != "a-space" || alt.Display != "Alt+Space" {
		t.Fatalf("option+space = %+v, want alt/space/a-space/Alt+Space", alt)
	}
}

func TestVoiceRecordKeyMalformedAndReservedFallback(t *testing.T) {
	for _, raw := range []any{"", "  ", "o", "ctrl+alt+r", "ctrl+spcae", "meta+b", "shift+b", true, 1, []string{"ctrl+b"}} {
		t.Run("invalid", func(t *testing.T) {
			got := ResolveVoiceRecordKey(raw, VoiceRecordKeyOptions{})
			if got.Raw != DefaultVoiceRecordKey || got.Display != "Ctrl+B" || got.Evidence != VoiceRecordKeyEvidenceInvalid || !got.Defaulted {
				t.Fatalf("ResolveVoiceRecordKey(%#v) = %+v, want invalid fallback to Ctrl+B", raw, got)
			}
		})
	}

	for _, raw := range []any{"ctrl+c", "ctrl+d", "ctrl+l"} {
		t.Run("reserved", func(t *testing.T) {
			got := ResolveVoiceRecordKey(raw, VoiceRecordKeyOptions{})
			if got.Raw != DefaultVoiceRecordKey || got.Display != "Ctrl+B" || got.Evidence != VoiceRecordKeyEvidenceReserved || !got.Defaulted {
				t.Fatalf("ResolveVoiceRecordKey(%#v) = %+v, want reserved fallback to Ctrl+B", raw, got)
			}
		})
	}
}

func TestVoiceRecordKeyMatchesConfiguredEvent(t *testing.T) {
	if !MatchesVoiceRecordKey("ctrl+o", VoiceRecordKeyEvent{Key: "o", Ctrl: true}, VoiceRecordKeyOptions{}) {
		t.Fatal("ctrl+o event did not match configured ctrl+o")
	}
	if MatchesVoiceRecordKey("ctrl+o", VoiceRecordKeyEvent{Key: "b", Ctrl: true}, VoiceRecordKeyOptions{}) {
		t.Fatal("ctrl+b event matched configured ctrl+o")
	}
	if !MatchesVoiceRecordKey("alt+escape", VoiceRecordKeyEvent{Key: "escape", Alt: true, Escape: true}, VoiceRecordKeyOptions{}) {
		t.Fatal("alt+escape event did not match configured alt+escape")
	}
	if MatchesVoiceRecordKey("alt+escape", VoiceRecordKeyEvent{Key: "escape", Meta: true, Escape: true}, VoiceRecordKeyOptions{}) {
		t.Fatal("bare escape/meta event matched configured alt+escape")
	}
}
