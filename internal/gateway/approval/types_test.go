package approval

import (
	"context"
	"testing"
)

func TestParseChoice(t *testing.T) {
	tests := []struct {
		value string
		want  Choice
		ok    bool
	}{
		{value: "once", want: ChoiceOnce, ok: true},
		{value: " SESSION ", want: ChoiceSession, ok: true},
		{value: "always", want: ChoiceAlways, ok: true},
		{value: "deny", want: ChoiceDeny, ok: true},
		{value: "approved", ok: false},
		{value: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, ok := ParseChoice(tt.value)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("choice = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolverFunc(t *testing.T) {
	var got Resolution
	resolver := ResolverFunc(func(_ context.Context, res Resolution) error {
		got = res
		return nil
	})

	err := resolver.ResolveGatewayApproval(context.Background(), Resolution{
		SessionKey: "slack:C123:sess-1",
		Choice:     ChoiceOnce,
		Platform:   "slack",
		ChatID:     "C123",
		MessageID:  "1711111111.000100",
		ActorID:    "U42",
	})
	if err != nil {
		t.Fatalf("ResolveGatewayApproval: %v", err)
	}
	if got.SessionKey != "slack:C123:sess-1" || got.Choice != ChoiceOnce || got.Platform != "slack" {
		t.Fatalf("resolution = %+v, want Slack once resolution", got)
	}
}
