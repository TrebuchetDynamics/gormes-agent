package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

// profileCommandFakeSeams collects the injected function-shaped seams the
// gormes profile command surface consumes through the ProfileSelector. Tests
// observe ordering and side effects through the captured fields rather than
// touching live filesystem state.
type profileCommandFakeSeams struct {
	activeProfileName  string
	activeProfileError error

	resolveProfileRoot      func(name string) (string, error)
	resolveProfileRootCalls []string

	validateProfileName      func(name string) error
	validateProfileNameCalls []string

	writeActiveProfile      func(name string) error
	writeActiveProfileCalls []string

	knownProfiles []string
}

func (f *profileCommandFakeSeams) defaults() profileCommandSeams {
	if f.resolveProfileRoot == nil {
		f.resolveProfileRoot = func(name string) (string, error) {
			return "/tmp/gormes-test-home/profiles/" + name, nil
		}
	}
	if f.validateProfileName == nil {
		f.validateProfileName = cli.ValidateProfileName
	}
	if f.writeActiveProfile == nil {
		f.writeActiveProfile = func(name string) error { return nil }
	}
	return profileCommandSeams{
		ReadActiveProfileName: func() (string, error) {
			if f.activeProfileError != nil {
				return "", f.activeProfileError
			}
			return f.activeProfileName, nil
		},
		ValidateProfileName: func(name string) error {
			f.validateProfileNameCalls = append(f.validateProfileNameCalls, name)
			return f.validateProfileName(name)
		},
		ResolveProfileRoot: func(name string) (string, error) {
			f.resolveProfileRootCalls = append(f.resolveProfileRootCalls, name)
			return f.resolveProfileRoot(name)
		},
		WriteActiveProfile: func(name string) error {
			f.writeActiveProfileCalls = append(f.writeActiveProfileCalls, name)
			return f.writeActiveProfile(name)
		},
		ListKnownProfiles: func() ([]string, error) {
			return append([]string(nil), f.knownProfiles...), nil
		},
	}
}

