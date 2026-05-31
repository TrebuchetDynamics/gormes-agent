package commandpatterns

import (
	"fmt"
	"regexp"
)

// Entry describes one command-safety regex and its audit description.
// It is shared by hardline and recoverable dangerous command policy tables.
type Entry struct {
	Regex       string
	Description string
}

// Compiled is a regex policy entry ready for command matching.
type Compiled struct {
	Regex       *regexp.Regexp
	Description string
}

// Compile compiles entries with the supplied regexp option prefix.
func Compile(prefix string, entries []Entry) []Compiled {
	compiled := make([]Compiled, 0, len(entries))
	for _, entry := range entries {
		re, err := regexp.Compile(prefix + entry.Regex)
		if err != nil {
			continue
		}
		compiled = append(compiled, Compiled{Regex: re, Description: entry.Description})
	}
	return compiled
}

// MustValidate panics if any entry does not compile with prefix.
func MustValidate(label, prefix string, entries []Entry) {
	for _, entry := range entries {
		if _, err := regexp.Compile(prefix + entry.Regex); err != nil {
			panic(fmt.Sprintf("tools: invalid %s pattern %q: %v", label, entry.Regex, err))
		}
	}
}
