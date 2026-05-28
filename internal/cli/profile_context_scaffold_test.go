package cli

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/TrebuchetDynamics/gormes-agent/internal/agenttemplate"
)

func TestMaterializeMainProfileContextScaffoldSeedsDefaultFiles(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")

	result, err := MaterializeMainProfileContextScaffold(ProfileContextScaffoldOptions{BaseHome: base})
	if err != nil {
		t.Fatalf("MaterializeMainProfileContextScaffold: %v", err)
	}

	wantRoot := filepath.Join(base, "profiles", "main")
	if result.ProfileName != "main" || result.Root != wantRoot {
		t.Fatalf("result = %+v, want main root %s", result, wantRoot)
	}
	if len(result.Templates.Files) == 0 {
		t.Fatalf("result templates empty: %+v", result)
	}
	if result.MemoryDB.Path != "memory.db" || result.MemoryDB.Action != agenttemplate.ActionCreate {
		t.Fatalf("memory db result = %+v, want create memory.db", result.MemoryDB)
	}
	for _, rel := range []string{
		"SOUL.md",
		"AGENTS.md",
		"IDENTITY.md",
		"TOOLS.md",
		filepath.Join("memory", "USER.md"),
		filepath.Join("memory", "MEMORY.md"),
	} {
		body, err := os.ReadFile(filepath.Join(wantRoot, rel))
		if err != nil {
			t.Fatalf("read seeded %s: %v", rel, err)
		}
		if len(body) == 0 {
			t.Fatalf("seeded %s is empty", rel)
		}
	}
	assertMemoryDBBootstrapped(t, filepath.Join(wantRoot, "memory.db"))
}

func TestApplyProfileContextScaffoldPersonalizesFreshProfileIdentity(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")

	_, err := ApplyProfileContextScaffold(ProfileContextScaffoldOptions{BaseHome: base, ProfileName: "tulin_sage", DisplayName: "Tulin Sage"})
	if err != nil {
		t.Fatalf("ApplyProfileContextScaffold: %v", err)
	}
	root := filepath.Join(base, "profiles", "tulin_sage")
	for _, tc := range []struct {
		rel  string
		want string
	}{
		{rel: "SOUL.md", want: "- Profile ID: `tulin_sage`\n- Agent name: Tulin Sage"},
		{rel: "IDENTITY.md", want: "- Name: Tulin Sage\n- Profile ID: `tulin_sage`"},
	} {
		body, err := os.ReadFile(filepath.Join(root, tc.rel))
		if err != nil {
			t.Fatalf("read %s: %v", tc.rel, err)
		}
		if !strings.Contains(string(body), tc.want) {
			t.Fatalf("%s = %q, want to contain %q", tc.rel, string(body), tc.want)
		}
	}
}

func TestMaterializeMainProfileContextScaffoldDoesNotOverwriteExistingFiles(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	existingSoul := filepath.Join(base, "profiles", "main", "SOUL.md")
	if err := os.MkdirAll(filepath.Dir(existingSoul), 0o700); err != nil {
		t.Fatalf("mkdir existing main profile: %v", err)
	}
	if err := os.WriteFile(existingSoul, []byte("custom main persona\n"), 0o600); err != nil {
		t.Fatalf("write existing SOUL.md: %v", err)
	}

	result, err := MaterializeMainProfileContextScaffold(ProfileContextScaffoldOptions{BaseHome: base})
	if err != nil {
		t.Fatalf("MaterializeMainProfileContextScaffold: %v", err)
	}
	if got := actionByPath(result.Templates)["SOUL.md"]; got != agenttemplate.ActionSkip {
		t.Fatalf("SOUL.md action = %q, want %q", got, agenttemplate.ActionSkip)
	}
	body, err := os.ReadFile(existingSoul)
	if err != nil {
		t.Fatalf("read existing SOUL.md: %v", err)
	}
	if string(body) != "custom main persona\n" {
		t.Fatalf("existing SOUL.md overwritten: %q", string(body))
	}
	if got := actionByPath(result.Templates)[filepath.ToSlash(filepath.Join("memory", "USER.md"))]; got != agenttemplate.ActionCreate {
		t.Fatalf("memory/USER.md action = %q, want %q", got, agenttemplate.ActionCreate)
	}
	if result.MemoryDB.Action != agenttemplate.ActionCreate {
		t.Fatalf("memory.db action = %q, want %q", result.MemoryDB.Action, agenttemplate.ActionCreate)
	}
}

func TestApplyProfileContextScaffoldDryRunDoesNotCreateMainProfile(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	root := filepath.Join(base, "profiles", "main")

	result, err := ApplyProfileContextScaffold(ProfileContextScaffoldOptions{BaseHome: base, ProfileName: "main", DryRun: true})
	if err != nil {
		t.Fatalf("ApplyProfileContextScaffold dry-run: %v", err)
	}
	if result.Root != root {
		t.Fatalf("dry-run root = %q, want %q", result.Root, root)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("dry-run created root, stat err=%v", err)
	}
	if got := actionByPath(result.Templates)["SOUL.md"]; got != agenttemplate.ActionWouldCreate {
		t.Fatalf("SOUL.md dry-run action = %q, want %q", got, agenttemplate.ActionWouldCreate)
	}
	if result.MemoryDB.Action != agenttemplate.ActionWouldCreate {
		t.Fatalf("memory.db dry-run action = %q, want %q", result.MemoryDB.Action, agenttemplate.ActionWouldCreate)
	}
}

func assertMemoryDBBootstrapped(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open memory.db: %v", err)
	}
	defer db.Close()
	var version string
	if err := db.QueryRow(`SELECT v FROM schema_meta WHERE k = 'version'`).Scan(&version); err != nil {
		t.Fatalf("read memory.db schema version: %v", err)
	}
	if version == "" {
		t.Fatalf("memory.db schema version is empty")
	}
}

func actionByPath(result agenttemplate.WriteResult) map[string]agenttemplate.Action {
	out := make(map[string]agenttemplate.Action, len(result.Files))
	for _, file := range result.Files {
		out[file.Path] = file.Action
	}
	return out
}
