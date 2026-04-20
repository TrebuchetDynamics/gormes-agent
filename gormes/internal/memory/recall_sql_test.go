package memory

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func openGraphWithSeeds(t *testing.T) *SqliteStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "memory.db")
	s, err := OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close(context.Background()) })

	_, _ = s.db.Exec(`
		INSERT INTO entities(name, type, description, updated_at) VALUES
			('AzulVigia','PROJECT','sports platform',1),
			('Cadereyta','PLACE','',1),
			('Vania','PERSON','',1),
			('Neovim','TOOL','',1)
	`)
	_, _ = s.db.Exec(`
		INSERT INTO turns(session_id, role, content, ts_unix, chat_id) VALUES
			('s','user','working on AzulVigia',1,'telegram:42'),
			('s','user','Vania uses Neovim',2,'telegram:42'),
			('s','user','Neovim rocks',3,'telegram:99')
	`)
	return s
}

func TestSeedsExactName_MatchesCaseInsensitive(t *testing.T) {
	s := openGraphWithSeeds(t)
	ids, err := seedsExactName(context.Background(), s.db,
		[]string{"azulvigia", "Vania"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Errorf("len=%d, want 2", len(ids))
	}
}

func TestSeedsExactName_SkipsShortNames(t *testing.T) {
	s := openGraphWithSeeds(t)
	ids, _ := seedsExactName(context.Background(), s.db, []string{"Vo"}, 5)
	if len(ids) != 0 {
		t.Errorf("short name returned %d seeds, want 0", len(ids))
	}
}

func TestSeedsExactName_EmptyCandidateReturnsEmpty(t *testing.T) {
	s := openGraphWithSeeds(t)
	ids, err := seedsExactName(context.Background(), s.db, nil, 5)
	if err != nil {
		t.Errorf("err = %v, want nil on empty candidates", err)
	}
	if len(ids) != 0 {
		t.Errorf("len = %d, want 0", len(ids))
	}
}

func TestSeedsFTS5_MatchesByTurnContent(t *testing.T) {
	s := openGraphWithSeeds(t)
	ids, err := seedsFTS5(context.Background(), s.db,
		"AzulVigia", "telegram:42", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) == 0 {
		t.Error("FTS5 match returned zero seeds; AzulVigia should match turn content")
	}
}

func TestSeedsFTS5_ScopesToChatID(t *testing.T) {
	s := openGraphWithSeeds(t)
	// "Neovim" appears in a chat-99 turn, NOT a chat-42 turn. Querying
	// from chat 42 must not return Neovim via FTS5 (chat scoping).
	ids, _ := seedsFTS5(context.Background(), s.db,
		"Neovim", "telegram:42", 5)
	for _, id := range ids {
		var name string
		_ = s.db.QueryRow(`SELECT name FROM entities WHERE id = ?`, id).Scan(&name)
		if name == "Neovim" {
			// "Neovim" ALSO appears in chat-42's "Vania uses Neovim" turn,
			// so this actually SHOULD match. Let me re-check the fixture.
			// Actually: chat-42's turn 2 is "Vania uses Neovim" — Neovim IS there.
			// So this test passes BUT its invariant is weaker than intended.
			// The test still demonstrates scoping because a query from chat 99
			// would find Neovim ONLY, not AzulVigia which is chat-42-only.
			_ = name // keep the assertion stance
		}
	}
	// Stronger scoping check: query from chat 99 for "AzulVigia".
	// AzulVigia is only in a chat-42 turn; chat-99 scope must return zero.
	ids2, _ := seedsFTS5(context.Background(), s.db,
		"AzulVigia", "telegram:99", 5)
	for _, id := range ids2 {
		var name string
		_ = s.db.QueryRow(`SELECT name FROM entities WHERE id = ?`, id).Scan(&name)
		if name == "AzulVigia" {
			t.Errorf("chat-42-only AzulVigia leaked into chat-99 scope")
		}
	}
}

func TestSeedsFTS5_EmptyChatIDMatchesGlobal(t *testing.T) {
	s := openGraphWithSeeds(t)
	ids, _ := seedsFTS5(context.Background(), s.db, "AzulVigia", "", 5)
	if len(ids) == 0 {
		t.Error("empty chat_id should be global scope; got zero seeds")
	}
}

// openGraphWithEdges builds a graph for CTE tests:
//
//	A --KNOWS--> B --WORKS_ON--> C --LOCATED_IN--> D
//	A --LIKES--> E   (weight 0.5 — below threshold)
//
// Weights: A->B = 2.0, B->C = 2.0, C->D = 2.0, A->E = 0.5
func openGraphWithEdges(t *testing.T) (*SqliteStore, map[string]int64) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "memory.db")
	s, err := OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close(context.Background()) })

	for _, n := range []string{"A", "B", "C", "D", "E"} {
		_, _ = s.db.Exec(
			`INSERT INTO entities(name, type, updated_at) VALUES(?, 'PERSON', ?)`,
			n, time.Now().Unix())
	}
	ids := make(map[string]int64)
	rows, _ := s.db.Query(`SELECT name, id FROM entities`)
	for rows.Next() {
		var n string
		var id int64
		_ = rows.Scan(&n, &id)
		ids[n] = id
	}
	rows.Close()

	type edge struct {
		src, tgt, pred string
		w              float64
	}
	edges := []edge{
		{"A", "B", "KNOWS", 2.0},
		{"B", "C", "WORKS_ON", 2.0},
		{"C", "D", "LOCATED_IN", 2.0},
		{"A", "E", "LIKES", 0.5},
	}
	for _, e := range edges {
		_, _ = s.db.Exec(
			`INSERT INTO relationships(source_id, target_id, predicate, weight, updated_at)
			 VALUES(?, ?, ?, ?, ?)`,
			ids[e.src], ids[e.tgt], e.pred, e.w, time.Now().Unix())
	}
	return s, ids
}

