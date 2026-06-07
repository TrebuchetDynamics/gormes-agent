package entry

import "testing"

func TestTargetDisplayNameSharesPlatformNamingPolicy(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		entry    Entry
		want     string
	}{
		{name: "discord guild channel", platform: " Discord ", entry: Entry{Name: " general ", Guild: " Sages ", Type: "channel"}, want: "#general"},
		{name: "non discord typed target", platform: "telegram", entry: Entry{Name: " Alice ", Type: " dm "}, want: "Alice (dm)"},
		{name: "plain target", platform: "slack", entry: Entry{Name: " ops "}, want: "ops"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TargetDisplayName(tt.platform, tt.entry); got != tt.want {
				t.Fatalf("TargetDisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}
