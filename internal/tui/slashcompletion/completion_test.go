package slashcompletion

import (
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
)

func TestCompletionsAreDeterministicAndMerged(t *testing.T) {
	first := CommandCompletions("/he")
	second := CommandCompletions("/he")
	if !reflect.DeepEqual(first, second) || len(first) == 0 || first[0].Name != "help" {
		t.Fatalf("CommandCompletions(/he) = %#v / %#v, want stable help", first, second)
	}
	catalog := prompttemplates.Catalog{Templates: []prompttemplates.Template{{Name: "review", Description: "Review", ArgumentHint: "<scope>"}}}
	got := WithPromptTemplates("/rev", catalog)
	if len(got) != 1 || got[0].Name != "review" || got[0].ArgumentHint != "<scope>" {
		t.Fatalf("WithPromptTemplates = %#v, want review template", got)
	}
}

func TestAcceptedText(t *testing.T) {
	if got, ok := AcceptedText("/he", Completion{Name: "help"}, false); got != "/help" || !ok {
		t.Fatalf("AcceptedText enter prefix = (%q, %v), want /help true", got, ok)
	}
	if got, ok := AcceptedText("/help", Completion{Name: "help"}, false); got != "/help" || ok {
		t.Fatalf("AcceptedText enter exact = (%q, %v), want /help false", got, ok)
	}
	if got, ok := AcceptedText("/reasoning ", Completion{Name: "show"}, false); got != "/reasoning show" || !ok {
		t.Fatalf("AcceptedText subcommand = (%q, %v), want /reasoning show true", got, ok)
	}
	if got, ok := AcceptedText("/rev", Completion{Name: "review", ArgumentHint: "<scope>"}, true); got != "/review " || !ok {
		t.Fatalf("AcceptedText tab arg command = (%q, %v), want /review space true", got, ok)
	}
}

func TestSubcommandCompletionsAndAutoSuggest(t *testing.T) {
	got := SubcommandCompletions("/reasoning sh")
	if len(got) != 1 || got[0].Name != "show" {
		t.Fatalf("SubcommandCompletions = %#v, want show", got)
	}
	if got := AutoSuggest("/hel"); got != "p" {
		t.Fatalf("AutoSuggest(/hel) = %q, want p", got)
	}
	if got := AutoSuggest("/reasoning sh"); got != "ow" {
		t.Fatalf("AutoSuggest(/reasoning sh) = %q, want ow", got)
	}
}
