package restore

import (
	"testing"
	"time"
)

func TestFormatSize(t *testing.T) {
	cases := []struct {
		bytes int64
		want  string
	}{
		{12, "12B"},
		{1536, "1.5KB"},
		{2 * 1024 * 1024, "2.0MB"},
		{3 * 1024 * 1024 * 1024, "3.0GB"},
	}
	for _, tt := range cases {
		if got := FormatSize(tt.bytes); got != tt.want {
			t.Fatalf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestFormatAge(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "just now"},
		{3 * time.Minute, "3m ago"},
		{2 * time.Hour, "2h ago"},
		{5 * 24 * time.Hour, "5d ago"},
	}
	for _, tt := range cases {
		if got := FormatAge(tt.d); got != tt.want {
			t.Fatalf("FormatAge(%s) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
