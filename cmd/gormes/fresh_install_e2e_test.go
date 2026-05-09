package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// TestFreshInstallE2E_NoNullArrayFieldsInJSON is the consolidated
// fresh-install conformance battery for the `--json` arc. It runs
// every documented read-only inventory command on a synthetic clean
// GORMES_HOME and asserts the convention we've enforced piecewise
// in slices 32-38: every empty inventory must serialize as an empty
// array (`[]`) or empty map (`{}`), never `null`. Fleet automation
// iterating those surfaces without nil-checks then can't crash on
// missing state.
//
// New `--json` surfaces inherit this contract by name: if a command
// is added to the suite below, it gets the no-null check for free.
// Add new entries here when shipping a new `--json` surface so the
// next nuke+reinstall+probe cycle finds nothing.
//
// This is the E2E layer above the per-command unit tests
// (kanban_list_json_empty_test.go, gateway_status_json_empty_test.go,
// etc.): unit tests assert per-command shape; this test asserts the
// invariant holds across the entire surface.
func TestFreshInstallE2E_NoNullArrayFieldsInJSON(t *testing.T) {
	root := freshInstallE2EHome(t)

	// Each entry: command-line args (without `gormes` prefix).
	// All commands are run in `--json` mode; commands that don't
	// support `--json` are excluded (they're covered by sibling
	// text-mode E2E batteries).
	cases := [][]string{
		{"session", "list", "--json"},
		{"memory", "status", "--json"},
		{"status", "--json", "--progress", filepath.Join(root, "no-such-progress.json")},
		{"gateway", "probe", "--json"},
		{"gateway", "discover", "--json", "--timeout", "50"},
		{"gateway", "status", "--json"},
		{"kanban", "list", "--json"},
		{"kanban", "boards", "list", "--json"},
		{"auth", "list", "--json"},
		{"profile", "list", "--json"},
		{"profile", "show", "--json"},
		{"checkpoints", "status", "--json"},
		{"channels", "capabilities", "--json"},
		{"config", "show", "--json"},
		{"config", "check", "--json"},
		{"onboard", "--json"},
		{"onboard", "--wizard", "--json", "--non-interactive"},
		{"version", "--json"},
		{"restore", "--list", "--json"},
		{"update", "--check", "--json"},
		{"plugins", "list", "--json"},
	}

	for _, args := range cases {
		args := args
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, _, _ := executeRootCommandForTest(cmd, args...)
			// Some commands return a non-zero exit on degraded
			// state (gateway probe, status with missing
			// progress.json, etc.) — that's fine. The contract
			// under test is JSON shape, not exit code.

			if strings.TrimSpace(stdout) == "" {
				// Some commands legitimately emit empty stdout on
				// error paths (e.g. flag-validation failures from
				// cobra). Skip those — sibling unit tests cover
				// the error-mode shape.
				return
			}

			var parsed map[string]any
			if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
				// Not JSON (some commands ignore --json on certain
				// flag combos). Sibling unit tests own the
				// per-command --json contract; here we only
				// enforce no-null on the JSON surfaces that emit
				// JSON.
				return
			}
			banned := findNullFields(parsed, "")
			if len(banned) > 0 {
				t.Fatalf("command %q --json emitted null fields where empty arrays/maps are required: %s\nfull stdout:\n%s",
					strings.Join(args, " "), strings.Join(banned, ", "), stdout)
			}
		})
	}
}

// TestFreshInstallE2E_TypoSuggestionsAcrossParents is the
// consolidated battery for slice 26 (parent-command typo
// suggestions). It sweeps every documented parent that exposes
// only subcommands and proves a single-edit-distance typo
// surfaces "did you mean" guidance.
//
// The historical regression was that cobra's `NoArgs` validator
// short-circuited the suggestion path; the in-tree
// `installParentUnknownSubcommandGuards` helper now installs the
// typo-aware guard. This test pins the contract per-parent so any
// new parent ships with suggestions for free, OR the regression
// is loud at CI time.
func TestFreshInstallE2E_TypoSuggestionsAcrossParents(t *testing.T) {
	freshInstallE2EHome(t)

	cases := []struct {
		parent           string
		typo             string
		wantedSuggestion string
	}{
		{"session", "lst", "list"},
		{"session", "expor", "export"},
		{"memory", "statuss", "status"},
		{"goncho", "doctorr", "doctor"},
		{"kanban", "lst", "list"},
		{"profile", "lst", "list"},
		{"profile", "infoo", "info"},
		{"channels", "capabilites", "capabilities"},
		{"checkpoints", "statuss", "status"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.parent+"_"+tc.typo, func(t *testing.T) {
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeRootCommandForTest(cmd, tc.parent, tc.typo)
			if err == nil {
				t.Fatalf("typo `%s %s` must error; stdout=%q stderr=%q", tc.parent, tc.typo, stdout, stderr)
			}
			combined := strings.ToLower(err.Error() + "\n" + stderr + "\n" + stdout)
			if !strings.Contains(combined, "did you mean") {
				t.Fatalf("typo `%s %s` must include `did you mean…`; got:\n%s", tc.parent, tc.typo, combined)
			}
			if !strings.Contains(combined, strings.ToLower(tc.wantedSuggestion)) {
				t.Fatalf("typo `%s %s` must suggest %q; got:\n%s", tc.parent, tc.typo, tc.wantedSuggestion, combined)
			}
		})
	}
}

