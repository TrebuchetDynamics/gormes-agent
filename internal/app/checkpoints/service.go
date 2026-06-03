package checkpoints

import (
	"fmt"
	"time"
)

// FormatBytes mirrors Hermes' checkpoints _fmt_bytes helper.
func FormatBytes(n int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(n)
	if n < 0 {
		size = 0
	}
	for _, unit := range units {
		if size < 1024 || unit == units[len(units)-1] {
			if unit == "B" {
				return fmt.Sprintf("%d B", int64(size))
			}
			return fmt.Sprintf("%.1f %s", size, unit)
		}
		size /= 1024
	}
	return fmt.Sprintf("%.1f TB", size)
}

// Ago returns the age duration for checkpoint timestamps, with -1 for unknown timestamps.
func Ago(now, t time.Time) time.Duration {
	if t.IsZero() {
		return -1
	}
	return now.Sub(t)
}

// FormatAge mirrors Hermes' checkpoints _fmt_age helper: Xs/m/h/d ago, "now", or "—".
func FormatAge(d time.Duration) string {
	if d < 0 {
		return "—"
	}
	if d < time.Second {
		return "now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}
