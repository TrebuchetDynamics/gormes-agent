package skills

import (
	"math"
	"sort"
)

// rrfK is the dampening constant in the Reciprocal Rank Fusion formula
// `1 / (k + rank)`. The literature converges on 60 for general-purpose
// hybrid search; it keeps top-rank contributions meaningful without
// letting either ranking utterly dominate the other.
const rrfK = 60

// rankedSkill is the workhorse pair used by every ranking step in this
// file: which skill (by index into the input slice) and what score it
// earned. Using indices rather than skill copies keeps Skill values out
// of the sort comparator and makes ranking allocation-light.
type rankedSkill struct {
	index int
	score float64
}

// SelectHybrid returns up to max skills ranked by a Reciprocal Rank
// Fusion (RRF) of two independent rankings:
//
//  1. Lexical: token-overlap scoring against the skill's name,
//     description, and body (the same heuristic used by Select).
//  2. Semantic: cosine similarity between queryEmbedding and each
//     skill's per-row embedding.
//
// embeddings is parallel to skills: embeddings[i] is the vector for
// skills[i], or nil when that skill has not been embedded yet. A
// non-positive max snaps to DefaultSelectionCap. A nil/empty
// queryEmbedding disables the semantic ranking, reducing the function
// to a pure lexical selector. A skill with no contribution from either
// ranking is dropped before the cap is applied. Ties are broken
// deterministically by skill name, then skill path.
func SelectHybrid(
	skills []Skill,
	embeddings [][]float32,
	queryText string,
	queryEmbedding []float32,
	max int,
) []Skill {
	if len(skills) == 0 {
		return nil
	}
	if max <= 0 {
		max = DefaultSelectionCap
	}

	tokens := tokenize(queryText)
	hasSemantic := len(queryEmbedding) > 0
	if len(tokens) == 0 && !hasSemantic {
		return nil
	}

	lexical := lexicalRanking(skills, tokens)
	semantic := semanticRanking(skills, embeddings, queryEmbedding, hasSemantic)

	fused := make(map[int]float64, len(skills))
	for rank, r := range lexical {
		fused[r.index] += 1.0 / float64(rrfK+rank+1)
	}
	for rank, r := range semantic {
		fused[r.index] += 1.0 / float64(rrfK+rank+1)
	}

	final := make([]rankedSkill, 0, len(fused))
	for i, score := range fused {
		final = append(final, rankedSkill{index: i, score: score})
	}
	sortRanked(final, skills)

	if len(final) == 0 {
		return nil
	}
	out := make([]Skill, 0, max)
	for _, r := range final {
		out = append(out, skills[r.index])
		if len(out) == max {
			break
		}
	}
	return out
}

// lexicalRanking returns the score-descending lexical ranking,
// excluding skills with zero token overlap. The output index in the
// slice is the rank used by the RRF caller (rank 0 = top match).
func lexicalRanking(skills []Skill, tokens []string) []rankedSkill {
	out := make([]rankedSkill, 0, len(skills))
	if len(tokens) == 0 {
		return out
	}
	for i, s := range skills {
		if score := scoreSkill(s, tokens); score > 0 {
			out = append(out, rankedSkill{index: i, score: float64(score)})
		}
	}
	sortRanked(out, skills)
	return out
}

// semanticRanking returns the cosine-descending semantic ranking,
// excluding skills with no embedding or non-positive similarity. When
// hasQuery is false the function returns an empty ranking immediately.
func semanticRanking(skills []Skill, embeddings [][]float32, queryEmbedding []float32, hasQuery bool) []rankedSkill {
	out := make([]rankedSkill, 0, len(skills))
	if !hasQuery {
		return out
	}
	for i := range skills {
		if i >= len(embeddings) {
			break
		}
		vec := embeddings[i]
		if len(vec) == 0 {
			continue
		}
		if score := cosineSimilarity(queryEmbedding, vec); score > 0 {
			out = append(out, rankedSkill{index: i, score: score})
		}
	}
	sortRanked(out, skills)
	return out
}

// sortRanked orders entries by score descending, breaking ties by the
// underlying skill's name and path. Callers rely on the resulting order
// to assign RRF ranks, so the sort must be deterministic across runs
// even when scores collide.
func sortRanked(rs []rankedSkill, skills []Skill) {
	sort.SliceStable(rs, func(i, j int) bool {
		if rs[i].score != rs[j].score {
			return rs[i].score > rs[j].score
		}
		a := skills[rs[i].index]
		b := skills[rs[j].index]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Path < b.Path
	})
}

// cosineSimilarity returns dot(a, b) / (|a| * |b|). Mismatched
// dimensions or zero-magnitude inputs return 0 so a corrupt or empty
// embedding can never panic the selector.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i, x := range a {
		y := b[i]
		dot += float64(x) * float64(y)
		na += float64(x) * float64(x)
		nb += float64(y) * float64(y)
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
