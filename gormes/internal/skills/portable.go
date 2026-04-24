package skills

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Provenance is the learning-loop metadata the portable SKILL.md format
// carries alongside the core Skill document. It mirrors the fields
// produced by learning.Candidate so a skill distilled by Phase 6.B can
// be promoted to disk without losing its origin, signal score, or tool
// trace — the audit trail that lets operators decide whether to keep,
// edit, or disable a distilled skill later.
//
// Score and Threshold are always rendered (0 is meaningful: it records a
// below-threshold promotion), while SessionID, DistilledAt, Reasons, and
// ToolNames are omitted when blank/zero/empty. Callers without any
// provenance should pass nil to RenderPortable; the parser treats a
// missing "learning:" block as "legacy document" and yields a nil
// Provenance pointer so downstream code can tell the two cases apart
// without inspecting every scalar.
type Provenance struct {
	SessionID   string
	DistilledAt time.Time
	Score       int
	Threshold   int
	Reasons     []string
	ToolNames   []string
}

// PortableSkill bundles a parsed Skill with its optional Provenance so
// callers can surface both in one hop. Provenance is nil when the source
// document omitted the `learning:` block.
type PortableSkill struct {
	Skill      Skill
	Provenance *Provenance
}

// RenderPortable serialises a Skill as a portable SKILL.md document. If
// prov is nil, the output matches RenderDocument so the legacy rendering
// contract is unchanged. If prov is non-nil, a `learning:` block is
// appended to the frontmatter after `description:`, carrying the
// provenance fields in a stable order so the output is byte-deterministic
// for the same inputs.
func RenderPortable(skill Skill, prov *Provenance) string {
	if prov == nil {
		return RenderDocument(skill)
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: ")
	b.WriteString(strconv.Quote(strings.TrimSpace(skill.Name)))
	b.WriteByte('\n')
	b.WriteString("description: ")
	b.WriteString(strconv.Quote(strings.TrimSpace(skill.Description)))
	b.WriteByte('\n')

	b.WriteString("learning:\n")
	if sid := strings.TrimSpace(prov.SessionID); sid != "" {
		b.WriteString("  session_id: ")
		b.WriteString(strconv.Quote(sid))
		b.WriteByte('\n')
	}
	if !prov.DistilledAt.IsZero() {
		b.WriteString("  distilled_at: ")
		b.WriteString(strconv.Quote(prov.DistilledAt.UTC().Format(time.RFC3339)))
		b.WriteByte('\n')
	}
	b.WriteString("  score: ")
	b.WriteString(strconv.Itoa(prov.Score))
	b.WriteByte('\n')
	b.WriteString("  threshold: ")
	b.WriteString(strconv.Itoa(prov.Threshold))
	b.WriteByte('\n')
	if len(prov.Reasons) > 0 {
		b.WriteString("  reasons: ")
		writeInlineStringList(&b, prov.Reasons)
		b.WriteByte('\n')
	}
	if len(prov.ToolNames) > 0 {
		b.WriteString("  tool_names: ")
		writeInlineStringList(&b, prov.ToolNames)
		b.WriteByte('\n')
	}

	b.WriteString("---\n\n")
	b.WriteString(strings.TrimSpace(skill.Body))
	b.WriteByte('\n')
	return b.String()
}

// ParsePortable reads a SKILL.md document and returns a PortableSkill
// that bundles the parsed Skill with optional Provenance. Documents that
// lack the `learning:` block produce a nil Provenance so callers can
// distinguish "never had provenance" from "had all-zero provenance".
func ParsePortable(raw []byte, maxBytes int) (PortableSkill, error) {
	skill, err := Parse(raw, maxBytes)
	if err != nil {
		return PortableSkill{}, err
	}

	prov, err := extractProvenance(raw)
	if err != nil {
		return PortableSkill{}, err
	}
	return PortableSkill{Skill: skill, Provenance: prov}, nil
}

func extractProvenance(raw []byte) (*Provenance, error) {
	doc := string(raw)
	doc = strings.TrimPrefix(doc, "\ufeff")
	doc = strings.ReplaceAll(doc, "\r\n", "\n")
	lines := strings.Split(doc, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, nil
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return nil, nil
	}

	blockStart := -1
	for i := 1; i < end; i++ {
		if strings.TrimSpace(lines[i]) == "learning:" && isTopLevelKeyLine(lines[i]) {
			blockStart = i + 1
			break
		}
	}
	if blockStart == -1 {
		return nil, nil
	}

	prov := &Provenance{}
	for i := blockStart; i < end; i++ {
		line := lines[i]
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue
		}
		// A non-indented line ends the learning block (we treat the next
		// top-level frontmatter key as the block terminator).
		if !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "\t") {
			break
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "session_id":
			prov.SessionID = unquoteScalar(value)
		case "distilled_at":
			if t, perr := time.Parse(time.RFC3339, unquoteScalar(value)); perr == nil {
				prov.DistilledAt = t.UTC()
			} else {
				return nil, fmt.Errorf("skill learning.distilled_at: %w", perr)
			}
		case "score":
			if n, perr := strconv.Atoi(strings.TrimSpace(value)); perr == nil {
				prov.Score = n
			} else {
				return nil, fmt.Errorf("skill learning.score: %w", perr)
			}
		case "threshold":
			if n, perr := strconv.Atoi(strings.TrimSpace(value)); perr == nil {
				prov.Threshold = n
			} else {
				return nil, fmt.Errorf("skill learning.threshold: %w", perr)
			}
		case "reasons":
			list, perr := parseInlineStringList(value)
			if perr != nil {
				return nil, fmt.Errorf("skill learning.reasons: %w", perr)
			}
			prov.Reasons = list
		case "tool_names":
			list, perr := parseInlineStringList(value)
			if perr != nil {
				return nil, fmt.Errorf("skill learning.tool_names: %w", perr)
			}
			prov.ToolNames = list
		}
	}
	return prov, nil
}

