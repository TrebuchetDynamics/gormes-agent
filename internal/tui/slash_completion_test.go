package tui

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

// TestHermesSlashCompletion_CommandPrefix proves typed slash prefixes resolve
// to matching canonical commands and aliases from internal/cli.CommandRegistry,
// preserving Hermes prompt_toolkit completion semantics: the leading "/" is
// stripped before prefix matching, completions are returned in stable
// alphabetical order, and an exact name match still surfaces (so the dropdown
// can stay open like Hermes does).
func TestHermesSlashCompletion_CommandPrefix(t *testing.T) {
	cases := []struct {
		input string
		want  []string // names without leading "/"
	}{
		{input: "/he", want: []string{"help"}},
		{input: "/r", want: collectRegistryNamesWithPrefix("r")},
		{input: "/", want: collectAllRegistryNames()},
		{input: "/RES", want: []string{"reset", "restart", "resume"}},
		{input: "/help", want: []string{"help"}},
		{input: "/no-such-command-xyzzy", want: nil},
	}
	for _, tc := range cases {
		got := HermesSlashCommandCompletions(tc.input)
		gotNames := completionNames(got)
		if !reflect.DeepEqual(gotNames, tc.want) {
			t.Errorf("HermesSlashCommandCompletions(%q) = %v, want %v", tc.input, gotNames, tc.want)
		}
	}
}

// TestHermesSlashCompletion_SubcommandPrefix proves reasoning subcommands
// surface in the Hermes-canonical order ("none", "minimal", "low", "medium",
// "high", "xhigh", "show", "hide", "on", "off") for `/reasoning ` (trailing
// space, no prefix) and that `/reasoning sh` filters down to the prefix-match
// subset preserving order.
func TestHermesSlashCompletion_SubcommandPrefix(t *testing.T) {
	wantAll := []string{"none", "minimal", "low", "medium", "high", "xhigh", "show", "hide", "on", "off"}
	got := HermesSlashSubcommandCompletions("/reasoning ")
	if !reflect.DeepEqual(completionNames(got), wantAll) {
		t.Errorf("HermesSlashSubcommandCompletions(\"/reasoning \") = %v, want %v", completionNames(got), wantAll)
	}

	wantSh := []string{"show"}
	gotSh := HermesSlashSubcommandCompletions("/reasoning sh")
	if !reflect.DeepEqual(completionNames(gotSh), wantSh) {
		t.Errorf("HermesSlashSubcommandCompletions(\"/reasoning sh\") = %v, want %v", completionNames(gotSh), wantSh)
	}

	// Exact subcommand match still surfaces so the user can keep editing the
	// completion menu (mirrors prompt_toolkit's _completion_text trailing-space
	// behavior at the helper level: the entry is kept; the UI layer is free to
	// append a trailing space).
	gotExact := HermesSlashSubcommandCompletions("/reasoning none")
	if !reflect.DeepEqual(completionNames(gotExact), []string{"none"}) {
		t.Errorf("HermesSlashSubcommandCompletions(\"/reasoning none\") = %v, want [none]", completionNames(gotExact))
	}

	// Subcommands only resolve while editing the first sub-token. After a
	// space inside the args (e.g. `/reasoning show extra`) no further static
	// subcommand completions are surfaced.
	gotPast := HermesSlashSubcommandCompletions("/reasoning show extra")
	if len(gotPast) != 0 {
		t.Errorf("HermesSlashSubcommandCompletions(\"/reasoning show extra\") = %v, want empty (past first sub-token)", completionNames(gotPast))
	}

	// Commands without a registered subcommand inventory return no static
	// subcommand completions; the dynamic /model, /skin, /personality menus
	// are intentionally not part of this slice.
	gotUnknownSub := HermesSlashSubcommandCompletions("/help ")
	if len(gotUnknownSub) != 0 {
		t.Errorf("HermesSlashSubcommandCompletions(\"/help \") = %v, want empty (no subcommand inventory)", completionNames(gotUnknownSub))
	}
}

func TestHermesSlashCompletion_LocalNativeHandlersAreAvailable(t *testing.T) {
	cases := []struct {
		input string
		name  string
	}{
		{input: "/bra", name: "branch"},
		{input: "/fo", name: "fork"},
		{input: "/sav", name: "save"},
		{input: "/cop", name: "copy"},
		{input: "/mou", name: "mouse"},
		{input: "/scr", name: "scroll"},
		{input: "/qui", name: "quit"},
		{input: "/exi", name: "exit"},
		{input: "/red", name: "redraw"},
		{input: "/det", name: "details"},
		{input: "/ind", name: "indicator"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			completions := HermesSlashCommandCompletions(tc.input)
			for _, completion := range completions {
				if completion.Name != tc.name {
					continue
				}
				if !completion.Available {
					t.Fatalf("completion for %s = unavailable; want available for shipped native handler (all=%+v)", tc.name, completions)
				}
				return
			}
			t.Fatalf("HermesSlashCommandCompletions(%q) = %+v, want %s", tc.input, completions, tc.name)
		})
	}
}

