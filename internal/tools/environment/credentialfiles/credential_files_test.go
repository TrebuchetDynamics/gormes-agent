package credentialfiles_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/environment/credentialfiles"
)

func makeHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return full
}

func TestRegistry_RegisterFoundFile(t *testing.T) {
	home := makeHome(t)
	writeFile(t, home, "google_token.json", `{"token":"abc"}`)
	r := credentialfiles.NewRegistry(home, nil)
	ok, err := r.Register("google_token.json", "/root/.gormes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for existing file")
	}
	mounts := r.Mounts()
	if len(mounts) != 1 {
		t.Fatalf("want 1 mount, got %d", len(mounts))
	}
	m := mounts[0]
	if !strings.HasSuffix(m.ContainerPath, "/google_token.json") {
		t.Errorf("unexpected container path: %q", m.ContainerPath)
	}
	if !m.ReadOnly {
		t.Error("expected read-only mount")
	}
	if m.HostPath == "" {
		t.Error("HostPath must not be empty")
	}
}

func TestRegistry_RegisterMissingFile(t *testing.T) {
	home := makeHome(t)
	r := credentialfiles.NewRegistry(home, nil)
	ok, err := r.Register("no_such_file.json", "/root/.gormes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for missing file")
	}
	if r.Len() != 0 {
		t.Fatalf("expected empty registry, got %d entries", r.Len())
	}
}

func TestRegistry_RejectsAbsolutePath(t *testing.T) {
	home := makeHome(t)
	r := credentialfiles.NewRegistry(home, nil)
	ok, err := r.Register("/etc/passwd", "/root/.gormes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for absolute path")
	}
}

func TestRegistry_RejectsTraversal(t *testing.T) {
	home := makeHome(t)
	// Create a file outside home that could be accessed via traversal.
	outsideDir := t.TempDir()
	writeFile(t, outsideDir, "secret.txt", "secret")
	r := credentialfiles.NewRegistry(home, nil)
	ok, _ := r.Register("../../../etc/passwd", "/root/.gormes")
	if ok {
		t.Fatal("traversal path should be rejected")
	}
}

func TestRegistry_RegisterMany(t *testing.T) {
	home := makeHome(t)
	writeFile(t, home, "a.json", "{}")
	writeFile(t, home, "b.json", "{}")
	r := credentialfiles.NewRegistry(home, nil)
	missing := r.RegisterMany([]string{"a.json", "b.json", "missing.json"}, "/root/.gormes")
	if len(missing) != 1 || missing[0] != "missing.json" {
		t.Fatalf("unexpected missing list: %v", missing)
	}
	if r.Len() != 2 {
		t.Fatalf("expected 2 registered, got %d", r.Len())
	}
}

func TestRegistry_Clear(t *testing.T) {
	home := makeHome(t)
	writeFile(t, home, "tok.json", "{}")
	r := credentialfiles.NewRegistry(home, nil)
	r.Register("tok.json", "/root/.gormes")
	if r.Len() != 1 {
		t.Fatal("expected 1 entry before clear")
	}
	r.Clear()
	if r.Len() != 0 {
		t.Fatal("expected 0 entries after clear")
	}
}

func TestRegistry_MountsStableOrder(t *testing.T) {
	home := makeHome(t)
	for _, name := range []string{"z.json", "a.json", "m.json"} {
		writeFile(t, home, name, "{}")
	}
	r := credentialfiles.NewRegistry(home, nil)
	r.Register("z.json", "/root/.gormes")
	r.Register("a.json", "/root/.gormes")
	r.Register("m.json", "/root/.gormes")
	mounts := r.Mounts()
	if len(mounts) != 3 {
		t.Fatalf("expected 3, got %d", len(mounts))
	}
	if !strings.HasSuffix(mounts[0].ContainerPath, "/a.json") {
		t.Errorf("first mount should be a.json, got %q", mounts[0].ContainerPath)
	}
	if !strings.HasSuffix(mounts[2].ContainerPath, "/z.json") {
		t.Errorf("last mount should be z.json, got %q", mounts[2].ContainerPath)
	}
}

func TestRegistry_ConfiguredPathsPreRegistered(t *testing.T) {
	home := makeHome(t)
	writeFile(t, home, "config_cred.json", "{}")
	r := credentialfiles.NewRegistry(home, []string{"config_cred.json"})
	if r.Len() != 1 {
		t.Fatalf("expected 1 pre-registered entry from config, got %d", r.Len())
	}
}

func TestSkillsDirectoryMount_Present(t *testing.T) {
	home := makeHome(t)
	if err := os.MkdirAll(filepath.Join(home, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := credentialfiles.SkillsDirectoryMount(home, "")
	if m == nil {
		t.Fatal("expected non-nil mount when skills dir exists")
	}
	if !strings.HasSuffix(m.HostPath, "/skills") {
		t.Errorf("unexpected host path: %q", m.HostPath)
	}
	if m.ContainerPath != "/root/.gormes/skills" {
		t.Errorf("unexpected container path: %q", m.ContainerPath)
	}
	if !m.ReadOnly {
		t.Error("expected read-only mount")
	}
}

func TestSkillsDirectoryMount_Absent(t *testing.T) {
	home := makeHome(t)
	m := credentialfiles.SkillsDirectoryMount(home, "")
	if m != nil {
		t.Fatalf("expected nil mount when skills dir absent, got %+v", m)
	}
}

func TestIterSkillsFiles(t *testing.T) {
	home := makeHome(t)
	skillsDir := filepath.Join(home, "skills")
	os.MkdirAll(skillsDir, 0o755)
	writeFile(t, skillsDir, "foo/SKILL.md", "# foo")
	writeFile(t, skillsDir, "bar/SKILL.md", "# bar")

	var visited []string
	err := credentialfiles.IterSkillsFiles(home, "", func(hostPath, containerPath string) {
		visited = append(visited, containerPath)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(visited) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(visited), visited)
	}
	for _, p := range visited {
		if !strings.HasPrefix(p, "/root/.gormes/skills/") {
			t.Errorf("unexpected container path: %q", p)
		}
	}
}

func TestIterSkillsFiles_EmptyDir(t *testing.T) {
	home := makeHome(t)
	// No skills dir — walk should return without error.
	var count int
	err := credentialfiles.IterSkillsFiles(home, "", func(_, _ string) { count++ })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 files, got %d", count)
	}
}
