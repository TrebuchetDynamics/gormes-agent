package delivery

import (
	"strings"
	"testing"

	entrymodel "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/entry"
)

func TestTargetRejectsBlankPlatform(t *testing.T) {
	got := Target(" \t ", entrymodel.Entry{ID: "42", ChatID: "42"})
	if got.Platform != "" || got.ChatID != "" || got.ThreadID != "" || got.IsExplicit {
		t.Fatalf("Target(blank platform) = %+v, want invalid target", got)
	}
}

func TestTargetRejectsControlCharactersInEntryIdentifiers(t *testing.T) {
	for _, entry := range []entrymodel.Entry{
		{ID: "42\nadmin"},
		{ID: "42", ChatID: "42\nadmin"},
		{ID: "42", ChatID: "42", ThreadID: "thread\nadmin"},
	} {
		got := Target("telegram", entry)
		if got.ChatID != "" || got.ThreadID != "" || got.IsExplicit {
			t.Fatalf("Target(%+v) = %+v, want invalid control-character target rejected", entry, got)
		}
	}
}

func TestTargetRejectsLocalEntryIdentifiers(t *testing.T) {
	for _, entry := range []entrymodel.Entry{
		{ID: "42"},
		{ChatID: "42"},
		{ChatID: "42", ThreadID: "99"},
	} {
		got := Target("local", entry)
		if got.ChatID != "" || got.ThreadID != "" || got.IsExplicit {
			t.Fatalf("Target(local, %+v) = %+v, want local directory target rejected", entry, got)
		}
	}
}

func TestTargetUsesExplicitChatAndThreadIDs(t *testing.T) {
	target := Target("telegram", entrymodel.Entry{ID: "ignored", ChatID: " -100 ", ThreadID: " 12 "})
	if target.Platform != "telegram" || target.ChatID != "-100" || target.ThreadID != "12" || !target.IsExplicit {
		t.Fatalf("Target() = %+v, want explicit trimmed chat/thread target", target)
	}
}

func TestTargetPreservesMatrixRoomIDWithHomeserverPort(t *testing.T) {
	target := Target("matrix", entrymodel.Entry{ID: " !room:matrix.org:8448 "})
	if target.Platform != "matrix" || target.ChatID != "!room:matrix.org:8448" || target.ThreadID != "" || !target.IsExplicit {
		t.Fatalf("Target() = %+v, want full matrix room ID with homeserver port", target)
	}
}

func TestTargetParsesPlatformColonThreadID(t *testing.T) {
	target := Target("matrix", entrymodel.Entry{ID: " !room:matrix.org:$thread:matrix.org "})
	if target.Platform != "matrix" || target.ChatID != "!room:matrix.org" || target.ThreadID != "$thread:matrix.org" || !target.IsExplicit {
		t.Fatalf("Target() = %+v, want matrix room/thread parsed from colon-bearing entry ID", target)
	}
}

func TestTargetPreservesPlatformColonChatID(t *testing.T) {
	for _, tc := range []struct {
		platform string
		entryID  string
	}{
		{platform: "matrix", entryID: " !room:matrix.org "},
		{platform: "simplex", entryID: " group:ops "},
	} {
		t.Run(tc.platform, func(t *testing.T) {
			target := Target(tc.platform, entrymodel.Entry{ID: tc.entryID})
			if target.Platform != tc.platform || target.ChatID != strings.TrimSpace(tc.entryID) || target.ThreadID != "" || !target.IsExplicit {
				t.Fatalf("Target() = %+v, want full colon-bearing chat ID without thread", target)
			}
		})
	}
}

func TestTargetFallsBackToCompositeEntryID(t *testing.T) {
	target := Target("discord", entrymodel.Entry{ID: " C01:99 "})
	if target.Platform != "discord" || target.ChatID != "C01" || target.ThreadID != "99" || !target.IsExplicit {
		t.Fatalf("Target() = %+v, want chat/thread parsed from entry ID", target)
	}
}