// TestHermesSlashCompletion_UnavailableCommandsStillComplete proves
// recognized-but-unported commands appear in completion (so users can discover
// them) while EvaluateActiveTurnVerdict still classifies their dispatch as
// ActiveTurnPolicyUnavailable with explicit evidence — never letting the slash
// text leak to the kernel.
func TestHermesSlashCompletion_UnavailableCommandsStillComplete(t *testing.T) {
	completions := HermesSlashCommandCompletions("/too")
	names := completionNames(completions)
	if !containsString(names, "tools") {
		t.Fatalf("HermesSlashCommandCompletions(\"/too\") = %v, want to include unavailable command \"tools\"", names)
	}

	verdict := cli.EvaluateActiveTurnVerdict("/tools", false)
	if !verdict.Known {
		t.Errorf("EvaluateActiveTurnVerdict(/tools) Known = false, want true (registry recognizes /tools)")
	}
	if verdict.Allowed {
		t.Errorf("EvaluateActiveTurnVerdict(/tools) Allowed = true, want false for unavailable command")
	}
	if verdict.Policy != cli.ActiveTurnPolicyUnavailable {
		t.Errorf("EvaluateActiveTurnVerdict(/tools) Policy = %q, want %q", verdict.Policy, cli.ActiveTurnPolicyUnavailable)
	}
	if !strings.Contains(strings.ToLower(verdict.Evidence), "unavailable") {
		t.Errorf("EvaluateActiveTurnVerdict(/tools) Evidence = %q, want to mention unavailable", verdict.Evidence)
	}
}

func TestHermesSlashCompletion_BrowserSubcommands(t *testing.T) {
	got := HermesSlashSubcommandCompletions("/browser ")
	if !reflect.DeepEqual(completionNames(got), []string{"status", "connect"}) {
		t.Fatalf("HermesSlashSubcommandCompletions(/browser) = %v, want status/connect", completionNames(got))
	}
	verdict := cli.EvaluateActiveTurnVerdict("/browser status", false)
	if !verdict.Known || !verdict.Allowed || verdict.Policy != cli.ActiveTurnPolicyBypass {
		t.Fatalf("EvaluateActiveTurnVerdict(/browser status) = %+v, want local ported bypass command", verdict)
	}
}

func TestRenderSlashCompletionMenu_CommandPrefix(t *testing.T) {
	got := renderSlashCompletionMenu("/n", 72)
	if !strings.Contains(got, "/new") {
		t.Fatalf("renderSlashCompletionMenu(/n) missing /new:\n%s", got)
	}
	if !strings.Contains(got, "Start a fresh session") {
		t.Fatalf("renderSlashCompletionMenu(/n) missing description:\n%s", got)
	}
	if strings.Contains(got, "mouse: disabled") {
		t.Fatalf("completion menu leaked unrelated status text:\n%s", got)
	}
}

func TestRenderSlashCompletionMenu_Subcommands(t *testing.T) {
	got := renderSlashCompletionMenu("/browser ", 72)
	if !strings.Contains(got, "status") || !strings.Contains(got, "connect") {
		t.Fatalf("renderSlashCompletionMenu(/browser ) missing static subcommands:\n%s", got)
	}
}

func TestRenderSlashCompletionMenu_SearchListChrome(t *testing.T) {
	got := renderSlashCompletionMenuWithSkin("/sta", 52, BuiltinSkins()["poseidon"])
	for _, want := range []string{"Search /sta", "❯", "/status", "↑/↓ select", "Enter complete"} {
		if !strings.Contains(got, want) {
			t.Fatalf("search completion menu missing %q:\n%s", want, got)
		}
	}
	for _, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > 52 {
			t.Fatalf("search completion line width %d exceeds 52:\n%q\n\n%s", w, line, got)
		}
	}
}

func TestRenderSlashCompletionMenu_NarrowWidthKeepsChromeCompact(t *testing.T) {
	got := renderSlashCompletionMenu("/", 36)
	lines := strings.Split(got, "\n")
	if len(lines) > 6 {
		t.Fatalf("narrow slash completion menu rendered %d lines, want at most 6 to avoid swallowing the chat UI:\n%s", len(lines), got)
	}
	for _, line := range lines {
		if w := lipgloss.Width(line); w > 36 {
			t.Fatalf("narrow completion line width %d exceeds 36:\n%q\n\n%s", w, line, got)
		}
	}
}

