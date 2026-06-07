package mapclone

import "testing"

func TestStringStringClonesAndPreservesNil(t *testing.T) {
	if got := StringString(nil); got != nil {
		t.Fatalf("StringString(nil) = %#v, want nil", got)
	}

	input := map[string]string{"source": "original"}
	got := StringString(input)
	got["source"] = "clone"
	got["added"] = "value"

	if input["source"] != "original" {
		t.Fatalf("input source = %q, want original", input["source"])
	}
	if _, ok := input["added"]; ok {
		t.Fatalf("input gained clone-only key: %#v", input)
	}
}

func TestOrEmptyClonersCloneAndConvertNilToEmpty(t *testing.T) {
	if got := StringStringOrEmpty(nil); got == nil || len(got) != 0 {
		t.Fatalf("StringStringOrEmpty(nil) = %#v, want empty non-nil map", got)
	}
	if got := StringBoolOrEmpty(nil); got == nil || len(got) != 0 {
		t.Fatalf("StringBoolOrEmpty(nil) = %#v, want empty non-nil map", got)
	}
	if got := NestedStringBoolOrEmpty(nil); got == nil || len(got) != 0 {
		t.Fatalf("NestedStringBoolOrEmpty(nil) = %#v, want empty non-nil map", got)
	}

	strings := map[string]string{"source": "original"}
	stringClone := StringStringOrEmpty(strings)
	stringClone["source"] = "clone"
	if strings["source"] != "original" {
		t.Fatalf("string input changed with clone mutation: %#v", strings)
	}

	bools := map[string]bool{"enabled": true}
	boolClone := StringBoolOrEmpty(bools)
	boolClone["enabled"] = false
	if !bools["enabled"] {
		t.Fatalf("bool input changed with clone mutation: %#v", bools)
	}

	nested := map[string]map[string]bool{"telegram": {"u1": true}}
	nestedClone := NestedStringBoolOrEmpty(nested)
	nestedClone["telegram"]["u1"] = false
	if !nested["telegram"]["u1"] {
		t.Fatalf("nested input changed with clone mutation: %#v", nested)
	}
}
