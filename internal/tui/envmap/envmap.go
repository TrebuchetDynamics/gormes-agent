package envmap

// Value returns the value for key from env, treating a nil map as empty.
func Value(env map[string]string, key string) string {
	if env == nil {
		return ""
	}
	return env[key]
}

// Has reports whether key is present in env, treating a nil map as empty.
func Has(env map[string]string, key string) bool {
	if env == nil {
		return false
	}
	_, ok := env[key]
	return ok
}
