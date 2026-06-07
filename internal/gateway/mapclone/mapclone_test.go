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