// TestFreshInstallE2E_FreshInstallReadOnlyCommandsExitZero pins the
// empty-state UX battery: read-only inventory commands on a fresh
// install (no memory.db, no kanban.db, no sessions, no plugins)
// must succeed with exit 0 and a friendly empty-state message.
// Slices 21 (session list), 23 (memory status), 25 (memory schema-less),
// 32-33 (kanban) all fixed individual cases; this test pins the
// invariant across the inventory surface so no new command
// regresses.
func TestFreshInstallE2E_FreshInstallReadOnlyCommandsExitZero(t *testing.T) {
	freshInstallE2EHome(t)

	cases := [][]string{
		{"session", "list"},
		{"memory", "status"},
		{"kanban", "list"},
		{"kanban", "boards", "list"},
		{"auth", "list"},
		{"profile", "list"},
		{"checkpoints", "list"},
		{"plugins", "list"},
	}
	for _, args := range cases {
		args := args
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeRootCommandForTest(cmd, args...)
			if err != nil {
				t.Fatalf("fresh-install `%s` must exit 0; got %v\nstdout=%s\nstderr=%s",
					strings.Join(args, " "), err, stdout, stderr)
			}
		})
	}
}

// TestFreshInstallE2E_BuildProvenancePresentInJSON is the
// build-attribution conformance battery for the `--json` arc. Every
// `--json` surface (except `version --json`, which IS the build
// report) must lead with a `{build: {version, git_commit}}` block so
// fleet automation aggregating snapshots across machines can
// attribute each document to a specific binary. Slices 32-40 added
// the per-command shape; this test consolidates the invariant.
func TestFreshInstallE2E_BuildProvenancePresentInJSON(t *testing.T) {
	freshInstallE2EHome(t)

	// Inventory commands that document `build` in their --json
	// shape. `version --json` is excluded — it IS the build report,
	// not a wrapper carrying build provenance.
	cases := [][]string{
		{"session", "list", "--json"},
		{"memory", "status", "--json"},
		{"gateway", "discover", "--json", "--timeout", "50"},
		{"gateway", "status", "--json"},
		{"kanban", "list", "--json"},
		{"kanban", "boards", "list", "--json"},
		{"auth", "list", "--json"},
		{"profile", "list", "--json"},
		{"profile", "show", "--json"},
		{"checkpoints", "status", "--json"},
		{"channels", "capabilities", "--json"},
		{"config", "show", "--json"},
		{"config", "check", "--json"},
		{"onboard", "--json"},
		{"onboard", "--wizard", "--json", "--non-interactive"},
		{"restore", "--list", "--json"},
		{"update", "--check", "--json"},
		{"plugins", "list", "--json"},
	}

	for _, args := range cases {
		args := args
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, _, _ := executeRootCommandForTest(cmd, args...)
			if strings.TrimSpace(stdout) == "" {
				return
			}
			var parsed map[string]any
			if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
				return
			}
			build, ok := parsed["build"].(map[string]any)
			if !ok {
				t.Fatalf("command %q --json missing `build` block (or wrong type); fleet log pipelines depend on this. Full stdout:\n%s",
					strings.Join(args, " "), stdout)
			}
			version, _ := build["version"].(string)
			if version == "" {
				t.Errorf("command %q --json: build.version is empty/missing", strings.Join(args, " "))
			}
			gitCommit, _ := build["git_commit"].(string)
			if gitCommit == "" {
				t.Errorf("command %q --json: build.git_commit is empty/missing", strings.Join(args, " "))
			}
		})
	}
}

