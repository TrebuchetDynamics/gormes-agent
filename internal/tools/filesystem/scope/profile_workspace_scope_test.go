package scope

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfileWorkspaceScope_AllowsProjectsAndOwnedProfileContent(t *testing.T) {
	root := t.TempDir()
	project1 := filepath.Join(root, "project1")
	project2 := filepath.Join(root, "project2")
	profilesRoot := filepath.Join(root, ".gormes", "profiles")
	profile := filepath.Join(profilesRoot, "coder")
	sibling := filepath.Join(profilesRoot, "researcher")
	for _, dir := range []string{
		project1,
		project2,
		filepath.Join(profile, "skills", "writer"),
		sibling,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeFile(t, filepath.Join(project1, "notes.md"), "p1")
	writeFile(t, filepath.Join(project2, "notes.md"), "p2")
	writeFile(t, filepath.Join(profile, "SOUL.md"), "identity")
	writeFile(t, filepath.Join(profile, "skills", "writer", "SKILL.md"), "skill")
	writeFile(t, filepath.Join(profile, ".env"), "OPENAI_API_KEY=secret")
	writeFile(t, filepath.Join(sibling, "SOUL.md"), "sibling")

	scope, err := NewProfileWorkspaceScope(ProfileWorkspaceScopeOptions{
		ProjectRoots: []string{project1, project2},
		ProfileRoot:  profile,
		OperatorHome: root,
	})
	if err != nil {
		t.Fatalf("NewProfileWorkspaceScope: %v", err)
	}

	cases := []struct {
		name    string
		path    string
		base    string
		access  ProfileWorkspaceAccess
		allowed bool
	}{
		{name: "relative project1", path: "notes.md", base: project1, access: ProfileWorkspaceAccessRead, allowed: true},
		{name: "absolute project2", path: filepath.Join(project2, "notes.md"), base: project1, access: ProfileWorkspaceAccessRead, allowed: true},
		{name: "profile soul", path: filepath.Join(profile, "SOUL.md"), base: project1, access: ProfileWorkspaceAccessWrite, allowed: true},
		{name: "profile identity", path: filepath.Join(profile, "IDENTITY.md"), base: project1, access: ProfileWorkspaceAccessWrite, allowed: true},
		{name: "profile skill", path: filepath.Join(profile, "skills", "writer", "SKILL.md"), base: project1, access: ProfileWorkspaceAccessWrite, allowed: true},
		{name: "profile env denied", path: filepath.Join(profile, ".env"), base: project1, access: ProfileWorkspaceAccessRead, allowed: false},
		{name: "sibling profile denied", path: filepath.Join(sibling, "SOUL.md"), base: project1, access: ProfileWorkspaceAccessRead, allowed: false},
		{name: "outside denied", path: filepath.Join(root, "outside.txt"), base: project1, access: ProfileWorkspaceAccessRead, allowed: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision := scope.Resolve(tc.path, tc.base, tc.access)
			if decision.Allowed != tc.allowed {
				t.Fatalf("Resolve allowed = %v, want %v: %#v", decision.Allowed, tc.allowed, decision)
			}
			if !tc.allowed && decision.Evidence != ProfileWorkspaceScopeViolation {
				t.Fatalf("denied evidence = %q, want %q: %#v", decision.Evidence, ProfileWorkspaceScopeViolation, decision)
			}
		})
	}
}

func TestProfileWorkspaceScope_BlocksSymlinkAndPrefixEscapes(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	projectSibling := filepath.Join(root, "project-other")
	outside := filepath.Join(root, "outside")
	for _, dir := range []string{project, projectSibling, outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeFile(t, filepath.Join(projectSibling, "secret.txt"), "prefix")
	writeFile(t, filepath.Join(outside, "secret.txt"), "outside")
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(project, "linked-secret.txt")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	scope, err := NewProfileWorkspaceScope(ProfileWorkspaceScopeOptions{
		ProjectRoots: []string{project},
		OperatorHome: root,
	})
	if err != nil {
		t.Fatalf("NewProfileWorkspaceScope: %v", err)
	}

	for _, path := range []string{
		filepath.Join(projectSibling, "secret.txt"),
		filepath.Join(project, "linked-secret.txt"),
	} {
		decision := scope.Resolve(path, project, ProfileWorkspaceAccessRead)
		if decision.Allowed {
			t.Fatalf("Resolve(%q) allowed symlink/prefix escape: %#v", path, decision)
		}
		if decision.Evidence != ProfileWorkspaceScopeViolation {
			t.Fatalf("Resolve(%q) evidence = %q, want %q", path, decision.Evidence, ProfileWorkspaceScopeViolation)
		}
	}
}

func TestProfileWorkspaceScope_EmptyListDefaultsToOperatorHome(t *testing.T) {
	operatorHome := t.TempDir()
	project := filepath.Join(operatorHome, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

	scope, err := NewProfileWorkspaceScope(ProfileWorkspaceScopeOptions{
		ProjectRoots: nil,
		OperatorHome: operatorHome,
	})
	if err != nil {
		t.Fatalf("NewProfileWorkspaceScope: %v", err)
	}
	if scope.Configured() {
		t.Fatalf("Configured = true, want false for empty agents.defaults.workspaces")
	}
	if got := scope.DefaultRoot(); got != operatorHome {
		t.Fatalf("DefaultRoot = %q, want operator home %q", got, operatorHome)
	}
	allowed := scope.Resolve(filepath.Join(project, "new.txt"), operatorHome, ProfileWorkspaceAccessWrite)
	if !allowed.Allowed {
		t.Fatalf("operator home project denied: %#v", allowed)
	}
	denied := scope.Resolve(filepath.Join(filepath.Dir(operatorHome), "outside.txt"), operatorHome, ProfileWorkspaceAccessRead)
	if denied.Allowed {
		t.Fatalf("outside operator home allowed: %#v", denied)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
