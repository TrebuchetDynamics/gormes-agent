package navigation

// ClampIndex returns selected constrained to a list of total items. Empty lists
// clamp to zero so callers can keep cursor fields non-negative.
func ClampIndex(selected, total int) int {
	if total <= 0 || selected < 0 {
		return 0
	}
	if selected >= total {
		return total - 1
	}
	return selected
}

// MoveIndex applies delta and clamps the result to the list bounds.
func MoveIndex(selected, total, delta int) int {
	return ClampIndex(selected+delta, total)
}

// Window returns a centered visible window around selected.
func Window(selected, total, size int) (int, int) {
	if total <= 0 || size <= 0 {
		return 0, 0
	}
	if total <= size {
		return 0, total
	}
	selected = ClampIndex(selected, total)
	start := selected - size/2
	if start < 0 {
		start = 0
	}
	if start+size > total {
		start = total - size
	}
	return start, start + size
}
