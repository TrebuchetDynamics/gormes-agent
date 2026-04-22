package skills

import (
	"fmt"
	"strings"
)

// Parse converts a SKILL.md document into a typed Skill.
func Parse(raw []byte, maxBytes int) (Skill, error) {
	doc := string(raw)
	if maxBytes > 0 && len(raw) > maxBytes {
		return Skill{}, fmt.Errorf("skill document too large: %d > %d bytes", len(raw), maxBytes)
	}

	doc = strings.TrimPrefix(doc, "\ufeff")
	doc = strings.ReplaceAll(doc, "\r\n", "\n")

	lines := strings.Split(doc, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return Skill{}, fmt.Errorf("skill frontmatter must start with ---")
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return Skill{}, fmt.Errorf("skill frontmatter closing --- not found")
	}

	var skill Skill
	skill.RawBytes = len(raw)

	for _, line := range lines[1:end] {
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = trimScalar(value)
		switch key {
		case "name":
			skill.Name = value
		case "description":
			skill.Description = value
		}
	}

	skill.Body = strings.Trim(strings.Join(lines[end+1:], "\n"), "\n")
	if err := skill.Validate(maxBytes); err != nil {
		return Skill{}, err
	}
	return skill, nil
}

func trimScalar(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if value[0] == '"' && value[len(value)-1] == '"' {
			return value[1 : len(value)-1]
		}
		if value[0] == '\'' && value[len(value)-1] == '\'' {
			return value[1 : len(value)-1]
		}
	}
	return value
}