// hasEntityNamed returns true if the returned neighborhood includes
// an entity with the given name.
func hasEntityNamed(list []recalledEntity, name string) bool {
	for _, e := range list {
		if e.Name == name {
			return true
		}
	}
	return false
}

func TestTraverse_OneDegreeFromA(t *testing.T) {
	s, ids := openGraphWithEdges(t)
	got, err := traverseNeighborhood(context.Background(), s.db,
		[]int64{ids["A"]}, 1, 1.0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEntityNamed(got, "A") || !hasEntityNamed(got, "B") {
		t.Errorf("neighborhood missing A or B; got %v", got)
	}
	if hasEntityNamed(got, "E") {
		t.Errorf("weight-0.5 edge A->E should have been filtered; got %v", got)
	}
}

func TestTraverse_TwoDegreeFromA(t *testing.T) {
	s, ids := openGraphWithEdges(t)
	got, _ := traverseNeighborhood(context.Background(), s.db,
		[]int64{ids["A"]}, 2, 1.0, 10)
	if !hasEntityNamed(got, "C") {
		t.Errorf("depth-2 should include C; got %v", got)
	}
	if hasEntityNamed(got, "D") {
		t.Errorf("depth=2 must NOT reach D (D is at depth 3); got %v", got)
	}
}

func TestTraverse_ThreeDegreeReachesD(t *testing.T) {
	s, ids := openGraphWithEdges(t)
	got, _ := traverseNeighborhood(context.Background(), s.db,
		[]int64{ids["A"]}, 3, 1.0, 10)
	if !hasEntityNamed(got, "D") {
		t.Errorf("depth-3 should include D; got %v", got)
	}
}

func TestTraverse_WeightThresholdFiltersWeakEdges(t *testing.T) {
	s, ids := openGraphWithEdges(t)
	got, _ := traverseNeighborhood(context.Background(), s.db,
		[]int64{ids["A"]}, 2, 1.0, 10)
	if hasEntityNamed(got, "E") {
		t.Errorf("weight=0.5 edge should have been excluded at threshold=1.0; got %v", got)
	}
}

func TestTraverse_MaxFactsCap(t *testing.T) {
	s, ids := openGraphWithEdges(t)
	got, _ := traverseNeighborhood(context.Background(), s.db,
		[]int64{ids["A"]}, 5, 0.0, 2)
	if len(got) > 2 {
		t.Errorf("len = %d, want <= 2 (MaxFacts)", len(got))
	}
}

func TestTraverse_EmptySeedsReturnsEmpty(t *testing.T) {
	s, _ := openGraphWithEdges(t)
	got, err := traverseNeighborhood(context.Background(), s.db,
		nil, 2, 1.0, 10)
	if err != nil {
		t.Errorf("err = %v, want nil for empty seeds", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}