// TestFreshInstallE2E_JSONIsParseable is the JSON-validity invariant.
// Every `--json` command must emit a single top-level JSON document
// when it produces stdout. Catches regressions where an error path
// or interactive prompt accidentally writes prose to stdout in JSON
// mode (a real bug class — `gormes setup terminal --non-interactive
// --json` historically printed the human menu when --json wasn't
// implemented for that section).
func TestFreshInstallE2E_JSONIsParseable(t *testing.T) {
	freshInstallE2EHome(t)

	cases := [][]string{
		{"session", "list", "--json"},
		{"memory", "status", "--json"},
		{"gateway", "discover", "--json", "--timeout", "50"},
		{"gateway", "status", "--json"},
		{"kanban", "list", "--json"},
		{"kanban", "boards", "list", "--json"},
		{"auth", "list", "--json"},
		{"profile", "list", "--json"},
		{"profile", "show", "--json"},
		{"checkpoints", "status", "--json"},
		{"channels", "capabilities", "--json"},
		{"config", "show", "--json"},
		{"config", "check", "--json"},
		{"config", "get", "hermes.model", "--json"},
		{"onboard", "--json"},
		{"onboard", "--wizard", "--json", "--non-interactive"},
		{"version", "--json"},
		{"restore", "--list", "--json"},
		{"update", "--check", "--json"},
		{"plugins", "list", "--json"},
	}
	for _, args := range cases {
		args := args
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, _, _ := executeRootCommandForTest(cmd, args...)
			trimmed := strings.TrimSpace(stdout)
			if trimmed == "" {
				return
			}
			var parsed any
			if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
				t.Fatalf("command %q --json must emit parseable JSON to stdout; got error %v\nstdout=%s",
					strings.Join(args, " "), err, stdout)
			}
		})
	}
}

// TestFreshInstallE2E_NotFoundJSONEmitsStructuredDocument is the
// "look up X" not-found conformance battery. Every command that
// resolves an operator-supplied ID (kanban show/claim/complete,
// session export, mcp login, etc.) must emit a parseable
// `{build, action: "not_found", id, ...}` document on stdout when
// `--json` is set and the target doesn't exist. Operators driving
// fleet automation can't otherwise distinguish "missing target"
// from "command crashed" — both yield empty stdout + non-zero exit.
//
// Each entry: (args without `gormes` prefix, optional pre-init
// command). Commands that need a pre-init step (kanban needs the
// store initialized) supply it; otherwise nil.
func TestFreshInstallE2E_NotFoundJSONEmitsStructuredDocument(t *testing.T) {
	cases := []struct {
		name    string
		preInit []string
		args    []string
	}{
		{name: "kanban_show", preInit: []string{"kanban", "init"}, args: []string{"kanban", "show", "missing-id", "--json"}},
		{name: "kanban_complete", preInit: []string{"kanban", "init"}, args: []string{"kanban", "complete", "missing-id", "--json"}},
		{name: "kanban_claim", preInit: []string{"kanban", "init"}, args: []string{"kanban", "claim", "missing-id", "--json"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			freshInstallE2EHome(t)
			if tc.preInit != nil {
				cmd := newRootCommandWithRuntime(rootRuntime{})
				if _, _, err := executeRootCommandForTest(cmd, tc.preInit...); err != nil {
					t.Fatalf("pre-init `%s`: %v", strings.Join(tc.preInit, " "), err)
				}
			}
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeRootCommandForTest(cmd, tc.args...)
			if err == nil {
				t.Fatalf("`%s` must error on missing target; stdout=%q stderr=%q",
					strings.Join(tc.args, " "), stdout, stderr)
			}
			if strings.TrimSpace(stdout) == "" {
				t.Fatalf("`%s` --json must emit JSON on stdout even on not-found; got empty stdout, stderr=%s",
					strings.Join(tc.args, " "), stderr)
			}
			var parsed map[string]any
			if jsonErr := json.Unmarshal([]byte(stdout), &parsed); jsonErr != nil {
				t.Fatalf("stdout must be parseable JSON: %v\nstdout=%s", jsonErr, stdout)
			}
			action, _ := parsed["action"].(string)
			if action != "not_found" {
				t.Errorf("`%s` --json action = %q, want %q",
					strings.Join(tc.args, " "), action, "not_found")
			}
		})
	}
}

