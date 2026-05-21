package lsp

import (
	"context"
	"fmt"
	"sort"
)

const (
	StatusDiagnostics = "diagnostics"
	StatusClean       = "clean"
	StatusSkipped     = "skipped"
)

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character,omitempty"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	Severity int    `json:"severity,omitempty"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
	Range    Range  `json:"range"`
}

type PostEditRequest struct {
	Path         string
	RelativePath string
	PreContent   *string
	PostContent  string
}

type PostEditResult struct {
	Status      string       `json:"status"`
	Success     bool         `json:"success"`
	Message     string       `json:"message,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

type PostEditService interface {
	PostEditDiagnostics(ctx context.Context, req PostEditRequest) PostEditResult
}

func NewPostEditResult(req PostEditRequest, baseline []Diagnostic, post []Diagnostic) PostEditResult {
	delta := DiagnosticDelta(req.PreContent, req.PostContent, baseline, post)
	if len(delta) == 0 {
		return PostEditResult{Status: StatusClean, Success: true, Message: "LSP diagnostics clean"}
	}
	return PostEditResult{
		Status:      StatusDiagnostics,
		Success:     false,
		Message:     fmt.Sprintf("%d new LSP diagnostic(s)", len(delta)),
		Diagnostics: delta,
	}
}

func DiagnosticDelta(preContent *string, postContent string, baseline []Diagnostic, post []Diagnostic) []Diagnostic {
	if len(post) == 0 {
		return nil
	}
	shiftedBaseline := baseline
	if preContent != nil {
		shift := BuildLineShift(*preContent, postContent)
		shiftedBaseline = ShiftBaseline(baseline, shift)
	}
	seen := make(map[string]struct{}, len(shiftedBaseline))
	for _, diagnostic := range shiftedBaseline {
		seen[DiagnosticKey(diagnostic)] = struct{}{}
	}
	var delta []Diagnostic
	for _, diagnostic := range post {
		if _, ok := seen[DiagnosticKey(diagnostic)]; ok {
			continue
		}
		delta = append(delta, diagnostic)
	}
	return delta
}

func DiagnosticKey(d Diagnostic) string {
	return fmt.Sprintf("%d\x00%s\x00%s\x00%s\x00%d:%d:%d:%d", d.Severity, d.Code, d.Source, d.Message, d.Range.Start.Line, d.Range.Start.Character, d.Range.End.Line, d.Range.End.Character)
}

type LineShift func(int) (int, bool)

func BuildLineShift(preText, postText string) LineShift {
	preLines := splitLines(preText)
	postLines := splitLines(postText)
	if equalStrings(preLines, postLines) {
		return func(line int) (int, bool) { return line, true }
	}
	table := lcsTable(preLines, postLines)
	pairs := make(map[int]int)
	var walk func(i, j int)
	walk = func(i, j int) {
		if i == 0 || j == 0 {
			return
		}
		if preLines[i-1] == postLines[j-1] {
			walk(i-1, j-1)
			pairs[i-1] = j - 1
			return
		}
		if table[i-1][j] >= table[i][j-1] {
			walk(i-1, j)
			return
		}
		walk(i, j-1)
	}
	walk(len(preLines), len(postLines))
	keys := make([]int, 0, len(pairs))
	for key := range pairs {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return func(line int) (int, bool) {
		if mapped, ok := pairs[line]; ok {
			return mapped, true
		}
		idx := sort.SearchInts(keys, line)
		if idx < len(keys) && keys[idx] == line {
			return pairs[line], true
		}
		return 0, false
	}
}

func ShiftBaseline(baseline []Diagnostic, shift LineShift) []Diagnostic {
	if len(baseline) == 0 || shift == nil {
		return baseline
	}
	out := make([]Diagnostic, 0, len(baseline))
	for _, diagnostic := range baseline {
		shifted, ok := ShiftDiagnosticRange(diagnostic, shift)
		if ok {
			out = append(out, shifted)
		}
	}
	return out
}

func ShiftDiagnosticRange(d Diagnostic, shift LineShift) (Diagnostic, bool) {
	start, ok := shift(d.Range.Start.Line)
	if !ok {
		return Diagnostic{}, false
	}
	end, endOK := shift(d.Range.End.Line)
	if !endOK {
		end = start
	}
	out := d
	out.Range.Start.Line = start
	out.Range.End.Line = end
	return out, true
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := []string{}
	start := 0
	for i, r := range text {
		if r == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func lcsTable(a, b []string) [][]int {
	table := make([][]int, len(a)+1)
	for i := range table {
		table[i] = make([]int, len(b)+1)
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				table[i][j] = table[i-1][j-1] + 1
			} else if table[i-1][j] >= table[i][j-1] {
				table[i][j] = table[i-1][j]
			} else {
				table[i][j] = table[i][j-1]
			}
		}
	}
	return table
}
