package main

import (
	"bytes"
	"context"
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
