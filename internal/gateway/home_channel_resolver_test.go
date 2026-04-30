package gateway

import "testing"

func TestResolveHomeChannelTarget_ChannelNeutralPlatformHome(t *testing.T) {
	homes := HomeChannelTargets{
		"discord": {Platform: "discord", ChatID: "D-home"},
		"slack":   {Platform: "slack", ChatID: "C-home"},
	}

	target, err := ParseDeliveryTarget(" DISCORD ", nil)
	if err != nil {
		t.Fatalf("ParseDeliveryTarget: %v", err)
	}
	got, ok := ResolveHomeChannelTarget(target, homes)
	if !ok {
		t.Fatal("ResolveHomeChannelTarget did not resolve configured discord home")
	}
	want := DeliveryTarget{Platform: "discord", ChatID: "D-home"}
	if got != want {
		t.Fatalf("resolved target = %+v, want %+v", got, want)
	}
}

func TestResolveHomeChannelTarget_PreservesExplicitOriginAndLocal(t *testing.T) {
	homes := HomeChannelTargets{"telegram": {Platform: "telegram", ChatID: "42"}}
	origin := &SessionSource{Platform: "telegram", ChatID: "origin-chat", ThreadID: "thread-1"}

	for _, raw := range []string{"telegram:99", "telegram:99:7", "origin", "local"} {
		t.Run(raw, func(t *testing.T) {
			target, err := ParseDeliveryTarget(raw, origin)
			if err != nil {
				t.Fatalf("ParseDeliveryTarget(%q): %v", raw, err)
			}
			got, ok := ResolveHomeChannelTarget(target, homes)
			if ok {
				t.Fatalf("ResolveHomeChannelTarget(%q) ok = true, want false", raw)
			}
			if got != target {
				t.Fatalf("ResolveHomeChannelTarget(%q) = %+v, want unchanged %+v", raw, got, target)
			}
		})
	}
}

func TestResolveHomeChannelTargetWithFallback_UsesDiscoveryOnlyWhenEnabled(t *testing.T) {
	target, err := ParseDeliveryTarget("bluebubbles", nil)
	if err != nil {
		t.Fatalf("ParseDeliveryTarget: %v", err)
	}
	fallback := HomeChannelDiscoveryFallback{
		DiscoveryEnabled: true,
		Source:           SessionSource{Platform: "bluebubbles", ChatID: "chat-guid", ThreadID: "thread-guid"},
	}
	got, err := ResolveHomeChannelTargetWithFallback(target, nil, fallback)
	if err != nil {
		t.Fatalf("ResolveHomeChannelTargetWithFallback error = %v", err)
	}
	want := DeliveryTarget{Platform: "bluebubbles", ChatID: "chat-guid", ThreadID: "thread-guid"}
	if got != want {
		t.Fatalf("resolved fallback = %+v, want %+v", got, want)
	}

	_, err = ResolveHomeChannelTargetWithFallback(target, nil, HomeChannelDiscoveryFallback{Source: fallback.Source})
	if _, ok := err.(MissingHomeChannelError); !ok {
		t.Fatalf("missing disabled discovery error = %T %v, want MissingHomeChannelError", err, err)
	}
}

func TestResolveHomeChannelTargetWithFallback_ConfiguredHomeWinsOverDiscovery(t *testing.T) {
	target, err := ParseDeliveryTarget("slack", nil)
	if err != nil {
		t.Fatalf("ParseDeliveryTarget: %v", err)
	}
	got, err := ResolveHomeChannelTargetWithFallback(
		target,
		HomeChannelTargets{"slack": {Platform: "slack", ChatID: "C-configured"}},
		HomeChannelDiscoveryFallback{DiscoveryEnabled: true, Source: SessionSource{Platform: "slack", ChatID: "C-discovered"}},
	)
	if err != nil {
		t.Fatalf("ResolveHomeChannelTargetWithFallback error = %v", err)
	}
	if got.ChatID != "C-configured" {
		t.Fatalf("ChatID = %q, want configured home channel", got.ChatID)
	}
}
