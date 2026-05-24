package compact

import (
	"strings"
	"testing"
)

func TestCompact_GoTestFailurePreservesActionableDiagnostics(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString("ok  \tgithub.com/example/project/pkg")
		b.WriteString(string(rune('a' + i%20)))
		b.WriteString("\t0.001s\n")
	}
	b.WriteString("--- FAIL: TestWidgetHandlesOverflow (0.00s)\n")
	b.WriteString("    widget_test.go:42: got overflow=false, want true\n")
	b.WriteString("FAIL\n")
	b.WriteString("FAIL\tgithub.com/example/project/widget\t0.123s\n")

	result := Compact(Request{
		ToolName: "execute_code",
		Command:  "go test ./...",
		Stream:   "stdout",
		Text:     b.String(),
		ExitCode: 1,
	}, Config{Mode: ModeAuto, ThresholdBytes: 128, HeadLines: 2, TailLines: 2})

	if !result.Applied {
		t.Fatal("Compact Applied = false, want true")
	}
	if result.Reducer != ReducerGoTest {
		t.Fatalf("Reducer = %q, want %q", result.Reducer, ReducerGoTest)
	}
	for _, want := range []string{
		"TestWidgetHandlesOverflow",
		"widget_test.go:42",
		"FAIL\tgithub.com/example/project/widget",
		"passing packages: 40",
	} {
		if !strings.Contains(result.Text, want) {
			t.Fatalf("compacted output missing %q:\n%s", want, result.Text)
		}
	}
	if strings.Contains(result.Text, "github.com/example/project/pkga") &&
		strings.Contains(result.Text, "github.com/example/project/pkgt") {
		t.Fatalf("compacted output kept noisy passing package wall:\n%s", result.Text)
	}
	if result.OriginalBytes <= result.CompactedBytes {
		t.Fatalf("bytes = original %d compacted %d, want reduction", result.OriginalBytes, result.CompactedBytes)
	}
	if !hasEvidence(result.Evidence, EvidenceFailingTestsKept) {
		t.Fatalf("evidence = %v, want %q", result.Evidence, EvidenceFailingTestsKept)
	}
}

func TestCompact_GoTestFailureAfterLeadingNoise(t *testing.T) {
	text := strings.Join([]string{
		"preparing test environment",
		"ok  \tgithub.com/example/project/pkg\t0.001s",
		"--- FAIL: TestWidgetHandlesOverflow (0.00s)",
		"    widget_test.go:42: got overflow=false, want true",
		"FAIL",
		"FAIL\tgithub.com/example/project/widget\t0.123s",
	}, "\n")

	result := Compact(Request{
		ToolName: "execute_code",
		Command:  "go test ./...",
		Stream:   "stdout",
		Text:     text,
		ExitCode: 1,
	}, Config{Mode: ModeAuto, ThresholdBytes: 16, HeadLines: 1, TailLines: 1})

	if result.Reducer != ReducerGoTest {
		t.Fatalf("Reducer = %q, want %q:\n%s", result.Reducer, ReducerGoTest, result.Text)
	}
	if !strings.Contains(result.Text, "TestWidgetHandlesOverflow") {
		t.Fatalf("compacted output lost failure:\n%s", result.Text)
	}
}

func TestCompact_GoTestSuccessCountsPassingPackages(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 24; i++ {
		b.WriteString("ok  \tgithub.com/example/project/pkg")
		b.WriteString(string(rune('a' + i%20)))
		b.WriteString("\t0.001s\n")
	}

	result := Compact(Request{
		ToolName: "execute_code",
		Command:  "go test ./...",
		Stream:   "stdout",
		Text:     b.String(),
		ExitCode: 0,
	}, Config{Mode: ModeAuto, ThresholdBytes: 128, HeadLines: 2, TailLines: 2})

	if !result.Applied {
		t.Fatal("Compact Applied = false, want true")
	}
	if result.Reducer != ReducerGoTest {
		t.Fatalf("Reducer = %q, want %q", result.Reducer, ReducerGoTest)
	}
	if !strings.Contains(result.Text, "passing packages: 24") {
		t.Fatalf("compacted success output missing package count:\n%s", result.Text)
	}
	if strings.Contains(result.Text, "github.com/example/project/pkga") {
		t.Fatalf("compacted success output kept passing package wall:\n%s", result.Text)
	}
	if !hasEvidence(result.Evidence, EvidencePassingPkgsCounted) {
		t.Fatalf("evidence = %v, want %q", result.Evidence, EvidencePassingPkgsCounted)
	}
}

