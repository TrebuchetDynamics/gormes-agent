package tools

import (
	"encoding/json"
	"strings"
	"time"
)

type PatchParserTool struct{}

func NewPatchParserTool() *PatchParserTool { return &PatchParserTool{} }

func (*PatchParserTool) Name() string        { return "patch_parser" }
func (*PatchParserTool) Description() string { return "Parse and validate a unified diff patch. Returns structured patch information: files changed, hunks, additions, deletions." }
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

type patchFileInfo struct {
	File      string `json:"file"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Hunks     int    `json:"hunks"`
}

func parsePatchFiles(patch string) []patchFileInfo {
	var files []patchFileInfo
	lines := strings.Split(patch, "\n")
	var cur *patchFileInfo
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			if cur != nil {
				files = append(files, *cur)
			}
			cur = &patchFileInfo{}
		} else if strings.HasPrefix(line, "--- ") && cur != nil && cur.File == "" {
			name := strings.TrimPrefix(line, "--- ")
			if strings.HasPrefix(name, "a/") {
				name = name[2:]
			}
			cur.File = name
		} else if cur != nil {
			if strings.HasPrefix(line, "@@") {
				cur.Hunks++
			} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				cur.Additions++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				cur.Deletions++
			}
		}
	}
	if cur != nil {
		files = append(files, *cur)
	}
	return files
}