// TestFreshInstallE2E_InvalidInputJSONEmitsStructuredError extends
// the conformance fence to the missing-arg / missing-required-flag /
// unknown-subcommand axes that slipped through in v0.2.0. Each
// command paired with `--json` must emit a parseable JSON document
// on stdout — never just a cobra error to stderr — so fleet
// automation can distinguish "user mistake" (missing arg) from
// "command crashed" (empty stdout, exit 1) the same way the
// not-found battery already does for kanban lookups.
//
// Cases:
//
//	auth status --json                  → missing required <provider> arg
//	logs --json                         → no log file exists yet
//	secrets audit --json                → missing required --plan flag
//	restore --json                      → missing required --list/--latest/--path
//	mcp <bad-subcommand> --json         → unknown subcommand under parent
//
// Contract: each must (1) exit non-zero, (2) write a parseable JSON
// document to stdout carrying at minimum `{build, action, error}`,
// (3) leave stderr free to mirror the human-readable error if
// useful. Same convention as the kanban not-found battery
// (TestFreshInstallE2E_NotFoundJSONEmitsStructuredDocument).
func TestFreshInstallE2E_InvalidInputJSONEmitsStructuredError(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantAction string
	}{
		{name: "auth_status_missing_arg", args: []string{"auth", "status", "--json"}, wantAction: "missing_argument"},
		{name: "logs_no_log_file", args: []string{"logs", "--json"}, wantAction: "no_logs"},
		{name: "secrets_audit_missing_plan_flag", args: []string{"secrets", "audit", "--json"}, wantAction: "missing_flag"},
		{name: "restore_missing_arg", args: []string{"restore", "--json"}, wantAction: "missing_argument"},
		{name: "mcp_unknown_subcommand", args: []string{"mcp", "definitely-not-a-subcommand", "--json"}, wantAction: "unknown_subcommand"},
		// Typo-with-suggestion paths short-circuit through cobra's
		// built-in `Find()`/`findSuggestions` before any parent's
		// RunE guard fires. installParentUnknownSubcommandGuards
		// only catches the no-suggestion case; these subtests pin
		// the conformance fence for the suggestion case too. The
		// fix lives at executeRootCommand's error wrapper.
		{name: "config_typo_with_suggestion", args: []string{"config", "gat", "--json"}, wantAction: "unknown_subcommand"},
		{name: "kanban_typo_with_suggestion", args: []string{"kanban", "shor", "--json"}, wantAction: "unknown_subcommand"},
		// Gateway parent has its own RunE (`runGateway`) that runs
		// the actual gateway when no subcommand matches. cobra
		// parses `--json` BEFORE the RunE fires and rejects it as
		// "unknown flag --json" because gateway parent doesn't
		// register a --json flag. The conformance fence wraps this
		// at the parent level so `gateway <bad-subcommand> --json`
		// emits the same `unknown_subcommand` document as every
		// other parent. Same intent: the user typed an invocation
		// containing --json; conformance demands JSON on stdout.
		{name: "gateway_unknown_subcommand_json", args: []string{"gateway", "definitely-not-a-subcommand", "--json"}, wantAction: "unknown_subcommand"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			freshInstallE2EHome(t)
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeRootCommandForTest(cmd, tc.args...)
			if err == nil {
				t.Fatalf("`gormes %s` must error on invalid input; stdout=%q stderr=%q",
					strings.Join(tc.args, " "), stdout, stderr)
			}
			if strings.TrimSpace(stdout) == "" {
				t.Fatalf("`gormes %s` --json must emit JSON on stdout even on invalid input; got empty stdout. stderr=%s",
					strings.Join(tc.args, " "), stderr)
			}
			var parsed map[string]any
			if jsonErr := json.Unmarshal([]byte(stdout), &parsed); jsonErr != nil {
				t.Fatalf("`gormes %s` stdout must be parseable JSON: %v\nstdout=%s",
					strings.Join(tc.args, " "), jsonErr, stdout)
			}
			action, _ := parsed["action"].(string)
			if action != tc.wantAction {
				t.Errorf("`gormes %s` action = %q, want %q",
					strings.Join(tc.args, " "), action, tc.wantAction)
			}
			build, _ := parsed["build"].(map[string]any)
			if build == nil {
				t.Errorf("`gormes %s` --json must include build provenance; got=%v",
					strings.Join(tc.args, " "), parsed)
			}
			if errStr, _ := parsed["error"].(string); strings.TrimSpace(errStr) == "" {
				t.Errorf("`gormes %s` --json must include a human-readable `error` field; got=%v",
					strings.Join(tc.args, " "), parsed)
			}
		})
	}
}

// freshInstallE2EHome sets up a synthetic GORMES_HOME the way a
// fresh install looks: empty directory, no DBs, no auth, no
// gateway state. Returns the root for tests that want to construct
// paths underneath.
func freshInstallE2EHome(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes-home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes-home"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex-home"))
	// Belt-and-suspenders: zero out any provider env that could
	// otherwise let auth status pick up the developer's real creds.
	t.Setenv("GORMES_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	return root
}

// findNullFields walks a decoded JSON document and returns the
// dot-paths of every field whose value is `null`. Used by the no-null
// E2E battery to flag fields that should be `[]`/`{}`.
func findNullFields(v any, path string) []string {
	switch t := v.(type) {
	case nil:
		return []string{path}
	case map[string]any:
		var out []string
		for k, child := range t {
			out = append(out, findNullFields(child, fmt.Sprintf("%s.%s", path, k))...)
		}
		return out
	case []any:
		// Empty arrays are explicitly fine; we only flag null
		// values, not empty containers. Items within the array are
		// not walked (they may legitimately contain null fields
		// like missing optional record sub-objects).
		return nil
	}
	return nil
}
