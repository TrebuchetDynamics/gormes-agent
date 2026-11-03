package navivox

import (
	"strings"
	"testing"
)

func TestNavivoxCommandPrintsDeprecationNotice(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	stdout, stderr, err := executeNavivoxCommandForTest(NewCommand(CommandOptions{VPNHostList: vpnhostList}))
	if err != nil {
		t.Fatalf("navivox: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, NavivoxDeprecationNotice) {
		t.Fatalf("navivox command missing deprecation notice:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestNavivoxPairHelpStaysInvocableWhileDeprecated(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	stdout, stderr, err := executeNavivoxCommandForTest(NewCommand(CommandOptions{VPNHostList: vpnhostList}), "pair", "--help")
	if err != nil {
		t.Fatalf("navivox pair --help: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"terminal handoff", "compact QR", "gormes navivox pair"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("navivox pair help missing %q:\n%s", want, stdout)
		}
	}
}
