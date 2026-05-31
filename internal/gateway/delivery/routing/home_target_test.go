package routing

import "testing"

func TestResolveHomeTarget_ChannelNeutralPlatformHome(t *testing.T) {
	homes := HomeTargets{
		"discord": {Platform: "discord", ChatID: "D-home"},
		"slack":   {Platform: "slack", ChatID: "C-home"},
	}

	target, err := ParseTarget(" DISCORD ", nil)
	if err != nil {
		t.Fatalf("ParseTarget: %v", err)
	}
	got, ok := ResolveHomeTarget(target, homes)
	if !ok {
		t.Fatal("ResolveHomeTarget did not resolve configured discord home")
	}
	want := Target{Platform: "discord", ChatID: "D-home"}
	if got != want {
		t.Fatalf("resolved target = %+v, want %+v", got, want)
	}
}

func TestResolveHomeTarget_PreservesExplicitOriginAndLocal(t *testing.T) {
	homes := HomeTargets{"telegram": {Platform: "telegram", ChatID: "42"}}
	origin := &OriginSource{Platform: "telegram", ChatID: "origin-chat", ThreadID: "thread-1"}

	for _, raw := range []string{"telegram:99", "telegram:99:7", "origin", "local"} {
		t.Run(raw, func(t *testing.T) {
			target, err := ParseTarget(raw, origin)
			if err != nil {
				t.Fatalf("ParseTarget(%q): %v", raw, err)
			}
			got, ok := ResolveHomeTarget(target, homes)
			if ok {
				t.Fatalf("ResolveHomeTarget(%q) ok = true, want false", raw)
			}
			if got != target {
				t.Fatalf("ResolveHomeTarget(%q) = %+v, want unchanged %+v", raw, got, target)
			}
		})
	}
}

func TestResolveHomeTargetWithFallback_UsesDiscoveryOnlyWhenEnabled(t *testing.T) {
	target, err := ParseTarget("bluebubbles", nil)
	if err != nil {
		t.Fatalf("ParseTarget: %v", err)
	}
	fallback := HomeDiscoveryFallback{
		DiscoveryEnabled: true,
		Source:           OriginSource{Platform: "bluebubbles", ChatID: "chat-guid", ThreadID: "thread-guid"},
	}
	got, err := ResolveHomeTargetWithFallback(target, nil, fallback)
	if err != nil {
		t.Fatalf("ResolveHomeTargetWithFallback error = %v", err)
	}
	want := Target{Platform: "bluebubbles", ChatID: "chat-guid", ThreadID: "thread-guid"}
	if got != want {
		t.Fatalf("resolved fallback = %+v, want %+v", got, want)
	}

	_, err = ResolveHomeTargetWithFallback(target, nil, HomeDiscoveryFallback{Source: fallback.Source})
	if _, ok := err.(MissingHomeError); !ok {
		t.Fatalf("missing disabled discovery error = %T %v, want MissingHomeError", err, err)
	}
}

func TestResolveHomeTargetWithFallback_ConfiguredHomeWinsOverDiscovery(t *testing.T) {
	target, err := ParseTarget("slack", nil)
	if err != nil {
		t.Fatalf("ParseTarget: %v", err)
	}
	got, err := ResolveHomeTargetWithFallback(
		target,
		HomeTargets{"slack": {Platform: "slack", ChatID: "C-configured"}},
		HomeDiscoveryFallback{DiscoveryEnabled: true, Source: OriginSource{Platform: "slack", ChatID: "C-discovered"}},
	)
	if err != nil {
		t.Fatalf("ResolveHomeTargetWithFallback error = %v", err)
	}
	if got.ChatID != "C-configured" {
		t.Fatalf("ChatID = %q, want configured home channel", got.ChatID)
	}
}
