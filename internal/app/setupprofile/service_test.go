package setupprofile

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestProfileIDUsesMaterializedProfileRootName(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	if got := ProfileID(filepath.Join(base, "profiles", "work"), "main"); got != "work" {
		t.Fatalf("ProfileID named = %q, want work", got)
	}
	if got := ProfileID(filepath.Join(base, "home"), "main"); got != "main" {
		t.Fatalf("ProfileID default = %q, want main", got)
	}
}

func TestCredentialIDNormalizesChannel(t *testing.T) {
	if got := CredentialID(" Main ", "slack/app one"); got != "main-slack_app_one" {
		t.Fatalf("CredentialID = %q", got)
	}
}

func TestCompactStringsTrimsAndDeduplicates(t *testing.T) {
	got := CompactStrings([]string{"  a ", "", "b", "a"})
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CompactStrings = %#v, want %#v", got, want)
	}
}

func TestInt64StringsFormatsValues(t *testing.T) {
	got := Int64Strings([]int64{1, 42})
	want := []string{"1", "42"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Int64Strings = %#v, want %#v", got, want)
	}
}
