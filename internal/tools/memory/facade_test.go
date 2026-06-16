package memory

import "testing"

func TestMemoryPackageFacadePreservesPublicSurface(t *testing.T) {
	if MemoryToolName != "memory" {
		t.Fatalf("MemoryToolName = %q, want memory", MemoryToolName)
	}
	if NewMemoryTool(MemoryToolConfig{MemoryDir: t.TempDir()}) == nil {
		t.Fatal("NewMemoryTool returned nil")
	}
	if _, ok := MemoryToolOperationSpec(StoreMemoryToolName); !ok {
		t.Fatalf("MemoryToolOperationSpec(%q) missing through facade", StoreMemoryToolName)
	}
	if len(MemoryToolDescriptors()) == 0 || len(MemoryToolOperationSpecs()) == 0 {
		t.Fatal("memory descriptor facade returned empty catalog")
	}
}
