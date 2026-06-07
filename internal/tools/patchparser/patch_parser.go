package patchparser

import (
	"encoding/json"
	"strings"
	"time"
)

type PatchParserTool struct{}

func NewPatchParserTool() *PatchParserTool { return &PatchParserTool{} }

func (*PatchParserTool) Name() string { return "patch_parser" }
func (*PatchParserTool) Description() string {
	return "Parse and validate a unified diff patch. Returns structured patch information: files changed, hunks, additions, deletions."
}
func (*PatchParserTool) Timeout() time.Duration { return 10 * time.Second }

func (*PatchParserTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"patch":{"type":"string","description":"The unified diff patch content to parse"}},"required":["patch"]}`)
}

func (*PatchParserTool) Execute(_ any, args json.RawMessage) (json.RawMessage, error) {
	var in struct{ Patch string }
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, err
	}
	files := parsePatchFiles(in.Patch)
	return json.Marshal(map[string]any{
		"success": true,
		"files":   files,
		"count":   len(files),
	})
}

type PatchFileInfo struct {
	File      string `json:"file"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Hunks     int    `json:"hunks"`
}

func parsePatchFiles(patch string) []PatchFileInfo {
	state := patchParseState{}
	for _, line := range strings.Split(patch, "\n") {
		state.accept(line)
	}
	state.finish()
	return state.files
}

type patchParseState struct {
	files []PatchFileInfo
	cur   *PatchFileInfo
}

func (s *patchParseState) accept(line string) {
	switch {
	case strings.HasPrefix(line, "diff --git"):
		s.finish()
		s.cur = &PatchFileInfo{}
	case strings.HasPrefix(line, "--- "):
		s.ensureCurrentFile()
		if s.cur.File == "" {
			name := normalizePatchFileName(strings.TrimPrefix(line, "--- "))
			if name != "/dev/null" {
				s.cur.File = name
			}
		}
	case strings.HasPrefix(line, "+++ "):
		s.ensureCurrentFile()
		if s.cur.File == "" {
			s.cur.File = normalizePatchFileName(strings.TrimPrefix(line, "+++ "))
		}
	case s.cur != nil:
		s.acceptBodyLine(line)
	}
}

func (s *patchParseState) acceptBodyLine(line string) {
	switch {
	case strings.HasPrefix(line, "@@"):
		s.cur.Hunks++
	case strings.HasPrefix(line, "+"):
		s.cur.Additions++
	case strings.HasPrefix(line, "-"):
		s.cur.Deletions++
	}
}

func (s *patchParseState) ensureCurrentFile() {
	if s.cur == nil {
		s.cur = &PatchFileInfo{}
	}
}

func (s *patchParseState) finish() {
	if s.cur != nil {
		s.files = append(s.files, *s.cur)
		s.cur = nil
	}
}

func normalizePatchFileName(name string) string {
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, "a/") || strings.HasPrefix(name, "b/") {
		return name[2:]
	}
	return name
}
