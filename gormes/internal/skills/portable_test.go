package skills

import (
	"strings"
	"testing"
	"time"
)

func TestRenderPortableMatchesLegacyDocumentWhenProvenanceNil(t *testing.T) {
	skill := Skill{
		Name:        "debug-restart-loop",
		Description: "Diagnose and patch a kernel restart loop",
		Body:        "## Steps\n1. Trace\n2. Patch",
	}

	got := RenderPortable(skill, nil)
	want := RenderDocument(skill)
	if got != want {
		t.Fatalf("RenderPortable(nil provenance) = %q,\nwant legacy RenderDocument() output = %q", got, want)
	}
}

func TestRenderPortableEmitsLearningBlockWithProvenance(t *testing.T) {
	skill := Skill{
		Name:        "debug-restart-loop",
		Description: "Diagnose and patch a kernel restart loop using tracing + focused tests.",
		Body:        "## Steps\n1. Trace the failure.\n2. Read the restart path.\n3. Run the targeted tests until green.",
	}
	prov := &Provenance{
		SessionID:   "sess-debug-restart",
		DistilledAt: time.Date(2026, 4, 23, 20, 30, 0, 0, time.UTC),
		Score:       3,
		Threshold:   2,
		Reasons:     []string{"tool_calls", "multi_tool_calls", "duration"},
		ToolNames:   []string{"read_file", "run_tests"},
	}

	got := RenderPortable(skill, prov)

	for _, must := range []string{
		`name: "debug-restart-loop"`,
		`description: "Diagnose and patch a kernel restart loop using tracing + focused tests."`,
		"learning:",
		`  session_id: "sess-debug-restart"`,
		`  distilled_at: "2026-04-23T20:30:00Z"`,
		"  score: 3",
		"  threshold: 2",
		`  reasons: ["tool_calls", "multi_tool_calls", "duration"]`,
		`  tool_names: ["read_file", "run_tests"]`,
		"## Steps",
	} {
		if !strings.Contains(got, must) {
			t.Fatalf("RenderPortable output missing %q\ngot:\n%s", must, got)
		}
	}

	if !strings.HasPrefix(got, "---\n") {
		t.Fatalf("RenderPortable output must start with '---\\n', got prefix %q", got[:min(len(got), 16)])
	}
	if !strings.Contains(got, "\n---\n\n## Steps") {
		t.Fatalf("RenderPortable must close frontmatter before the body.\ngot:\n%s", got)
	}
}

func TestRenderPortableIsDeterministic(t *testing.T) {
	skill := Skill{
		Name:        "observe",
		Description: "short",
		Body:        "body",
	}
	prov := &Provenance{
		SessionID:   "sess-x",
		DistilledAt: time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC),
		Score:       2,
		Threshold:   2,
		Reasons:     []string{"a", "b"},
		ToolNames:   []string{"x"},
	}

	first := RenderPortable(skill, prov)
	second := RenderPortable(skill, prov)
	if first != second {
		t.Fatalf("RenderPortable not deterministic:\nfirst=%q\nsecond=%q", first, second)
	}
}

func TestRenderPortableEscapesQuotesInScalars(t *testing.T) {
	skill := Skill{
		Name:        "escape",
		Description: `Short "quoted" description with \backslashes\`,
		Body:        "body",
	}
	prov := &Provenance{
		SessionID:   `sess "quoted"`,
		DistilledAt: time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC),
		Score:       1,
		Threshold:   1,
		Reasons:     []string{`r"1`},
		ToolNames:   []string{`t\1`},
	}

	doc := RenderPortable(skill, prov)

	// Round-tripping is the contract that proves escaping is correct.
	ps, err := ParsePortable([]byte(doc), 8*1024)
	if err != nil {
		t.Fatalf("ParsePortable after RenderPortable returned %v\ndoc=\n%s", err, doc)
	}
	if ps.Skill.Description != skill.Description {
		t.Fatalf("description round-trip lost: got %q want %q", ps.Skill.Description, skill.Description)
	}
	if ps.Provenance == nil {
		t.Fatalf("provenance dropped on parse")
	}
	if ps.Provenance.SessionID != prov.SessionID {
		t.Fatalf("session_id round-trip lost: got %q want %q", ps.Provenance.SessionID, prov.SessionID)
	}
	if len(ps.Provenance.Reasons) != 1 || ps.Provenance.Reasons[0] != prov.Reasons[0] {
		t.Fatalf("reasons round-trip lost: got %#v want %#v", ps.Provenance.Reasons, prov.Reasons)
	}
	if len(ps.Provenance.ToolNames) != 1 || ps.Provenance.ToolNames[0] != prov.ToolNames[0] {
		t.Fatalf("tool_names round-trip lost: got %#v want %#v", ps.Provenance.ToolNames, prov.ToolNames)
	}
}

