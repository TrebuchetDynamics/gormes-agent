package sandbox

import (
	"testing"
)

func TestVirtualPathResolver_ResolveWorkspace(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	hostPath, err := r.Resolve("/mnt/user-data/workspace/myfile.txt")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	expected := "/host/data/workspace/myfile.txt"
	if hostPath != expected {
		t.Errorf("expected %q, got %q", expected, hostPath)
	}
}

func TestVirtualPathResolver_ResolveUploads(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	hostPath, err := r.Resolve("/mnt/user-data/uploads/image.png")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	expected := "/host/data/uploads/image.png"
	if hostPath != expected {
		t.Errorf("expected %q, got %q", expected, hostPath)
	}
}

func TestVirtualPathResolver_ResolveOutputs(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	hostPath, err := r.Resolve("/mnt/user-data/outputs/result.json")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	expected := "/host/data/outputs/result.json"
	if hostPath != expected {
		t.Errorf("expected %q, got %q", expected, hostPath)
	}
}

func TestVirtualPathResolver_RejectsPathTraversal(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	traversalPaths := []string{
		"/mnt/user-data/workspace/../../../etc/passwd",
		"/mnt/user-data/../../etc/shadow",
		"/mnt/user-data/workspace/..\\..\\..\\etc\\passwd",
	}

	for _, p := range traversalPaths {
		_, err := r.Resolve(p)
		if err == nil {
			t.Errorf("expected error for path traversal: %q", p)
		}
	}
}

func TestVirtualPathResolver_RejectsOutsideVirtualRoot(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	outsidePaths := []string{
		"/etc/passwd",
		"/tmp/foo",
		"/var/log/syslog",
		"relative/path",
	}

	for _, p := range outsidePaths {
		_, err := r.Resolve(p)
		if err == nil {
			t.Errorf("expected error for path outside virtual root: %q", p)
		}
	}
}

func TestVirtualPathResolver_HostToVirtual(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	virtualPath, err := r.HostToVirtual("/host/data/workspace/myfile.txt")
	if err != nil {
		t.Fatalf("HostToVirtual failed: %v", err)
	}

	expected := "/mnt/user-data/workspace/myfile.txt"
	if virtualPath != expected {
		t.Errorf("expected %q, got %q", expected, virtualPath)
	}
}

func TestVirtualPathResolver_HostToVirtual_RejectsOutside(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	_, err := r.HostToVirtual("/etc/passwd")
	if err == nil {
		t.Error("expected error for path outside host root")
	}
}

func TestVirtualPathResolver_MaskOutput_ReplacesHostPaths(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	input := "/host/data/workspace/myfile.txt"
	want := "/mnt/user-data/workspace/myfile.txt"
	got := r.MaskOutput(input)
	if got != want {
		t.Errorf("MaskOutput(%q) = %q, want %q", input, got, want)
	}
}

func TestVirtualPathResolver_MaskOutput_MultiplePaths(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	input := "Wrote /host/data/workspace/a.go and /host/data/outputs/b.json"
	want := "Wrote /mnt/user-data/workspace/a.go and /mnt/user-data/outputs/b.json"
	got := r.MaskOutput(input)
	if got != want {
		t.Errorf("MaskOutput(%q) = %q, want %q", input, got, want)
	}
}

func TestVirtualPathResolver_MaskOutput_LeavesNonHostPaths(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	input := "Found at /etc/passwd and /usr/local/bin"
	got := r.MaskOutput(input)
	if got != input {
		t.Errorf("MaskOutput should leave non-host paths unchanged, got %q", got)
	}
}

func TestVirtualPathResolver_MaskOutput_EmptyString(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	got := r.MaskOutput("")
	if got != "" {
		t.Errorf("MaskOutput(\"\") = %q, want \"\"", got)
	}
}

func TestVirtualPathResolver_MaskOutput_PathWithinToolOutput(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	input := "Tool completed. File saved to /host/data/workspace/result.csv (42 bytes)"
	want := "Tool completed. File saved to /mnt/user-data/workspace/result.csv (42 bytes)"
	got := r.MaskOutput(input)
	if got != want {
		t.Errorf("MaskOutput(%q) = %q, want %q", input, got, want)
	}
}

func TestVirtualPathResolver_PathFamily(t *testing.T) {
	r := NewVirtualPathResolver("/mnt/user-data", "/host/data")

	// Workspace should be read-write
	family, err := r.PathFamily("/mnt/user-data/workspace/foo")
	if err != nil {
		t.Fatalf("PathFamily failed: %v", err)
	}
	if family != PathFamilyReadWrite {
		t.Errorf("expected read-write for workspace, got %v", family)
	}

	// Uploads should be read-write
	family, err = r.PathFamily("/mnt/user-data/uploads/foo")
	if err != nil {
		t.Fatalf("PathFamily failed: %v", err)
	}
	if family != PathFamilyReadWrite {
		t.Errorf("expected read-write for uploads, got %v", family)
	}

	// Outputs should be read-write
	family, err = r.PathFamily("/mnt/user-data/outputs/foo")
	if err != nil {
		t.Fatalf("PathFamily failed: %v", err)
	}
	if family != PathFamilyReadWrite {
		t.Errorf("expected read-write for outputs, got %v", family)
	}
}