// TestHermesSlashCompletion_AutoSuggest proves the inline ghost suffix matches
// what Hermes' SlashCommandAutoSuggest returns: the unique remaining tail of a
// command or subcommand name, or empty when ambiguous, unknown, or already
// complete. Mirrors hermes_cli/commands.py:SlashCommandAutoSuggest.
func TestHermesSlashCompletion_AutoSuggest(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		// Unique command prefix → suffix to ghost.
		{input: "/he", want: "lp"},
		{input: "/restar", want: "t"},
		// Ambiguous prefix → no suggestion (multiple commands match).
		{input: "/re", want: ""},
		// Already a complete command → no suggestion.
		{input: "/help", want: ""},
		// Unknown prefix → no suggestion.
		{input: "/zzz", want: ""},
		// Empty / non-slash input → no suggestion.
		{input: "", want: ""},
		{input: "hello", want: ""},
		// Subcommand suggestion: unique prefix after command + space.
		{input: "/reasoning mi", want: "nimal"},
		{input: "/reasoning xh", want: "igh"},
		// Ambiguous subcommand → no suggestion (`/reasoning ` matches all 10).
		{input: "/reasoning ", want: ""},
		// Subcommand exact match → no suggestion.
		{input: "/reasoning none", want: ""},
		// Subcommand suggestion only resolves on the first sub-token.
		{input: "/reasoning show ex", want: ""},
		// Commands without a subcommand inventory return no subcommand
		// suggestion even after the space.
		{input: "/help arg", want: ""},
	}
	for _, tc := range cases {
		got := HermesSlashAutoSuggest(tc.input)
		if got != tc.want {
			t.Errorf("HermesSlashAutoSuggest(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestHermesSlashCompletion_DeterministicAndPure proves the helpers run with
// no provider, config, plugin, or filesystem dependency: repeated calls return
// the exact same slice of completions, and the slice is independent across
// calls (mutating one does not poison cached state).
func TestHermesSlashCompletion_DeterministicAndPure(t *testing.T) {
	first := HermesSlashCommandCompletions("/r")
	second := HermesSlashCommandCompletions("/r")
	if !reflect.DeepEqual(completionNames(first), completionNames(second)) {
		t.Errorf("HermesSlashCommandCompletions is non-deterministic: %v vs %v", completionNames(first), completionNames(second))
	}
	// Mutate the returned slice; a subsequent call must still return the
	// canonical list (proves we did not hand out a shared backing array).
	if len(first) > 0 {
		first[0] = SlashCompletion{Name: "POISON"}
	}
	third := HermesSlashCommandCompletions("/r")
	if !reflect.DeepEqual(completionNames(second), completionNames(third)) {
		t.Errorf("HermesSlashCommandCompletions leaks shared state: %v vs %v", completionNames(second), completionNames(third))
	}

	subFirst := HermesSlashSubcommandCompletions("/reasoning ")
	subSecond := HermesSlashSubcommandCompletions("/reasoning ")
	if !reflect.DeepEqual(completionNames(subFirst), completionNames(subSecond)) {
		t.Errorf("HermesSlashSubcommandCompletions is non-deterministic")
	}
}

// completionNames extracts the Name field from a slice of SlashCompletion so
// table tests can compare on a single dimension while still proving the
// returned struct carries a name.
func completionNames(c []SlashCompletion) []string {
	if len(c) == 0 {
		return nil
	}
	out := make([]string, 0, len(c))
	for _, x := range c {
		out = append(out, x.Name)
	}
	return out
}

func containsString(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// collectRegistryNamesWithPrefix returns all canonical command names plus
// aliases that match the prefix, in stable alphabetical order. Used by the
// command-prefix table test to avoid hard-coding a list that drifts as the
// registry grows.
func collectRegistryNamesWithPrefix(prefix string) []string {
	prefix = strings.ToLower(prefix)
	seen := map[string]struct{}{}
	for _, cmd := range cli.CommandRegistry {
		if strings.HasPrefix(cmd.Name, prefix) {
			seen[cmd.Name] = struct{}{}
		}
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, prefix) {
				seen[alias] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func collectAllRegistryNames() []string {
	seen := map[string]struct{}{}
	for _, cmd := range cli.CommandRegistry {
		seen[cmd.Name] = struct{}{}
		for _, alias := range cmd.Aliases {
			seen[alias] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}
