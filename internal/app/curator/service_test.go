package curator

import (
	"testing"
	"time"
)

func TestResolveSkillsRootUsesInjectedRoot(t *testing.T) {
	got := ResolveSkillsRoot(CommandDeps{SkillsRoot: func() string { return " /tmp/skills " }})
	if got != "/tmp/skills" {
		t.Fatalf("ResolveSkillsRoot = %q", got)
	}
}

func TestFormatTimestampBuckets(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	deps := CommandDeps{Now: func() time.Time { return now }}
	then := now.Add(-2 * time.Hour)
	if got := FormatTimestamp(&then, deps); got != "2h ago" {
		t.Fatalf("FormatTimestamp = %q", got)
	}
	if got := FormatTimestamp(nil, deps); got != "never" {
		t.Fatalf("nil FormatTimestamp = %q", got)
	}
}
