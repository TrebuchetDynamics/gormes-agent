package slashcompletion

import (
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
)

func TestPromptTemplateCompletionMatchingIsCaseInsensitive(t *testing.T) {
	catalog := prompttemplates.Catalog{Templates: []prompttemplates.Template{{Name: "Review", Description: "Review", ArgumentHint: "<scope>"}}}
	got := PromptTemplateCompletions("/rev", catalog)
	if len(got) != 1 || got[0].Name != "Review" {
		t.Fatalf("PromptTemplateCompletions(/rev) = %#v, want Review regardless of catalog name case", got)
	}
}

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

func TestPromptTemplateCompletionsDeduplicateCaseInsensitiveNames(t *testing.T) {
	catalog := prompttemplates.Catalog{Templates: []prompttemplates.Template{
		{Name: "Review", Description: "first"},
		{Name: "review", Description: "duplicate"},
	}}

	got := PromptTemplateCompletions("/rev", catalog)
	if len(got) != 1 || got[0].Name != "Review" || got[0].Description != "first" {
		t.Fatalf("PromptTemplateCompletions duplicate case variants = %#v, want first canonical Review completion", got)
	}
}

func TestSkillCompletionsDeduplicateCaseInsensitiveNames(t *testing.T) {
	commands := []skills.SkillSlashCommand{
		{Command: "/Review", Description: "first"},
		{Command: "review", Description: "duplicate"},
	}

	got := SkillCompletions("/rev", commands)
	if len(got) != 1 || got[0].Name != "review" || got[0].Description != "first" {
		t.Fatalf("SkillCompletions duplicate case variants = %#v, want first canonical review completion", got)
	}
}

func TestWithDynamicDeduplicatesCaseInsensitiveNames(t *testing.T) {
	commands := []skills.SkillSlashCommand{{Command: "/Review", Description: "skill review"}}
	catalog := prompttemplates.Catalog{Templates: []prompttemplates.Template{{Name: "Review", Description: "template review"}}}

	got := WithDynamic("/rev", commands, catalog)
	if len(got) != 1 || got[0].Name != "review" {
		t.Fatalf("WithDynamic duplicate case variants = %#v, want one canonical review completion", got)
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
	if got := AutoSuggest("/Hel"); got != "p" {
		t.Fatalf("AutoSuggest(/Hel) = %q, want case-insensitive p", got)
	}
	if got := AutoSuggest("/reasoning sh"); got != "ow" {
		t.Fatalf("AutoSuggest(/reasoning sh) = %q, want ow", got)
	}
}

func TestSubcommandCompletionIsCaseInsensitiveAndCanonicalizesBase(t *testing.T) {
	got := SubcommandCompletions("/Reasoning sh")
	if len(got) != 1 || got[0].Name != "show" {
		t.Fatalf("SubcommandCompletions(/Reasoning sh) = %#v, want show", got)
	}
	if suffix := AutoSuggest("/Reasoning sh"); suffix != "ow" {
		t.Fatalf("AutoSuggest(/Reasoning sh) = %q, want ow", suffix)
	}
	accepted, ok := AcceptedText("/Reasoning sh", Completion{Name: "show"}, false)
	if !ok || accepted != "/reasoning show" {
		t.Fatalf("AcceptedText(/Reasoning sh, show) = (%q, %v), want /reasoning show true", accepted, ok)
	}
}

func TestCompletionCandidateFlowDropsEmptyGroups(t *testing.T) {
	if got := flattenCompletionGroups(nil); got != nil {
		t.Fatalf("flattenCompletionGroups(nil) = %#v, want nil", got)
	}

	groups := [][]Completion{{}, {{Name: "beta"}}, nil, {{Name: "alpha"}}}
	got := uniqueSortedCompletions(flattenCompletionGroups(groups))
	want := []Completion{{Name: "alpha"}, {Name: "beta"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueSortedCompletions(flattened groups) = %#v, want %#v", got, want)
	}
}

func TestCompletionRequestParsingClassifiesCommandAndSubcommandFlow(t *testing.T) {
	command, ok := parseCompletionRequest("/he")
	if !ok || !command.commandOnly() || command.commandPrefix != "he" {
		t.Fatalf("parseCompletionRequest(/he) = (%#v, %v), want command prefix he", command, ok)
	}

	sub, ok := parseCompletionRequest("/Reasoning   Sh")
	if !ok || !sub.subcommandOnly() || sub.base != "/reasoning" || sub.subPrefix != "sh" {
		t.Fatalf("parseCompletionRequest(/Reasoning   Sh) = (%#v, %v), want canonical subcommand request", sub, ok)
	}

	if got, ok := parseCompletionRequest("/reasoning show now"); ok || got != (completionRequest{}) {
		t.Fatalf("parseCompletionRequest with subcommand args = (%#v, %v), want rejected", got, ok)
	}
}

func TestSubcommandCompletionCandidatesAreSortedAndDeduplicated(t *testing.T) {
	got := matchingSubcommandCompletions([]string{"Show", "hide", "show", "", "status"}, "s")
	want := []Completion{
		{Name: "Show", Display: "Show", Available: true},
		{Name: "status", Display: "status", Available: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("matchingSubcommandCompletions unsorted duplicate candidates = %#v, want %#v", got, want)
	}
}

func TestSubcommandCompletionWhitespaceIsConsistent(t *testing.T) {
	for _, input := range []string{"/reasoning sh", "/reasoning\tsh", "/reasoning   sh"} {
		got := SubcommandCompletions(input)
		if len(got) != 1 || got[0].Name != "show" {
			t.Fatalf("SubcommandCompletions(%q) = %#v, want show", input, got)
		}
		if suffix := AutoSuggest(input); suffix != "ow" {
			t.Fatalf("AutoSuggest(%q) = %q, want ow", input, suffix)
		}
		accepted, ok := AcceptedText(input, Completion{Name: "show"}, false)
		if !ok || accepted != "/reasoning show" {
			t.Fatalf("AcceptedText(%q, show) = (%q, %v), want /reasoning show true", input, accepted, ok)
		}
	}

	if got := SubcommandCompletions("/reasoning show now"); got != nil {
		t.Fatalf("SubcommandCompletions with completed subcommand args = %#v, want nil", got)
	}
}
