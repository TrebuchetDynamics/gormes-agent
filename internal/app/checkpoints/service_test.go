package checkpoints

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{-1, "0 B"},
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
	}
	for _, tc := range cases {
		if got := FormatBytes(tc.in); got != tc.want {
			t.Fatalf("FormatBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatAge(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{-1, "—"},
		{0, "now"},
		{5 * time.Second, "5s ago"},
		{2 * time.Minute, "2m ago"},
		{3 * time.Hour, "3h ago"},
		{49 * time.Hour, "2d ago"},
	}
	for _, tc := range cases {
		if got := FormatAge(tc.in); got != tc.want {
			t.Fatalf("FormatAge(%s) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAgoZeroTimestamp(t *testing.T) {
	if got := Ago(time.Now(), time.Time{}); got != -1 {
		t.Fatalf("Ago(zero) = %s, want -1", got)
	}
}
