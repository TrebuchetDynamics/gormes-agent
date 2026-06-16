package toolsets

import (
	"errors"
	"reflect"
	"testing"
)

func TestHermesToolsetDistributionManifest(t *testing.T) {
	distributions := ListToolsetDistributions()
	if got, want := len(distributions), 17; got != want {
		t.Fatalf("distribution count = %d, want %d", got, want)
	}
	wantNames := []string{
		"default",
		"image_gen",
		"research",
		"science",
		"development",
		"safe",
		"balanced",
		"minimal",
		"terminal_only",
		"terminal_web",
		"creative",
		"reasoning",
		"browser_use",
		"browser_only",
		"browser_tasks",
		"terminal_tasks",
		"mixed_tasks",
	}
	if got := distributionNames(distributions); !reflect.DeepEqual(got, wantNames) {
		t.Fatalf("distribution names = %v, want %v", got, wantNames)
	}

	imageGen, ok := GetToolsetDistribution("image_gen")
	if !ok {
		t.Fatal("missing image_gen distribution")
	}
	if got, want := imageGen.Description, "Heavy focus on image generation with vision and web support"; got != want {
		t.Fatalf("image_gen description = %q, want %q", got, want)
	}
	if got, want := distributionWeights(imageGen), map[string]int{"image_gen": 90, "vision": 90, "web": 55, "terminal": 45, "moa": 10}; !reflect.DeepEqual(got, want) {
		t.Fatalf("image_gen weights = %v, want %v", got, want)
	}

	terminalOnly, ok := GetToolsetDistribution("terminal_only")
	if !ok {
		t.Fatal("missing terminal_only distribution")
	}
	if got, want := distributionWeights(terminalOnly), map[string]int{"terminal": 100, "file": 100}; !reflect.DeepEqual(got, want) {
		t.Fatalf("terminal_only weights = %v, want %v", got, want)
	}
}

func TestToolsetDistributionSamplerDeterministicSelection(t *testing.T) {
	sample, err := SampleToolsetsFromDistribution("image_gen", validToolsetOptions(0.89, 0.91, 0.54, 0.44, 0.09))
	if err != nil {
		t.Fatalf("SampleToolsetsFromDistribution: %v", err)
	}
	want := []string{"image_gen", "web", "terminal", "moa"}
	if !reflect.DeepEqual(sample.Toolsets, want) {
		t.Fatalf("sampled toolsets = %v, want %v", sample.Toolsets, want)
	}
	if len(sample.Issues) != 0 {
		t.Fatalf("issues = %#v, want none", sample.Issues)
	}
}

func TestToolsetDistributionSamplerNamedDistributions(t *testing.T) {
	cases := []struct {
		name  string
		rolls []float64
		want  []string
	}{
		{name: "default", rolls: []float64{0.99, 0.99, 0.99, 0.99, 0.99, 0.99, 0.99}, want: []string{"web", "vision", "image_gen", "terminal", "file", "moa", "browser"}},
		{name: "safe", rolls: []float64{0.79, 0.69, 0.59, 0.59, 0.49}, want: []string{"web", "browser", "vision", "image_gen", "moa"}},
		{name: "terminal_only", rolls: []float64{0.99, 0.99}, want: []string{"terminal", "file"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sample, err := SampleToolsetsFromDistribution(tc.name, validToolsetOptions(tc.rolls...))
			if err != nil {
				t.Fatalf("SampleToolsetsFromDistribution: %v", err)
			}
			if !reflect.DeepEqual(sample.Toolsets, tc.want) {
				t.Fatalf("sampled toolsets = %v, want %v", sample.Toolsets, tc.want)
			}
		})
	}
}

func TestToolsetDistributionSamplerUnknownDistribution(t *testing.T) {
	sample, err := SampleToolsetsFromDistribution("does_not_exist", ToolsetDistributionSampleOptions{Random: sequenceRandom(0)})
	if !errors.Is(err, ErrUnknownToolsetDistribution) {
		t.Fatalf("error = %v, want ErrUnknownToolsetDistribution", err)
	}
	if len(sample.Toolsets) != 0 {
		t.Fatalf("toolsets = %v, want none", sample.Toolsets)
	}
	assertDistributionIssue(t, sample.Issues, ToolsetDistributionIssueUnknownDistribution, "does_not_exist")
}

func TestToolsetDistributionSamplerSkipsInvalidToolsets(t *testing.T) {
	sample, err := SampleToolsetsFromDistribution("default", ToolsetDistributionSampleOptions{
		Random: sequenceRandom(0.99, 0.99, 0.99, 0.99, 0.99, 0.99, 0.99),
		ValidateToolset: func(name string) bool {
			return name != "vision"
		},
	})
	if err != nil {
		t.Fatalf("SampleToolsetsFromDistribution: %v", err)
	}
	if containsString(sample.Toolsets, "vision") {
		t.Fatalf("sampled invalid vision toolset: %v", sample.Toolsets)
	}
	assertDistributionIssue(t, sample.Issues, ToolsetDistributionIssueInvalidToolsetSkipped, "vision")
}

func TestToolsetDistributionSamplerFallsBackToHighestProbabilityValidToolset(t *testing.T) {
	sample, err := SampleToolsetsFromDistribution("image_gen", ToolsetDistributionSampleOptions{
		Random: sequenceRandom(0.99, 0.99, 0.99, 0.99, 0.99),
		ValidateToolset: func(name string) bool {
			return name != "image_gen"
		},
	})
	if err != nil {
		t.Fatalf("SampleToolsetsFromDistribution: %v", err)
	}
	if got, want := sample.Toolsets, []string{"vision"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("fallback toolsets = %v, want %v", got, want)
	}
	assertDistributionIssue(t, sample.Issues, ToolsetDistributionIssueInvalidToolsetSkipped, "image_gen")
	assertDistributionIssue(t, sample.Issues, ToolsetDistributionIssueFallbackSelected, "vision")
}

func distributionNames(distributions []ToolsetDistribution) []string {
	names := make([]string, 0, len(distributions))
	for _, distribution := range distributions {
		names = append(names, distribution.Name)
	}
	return names
}

func distributionWeights(distribution ToolsetDistribution) map[string]int {
	weights := make(map[string]int, len(distribution.Toolsets))
	for _, entry := range distribution.Toolsets {
		weights[entry.Toolset] = entry.Probability
	}
	return weights
}

func validToolsetOptions(rolls ...float64) ToolsetDistributionSampleOptions {
	return ToolsetDistributionSampleOptions{
		Random:          sequenceRandom(rolls...),
		ValidateToolset: func(string) bool { return true },
	}
}

func sequenceRandom(values ...float64) func() float64 {
	index := 0
	return func() float64 {
		if index >= len(values) {
			return 0
		}
		value := values[index]
		index++
		return value
	}
}

func assertDistributionIssue(t *testing.T, issues []ToolsetDistributionIssue, kind ToolsetDistributionIssueKind, toolset string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Kind == kind && issue.Toolset == toolset {
			return
		}
	}
	t.Fatalf("missing issue kind=%s toolset=%s in %#v", kind, toolset, issues)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
