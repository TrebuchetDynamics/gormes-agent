package cli

import (
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands"
)

func TestCommandAliasFacadeDelegatesToCommandsPackage(t *testing.T) {
	got := ResolveCommandAlias("/provider openrouter --global")
	want := commands.ResolveCommandAlias("/provider openrouter --global")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveCommandAlias facade = %+v, want commands package result %+v", got, want)
	}

	quick := map[string]QuickCommandAlias{
		"g":    {Type: "alias", Target: "/goal"},
		"ship": {Type: "alias", Target: "/g now"},
	}
	gotQuick := ResolveQuickCommandAlias("/ship with tests", quick)
	wantQuick := commands.ResolveQuickCommandAlias("/ship with tests", quick)
	if !reflect.DeepEqual(gotQuick, wantQuick) {
		t.Fatalf("ResolveQuickCommandAlias facade = %+v, want commands package result %+v", gotQuick, wantQuick)
	}
}