func TestParsePortableWithoutLearningBlockHasNilProvenance(t *testing.T) {
	raw := strings.Join([]string{
		"---",
		`name: "legacy-skill"`,
		`description: "Legacy skills without provenance must still parse."`,
		"---",
		"",
		"body",
		"",
	}, "\n")

	ps, err := ParsePortable([]byte(raw), 8*1024)
	if err != nil {
		t.Fatalf("ParsePortable legacy doc: %v", err)
	}
	if ps.Skill.Name != "legacy-skill" {
		t.Fatalf("Skill.Name = %q", ps.Skill.Name)
	}
	if ps.Provenance != nil {
		t.Fatalf("Provenance = %#v, want nil for legacy doc", ps.Provenance)
	}
}

func TestParsePortableSurfacesLearningBlockFields(t *testing.T) {
	raw := strings.Join([]string{
		"---",
		`name: "debug-restart-loop"`,
		`description: "Diagnose and patch a kernel restart loop"`,
		"learning:",
		`  session_id: "sess-debug-restart"`,
		`  distilled_at: "2026-04-23T20:30:00Z"`,
		"  score: 3",
		"  threshold: 2",
		`  reasons: ["tool_calls", "multi_tool_calls", "duration"]`,
		`  tool_names: ["read_file", "run_tests"]`,
		"---",
		"",
		"body",
		"",
	}, "\n")

	ps, err := ParsePortable([]byte(raw), 8*1024)
	if err != nil {
		t.Fatalf("ParsePortable: %v", err)
	}
	if ps.Provenance == nil {
		t.Fatalf("Provenance = nil, want populated")
	}
	want := Provenance{
		SessionID:   "sess-debug-restart",
		DistilledAt: time.Date(2026, 4, 23, 20, 30, 0, 0, time.UTC),
		Score:       3,
		Threshold:   2,
		Reasons:     []string{"tool_calls", "multi_tool_calls", "duration"},
		ToolNames:   []string{"read_file", "run_tests"},
	}
	if ps.Provenance.SessionID != want.SessionID {
		t.Errorf("SessionID = %q, want %q", ps.Provenance.SessionID, want.SessionID)
	}
	if !ps.Provenance.DistilledAt.Equal(want.DistilledAt) {
		t.Errorf("DistilledAt = %v, want %v", ps.Provenance.DistilledAt, want.DistilledAt)
	}
	if ps.Provenance.Score != want.Score || ps.Provenance.Threshold != want.Threshold {
		t.Errorf("Score/Threshold = %d/%d, want %d/%d",
			ps.Provenance.Score, ps.Provenance.Threshold, want.Score, want.Threshold)
	}
	if !stringSliceEq(ps.Provenance.Reasons, want.Reasons) {
		t.Errorf("Reasons = %#v, want %#v", ps.Provenance.Reasons, want.Reasons)
	}
	if !stringSliceEq(ps.Provenance.ToolNames, want.ToolNames) {
		t.Errorf("ToolNames = %#v, want %#v", ps.Provenance.ToolNames, want.ToolNames)
	}
}

