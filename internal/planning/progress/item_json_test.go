package progress

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestItemModuleRoundTripUsesTypedFieldOrder(t *testing.T) {
	var it Item
	if err := json.Unmarshal([]byte(`{"name":"x","status":"planned","module":"fleet","trust_class":["system"]}`), &it); err != nil {
		t.Fatalf("Unmarshal item: %v", err)
	}
	if it.Module != "fleet" {
		t.Fatalf("Module = %q, want fleet", it.Module)
	}
	if _, ok := it.Extra["module"]; ok {
		t.Fatal("module must not be preserved as an extra field")
	}
	out, err := json.Marshal(it)
	if err != nil {
		t.Fatalf("Marshal item: %v", err)
	}
	if strings.Index(string(out), `"module"`) > strings.Index(string(out), `"trust_class"`) {
		t.Fatalf("module must marshal in the typed field order, got %s", out)
	}
}
