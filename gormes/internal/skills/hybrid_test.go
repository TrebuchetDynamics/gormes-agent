package skills

import (
	"reflect"
	"testing"
)

// TestSelectHybridCombinesLexicalAndSemantic proves the core contract of
// Phase 6.D: a skill that ranks moderately in BOTH the lexical and
// semantic rankings beats skills that rank highly in only one dimension.
//
// This is the defining property of hybrid search — neither a pure
// keyword match nor a pure embedding match should dominate when the
// query has signal on both axes.
func TestSelectHybridCombinesLexicalAndSemantic(t *testing.T) {
	skills := []Skill{
		{Name: "alpha-keyword-hit", Description: "lexical only", Body: ""},
		{Name: "beta-unrelated", Description: "wholly different topic", Body: ""},
		{Name: "gamma-keyword-shared", Description: "shared hit", Body: ""},
	}
	// Parallel embeddings aligned to skills above.
	embeddings := [][]float32{
		{0, 1, 0},     // alpha: strong lex, zero semantic overlap
		{1, 0, 0},     // beta:  zero lex, perfect semantic overlap
		{0.9, 0.1, 0}, // gamma: top-ranked in both
	}

	out := SelectHybrid(skills, embeddings, "keyword", []float32{1, 0, 0}, 3)

	got := skillNames(out)
	// gamma scores in BOTH rankings (lex-rank 2, sem-rank 2), so RRF
	// lifts it above alpha (lex-only, rank 1) and beta (sem-only, rank 1).
	// alpha < beta alphabetically breaks the tie between the single-side hits.
	want := []string{"gamma-keyword-shared", "alpha-keyword-hit", "beta-unrelated"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectHybrid() = %#v, want %#v", got, want)
	}
}

// TestSelectHybridFallsBackToLexicalWhenNoQueryEmbedding verifies that
// supplying a nil queryEmbedding reduces the function to a pure lexical
// ranking — the existing Select() contract, preserved so that callers
// with no embedder can still use the hybrid entrypoint.
func TestSelectHybridFallsBackToLexicalWhenNoQueryEmbedding(t *testing.T) {
	skills := []Skill{
		{Name: "careful-review", Description: "Review changes carefully", Body: "Work step by step."},
		{Name: "review-tests", Description: "Review tests and failure modes", Body: "Check assertions first."},
		{Name: "ship-checklist", Description: "Release checklist", Body: "Cut the tag after verification."},
	}

	out := SelectHybrid(skills, nil, "please review this carefully and check tests", nil, 2)

	got := skillNames(out)
	want := []string{"review-tests", "careful-review"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectHybrid() = %#v, want %#v", got, want)
	}
}

// TestSelectHybridHonorsSemanticWhenLexicalMisses proves that a skill
// with zero lexical overlap still surfaces when its embedding matches
// the query vector. Without semantic fusion, such a skill would be
// invisible to callers using the legacy lexical selector.
func TestSelectHybridHonorsSemanticWhenLexicalMisses(t *testing.T) {
	skills := []Skill{
		{Name: "auth-flow", Description: "token exchange", Body: ""},
		{Name: "billing-ledger", Description: "track billing", Body: ""},
	}
	embeddings := [][]float32{
		{1, 0, 0}, // auth-flow: perfect semantic overlap with query
		{0, 0, 1}, // billing-ledger: orthogonal
	}

	out := SelectHybrid(skills, embeddings, "zebra elephant giraffe", []float32{1, 0, 0}, 1)

	got := skillNames(out)
	want := []string{"auth-flow"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectHybrid() = %#v, want %#v", got, want)
	}
}

// TestSelectHybridTolerantsMissingEmbeddings ensures that skills with a
// nil embedding slot (not yet embedded) still participate via the
// lexical ranking, without being penalized into oblivion.
func TestSelectHybridTolerantsMissingEmbeddings(t *testing.T) {
	skills := []Skill{
		{Name: "unembedded-keyword", Description: "keyword only", Body: ""},
		{Name: "embedded-no-match", Description: "nothing", Body: ""},
	}
	embeddings := [][]float32{
		nil,       // unembedded-keyword: no vector, lexical only
		{0, 1, 0}, // embedded-no-match: orthogonal to query
	}

	out := SelectHybrid(skills, embeddings, "keyword", []float32{1, 0, 0}, 2)

	got := skillNames(out)
	want := []string{"unembedded-keyword"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectHybrid() = %#v, want %#v", got, want)
	}
}

// TestSelectHybridDeterministicAndCapped re-asserts the original Select
// guarantees on the hybrid entrypoint: identical inputs produce
// identical outputs, tied scores are ordered by name then path, and the
// result never exceeds the requested cap.
func TestSelectHybridDeterministicAndCapped(t *testing.T) {
	skills := []Skill{
		{Name: "careful-review", Description: "Review changes carefully", Path: "a/careful"},
		{Name: "review-tests", Description: "Review tests and failure modes", Path: "a/tests"},
		{Name: "ship-checklist", Description: "Release checklist", Path: "a/ship"},
		{Name: "zzz-noise", Description: "nothing relevant", Path: "a/zzz"},
	}
	embeddings := [][]float32{nil, nil, nil, nil}

	first := SelectHybrid(skills, embeddings, "please review this carefully and check tests", nil, 2)
	second := SelectHybrid(skills, embeddings, "please review this carefully and check tests", nil, 2)

	if len(first) != 2 {
		t.Fatalf("len(first) = %d, want 2", len(first))
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("SelectHybrid not deterministic: %#v vs %#v", first, second)
	}
	want := []string{"review-tests", "careful-review"}
	if got := skillNames(first); !reflect.DeepEqual(got, want) {
		t.Fatalf("skillNames(first) = %#v, want %#v", got, want)
	}
}

// TestSelectHybridEmptyQueryReturnsNil keeps the no-signal contract:
// when neither the query text nor the query embedding carry information,
// no skill is recommended. Callers must not see a default top-K list
// back just because a vector happened to be non-nil.
func TestSelectHybridEmptyQueryReturnsNil(t *testing.T) {
	skills := []Skill{{Name: "anything", Description: "desc", Body: "body"}}
	embeddings := [][]float32{{1, 0, 0}}

	if got := SelectHybrid(skills, embeddings, "", nil, 3); got != nil {
		t.Fatalf("SelectHybrid(empty query) = %#v, want nil", got)
	}
	if got := SelectHybrid(nil, nil, "keyword", []float32{1, 0, 0}, 3); got != nil {
		t.Fatalf("SelectHybrid(nil skills) = %#v, want nil", got)
	}
}

// TestSelectHybridIgnoresMismatchedDimensions verifies defensive
// behavior: a skill embedding with a different dimension from the query
// contributes zero semantic score (rather than panicking or crashing
// the whole selector), letting lexical signal still carry that skill.
func TestSelectHybridIgnoresMismatchedDimensions(t *testing.T) {
	skills := []Skill{
		{Name: "keyword-match", Description: "hit", Body: ""},
		{Name: "semantic-only", Description: "miss", Body: ""},
	}
	embeddings := [][]float32{
		{1, 0, 0, 0}, // dim=4 — mismatched with 3-dim query, cosine must be 0
		{1, 0, 0},    // dim=3 — aligned
	}

	out := SelectHybrid(skills, embeddings, "keyword", []float32{1, 0, 0}, 2)

	got := skillNames(out)
	// keyword-match carries lexical only; semantic-only carries semantic only.
	// With equal RRF contributions (rank 1 each), tie-break falls to name order.
	want := []string{"keyword-match", "semantic-only"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectHybrid() = %#v, want %#v", got, want)
	}
}