func TestPortableRoundTripPreservesEveryField(t *testing.T) {
	skill := Skill{
		Name:        "debug-restart-loop",
		Description: "Diagnose and patch a kernel restart loop using tracing + focused tests.",
		Body:        "## Steps\n1. Trace the failure.\n2. Read the restart path.\n3. Run tests.",
	}
	prov := &Provenance{
		SessionID:   "sess-debug-restart",
		DistilledAt: time.Date(2026, 4, 23, 20, 30, 0, 0, time.UTC),
		Score:       3,
		Threshold:   2,
		Reasons:     []string{"tool_calls", "multi_tool_calls", "duration"},
		ToolNames:   []string{"read_file", "run_tests"},
	}

	doc := RenderPortable(skill, prov)
	ps, err := ParsePortable([]byte(doc), 16*1024)
	if err != nil {
		t.Fatalf("ParsePortable: %v\ndoc=\n%s", err, doc)
	}

	if ps.Skill.Name != skill.Name {
		t.Errorf("Name round-trip: got %q want %q", ps.Skill.Name, skill.Name)
	}
	if ps.Skill.Description != skill.Description {
		t.Errorf("Description round-trip: got %q want %q", ps.Skill.Description, skill.Description)
	}
	if ps.Skill.Body != skill.Body {
		t.Errorf("Body round-trip: got %q want %q", ps.Skill.Body, skill.Body)
	}
	if ps.Provenance == nil {
		t.Fatalf("Provenance dropped on parse")
	}
	if ps.Provenance.SessionID != prov.SessionID {
		t.Errorf("SessionID round-trip: got %q want %q", ps.Provenance.SessionID, prov.SessionID)
	}
	if !ps.Provenance.DistilledAt.Equal(prov.DistilledAt) {
		t.Errorf("DistilledAt round-trip: got %v want %v", ps.Provenance.DistilledAt, prov.DistilledAt)
	}
	if ps.Provenance.Score != prov.Score {
		t.Errorf("Score round-trip: got %d want %d", ps.Provenance.Score, prov.Score)
	}
	if ps.Provenance.Threshold != prov.Threshold {
		t.Errorf("Threshold round-trip: got %d want %d", ps.Provenance.Threshold, prov.Threshold)
	}
	if !stringSliceEq(ps.Provenance.Reasons, prov.Reasons) {
		t.Errorf("Reasons round-trip: got %#v want %#v", ps.Provenance.Reasons, prov.Reasons)
	}
	if !stringSliceEq(ps.Provenance.ToolNames, prov.ToolNames) {
		t.Errorf("ToolNames round-trip: got %#v want %#v", ps.Provenance.ToolNames, prov.ToolNames)
	}
}

func TestParsePortableRejectsMalformedDistilledAt(t *testing.T) {
	raw := strings.Join([]string{
		"---",
		`name: "x"`,
		`description: "y"`,
		"learning:",
		`  distilled_at: "not-a-time"`,
		"---",
		"body",
	}, "\n")

	_, err := ParsePortable([]byte(raw), 8*1024)
	if err == nil {
		t.Fatal("ParsePortable returned nil, want error for malformed distilled_at")
	}
	if !strings.Contains(err.Error(), "distilled_at") {
		t.Fatalf("error %q does not mention distilled_at", err)
	}
}

func TestLegacyParseIgnoresLearningBlock(t *testing.T) {
	// A document carrying the learning block must still parse through the
	// existing skills.Parse seam. 6.D retrieval, 6.F browsing, and the
	// runtime snapshot all share the same parser, so portable SKILL.md
	// files dropped into any active dir must not regress them.
	raw := strings.Join([]string{
		"---",
		`name: "portable-doc"`,
		`description: "Has learning block but must still parse via legacy Parse."`,
		"learning:",
		`  session_id: "sess-x"`,
		`  distilled_at: "2026-04-23T20:30:00Z"`,
		"  score: 3",
		"  threshold: 2",
		"---",
		"body text",
		"",
	}, "\n")

	skill, err := Parse([]byte(raw), 8*1024)
	if err != nil {
		t.Fatalf("legacy Parse() must still accept portable docs, got %v", err)
	}
	if skill.Name != "portable-doc" {
		t.Fatalf("Skill.Name = %q", skill.Name)
	}
	if skill.Description != "Has learning block but must still parse via legacy Parse." {
		t.Fatalf("Skill.Description = %q", skill.Description)
	}
	if !strings.Contains(skill.Body, "body text") {
		t.Fatalf("Skill.Body = %q, want body preserved", skill.Body)
	}
}

func stringSliceEq(a, b []string) bool {
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
