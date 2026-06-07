package mapclone

// StringString returns an independent copy of input while preserving nil maps.
func StringString(input map[string]string) map[string]string {
	return copyMap(input)
}

// StringStringOrEmpty returns an independent copy of input and converts nil to
// an empty map for callers that historically expose non-nil reloadable maps.
func StringStringOrEmpty(input map[string]string) map[string]string {
	return copyMapOrEmpty(input)
}

// StringBoolOrEmpty returns an independent copy of input and converts nil to
// an empty map for callers that historically expose non-nil reloadable maps.
func StringBoolOrEmpty(input map[string]bool) map[string]bool {
	return copyMapOrEmpty(input)
}

// NestedStringBoolOrEmpty returns an independent deep copy of input and converts
// nil to an empty map for callers that historically expose non-nil reloadable maps.
func NestedStringBoolOrEmpty(input map[string]map[string]bool) map[string]map[string]bool {
	out := make(map[string]map[string]bool, len(input))
	for key, value := range input {
		out[key] = StringBoolOrEmpty(value)
	}
	return out
}

func copyMap[K comparable, V any](input map[K]V) map[K]V {
	if input == nil {
		return nil
	}
	return copyMapOrEmpty(input)
}

func copyMapOrEmpty[K comparable, V any](input map[K]V) map[K]V {
	out := make(map[K]V, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
