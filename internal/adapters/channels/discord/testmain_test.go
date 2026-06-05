package discord

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain isolates GORMES_HOME so the discord package's tests can
// never persist Discord thread fixtures to the developer's real
// `~/.gormes/discord_threads.json`. Observed in production: an
// operator who ran `./install.sh --local` (which builds the binary —
// `go build` doesn't run tests) and then later ran the test suite
// found their fresh-install `~/.gormes/discord_threads.json` seeded
// with `["thread-99","thread-200","thread-1","thread-seed"]` —
// exactly the IDs from `attachment_download_test.go`,
// `session_source_metadata_test.go`, `slash_commands_test.go`, and
// `thread_command_test.go`.
//
// Root cause: many of those tests construct Bot via `New(Config{...},
// ms, nil)` without setting `cfg.ThreadStatePath`, so the tracker
// inside the Bot falls back to `defaultDiscordThreadParticipationPath()`
// which resolves to `~/.gormes/discord_threads.json` whenever
// GORMES_HOME is unset. Calling `Mark()` from those tests then
// persists the fixture IDs into the operator's real home.
//
// Setting GORMES_HOME at the package boundary fixes ALL existing AND
// future discord tests with one defensive guard, instead of asking
// every test author to remember t.Setenv at every New() callsite.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "discord-test-home-")
	if err != nil {
		// Fall back to running anyway — the polluting tests will at
		// worst keep doing what they did before. We never fail the
		// process here because tests were running fine sans isolation.
		os.Exit(m.Run())
	}
	prev, hadPrev := os.LookupEnv("GORMES_HOME")
	os.Setenv("GORMES_HOME", tmp)
	code := m.Run()
	if hadPrev {
		os.Setenv("GORMES_HOME", prev)
	} else {
		os.Unsetenv("GORMES_HOME")
	}
	os.RemoveAll(tmp)
	os.Exit(code)
}

// TestThreadTracker_DoesNotPersistToOperatorHome is the regression
// fence: even with no explicit ThreadStatePath and no per-test
// t.Setenv, Mark() must not touch
// `${REAL_HOME}/.gormes/discord_threads.json`. The TestMain above
// makes that true; this test proves it.
func TestThreadTracker_DoesNotPersistToOperatorHome(t *testing.T) {
	realHome, err := os.UserHomeDir()
	if err != nil || realHome == "" {
		t.Skip("could not resolve operator home; skipping pollution fence")
	}
	realPath := filepath.Join(realHome, ".gormes", "discord_threads.json")

	// Snapshot whatever's there now so we can compare after.
	before, _ := os.ReadFile(realPath)

	tracker := NewThreadParticipationTracker(ThreadParticipationOptions{})
	if _, err := tracker.Mark("pollution-fence-sentinel"); err != nil {
		t.Fatalf("Mark: %v", err)
	}

	after, _ := os.ReadFile(realPath)
	if string(before) != string(after) {
		t.Fatalf("operator's real %s changed after Mark — TestMain isolation broke; before=%q after=%q", realPath, before, after)
	}
	if strings.Contains(string(after), "pollution-fence-sentinel") {
		t.Fatalf("pollution fixture leaked into %s: %s", realPath, after)
	}
}
