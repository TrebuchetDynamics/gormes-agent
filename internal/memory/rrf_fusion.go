package memory

// RRFFuseIDs merges two ranked ID lists using Reciprocal Rank Fusion.
// Higher fused scores rank first. Duplicate IDs across sources get boosted.
func RRFFuseIDs(ftsIDs, semIDs []int64, k, ftsWeight, semWeight float64) []int64 {
	if k <= 0 {
		k = 60
	}
	if ftsWeight <= 0 {
		ftsWeight = 1.0
	}
	if semWeight <= 0 {
		semWeight = 1.0
	}

	scores := make(map[int64]float64)
	var order []int64
	seen := make(map[int64]bool)

	for i, id := range ftsIDs {
		score := ftsWeight / (k + float64(i+1))
		if _, ok := scores[id]; !ok {
			scores[id] = score
			order = append(order, id)
			seen[id] = true
		} else {
			scores[id] += score
		}
	}

	for i, id := range semIDs {
		score := semWeight / (k + float64(i+1))
		if _, ok := scores[id]; !ok {
			scores[id] = score
			order = append(order, id)
			seen[id] = true
		} else {
			scores[id] += score
		}
	}

	// Sort by descending score (simple insertion sort, small N).
	for i := 1; i < len(order); i++ {
		j := i
		for j > 0 && scores[order[j]] > scores[order[j-1]] {
			order[j], order[j-1] = order[j-1], order[j]
			j--
		}
	}

	return order
}