func writeInlineStringList(b *strings.Builder, items []string) {
	b.WriteByte('[')
	for i, item := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(strconv.Quote(item))
	}
	b.WriteByte(']')
}

// parseInlineStringList accepts a YAML-style flow sequence of strings:
//
//	["a", "b"]
//
// It returns a non-nil slice whenever the brackets are well-formed,
// including the zero-element case `[]`, so callers can distinguish
// "missing" (nil) from "explicitly empty" semantically.
func parseInlineStringList(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil, fmt.Errorf("expected flow sequence [...], got %q", value)
	}
	inner := strings.TrimSpace(value[1 : len(value)-1])
	if inner == "" {
		return []string{}, nil
	}

	out := make([]string, 0, 4)
	i := 0
	for i < len(inner) {
		// Skip whitespace + separators between items.
		for i < len(inner) && (inner[i] == ' ' || inner[i] == ',') {
			i++
		}
		if i >= len(inner) {
			break
		}
		if inner[i] != '"' {
			return nil, fmt.Errorf("expected double-quoted element at offset %d in %q", i, inner)
		}
		j := i + 1
		var item strings.Builder
		for j < len(inner) {
			c := inner[j]
			if c == '\\' && j+1 < len(inner) {
				next := inner[j+1]
				switch next {
				case '"', '\\':
					item.WriteByte(next)
				case 'n':
					item.WriteByte('\n')
				case 't':
					item.WriteByte('\t')
				default:
					item.WriteByte('\\')
					item.WriteByte(next)
				}
				j += 2
				continue
			}
			if c == '"' {
				break
			}
			item.WriteByte(c)
			j++
		}
		if j >= len(inner) {
			return nil, fmt.Errorf("unterminated quoted element in %q", inner)
		}
		out = append(out, item.String())
		i = j + 1
	}
	return out, nil
}

func unquoteScalar(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		if unq, err := strconv.Unquote(value); err == nil {
			return unq
		}
		return value[1 : len(value)-1]
	}
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}
	return value
}

func isTopLevelKeyLine(line string) bool {
	if line == "" {
		return true
	}
	return line[0] != ' ' && line[0] != '\t'
}
