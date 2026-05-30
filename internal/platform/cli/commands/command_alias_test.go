package commands

import (
	"reflect"
	"strings"
	"testing"
)

func TestCommandAlias_SlashRegistryAliasAndPrefix(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKind  CommandAliasKind
		wantRaw   string
		wantArgs  string
		wantCanon string
		wantLine  string
		wantMatch []string
	}{
		{
			name:      "typed alias canonicalizes before dispatch",
			input:     "/provider openrouter --global",
			wantKind:  CommandAliasAlias,
			wantRaw:   "provider",
			wantArgs:  "openrouter --global",
			wantCanon: "model",
			wantLine:  "/model openrouter --global",
		},
		{
			name:      "unique prefix canonicalizes and preserves args",
			input:     "/prof --json",
			wantKind:  CommandAliasPrefix,
			wantRaw:   "prof",
			wantArgs:  "--json",
			wantCanon: "profile",
			wantLine:  "/profile --json",
		},
		{
			name:      "exact command wins over longer prefix matches",
			input:     "/status verbose",
			wantKind:  CommandAliasExact,
			wantRaw:   "status",
			wantArgs:  "verbose",
			wantCanon: "status",
			wantLine:  "/status verbose",
		},
		{
			name:      "ambiguous prefix gives bounded candidates",
			input:     "/stat now",
			wantKind:  CommandAliasAmbiguous,
			wantRaw:   "stat",
			wantArgs:  "now",
			wantMatch: []string{"/status", "/statusbar"},
		},
		{
			name:      "platform prefix is ambiguous with plural status command",
			input:     "/platf --json",
			wantKind:  CommandAliasAmbiguous,
			wantRaw:   "platf",
			wantArgs:  "--json",
			wantMatch: []string{"/platform", "/platforms"},
		},
		{
			name:     "unknown slash stays unknown",
			input:    "/no-such-command-xyzzy",
			wantKind: CommandAliasUnknown,
			wantRaw:  "no-such-command-xyzzy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveCommandAlias(tt.input)
			if got.Kind != tt.wantKind {
				t.Fatalf("ResolveCommandAlias(%q).Kind = %q, want %q (got=%+v)", tt.input, got.Kind, tt.wantKind, got)
			}
			if got.RawCommand != tt.wantRaw || got.RawArgs != tt.wantArgs || got.Canonical != tt.wantCanon || got.Rewrite != tt.wantLine {
				t.Fatalf("ResolveCommandAlias(%q) = %+v, want raw=%q args=%q canonical=%q rewrite=%q", tt.input, got, tt.wantRaw, tt.wantArgs, tt.wantCanon, tt.wantLine)
			}
			if !reflect.DeepEqual(got.Matches, tt.wantMatch) {
				t.Fatalf("ResolveCommandAlias(%q).Matches = %v, want %v", tt.input, got.Matches, tt.wantMatch)
			}
		})
	}
}

func TestCommandAlias_QuickCommandAliasPreservesArgsAndDetectsCycles(t *testing.T) {
	quick := map[string]QuickCommandAlias{
		"g":      {Type: "alias", Target: "/goal"},
		"ship":   {Type: "alias", Target: "/g now"},
		"loop-a": {Type: "alias", Target: "/loop-b"},
		"loop-b": {Type: "alias", Target: "/loop-a"},
		"bad":    {Type: "alias", Target: "/no-such-command-xyzzy"},
		"exec":   {Type: "exec", Target: "echo hi"},
	}

	resolved := ResolveQuickCommandAlias("/ship with tests", quick)
	if resolved.Kind != QuickCommandAliasResolved {
		t.Fatalf("ResolveQuickCommandAlias(/ship with tests).Kind = %q, want %q (got=%+v)", resolved.Kind, QuickCommandAliasResolved, resolved)
	}
	if resolved.Rewrite != "/goal now with tests" || resolved.Canonical != "goal" || resolved.RawArgs != "with tests" {
		t.Fatalf("resolved quick alias = %+v, want rewrite /goal now with tests and preserved raw args", resolved)
	}
	if !reflect.DeepEqual(resolved.Chain, []string{"ship", "g"}) {
		t.Fatalf("resolved.Chain = %v, want [ship g]", resolved.Chain)
	}

	cycled := ResolveQuickCommandAlias("/loop-a later", quick)
	if cycled.Kind != QuickCommandAliasCycle {
		t.Fatalf("cycle Kind = %q, want %q (got=%+v)", cycled.Kind, QuickCommandAliasCycle, cycled)
	}
	if !strings.Contains(strings.ToLower(cycled.Evidence), "cycle") {
		t.Fatalf("cycle Evidence = %q, want cycle guidance", cycled.Evidence)
	}

	unsupported := ResolveQuickCommandAlias("/bad later", quick)
	if unsupported.Kind != QuickCommandAliasUnsupportedTarget {
		t.Fatalf("unsupported Kind = %q, want %q (got=%+v)", unsupported.Kind, QuickCommandAliasUnsupportedTarget, unsupported)
	}

	exec := ResolveQuickCommandAlias("/exec", quick)
	if exec.Kind != QuickCommandAliasUnsupportedType {
		t.Fatalf("exec Kind = %q, want %q (got=%+v)", exec.Kind, QuickCommandAliasUnsupportedType, exec)
	}
}
