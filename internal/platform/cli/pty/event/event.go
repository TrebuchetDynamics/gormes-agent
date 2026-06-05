package event

// Clone returns a shallow copy of a structured PTY sidecar event.
// Nil events become an empty map so publishers and sinks can treat a missing
// payload as a valid structured event without sharing mutable state.
func Clone(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