func TestCompact_HeadTailFallbackKeepsBoundaries(t *testing.T) {
	text := strings.Join([]string{
		"line 01",
		"line 02",
		"line 03",
		"line 04",
		"line 05",
		"line 06",
		"line 07",
		"line 08",
	}, "\n")

	result := Compact(Request{
		ToolName: "execute_code",
		Command:  "custom noisy command",
		Stream:   "stderr",
		Text:     text,
		ExitCode: 2,
	}, Config{Mode: ModeAuto, ThresholdBytes: 16, HeadLines: 2, TailLines: 2})

	if !result.Applied {
		t.Fatal("Compact Applied = false, want true")
	}
	if result.Reducer != ReducerHeadTail {
		t.Fatalf("Reducer = %q, want %q", result.Reducer, ReducerHeadTail)
	}
	for _, want := range []string{"line 01", "line 02", "line 07", "line 08", "4 line(s) omitted"} {
		if !strings.Contains(result.Text, want) {
			t.Fatalf("head/tail output missing %q:\n%s", want, result.Text)
		}
	}
	if strings.Contains(result.Text, "line 04") || strings.Contains(result.Text, "line 05") {
		t.Fatalf("head/tail output kept middle lines:\n%s", result.Text)
	}
}

func TestCompact_BuildDiagnosticsKeepsFileLineErrors(t *testing.T) {
	text := strings.Join([]string{
		"# github.com/example/project/internal/widget",
		"internal/widget/widget.go:42:13: undefined: missingThing",
		"internal/widget/widget.go:43:2: declared and not used: unused",
	}, "\n") + "\n" + strings.Repeat("dependency cache noise\n", 40) + "FAIL\tgithub.com/example/project/internal/widget [build failed]\n"

	result := Compact(Request{
		ToolName: "execute_code",
		Command:  "go build ./...",
		Stream:   "stderr",
		Text:     text,
		ExitCode: 1,
	}, Config{Mode: ModeAuto, ThresholdBytes: 128, HeadLines: 2, TailLines: 2})

	if !result.Applied {
		t.Fatal("Compact Applied = false, want true")
	}
	if result.Reducer != ReducerBuildDiagnostics {
		t.Fatalf("Reducer = %q, want %q", result.Reducer, ReducerBuildDiagnostics)
	}
	for _, want := range []string{
		"internal/widget/widget.go:42:13",
		"undefined: missingThing",
		"internal/widget/widget.go:43:2",
		"FAIL\tgithub.com/example/project/internal/widget [build failed]",
	} {
		if !strings.Contains(result.Text, want) {
			t.Fatalf("build diagnostics output missing %q:\n%s", want, result.Text)
		}
	}
	if strings.Contains(result.Text, "dependency cache noise") {
		t.Fatalf("build diagnostics kept repeated noise:\n%s", result.Text)
	}
	if !hasEvidence(result.Evidence, EvidenceDiagnosticsKept) {
		t.Fatalf("evidence = %v, want %q", result.Evidence, EvidenceDiagnosticsKept)
	}
}

