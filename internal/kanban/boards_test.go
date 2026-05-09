package kanban

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBoardSlugValidation(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{name: "default", slug: "default", wantErr: false},
		{name: "single char", slug: "a", wantErr: false},
		{name: "hyphen mid", slug: "atm10-server", wantErr: false},
		{name: "underscore mid", slug: "proj_1", wantErr: false},
		{name: "numbers", slug: "1234-project", wantErr: false},
		{name: "long but valid", slug: "very-long-but-still-ok-slug-with-hyphens-and-numbers-1234", wantErr: false},
		{name: "empty", slug: "", wantErr: true},
		{name: "leading hyphen", slug: "-leading-hyphen", wantErr: true},
		{name: "leading underscore", slug: "_leading_underscore", wantErr: true},
		{name: "trailing hyphen", slug: "trailing-", wantErr: true},
		{name: "trailing underscore", slug: "trailing_", wantErr: true},
		{name: "uppercase", slug: "UPPERCASE", wantErr: true},
		{name: "spaces", slug: "has spaces", wantErr: true},
		{name: "special chars", slug: "has/slash", wantErr: true},
		{name: "dot segment", slug: "..", wantErr: true},
		{name: "dot segment nested", slug: "a/../b", wantErr: true},
		{name: "too long", slug: strings.Repeat("a", 65), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBoardSlug(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBoardSlug(%q) error=%v, wantErr=%v", tt.slug, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeBoardSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  default  ", "default"},
		{" atm10-server ", "atm10-server"},
		{"DEFAULT", "default"},
		{"HELLO-WORLD", "hello-world"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeBoardSlug(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeBoardSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBoardRegistry_DefaultBoardExists(t *testing.T) {
	root := t.TempDir()
	reg := NewBoardRegistry(root)

	boards, err := reg.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(boards) != 0 {
		t.Fatalf("fresh registry should be empty, got %d boards", len(boards))
	}

	// The default board is implicit; it exists when we resolve its path.
	defPath := reg.BoardPath("default")
	if defPath == "" {
		t.Fatal("BoardPath(default) returned empty")
	}
	expected := filepath.Join(root, "kanban.db")
	if defPath != expected {
		t.Errorf("default board path = %q, want %q", defPath, expected)
	}
}

func TestBoardRegistry_CreateListSwitchRemove(t *testing.T) {
	root := t.TempDir()
	reg := NewBoardRegistry(root)

	// Create a named board.
	if err := reg.Create("atm10-server"); err != nil {
		t.Fatalf("Create(atm10-server): %v", err)
	}

	// Create another board.
	if err := reg.Create("hermes-agent"); err != nil {
		t.Fatalf("Create(hermes-agent): %v", err)
	}

	// List boards (default is not in the registry until explicitly created).
	boards, err := reg.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(boards) != 2 {
		t.Fatalf("List returned %d boards, want 2", len(boards))
	}
	found := map[string]bool{}
	for _, b := range boards {
		found[b.Name] = true
	}
	for _, want := range []string{"atm10-server", "hermes-agent"} {
		if !found[want] {
			t.Errorf("board %q not found in list", want)
		}
	}

	// Switch to atm10-server.
	if err := reg.Switch("atm10-server"); err != nil {
		t.Fatalf("Switch(atm10-server): %v", err)
	}
	cur, err := reg.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if cur.Name != "atm10-server" {
		t.Errorf("current board = %q, want atm10-server", cur.Name)
	}

	// Switch to default (implicit board).
	if err := reg.Switch("default"); err != nil {
		t.Fatalf("Switch(default): %v", err)
	}
	cur, err = reg.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if cur.Name != "default" {
		t.Errorf("current board = %q, want default", cur.Name)
	}

	// Switch back to hermes-agent.
	if err := reg.Switch("hermes-agent"); err != nil {
		t.Fatalf("Switch(hermes-agent): %v", err)
	}

	// Remove atm10-server (not the active board).
	if err := reg.Remove("atm10-server"); err != nil {
		t.Fatalf("Remove(atm10-server): %v", err)
	}
	boards, err = reg.List()
	if err != nil {
		t.Fatalf("List after remove: %v", err)
	}
	if len(boards) != 1 {
		t.Fatalf("after remove, got %d boards, want 1", len(boards))
	}
	if boards[0].Name != "hermes-agent" {
		t.Errorf("remaining board = %q, want hermes-agent", boards[0].Name)
	}

	// Current board is still hermes-agent (we removed atm10-server, not hermes-agent).
	cur, err = reg.Current()
	if err != nil {
		t.Fatalf("Current after remove: %v", err)
	}
	if cur.Name != "hermes-agent" {
		t.Errorf("current after remove of atm10-server = %q, want hermes-agent", cur.Name)
	}

	// Now remove the active board — current should revert to default.
	if err := reg.Remove("hermes-agent"); err != nil {
		t.Fatalf("Remove(hermes-agent): %v", err)
	}
	cur, err = reg.Current()
	if err != nil {
		t.Fatalf("Current after remove of active: %v", err)
	}
	if cur.Name != "default" {
		t.Errorf("current after remove of active hermes-agent = %q, want default", cur.Name)
	}
}

func TestBoardRegistry_Rename(t *testing.T) {
	root := t.TempDir()
	reg := NewBoardRegistry(root)

	if err := reg.Create("old-name"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := reg.Switch("old-name"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	if err := reg.Rename("old-name", "new-name"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	boards, err := reg.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(boards) != 1 || boards[0].Name != "new-name" {
		t.Errorf("after rename, boards = %v, want [new-name]", boards)
	}

	// Current board should now be new-name.
	cur, err := reg.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if cur.Name != "new-name" {
		t.Errorf("current after rename = %q, want new-name", cur.Name)
	}

	// Old-name should not exist.
	if err := reg.Switch("old-name"); err == nil {
		t.Error("Switch to old-name should fail after rename")
	}
}

func TestBoardRegistry_DuplicateCreate(t *testing.T) {
	root := t.TempDir()
	reg := NewBoardRegistry(root)

	if err := reg.Create("my-board"); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if err := reg.Create("my-board"); err == nil {
		t.Error("duplicate Create should fail")
	}
}

func TestBoardRegistry_RejectInvalidSlug(t *testing.T) {
	root := t.TempDir()
	reg := NewBoardRegistry(root)

	badSlugs := []string{"", "-bad", "_bad", "bad-", "bad_", "has spaces", "../escape"}
	for _, slug := range badSlugs {
		t.Run(slug, func(t *testing.T) {
			if err := reg.Create(slug); err == nil {
				t.Errorf("Create(%q) should fail", slug)
			}
		})
	}
}

func TestBoardRegistry_BoardIsolation(t *testing.T) {
	root := t.TempDir()
	reg := NewBoardRegistry(root)

	if err := reg.Create("board-a"); err != nil {
		t.Fatal(err)
	}
	if err := reg.Create("board-b"); err != nil {
		t.Fatal(err)
	}

	// Open board-a and create a task.
	reg.Switch("board-a")
	pathA := reg.BoardPath("board-a")
	storeA, err := Open(t.Context(), pathA)
	if err != nil {
		t.Fatalf("Open(board-a): %v", err)
	}
	defer storeA.Close()

	taskA, err := storeA.CreateTask(t.Context(), CreateTaskInput{Title: "task on A"})
	if err != nil {
		t.Fatalf("CreateTask(A): %v", err)
	}
	if taskA.Title != "task on A" {
		t.Fatalf("unexpected task A: %+v", taskA)
	}

	// Open board-b and verify it does NOT see board-a's task.
	reg.Switch("board-b")
	pathB := reg.BoardPath("board-b")
	storeB, err := Open(t.Context(), pathB)
	if err != nil {
		t.Fatalf("Open(board-b): %v", err)
	}
	defer storeB.Close()

	tasksB, err := storeB.ListTasks(t.Context(), ListFilter{})
	if err != nil {
		t.Fatalf("ListTasks(B): %v", err)
	}
	if len(tasksB) != 0 {
		t.Errorf("board-b has %d tasks, want 0 (board isolation failed)", len(tasksB))
	}

	// Board-a still has its task.
	tasksA, err := storeA.ListTasks(t.Context(), ListFilter{})
	if err != nil {
		t.Fatalf("ListTasks(A): %v", err)
	}
	if len(tasksA) != 1 || tasksA[0].Title != "task on A" {
		t.Errorf("board-a has %d tasks, want 1 with title 'task on A': %+v", len(tasksA), tasksA)
	}
}

func TestBoardRegistry_PersistenceAcrossInstances(t *testing.T) {
	root := t.TempDir()
	reg1 := NewBoardRegistry(root)
	if err := reg1.Create("persistent"); err != nil {
		t.Fatal(err)
	}
	if err := reg1.Switch("persistent"); err != nil {
		t.Fatal(err)
	}

	// New registry instance at same root should see the board and current state.
	reg2 := NewBoardRegistry(root)
	boards, err := reg2.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 1 || boards[0].Name != "persistent" {
		t.Fatalf("reg2 List = %v, want [persistent]", boards)
	}
	cur, err := reg2.Current()
	if err != nil {
		t.Fatal(err)
	}
	if cur.Name != "persistent" {
		t.Errorf("reg2 current = %q, want persistent", cur.Name)
	}
}

func TestBoardRegistry_RemoveCannotRemoveDefault(t *testing.T) {
	root := t.TempDir()
	reg := NewBoardRegistry(root)

	if err := reg.Remove("default"); err == nil {
		t.Error("Remove(default) should fail")
	}
}

func TestBoardRegistry_CurrentFileRemovedWhenNoBoards(t *testing.T) {
	root := t.TempDir()
	reg := NewBoardRegistry(root)

	// Create a board and switch to it.
	reg.Create("temp-board")
	reg.Switch("temp-board")

	// Remove it — current should revert to default and the current file should
	// not contain a stale board reference.
	reg.Remove("temp-board")

	curFile := filepath.Join(root, "kanban", "current")
	if _, err := os.Stat(curFile); err == nil {
		data, _ := os.ReadFile(curFile)
		if strings.TrimSpace(string(data)) != "" {
			t.Errorf("current file should be empty or removed, got %q", string(data))
		}
	}
}
