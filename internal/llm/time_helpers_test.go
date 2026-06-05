package llm

import (
	"os"
	"strings"
	"testing"
)

func TestGetTimezone_FromGormesEnv(t *testing.T) {
	t.Setenv("GORMES_TIMEZONE", "America/New_York")
	os.Unsetenv("HERMES_TIMEZONE")
	loc := GetTimezone()
	if loc == nil {
		t.Fatal("GetTimezone() = nil, want America/New_York")
	}
	if loc.String() != "America/New_York" {
		t.Fatalf("GetTimezone() = %s, want America/New_York", loc.String())
	}
}

func TestGetTimezone_FromHermesEnv(t *testing.T) {
	os.Unsetenv("GORMES_TIMEZONE")
	t.Setenv("HERMES_TIMEZONE", "Europe/London")
	loc := GetTimezone()
	if loc == nil {
		t.Fatal("GetTimezone() = nil, want Europe/London")
	}
	if !strings.Contains(loc.String(), "Europe/London") {
		t.Fatalf("GetTimezone() = %s, want Europe/London", loc.String())
	}
}

func TestGetTimezone_EmptyReturnsNil(t *testing.T) {
	os.Unsetenv("GORMES_TIMEZONE")
	os.Unsetenv("HERMES_TIMEZONE")
	if loc := GetTimezone(); loc != nil {
		t.Fatalf("GetTimezone() = %v, want nil when no timezone set", loc)
	}
}

func TestGetTimezone_InvalidZoneReturnsNil(t *testing.T) {
	t.Setenv("GORMES_TIMEZONE", "NotAReal/Timezone")
	loc := GetTimezone()
	if loc != nil {
		t.Fatalf("GetTimezone() = %v, want nil for invalid zone name", loc)
	}
}

func TestNow_WithTimezone(t *testing.T) {
	t.Setenv("GORMES_TIMEZONE", "Asia/Tokyo")
	now := Now()
	// The zone should be JST
	zone, _ := now.Zone()
	if zone != "JST" {
		t.Fatalf("Now().Zone() = %s, want JST with Asia/Tokyo timezone", zone)
	}
}

func TestNow_WithoutTimezone(t *testing.T) {
	os.Unsetenv("GORMES_TIMEZONE")
	os.Unsetenv("HERMES_TIMEZONE")
	now := Now()
	if now.IsZero() {
		t.Fatal("Now() returned zero time")
	}
}

func TestGormesTimezonePrecedesHermes(t *testing.T) {
	t.Setenv("GORMES_TIMEZONE", "US/Pacific")
	t.Setenv("HERMES_TIMEZONE", "Europe/Paris")
	loc := GetTimezone()
	if loc == nil {
		t.Fatal("GetTimezone() = nil, want US/Pacific")
	}
	if !strings.Contains(loc.String(), "US/Pacific") && !strings.Contains(loc.String(), "America/Los_Angeles") {
		t.Fatalf("GetTimezone() = %s, want US/Pacific (GORMES_TIMEZONE should win over HERMES_TIMEZONE)", loc.String())
	}
}

func init() {
	os.Unsetenv("GORMES_TIMEZONE")
	os.Unsetenv("HERMES_TIMEZONE")
}
