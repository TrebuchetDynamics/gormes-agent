package slashcompletion

import (
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
)

func TestPromptTemplateCompletionMatchingIsCaseInsensitive(t *testing.T) {
	catalog := prompttemplates.Catalog{Templates: []prompttemplates.Template{{Name: "Review", Description: "Review", ArgumentHint: "<scope>"}}}
	got := PromptTemplateCompletions("/rev", catalog)
	if len(got) != 1 || got[0].Name != "Review" {
		t.Fatalf("PromptTemplateCompletions(/rev) = %#v, want Review regardless of catalog name case", got)
	}
}

func TestCommandCompletionPlanExposesSortedDedupedCandidates(t *testing.T) {
	registry := []cli.CommandPolicy{
		{Name: "Beta", Description: "canonical beta"},
		{Name: "alpha", Description: "alpha", ActiveTurnPolicy: cli.ActiveTurnPolicyUnavailable},
		{Name: "backup", Description: "backup", Aliases: []string{"Beta"}},
	}

	plan := planCommandCompletionCandidates(newCompletionPrefix("b"), registry)
	if !reflect.DeepEqual(plan.SortedNames, []string{"backup", "beta"}) {
		t.Fatalf("planCommandCompletionCandidates sorted names = %#v, want backup/beta", plan.SortedNames)
	}

	got := renderCommandCompletionPlan(plan)
	want := []Completion{
		{Name: "backup", Display: "/backup", Description: "backup", Available: true},
		{Name: "beta", Display: "/beta", Description: "canonical beta", Available: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("renderCommandCompletionPlan = %#v, want %#v", got, want)
	}
}

func TestCommandCompletionPlanCanonicalNamesOutrankEarlierAliases(t *testing.T) {
	registry := []cli.CommandPolicy{
		{Name: "backup", Description: "backup", Aliases: []string{"Beta"}},
		{Name: "Beta", Description: "canonical beta"},
	}

	plan := planCommandCompletionCandidates(newCompletionPrefix("b"), registry)
	got := renderCommandCompletionPlan(plan)
	want := []Completion{
		{Name: "backup", Display: "/backup", Description: "backup", Available: true},
		{Name: "beta", Display: "/beta", Description: "canonical beta", Available: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("renderCommandCompletionPlan canonical collision = %#v, want %#v", got, want)
	}
}

func TestCommandCompletionPlanDropsEmptyNamesAndReportsDuplicates(t *testing.T) {
	registry := []cli.CommandPolicy{
		{Name: "", Description: "empty", Aliases: []string{" ", "Help"}},
		{Name: "help", Description: "canonical help"},
		{Name: "Help", Description: "duplicate canonical help"},
	}

	plan := planCommandCompletionCandidates(newCompletionPrefix(""), registry)
	if !reflect.DeepEqual(plan.SortedNames, []string{"help"}) || plan.EmptyDropped != 2 || !reflect.DeepEqual(plan.DuplicateKeys, []string{"help"}) {
		t.Fatalf("planCommandCompletionCandidates empty/duplicate evidence = %#v, want help only, two empty drops, duplicate help", plan)
	}

	got := renderCommandCompletionPlan(plan)
	want := []Completion{{Name: "help", Display: "/help", Description: "canonical help", Available: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("renderCommandCompletionPlan empty/duplicate registry = %#v, want %#v", got, want)
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

func TestPromptTemplateCompletionsNormalizeSlashPrefixedTemplateNames(t *testing.T) {
	catalog := prompttemplates.Catalog{Templates: []prompttemplates.Template{
		{Name: " /Review ", Description: "review", ArgumentHint: "<scope>"},
	}}

	got := PromptTemplateCompletions("/rev", catalog)
	want := []Completion{{Name: "Review", Display: "/Review", Description: "review", ArgumentHint: "<scope>", Available: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PromptTemplateCompletions slash-prefixed template = %#v, want %#v", got, want)
	}
}

func TestCompletionNameNormalizationTrimsAfterSlashPrefix(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{raw: " / Review ", want: "Review"},
		{raw: "// review", want: "review"},
	}
	for _, tc := range cases {
		if got := completionName(tc.raw); got != tc.want {
			t.Fatalf("completionName(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}

	catalog := prompttemplates.Catalog{Templates: []prompttemplates.Template{{Name: " / Review ", Description: "review"}}}
	got := PromptTemplateCompletions("/rev", catalog)
	want := []Completion{{Name: "Review", Display: "/Review", Description: "review", Available: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PromptTemplateCompletions with spaced slash prefix = %#v, want %#v", got, want)
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

func TestDynamicCompletionPlansExposeDroppedCandidates(t *testing.T) {
	catalog := prompttemplates.Catalog{Templates: []prompttemplates.Template{
		{Name: " Review ", Description: "first", ArgumentHint: "<scope>"},
		{Name: "/review", Description: "duplicate"},
		{Name: "   ", Description: "empty"},
	}}
	promptPlan := planPromptTemplateCompletions(newCompletionPrefix("rev"), catalog)
	wantPrompt := []Completion{{Name: "Review", Display: "/Review", Description: "first", ArgumentHint: "<scope>", Available: true}}
	if !reflect.DeepEqual(promptPlan.Completions, wantPrompt) || promptPlan.EmptyDropped != 1 || !reflect.DeepEqual(promptPlan.DuplicateKeys, []string{"review"}) {
		t.Fatalf("planPromptTemplateCompletions = %#v, want completions %#v, one empty drop, duplicate review", promptPlan, wantPrompt)
	}

	skillPlan := planSkillCompletions(newCompletionPrefix("rev"), []skills.SkillSlashCommand{
		{Command: "/Review", Description: "first"},
		{Command: "review", Description: "duplicate"},
		{Command: "   ", Description: "empty"},
	})
	wantSkill := []Completion{{Name: "review", Display: "/review", Description: "first", Available: true}}
	if !reflect.DeepEqual(skillPlan.Completions, wantSkill) || skillPlan.EmptyDropped != 1 || !reflect.DeepEqual(skillPlan.DuplicateKeys, []string{"review"}) {
		t.Fatalf("planSkillCompletions = %#v, want completions %#v, one empty drop, duplicate review", skillPlan, wantSkill)
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

func TestAcceptedTextNormalizesSlashPrefixedCompletionNames(t *testing.T) {
	got, ok := AcceptedText("/rev", Completion{Name: " /review ", ArgumentHint: "<scope>"}, true)
	if !ok || got != "/review " {
		t.Fatalf("AcceptedText with slash-prefixed completion = (%q, %v), want /review space true", got, ok)
	}

	got, ok = AcceptedText("/sk", Completion{Name: " /skin "}, true)
	if !ok || got != "/skin" {
		t.Fatalf("AcceptedText with slash-prefixed no-space command = (%q, %v), want /skin true", got, ok)
	}
}

func TestAcceptedTextSpacePolicyMapsOnlyReferenceRegisteredCommands(t *testing.T) {
	for name := range noTrailingSpaceCommands {
		if _, ok := cli.ResolveCommandPolicy(name); !ok {
			t.Fatalf("noTrailingSpaceCommands contains unregistered command %q", name)
		}
	}
	for name := range argumentCommandNames {
		if _, ok := cli.ResolveCommandPolicy(name); !ok {
			t.Fatalf("argumentCommandNames contains unregistered command %q", name)
		}
	}
}

func TestAcceptedTextUsesRegistryContractForTrailingSpace(t *testing.T) {
	branch := CommandCompletions("/bran")
	if len(branch) != 1 || branch[0].Name != "branch" {
		t.Fatalf("CommandCompletions(/bran) = %#v, want branch", branch)
	}
	if got, ok := AcceptedText("/bran", branch[0], true); !ok || got != "/branch " {
		t.Fatalf("AcceptedText(/bran, branch, tab) = (%q, %v), want /branch space true", got, ok)
	}

	model := CommandCompletions("/mod")
	if len(model) != 1 || model[0].Name != "model" {
		t.Fatalf("CommandCompletions(/mod) = %#v, want model", model)
	}
	if got, ok := AcceptedText("/mod", model[0], true); !ok || got != "/model" {
		t.Fatalf("AcceptedText(/mod, model, tab) = (%q, %v), want /model true", got, ok)
	}
}

func TestAcceptedTextSubcommandExactWithExtraWhitespaceIsInertOnEnter(t *testing.T) {
	got, ok := AcceptedText("/reasoning   show", Completion{Name: "show"}, false)
	if ok || got != "/reasoning   show" {
		t.Fatalf("AcceptedText exact subcommand with extra spacing = (%q, %v), want original text false", got, ok)
	}

	plan := planAcceptedText("/reasoning   show", Completion{Name: "show"}, false)
	want := acceptedTextPlan{Text: "/reasoning   show", Reason: acceptedTextReasonExactRejected}
	if plan != want {
		t.Fatalf("planAcceptedText exact subcommand with extra spacing = %#v, want %#v", plan, want)
	}
}

func TestAcceptedTextRejectsStaleCompletionAfterSubcommandArguments(t *testing.T) {
	input := "/reasoning show now"
	got, ok := AcceptedText(input, Completion{Name: "help"}, true)
	if ok || got != input {
		t.Fatalf("AcceptedText with stale completion after subcommand args = (%q, %v), want original text false", got, ok)
	}

	plan := planAcceptedText(input, Completion{Name: "help"}, true)
	want := acceptedTextPlan{Text: input, Reason: acceptedTextReasonUnsupportedInput}
	if plan != want {
		t.Fatalf("planAcceptedText with stale completion after subcommand args = %#v, want %#v", plan, want)
	}
}

func TestAcceptedTextRejectsStaleSubcommandCompletionOutsideCurrentPrefix(t *testing.T) {
	input := "/reasoning sh"
	got, ok := AcceptedText(input, Completion{Name: "hide"}, true)
	if ok || got != input {
		t.Fatalf("AcceptedText with stale subcommand completion = (%q, %v), want original text false", got, ok)
	}

	plan := planAcceptedText(input, Completion{Name: "hide"}, true)
	want := acceptedTextPlan{Text: input, Reason: acceptedTextReasonUnsupportedInput}
	if plan != want {
		t.Fatalf("planAcceptedText with stale subcommand completion = %#v, want %#v", plan, want)
	}
}

func TestAcceptedTextRejectsStaleCommandCompletionOutsideCurrentPrefix(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "slash prefix mismatch", input: "/he"},
		{name: "not a slash completion request", input: "he"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := AcceptedText(tc.input, Completion{Name: "model"}, true)
			if ok || got != tc.input {
				t.Fatalf("AcceptedText with stale command completion = (%q, %v), want original text false", got, ok)
			}

			plan := planAcceptedText(tc.input, Completion{Name: "model"}, true)
			want := acceptedTextPlan{Text: tc.input, Reason: acceptedTextReasonUnsupportedInput}
			if plan != want {
				t.Fatalf("planAcceptedText with stale command completion = %#v, want %#v", plan, want)
			}
		})
	}
}

func TestAcceptedTextPlanExposesCompletionPath(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		completion  Completion
		acceptExact bool
		want        acceptedTextPlan
	}{
		{
			name:       "empty completion is inert",
			input:      "/he",
			completion: Completion{Name: "   "},
			want:       acceptedTextPlan{Text: "/he", Reason: acceptedTextReasonEmptyCompletion},
		},
		{
			name:       "subcommand preserves base and adds candidate",
			input:      "/reasoning sh",
			completion: Completion{Name: "show"},
			want:       acceptedTextPlan{Text: "/reasoning show", Changed: true, Reason: acceptedTextReasonSubcommand},
		},
		{
			name:        "tab exact subcommand appends argument separator",
			input:       "/reasoning none",
			completion:  Completion{Name: "none"},
			acceptExact: true,
			want:        acceptedTextPlan{Text: "/reasoning none ", Changed: true, Reason: acceptedTextReasonSubcommand},
		},
		{
			name:       "enter exact command is rejected",
			input:      "/help",
			completion: Completion{Name: "help"},
			want:       acceptedTextPlan{Text: "/help", Reason: acceptedTextReasonExactRejected},
		},
		{
			name:        "tab accepted argument command appends space",
			input:       "/rev",
			completion:  Completion{Name: "review", ArgumentHint: "<scope>"},
			acceptExact: true,
			want:        acceptedTextPlan{Text: "/review ", Changed: true, Reason: acceptedTextReasonCommand},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := planAcceptedText(tc.input, tc.completion, tc.acceptExact); got != tc.want {
				t.Fatalf("planAcceptedText(%q, %#v, %v) = %#v, want %#v", tc.input, tc.completion, tc.acceptExact, got, tc.want)
			}
		})
	}
}

func TestCommandAutoSuggestUsesCompletionPrefixNormalization(t *testing.T) {
	if got := CommandCompletions("//hel"); len(got) != 1 || got[0].Name != "help" {
		t.Fatalf("CommandCompletions(//hel) = %#v, want normalized help completion", got)
	}
	if got := AutoSuggest("//hel"); got != "p" {
		t.Fatalf("AutoSuggest(//hel) = %q, want p to match command completion normalization", got)
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

func TestCommandAutoSuggestDoesNotExtendExactAlias(t *testing.T) {
	if got := AutoSuggest("/q"); got != "" {
		t.Fatalf("AutoSuggest(/q) = %q, want no ghost text because /q is an exact queue alias", got)
	}
	plan := commandAutoSuggestPlanFor("q")
	if !plan.Exact || len(plan.Extending) == 0 {
		t.Fatalf("commandAutoSuggestPlanFor(q) = %#v, want exact alias with visible longer candidates", plan)
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

func TestUniqueCompletionPlanExposesDroppedCandidates(t *testing.T) {
	plan := planUniqueCompletions([]Completion{
		{Name: "Beta", Description: "kept first beta"},
		{Name: "   ", Description: "empty candidate"},
		{Name: "/alpha", Description: "sorted first"},
		{Name: "beta", Description: "duplicate beta"},
		{Name: "//ALPHA", Description: "duplicate alpha"},
	})

	wantCompletions := []Completion{
		{Name: "/alpha", Description: "sorted first"},
		{Name: "Beta", Description: "kept first beta"},
	}
	wantDuplicates := []string{"beta", "alpha"}
	if !reflect.DeepEqual(plan.Completions, wantCompletions) || plan.EmptyDropped != 1 || !reflect.DeepEqual(plan.DuplicateKeys, wantDuplicates) {
		t.Fatalf("planUniqueCompletions = %#v, want completions %#v, one empty drop, duplicates %#v", plan, wantCompletions, wantDuplicates)
	}
}

func TestCompletionRequestParsingClassifiesCommandAndSubcommandFlow(t *testing.T) {
	command, ok := parseCompletionRequest("/he")
	if !ok || !command.commandOnly() || command.commandPrefix.string() != "he" {
		t.Fatalf("parseCompletionRequest(/he) = (%#v, %v), want command prefix he", command, ok)
	}

	sub, ok := parseCompletionRequest("/Reasoning   Sh")
	if !ok || !sub.subcommandOnly() || sub.base != "/reasoning" || sub.subPrefix.string() != "sh" {
		t.Fatalf("parseCompletionRequest(/Reasoning   Sh) = (%#v, %v), want canonical subcommand request", sub, ok)
	}

	emptySub, ok := parseCompletionRequest("/reasoning   ")
	if !ok || !emptySub.subcommandOnly() || emptySub.base != "/reasoning" || emptySub.subPrefix.string() != "" {
		t.Fatalf("parseCompletionRequest(/reasoning spaces) = (%#v, %v), want empty subcommand prefix", emptySub, ok)
	}

	if got, ok := parseCompletionRequest("/reasoning show now"); ok || got != (completionRequest{}) {
		t.Fatalf("parseCompletionRequest with subcommand args = (%#v, %v), want rejected", got, ok)
	}
}

func TestCompletionInputSplitExposesWhitespaceAndArgs(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  completionInputParts
		ok    bool
	}{
		{name: "not slash", input: "help", ok: false},
		{name: "command only", input: "/he", want: completionInputParts{command: "/he"}, ok: true},
		{name: "empty subcommand slot", input: "/reasoning   ", want: completionInputParts{command: "/reasoning", hasSubcommandSlot: true}, ok: true},
		{name: "subcommand prefix", input: "/Reasoning\tSh", want: completionInputParts{command: "/Reasoning", subword: "Sh", hasSubcommandSlot: true}, ok: true},
		{name: "subcommand args", input: "/reasoning show now", want: completionInputParts{command: "/reasoning", subword: "show", hasSubcommandSlot: true, hasArgs: true}, ok: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := splitCompletionInput(tc.input)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("splitCompletionInput(%q) = (%#v, %v), want (%#v, %v)", tc.input, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestResolveSubcommandFlowCentralizesPolicyGate(t *testing.T) {
	flow, ok := resolveSubcommandFlow("/Reasoning   Sh")
	if !ok || flow.Base != "/reasoning" || flow.Prefix.string() != "sh" || len(flow.Subcommands) == 0 {
		t.Fatalf("resolveSubcommandFlow(/Reasoning   Sh) = (%#v, %v), want canonical reasoning flow", flow, ok)
	}

	for _, input := range []string{"/he", "/help ", "/reasoning show now"} {
		if got, ok := resolveSubcommandFlow(input); ok || got.Base != "" || got.Prefix.string() != "" || got.Subcommands != nil {
			t.Fatalf("resolveSubcommandFlow(%q) = (%#v, %v), want rejected", input, got, ok)
		}
	}
}

func TestCompletionPrefixesNormalizeBeforeMatching(t *testing.T) {
	commandPrefix := newCompletionPrefix(" /REV ")
	if !commandPrefix.matches("Review") || commandPrefix.matches("help") {
		t.Fatalf("completionPrefix matching did not normalize command prefix: %#v", commandPrefix)
	}

	subPrefix := newSubcommandPrefix(" /Sh ")
	candidates := matchingSubcommandCandidates([]string{"Show", "hide"}, subPrefix)
	want := []subcommandCandidate{{name: "Show", key: "show"}}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("matchingSubcommandCandidates with slash-prefixed mixed-case prefix = %#v, want %#v", candidates, want)
	}
}

func TestSubcommandCompletionCandidatesPreservePolicyOrderAndDeduplicate(t *testing.T) {
	candidates := matchingSubcommandCandidates([]string{"status", "Show", "hide", "show", "", "start"}, newSubcommandPrefix("s"))
	wantCandidates := []subcommandCandidate{
		{name: "status", key: "status"},
		{name: "Show", key: "show"},
		{name: "start", key: "start"},
	}
	if !reflect.DeepEqual(candidates, wantCandidates) {
		t.Fatalf("matchingSubcommandCandidates policy-order duplicate candidates = %#v, want %#v", candidates, wantCandidates)
	}

	got := matchingSubcommandCompletions([]string{"status", "Show", "hide", "show", "", "start"}, newSubcommandPrefix("s"))
	want := []Completion{
		{Name: "status", Display: "status", Available: true},
		{Name: "Show", Display: "Show", Available: true},
		{Name: "start", Display: "start", Available: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("matchingSubcommandCompletions policy-order duplicate candidates = %#v, want %#v", got, want)
	}
}

func TestSubcommandCandidatePlanNormalizesRawPolicyValues(t *testing.T) {
	plan := planMatchingSubcommandCandidates([]string{" /Show ", "show", "", "  "}, newSubcommandPrefix("sh"))
	wantCandidates := []subcommandCandidate{{name: "Show", key: "show"}}
	wantDuplicates := []string{"show"}
	if !reflect.DeepEqual(plan.Candidates, wantCandidates) || plan.EmptyDropped != 2 || !reflect.DeepEqual(plan.DuplicateKeys, wantDuplicates) {
		t.Fatalf("planMatchingSubcommandCandidates = %#v, want candidates %#v, two empty drops, duplicates %#v", plan, wantCandidates, wantDuplicates)
	}

	got := matchingSubcommandCompletions([]string{" /Show "}, newSubcommandPrefix("sh"))
	want := []Completion{{Name: "Show", Display: "Show", Available: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("matchingSubcommandCompletions normalizes raw policy values = %#v, want %#v", got, want)
	}
}

func TestSubcommandAutoSuggestDeduplicatesCaseVariantCandidates(t *testing.T) {
	policySubcommands := []string{"Show", "show"}
	matches := matchingSubcommandCandidates(policySubcommands, newSubcommandPrefix("sh"))
	if len(matches) != 1 || matches[0].key != "show" {
		t.Fatalf("matchingSubcommandCandidates duplicate case variants = %#v, want one show candidate", matches)
	}

	if suffix := singleSubcommandCandidateSuffix(newSubcommandPrefix("sh"), matches); suffix != "ow" {
		t.Fatalf("singleSubcommandCandidateSuffix duplicate case variants = %q, want ow", suffix)
	}
}

func TestSubcommandAutoSuggestIgnoresInconsistentCandidatePlans(t *testing.T) {
	matches := []subcommandCandidate{{name: "Show", key: "show"}}
	if suffix := singleSubcommandCandidateSuffix(newSubcommandPrefix("status"), matches); suffix != "" {
		t.Fatalf("singleSubcommandCandidateSuffix inconsistent plan = %q, want empty suffix", suffix)
	}

	if suffix, ok := subcommandCandidateSuffix("status", "show"); ok || suffix != "" {
		t.Fatalf("subcommandCandidateSuffix(status, show) = (%q, %v), want empty false", suffix, ok)
	}
}

func TestSubcommandCompletionWhitespaceIsConsistent(t *testing.T) {
	for _, input := range []string{"/reasoning sh", "/reasoning\tsh", "/reasoning   sh", "/reasoning /sh"} {
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