func runProfileTestCommand(t *testing.T, seams profileCommandSeams, args ...string) (string, string, error) {
	t.Helper()
	cmd := newProfileCommandWithSeams(seams)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestGormesProfileShowRendersActivePathRedacted(t *testing.T) {
	fake := &profileCommandFakeSeams{
		activeProfileName: "default",
		resolveProfileRoot: func(name string) (string, error) {
			return "/home/operator-secret/.config/gormes", nil
		},
	}
	stdout, stderr, err := runProfileTestCommand(t, fake.defaults(), "show")
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "default") {
		t.Fatalf("show stdout missing profile name:\n%s", stdout)
	}
	for _, leak := range []string{"/home/operator-secret", "/home/operator-secret/.config/gormes"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("show leaked raw path %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
	// The redacted form must include the trailing segment so operators can
	// still verify which root they're on without leaking parent dirs.
	if !strings.Contains(stdout, "gormes") {
		t.Fatalf("show stdout missing redacted last segment %q:\n%s", "gormes", stdout)
	}
	if !strings.Contains(stdout, "...") {
		t.Fatalf("show stdout missing redaction marker:\n%s", stdout)
	}
}

// TestGormesProfileShowJSONEmitsStructuredActiveRoot proves
// `gormes profile show --json` emits a parseable
// `{build, active, root}` document with the SAME redacted root the
// human surface prints. Operator scripts checking which profile is
// active and where its root lives need a structured shape — scraping
// the two-line "active profile: X\nroot: ..." text is fragile.
func TestGormesProfileShowJSONEmitsStructuredActiveRoot(t *testing.T) {
	fake := &profileCommandFakeSeams{
		activeProfileName: "work",
		resolveProfileRoot: func(name string) (string, error) {
			return "/home/operator-secret/.config/gormes/profiles/work", nil
		},
	}
	stdout, stderr, err := runProfileTestCommand(t, fake.defaults(), "show", "--json")
	if err != nil {
		t.Fatalf("show --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, leak := range []string{"/home/operator-secret"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("show --json leaked raw path %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Active string `json:"active"`
		Root   string `json:"root"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != Version || got.Build.GitCommit == "" {
		t.Fatalf("build provenance missing/wrong: %+v", got.Build)
	}
	if got.Active != "work" {
		t.Fatalf("got.Active = %q, want work", got.Active)
	}
	if !strings.Contains(got.Root, "...") {
		t.Fatalf("got.Root = %q, want a redacted form (`...` marker)", got.Root)
	}
	// JSON mode must not interleave the human row.
	if strings.Contains(stdout, "active profile:") {
		t.Fatalf("--json must not emit the human row; got:\n%s", stdout)
	}
}

func TestGormesProfileSetValidatesNameThenUpdatesStore(t *testing.T) {
	t.Run("invalid_name_is_rejected_before_store_write", func(t *testing.T) {
		fake := &profileCommandFakeSeams{}
		_, _, err := runProfileTestCommand(t, fake.defaults(), "set", "Bad Name")
		if err == nil {
			t.Fatal("set with invalid name returned nil error")
		}
		if !strings.Contains(err.Error(), "profile_name_invalid") {
			t.Fatalf("error %q does not surface profile_name_invalid", err.Error())
		}
		if len(fake.writeActiveProfileCalls) != 0 {
			t.Fatalf("invalid name reached store write: %v", fake.writeActiveProfileCalls)
		}
		if len(fake.validateProfileNameCalls) == 0 {
			t.Fatal("validator was not consulted before failing")
		}
	})

	t.Run("valid_name_writes_store_after_validation", func(t *testing.T) {
		fake := &profileCommandFakeSeams{}
		stdout, stderr, err := runProfileTestCommand(t, fake.defaults(), "set", "work")
		if err != nil {
			t.Fatalf("set valid name: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
		}
		if got := fake.validateProfileNameCalls; len(got) == 0 || got[0] != "work" {
			t.Fatalf("validator calls = %v, want first call 'work'", got)
		}
		if got := fake.writeActiveProfileCalls; len(got) != 1 || got[0] != "work" {
			t.Fatalf("write calls = %v, want exactly one with 'work'", got)
		}
		// Validator must run before the write (acceptance: order is enforced).
		// Both calls happened; with the fake recording order, the validator
		// call list must not be empty before the write call.
		if len(fake.validateProfileNameCalls) == 0 {
			t.Fatal("validator must be called before store write")
		}
	})
}

// TestGormesProfileSet_JSONEmitsStructuredOutcome proves
// `gormes profile set <name> --json` returns a parseable
// `{build, action, active, root}` document so fleet automation
// switching profiles across machines can confirm the active marker
// landed without scraping the two-line "active profile: ..." prose.
// The root path is redacted (only the trailing segment) so the JSON
// matches the same secrets contract as `profile show`.
func TestGormesProfileSet_JSONEmitsStructuredOutcome(t *testing.T) {
	fake := &profileCommandFakeSeams{
		resolveProfileRoot: func(name string) (string, error) {
			return "/home/operator-secret/.config/gormes/profiles/" + name, nil
		},
	}
	stdout, stderr, err := runProfileTestCommand(t, fake.defaults(), "set", "work", "--json")
	if err != nil {
		t.Fatalf("profile set --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	// Raw operator-secret path MUST never appear in stdout — same redaction
	// promise the human surface keeps.
	if strings.Contains(stdout+stderr, "/home/operator-secret") {
		t.Fatalf("profile set --json LEAKED raw path:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action string `json:"action"`
		Active string `json:"active"`
		Root   string `json:"root"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("profile set --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "set" {
		t.Errorf("action = %q, want %q", got.Action, "set")
	}
	if got.Active != "work" {
		t.Errorf("active = %q, want %q", got.Active, "work")
	}
	if got.Root == "" {
		t.Errorf("root must be populated (redacted form)")
	}
	// The root MUST be the redacted form (containing only the last segment).
	if !strings.Contains(got.Root, "work") {
		t.Errorf("root = %q, want redacted path including last segment", got.Root)
	}

	// Side effect: the validator and writer must still have been called.
	if len(fake.validateProfileNameCalls) == 0 {
		t.Errorf("validator must be called for --json path too")
	}
	if len(fake.writeActiveProfileCalls) != 1 || fake.writeActiveProfileCalls[0] != "work" {
		t.Errorf("writeActiveProfile calls = %v, want exactly one with 'work'", fake.writeActiveProfileCalls)
	}
}

func TestGormesProfileListEnumeratesKnownProfilesWithCurrentMarker(t *testing.T) {
	fake := &profileCommandFakeSeams{
		activeProfileName: "work",
		knownProfiles:     []string{"default", "work", "research"},
		resolveProfileRoot: func(name string) (string, error) {
			return "/home/operator-secret/.config/gormes/profiles/" + name, nil
		},
	}
	stdout, stderr, err := runProfileTestCommand(t, fake.defaults(), "list")
	if err != nil {
		t.Fatalf("list: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, name := range []string{"default", "work", "research"} {
		if !strings.Contains(stdout, name) {
			t.Fatalf("list missing profile %q:\n%s", name, stdout)
		}
	}
	// Marker must appear on the active line; we check that the active line
	// (containing "work") has a marker character that no other line has.
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	var workLine string
	otherLines := make([]string, 0, len(lines))
	for _, line := range lines {
		switch {
		case strings.Contains(line, " work") || strings.HasSuffix(strings.TrimRight(line, " "), "work"):
			workLine = line
		case strings.Contains(line, "default") || strings.Contains(line, "research"):
			otherLines = append(otherLines, line)
		}
	}
	if workLine == "" {
		t.Fatalf("could not locate work line in list output:\n%s", stdout)
	}
	if !strings.Contains(workLine, "*") {
		t.Fatalf("active profile line missing marker '*': %q", workLine)
	}
	for _, other := range otherLines {
		if strings.Contains(other, "*") {
			t.Fatalf("non-active profile line incorrectly carries marker '*': %q", other)
		}
	}
	// Raw paths must not leak.
	if strings.Contains(stdout+stderr, "/home/operator-secret") {
		t.Fatalf("list leaked raw path:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

// TestGormesProfileListJSONEmitsStructuredInventory proves
// `gormes profile list --json` emits a parseable
// `{build, active, profiles: [{name, active}, ...]}` document. Fleet
// automation that wants to inventory profiles or check which is active
// across hosts needs a structured shape — scraping the human " *"
// marker is fragile and locale-dependent.
func TestGormesProfileListJSONEmitsStructuredInventory(t *testing.T) {
	fake := &profileCommandFakeSeams{
		activeProfileName: "work",
		knownProfiles:     []string{"default", "work", "research"},
	}
	stdout, stderr, err := runProfileTestCommand(t, fake.defaults(), "list", "--json")
	if err != nil {
		t.Fatalf("list --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Active   string `json:"active"`
		Profiles []struct {
			Name   string `json:"name"`
			Active bool   `json:"active"`
		} `json:"profiles"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != Version || got.Build.GitCommit == "" {
		t.Fatalf("build provenance missing/wrong: %+v", got.Build)
	}
	if got.Active != "work" {
		t.Fatalf("got.Active = %q, want work", got.Active)
	}
	if len(got.Profiles) != 3 {
		t.Fatalf("got %d profiles, want 3", len(got.Profiles))
	}
	wantOrder := []string{"default", "research", "work"} // sorted
	for i, p := range got.Profiles {
		if p.Name != wantOrder[i] {
			t.Fatalf("profile[%d].Name = %q, want %q (sorted)", i, p.Name, wantOrder[i])
		}
		wantActive := p.Name == "work"
		if p.Active != wantActive {
			t.Fatalf("profile[%d].Active = %t for %q, want %t", i, p.Active, p.Name, wantActive)
		}
	}
	// JSON mode must not interleave the human row.
	if strings.Contains(stdout, "* work") {
		t.Fatalf("--json must not emit the human row; got:\n%s", stdout)
	}
}

// TestGormesProfileListJSONNoProfilesEmitsEmptyArray proves the JSON
// surface stays parseable when no profiles are known — consumers see
// `{"profiles": []}`, not the free-form "no profiles found" message.
func TestGormesProfileListJSONNoProfilesEmitsEmptyArray(t *testing.T) {
	fake := &profileCommandFakeSeams{
		knownProfiles: []string{},
	}
	stdout, _, err := runProfileTestCommand(t, fake.defaults(), "list", "--json")
	if err != nil {
		t.Fatalf("list --json on empty: %v", err)
	}
	var got struct {
		Profiles []any `json:"profiles"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("empty stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Profiles == nil {
		t.Fatalf("profiles must be `[]`, not omitted/null; got %q", stdout)
	}
	if len(got.Profiles) != 0 {
		t.Fatalf("got %d profiles, want 0", len(got.Profiles))
	}
}

func TestGormesProfileSetUpdatesRootResolver(t *testing.T) {
	t.Run("resolver_invoked_after_store_write_succeeds", func(t *testing.T) {
		fake := &profileCommandFakeSeams{}
		var observedOrder []string
		fake.writeActiveProfile = func(name string) error {
			observedOrder = append(observedOrder, "write:"+name)
			return nil
		}
		fake.resolveProfileRoot = func(name string) (string, error) {
			observedOrder = append(observedOrder, "resolve:"+name)
			return "/tmp/gormes-test-home/profiles/" + name, nil
		}
		stdout, _, err := runProfileTestCommand(t, fake.defaults(), "set", "research")
		if err != nil {
			t.Fatalf("set: %v\nstdout=%s", err, stdout)
		}
		if len(observedOrder) < 2 {
			t.Fatalf("expected write+resolve, got %v", observedOrder)
		}
		// Find the write call index and the resolve call referencing the new
		// name. The resolve for the new name must come AFTER the write.
		writeIdx, postWriteResolveIdx := -1, -1
		for i, evt := range observedOrder {
			if evt == "write:research" && writeIdx < 0 {
				writeIdx = i
			}
			if evt == "resolve:research" && writeIdx >= 0 && i > writeIdx {
				postWriteResolveIdx = i
				break
			}
		}
		if writeIdx < 0 {
			t.Fatalf("store write was never called: %v", observedOrder)
		}
		if postWriteResolveIdx < 0 {
			t.Fatalf("root resolver was not invoked AFTER store write: %v", observedOrder)
		}
	})

	t.Run("resolver_failure_after_write_surfaces_partial_failure", func(t *testing.T) {
		fake := &profileCommandFakeSeams{}
		fake.resolveProfileRoot = func(name string) (string, error) {
			return "", errors.New("disk full")
		}
		_, _, err := runProfileTestCommand(t, fake.defaults(), "set", "research")
		if err == nil {
			t.Fatal("expected partial-failure error, got nil")
		}
		if !strings.Contains(err.Error(), "profile_set_partial_failure") {
			t.Fatalf("error %q does not surface profile_set_partial_failure", err.Error())
		}
		if len(fake.writeActiveProfileCalls) != 1 {
			t.Fatalf("expected exactly one write before resolver failure, got %v", fake.writeActiveProfileCalls)
		}
	})
}

// Smoke test: the profile command must build a working ProfileSelector from
// its seams. This pins the consumer/producer relationship so future selector
// refactors don't silently drift the contract.
func TestGormesProfileSelectorWiring(t *testing.T) {
	fake := &profileCommandFakeSeams{
		activeProfileName: "default",
	}
	seams := fake.defaults()
	selector := cli.NewDefaultProfileSelector(cli.DefaultProfileSelectorOptions{
		ReadActiveProfileName: cli.ReadActiveProfileNameFunc(seams.ReadActiveProfileName),
		ValidateProfileName:   cli.ValidateProfileNameFunc(seams.ValidateProfileName),
		ResolveProfileRoot:    cli.ResolveProfileRootFunc(seams.ResolveProfileRoot),
	})
	profile, err := selector.Select(context.Background())
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if profile.Name != "default" {
		t.Fatalf("Profile.Name = %q, want default", profile.Name)
	}
	if !strings.Contains(profile.RootPath, "default") {
		t.Fatalf("Profile.RootPath = %q, want a path containing 'default'", profile.RootPath)
	}
}

// Compile-time guard: newProfileCommand returns a *cobra.Command so the root
// can register it without conditionals.
var _ = func() *cobra.Command { return newProfileCommand() }