func TestCompact_GitStatusSummarizesFileStates(t *testing.T) {
	text := strings.Join([]string{
		"## development...origin/development [ahead 1]",
		" M internal/tools/execute_code.go",
		"M  internal/tools/compact/compact.go",
		"A  internal/tools/compact/compact_test.go",
		" D old/file.go",
		"R  old/name.go -> new/name.go",
		"UU conflicted/file.go",
		"?? scratch/output.log",
		"?? scratch/another.log",
	}, "\n") + "\n"

	result := Compact(Request{
		ToolName: "execute_code",
		Command:  "git status --short --branch",
		Stream:   "stdout",
		Text:     text,
		ExitCode: 0,
	}, Config{Mode: ModeAuto, ThresholdBytes: 32, HeadLines: 2, TailLines: 2})

	if !result.Applied {
		t.Fatal("Compact Applied = false, want true")
	}
	if result.Reducer != ReducerGitStatus {
		t.Fatalf("Reducer = %q, want %q", result.Reducer, ReducerGitStatus)
	}
	for _, want := range []string{
		"branch: development...origin/development [ahead 1]",
		"modified: 2",
		"added: 1",
		"deleted: 1",
		"renamed: 1",
		"untracked: 2",
		"conflicted: 1",
		"UU conflicted/file.go",
	} {
		if !strings.Contains(result.Text, want) {
			t.Fatalf("git status output missing %q:\n%s", want, result.Text)
		}
	}
	if !hasEvidence(result.Evidence, EvidenceGitStatusCounted) {
		t.Fatalf("evidence = %v, want %q", result.Evidence, EvidenceGitStatusCounted)
	}
}

func TestCompact_RawRequestedBypassesCompaction(t *testing.T) {
	raw := strings.Repeat("raw line\n", 100)

	result := Compact(Request{
		ToolName:     "execute_code",
		Command:      "go test ./...",
		Stream:       "stdout",
		Text:         raw,
		RawRequested: true,
	}, Config{Mode: ModeAuto, ThresholdBytes: 16})

	if result.Applied {
		t.Fatal("Compact Applied = true, want false")
	}
	if result.Text != raw {
		t.Fatal("raw bypass changed output")
	}
	if !hasEvidence(result.Evidence, EvidenceRawRequested) {
		t.Fatalf("evidence = %v, want %q", result.Evidence, EvidenceRawRequested)
	}
}

func TestCompact_UnderThresholdKeepsOriginalText(t *testing.T) {
	raw := "short output\n"

	result := Compact(Request{
		ToolName: "execute_code",
		Command:  "go test ./...",
		Stream:   "stdout",
		Text:     raw,
	}, Config{Mode: ModeAuto, ThresholdBytes: 1024})

	if result.Applied {
		t.Fatal("Compact Applied = true, want false")
	}
	if result.Text != raw {
		t.Fatal("under-threshold output changed")
	}
	if !hasEvidence(result.Evidence, EvidenceUnderThreshold) {
		t.Fatalf("evidence = %v, want %q", result.Evidence, EvidenceUnderThreshold)
	}
}

func hasEvidence(evidence []string, want string) bool {
	for _, got := range evidence {
		if got == want {
			return true
		}
	}
	return false
}

func BenchmarkCompactGoTestFailure(b *testing.B) {
	var text strings.Builder
	for i := 0; i < 1000; i++ {
		text.WriteString("ok  \tgithub.com/example/project/pkg\t0.001s\n")
	}
	text.WriteString("--- FAIL: TestWidgetHandlesOverflow (0.00s)\n")
	text.WriteString("    widget_test.go:42: got overflow=false, want true\n")
	text.WriteString("FAIL\n")
	text.WriteString("FAIL\tgithub.com/example/project/widget\t0.123s\n")
	input := text.String()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result := Compact(Request{
			ToolName: "execute_code",
			Command:  "go test ./...",
			Stream:   "stdout",
			Text:     input,
			ExitCode: 1,
		}, Config{Mode: ModeAuto})
		b.ReportMetric(float64(result.OriginalBytes), "original_bytes/op")
		b.ReportMetric(float64(result.CompactedBytes), "compacted_bytes/op")
	}
}

func BenchmarkCompactHeadTail(b *testing.B) {
	input := strings.Repeat("this is a noisy line of command output\n", 2000)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result := Compact(Request{
			ToolName: "execute_code",
			Command:  "custom command",
			Stream:   "stdout",
			Text:     input,
		}, Config{Mode: ModeAuto})
		b.ReportMetric(float64(result.OriginalBytes), "original_bytes/op")
		b.ReportMetric(float64(result.CompactedBytes), "compacted_bytes/op")
	}
}
