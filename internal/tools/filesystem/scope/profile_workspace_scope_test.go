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
		ProfileName:  "coder",
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
		{name: "profile env inside workspace", path: filepath.Join(profile, ".env"), base: project1, access: ProfileWorkspaceAccessRead, allowed: true},
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

func TestProfileWorkspaceScope_DefaultsToProfileRootAndDeniesHome(t *testing.T) {
	operatorHome := t.TempDir()
	profileRoot := filepath.Join(operatorHome, ".gormes", "profiles", "coder")
	siblingRoot := filepath.Join(operatorHome, ".gormes", "profiles", "researcher")
	for _, dir := range []string{profileRoot, siblingRoot, filepath.Join(operatorHome, "git", "gormes")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeFile(t, filepath.Join(profileRoot, "notes.md"), "profile workspace")
	writeFile(t, filepath.Join(siblingRoot, "notes.md"), "sibling")
	writeFile(t, filepath.Join(operatorHome, "git", "gormes", "repo.md"), "repo")

	scope, err := NewProfileWorkspaceScope(ProfileWorkspaceScopeOptions{
		ProfileName:  "coder",
		ProfileRoot:  profileRoot,
		OperatorHome: operatorHome,
	})
	if err != nil {
		t.Fatalf("NewProfileWorkspaceScope: %v", err)
	}
	if !scope.Configured() {
		t.Fatalf("Configured = false, want true for profile-root default policy")
	}
	if got := scope.DefaultRoot(); got != profileRoot {
		t.Fatalf("DefaultRoot = %q, want active profile root %q", got, profileRoot)
	}
	allowed := scope.Resolve("notes.md", profileRoot, ProfileWorkspaceAccessRead)
	if !allowed.Allowed || allowed.Root != profileRoot || allowed.Relative != "notes.md" {
		t.Fatalf("profile notes decision = %#v, want allowed inside profile root", allowed)
	}
	for _, path := range []string{
		filepath.Join(siblingRoot, "notes.md"),
		filepath.Join(operatorHome, "git", "gormes", "repo.md"),
	} {
		decision := scope.Resolve(path, profileRoot, ProfileWorkspaceAccessRead)
		if decision.Allowed {
			t.Fatalf("Resolve(%q) allowed outside profile root: %#v", path, decision)
		}
		if decision.Evidence != ProfileWorkspaceScopeViolation {
			t.Fatalf("Resolve(%q) evidence = %q, want %q", path, decision.Evidence, ProfileWorkspaceScopeViolation)
		}
		if decision.Message != ProfileWorkspaceDeniedMessage {
			t.Fatalf("Resolve(%q) message = %q, want stable allow-list guidance", path, decision.Message)
		}
	}
}

func TestProfileWorkspaceScope_ExplicitAllowedPathExtendsProfileRoot(t *testing.T) {
	operatorHome := t.TempDir()
	profileRoot := filepath.Join(operatorHome, ".gormes", "profiles", "coder")
	repoRoot := filepath.Join(operatorHome, "git", "gormes")
	for _, dir := range []string{profileRoot, repoRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeFile(t, filepath.Join(repoRoot, "repo.md"), "repo")

	scope, err := NewProfileWorkspaceScope(ProfileWorkspaceScopeOptions{
		ProfileName:   "coder",
		ProfileRoot:   profileRoot,
		WorkspaceRoot: profileRoot,
		ProjectRoots:  []string{repoRoot},
		OperatorHome:  operatorHome,
	})
	if err != nil {
		t.Fatalf("NewProfileWorkspaceScope: %v", err)
	}
	decision := scope.Resolve(filepath.Join(repoRoot, "repo.md"), profileRoot, ProfileWorkspaceAccessRead)
	if !decision.Allowed || decision.Root != repoRoot || decision.Relative != "repo.md" {
		t.Fatalf("allowlisted repo decision = %#v, want allowed under explicit path", decision)
	}
}

func TestProfileWorkspaceScope_TildeUsesConfiguredOperatorHome(t *testing.T) {
	processHome := t.TempDir()
	operatorHome := t.TempDir()
	profileRoot := filepath.Join(operatorHome, ".gormes", "profiles", "coder")
	repoRoot := filepath.Join(operatorHome, "git", "gormes")
	sshRoot := filepath.Join(operatorHome, ".ssh")
	for _, dir := range []string{profileRoot, repoRoot, sshRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeFile(t, filepath.Join(repoRoot, "repo.md"), "repo")
	t.Setenv("HOME", processHome)

	scope, err := NewProfileWorkspaceScope(ProfileWorkspaceScopeOptions{
		ProfileName:  "coder",
		ProfileRoot:  profileRoot,
		ProjectRoots: []string{"~/git/gormes"},
		OperatorHome: operatorHome,
	})
	if err != nil {
		t.Fatalf("NewProfileWorkspaceScope: %v", err)
	}

	allowed := scope.Resolve("~/git/gormes/repo.md", profileRoot, ProfileWorkspaceAccessRead)
	if !allowed.Allowed || allowed.Root != repoRoot || allowed.Relative != "repo.md" {
		t.Fatalf("tilde allowlist decision = %#v, want repo under operator home", allowed)
	}
	denied := scope.Resolve("~/.ssh/id_rsa", profileRoot, ProfileWorkspaceAccessRead)
	if denied.Allowed {
		t.Fatalf("tilde home secret allowed: %#v", denied)
	}
	if want := filepath.Join(operatorHome, ".ssh", "id_rsa"); denied.Normalized != want {
		t.Fatalf("tilde normalized = %q, want operator home path %q", denied.Normalized, want)
	}
	if denied.Message != ProfileWorkspaceDeniedMessage {
		t.Fatalf("message = %q, want stable allow-list guidance", denied.Message)
	}
}

func TestProfileWorkspaceScopeRejectsUnsafeProfileNamesAndBlanketRoots(t *testing.T) {
	operatorHome := t.TempDir()
	profileRoot := filepath.Join(operatorHome, ".gormes", "profiles", "coder")
	if err := os.MkdirAll(profileRoot, 0o755); err != nil {
		t.Fatalf("mkdir profile: %v", err)
	}

	for _, name := range []string{"../coder", "coder/slash", ".", ".."} {
		t.Run("profile name "+name, func(t *testing.T) {
			_, err := NewProfileWorkspaceScope(ProfileWorkspaceScopeOptions{
				ProfileName:  name,
				ProfileRoot:  profileRoot,
				OperatorHome: operatorHome,
			})
			if err == nil {
				t.Fatalf("NewProfileWorkspaceScope accepted unsafe profile name %q", name)
			}
		})
	}

	for _, root := range []string{string(filepath.Separator), operatorHome} {
		t.Run("allowed root "+root, func(t *testing.T) {
			_, err := NewProfileWorkspaceScope(ProfileWorkspaceScopeOptions{
				ProfileName:  "coder",
				ProfileRoot:  profileRoot,
				ProjectRoots: []string{root},
				OperatorHome: operatorHome,
			})
			if err == nil {
				t.Fatalf("NewProfileWorkspaceScope accepted blanket allowed root %q", root)
			}
		})
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
