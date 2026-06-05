package profile

import (
	"strings"
	"testing"
)

// TestGormesProfileSet_RejectsUnknownProfile guards a UX bug found
// during install testing: `gormes profile set non-existent-profile`
// previously succeeded with exit 0 and silently wrote a marker
// pointing at a profile root that does not exist. Subsequent commands
// then either silently fell back to default (confusing) or failed
// with cryptic "no such file or directory" errors against a path
// the operator never knowingly created.
//
// New contract: profile set must fail when the requested name is not
// in ListKnownProfiles, with a hint listing the valid options.
// "main" is always known (defaultListKnownProfiles guarantees it),
// so the existing happy path stays green.
func TestGormesProfileSet_RejectsUnknownProfile(t *testing.T) {
	fake := &profileCommandFakeSeams{
		knownProfiles: []string{"main", "work"},
	}
	stdout, stderr, err := runProfileTestCommand(t, fake.defaults(), "set", "ghost")
	if err == nil {
		t.Fatalf("profile set on unknown name should error; stdout=%q stderr=%q", stdout, stderr)
	}
	combined := err.Error() + stderr
	if !strings.Contains(combined, "ghost") {
		t.Errorf("error must name the rejected profile %q; got %s", "ghost", combined)
	}
	// The error must mention at least one valid profile so operators
	// can self-correct without a separate `profile list` lookup.
	if !strings.Contains(combined, "main") {
		t.Errorf("error must list known profiles to guide the operator; got %s", combined)
	}
	// Must NOT have written the marker — the operator's previous
	// active profile stays in place when the requested name is bogus.
	if len(fake.writeActiveProfileCalls) > 0 {
		t.Errorf("profile set on unknown name must not call WriteActiveProfile; got calls %v", fake.writeActiveProfileCalls)
	}
}

// TestGormesProfileSet_AcceptsKnownProfile is the regression fence:
// the new validation must not break the happy path. Setting a known
// profile (e.g. "main" or any directory under ~/.gormes/profiles)
// must still succeed with exit 0 and write the marker.
func TestGormesProfileSet_AcceptsKnownProfile(t *testing.T) {
	fake := &profileCommandFakeSeams{
		knownProfiles: []string{"main", "work"},
	}
	stdout, stderr, err := runProfileTestCommand(t, fake.defaults(), "set", "work")
	if err != nil {
		t.Fatalf("profile set on known name failed: %v\nstdout=%s stderr=%s", err, stdout, stderr)
	}
	if got := len(fake.writeActiveProfileCalls); got != 1 || fake.writeActiveProfileCalls[0] != "work" {
		t.Errorf("WriteActiveProfile must be called once with %q; got %v", "work", fake.writeActiveProfileCalls)
	}
}

// TestGormesProfileSet_MainProfileAlwaysAccepted: even when the
// known list returned only "main" (fresh-install state), `profile
// set default` must succeed. Guards against a regression that locks
// operators out of the bootstrap profile.
func TestGormesProfileSet_MainProfileAlwaysAccepted(t *testing.T) {
	fake := &profileCommandFakeSeams{
		knownProfiles: []string{"main"},
	}
	stdout, stderr, err := runProfileTestCommand(t, fake.defaults(), "set", "main")
	if err != nil {
		t.Fatalf("profile set default must always succeed: %v\nstdout=%s stderr=%s", err, stdout, stderr)
	}
}
