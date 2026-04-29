package hermes

import (
	"path"
	"sort"
	"strings"
)

type ToolCallParserStatus string

const (
	ToolCallParserStatusImplemented ToolCallParserStatus = "implemented"
	ToolCallParserStatusRowBacked   ToolCallParserStatus = "row_backed"
	ToolCallParserStatusExcluded    ToolCallParserStatus = "excluded"
)

const ToolCallParserFamilyRowBackedEvidence = "parser_family_row_backed"

type ToolCallParserFamily struct {
	File          string
	Family        string
	InputStyle    string
	Status        ToolCallParserStatus
	TargetPackage string
	ProgressRow   string
	EvidenceCode  string
}

type ToolCallParserManifest struct {
	Families []ToolCallParserFamily
}

func DefaultToolCallParserManifest() ToolCallParserManifest {
	rows := []ToolCallParserFamily{
		rowBackedParser("deepseek_v3_parser.py", "deepseek_v3", "deepseek-v3"),
		rowBackedParser("deepseek_v3_1_parser.py", "deepseek_v31", "deepseek-v3.1"),
		rowBackedParser("glm45_parser.py", "glm45", "glm-4.5"),
		rowBackedParser("glm47_parser.py", "glm47", "glm-4.7"),
		rowBackedParser("hermes_parser.py", "hermes", "hermes-xml"),
		rowBackedParser("kimi_k2_parser.py", "kimi_k2", "kimi-k2"),
		rowBackedParser("llama_parser.py", "llama", "llama"),
		rowBackedParser("longcat_parser.py", "longcat", "longcat"),
		rowBackedParser("mistral_parser.py", "mistral", "mistral"),
		rowBackedParser("qwen3_coder_parser.py", "qwen3_coder", "qwen3-coder"),
		rowBackedParser("qwen_parser.py", "qwen", "qwen"),
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].File < rows[j].File })
	return ToolCallParserManifest{Families: rows}
}

func (m ToolCallParserManifest) Lookup(file string) (ToolCallParserFamily, bool) {
	file = normalizeParserFile(file)
	for _, row := range m.Families {
		if row.File == file {
			return row, true
		}
	}
	return ToolCallParserFamily{}, false
}

func (m ToolCallParserManifest) MissingUpstreamFiles(upstream []string) []string {
	missing := make([]string, 0)
	for _, file := range upstream {
		file = normalizeParserFile(file)
		if !isParserPythonFile(file) {
			continue
		}
		if _, ok := m.Lookup(file); !ok {
			missing = append(missing, file)
		}
	}
	sort.Strings(missing)
	return missing
}

func (m ToolCallParserManifest) UnknownFamilies(upstream []string) []string {
	return m.MissingUpstreamFiles(upstream)
}

func rowBackedParser(file, family, inputStyle string) ToolCallParserFamily {
	return ToolCallParserFamily{
		File:          file,
		Family:        family,
		InputStyle:    inputStyle,
		Status:        ToolCallParserStatusRowBacked,
		TargetPackage: "internal/hermes",
		ProgressRow:   "5.M Raw tool-call parser fixture matrix",
		EvidenceCode:  ToolCallParserFamilyRowBackedEvidence,
	}
}

func normalizeParserFile(file string) string {
	file = strings.ReplaceAll(file, "\\", "/")
	return path.Base(file)
}

func isParserPythonFile(file string) bool {
	return strings.HasSuffix(file, "_parser.py")
}
