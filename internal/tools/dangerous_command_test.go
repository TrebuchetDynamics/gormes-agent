package tools

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestDetectDangerous(t *testing.T) {
	cases := []struct {
		name         string
		cmd          string
		wantMatched  bool
		wantContains string
	}{
		{
			name:         "recursive rm under tmp",
			cmd:          "rm -rf /tmp/x",
			wantMatched:  true,
			wantContains: "delete",
		},
		{
			name:         "git reset hard",
			cmd:          "git reset --hard",
			wantMatched:  true,
			wantContains: "git reset --hard",
		},
		{
			name:         "sql drop",
			cmd:          "DROP TABLE users",
			wantMatched:  true,
			wantContains: "sql drop",
		},
		{
			name:        "benign ls",
			cmd:         "ls",
			wantMatched: false,
		},
		{
			name:        "empty",
			cmd:         "",
			wantMatched: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			matched, desc := DetectDangerous(tc.cmd)
			if matched != tc.wantMatched {
				t.Fatalf("DetectDangerous(%q) matched = %v, want %v", tc.cmd, matched, tc.wantMatched)
			}
			if !tc.wantMatched {
				if desc != "" {
					t.Fatalf("DetectDangerous(%q) description = %q, want empty", tc.cmd, desc)
				}
				return
			}
			if desc == "" {
				t.Fatalf("DetectDangerous(%q) returned empty description", tc.cmd)
			}
			if !strings.Contains(strings.ToLower(desc), tc.wantContains) {
				t.Fatalf("DetectDangerous(%q) description = %q, want contains %q", tc.cmd, desc, tc.wantContains)
			}
		})
	}
}

func TestGuardCommandHardlineWins(t *testing.T) {
	result := GuardCommand("rm -rf /", "manual")
	if !result.Hardline {
		t.Fatalf("GuardCommand hardline = false, want true: %#v", result)
	}
	if result.Approved {
		t.Fatalf("GuardCommand approved = true, want false: %#v", result)
	}
	if result.ApprovalRequired {
		t.Fatalf("GuardCommand approval required = true, want false for hardline: %#v", result)
	}
	if !strings.Contains(strings.ToLower(result.Description), "root filesystem") {
		t.Fatalf("GuardCommand description = %q, want root filesystem", result.Description)
	}
	if got := result.Evidence["detector"]; got != "hardline" {
		t.Fatalf("GuardCommand evidence detector = %q, want hardline", got)
	}
}

func TestGuardCommandRecoverable(t *testing.T) {
	for _, cmd := range []string{"rm -rf /tmp/x", "git reset --hard"} {
		t.Run(cmd, func(t *testing.T) {
			result := GuardCommand(cmd, "manual")
			if result.Hardline {
				t.Fatalf("GuardCommand(%q) hardline = true, want false: %#v", cmd, result)
			}
			if result.Approved {
				t.Fatalf("GuardCommand(%q) approved = true, want false: %#v", cmd, result)
			}
			if !result.ApprovalRequired {
				t.Fatalf("GuardCommand(%q) approval required = false, want true: %#v", cmd, result)
			}
			if result.Description == "" {
				t.Fatalf("GuardCommand(%q) returned empty description", cmd)
			}
			if result.Operator != "manual" {
				t.Fatalf("GuardCommand(%q) operator = %q, want manual", cmd, result.Operator)
			}
		})
	}
}

func TestGuardCommandBenign(t *testing.T) {
	result := GuardCommand("ls", "manual")
	if !reflect.DeepEqual(result, BlockedResult{}) {
		t.Fatalf("GuardCommand benign = %#v, want zero value", result)
	}
}

func TestBlockedResultEvidenceFields(t *testing.T) {
	cmd := "git reset --hard"
	result := GuardCommand(cmd, "manual")
	if result.Evidence == nil {
		t.Fatalf("GuardCommand evidence is nil")
	}
	if got := result.Evidence["pattern_description"]; got != result.Description {
		t.Fatalf("pattern_description evidence = %q, want %q", got, result.Description)
	}
	if got := result.Evidence["command"]; got != cmd {
		t.Fatalf("command evidence = %q, want input command %q", got, cmd)
	}
}

func TestGuardCommandPure(t *testing.T) {
	src, err := os.ReadFile("dangerous_command.go")
	if err != nil {
		t.Fatalf("read dangerous_command.go: %v", err)
	}
	for _, forbidden := range []string{"os/exec", "net/http", "os.Open"} {
		if strings.Contains(string(src), forbidden) {
			t.Fatalf("dangerous_command.go contains side-effect API %q", forbidden)
		}
	}
}
