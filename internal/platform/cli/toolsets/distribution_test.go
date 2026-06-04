package toolsets

import (
	"errors"
	"reflect"
	"testing"
)

func TestHermesToolsetDistributionFacade(t *testing.T) {
	distributions := ListToolsetDistributions()
	if got, want := len(distributions), 17; got != want {
		t.Fatalf("distribution count = %d, want %d", got, want)
	}

	for _, name := range []string{"default", "image_gen", "safe", "terminal_only"} {
		if _, ok := GetToolsetDistribution(name); !ok {
			t.Fatalf("missing distribution %q", name)
		}
	}

	imageGen, _ := GetToolsetDistribution("image_gen")
	if got, want := distributionWeights(imageGen), map[string]int{"image_gen": 90, "vision": 90, "web": 55, "terminal": 45, "moa": 10}; !reflect.DeepEqual(got, want) {
		t.Fatalf("image_gen weights = %v, want %v", got, want)
	}
}

func TestToolsetDistributionFacadeSampler(t *testing.T) {
	sample, err := SampleToolsetsFromDistribution("image_gen", ToolsetDistributionSampleOptions{
		Random: sequenceRandom(0.89, 0.91, 0.54, 0.44, 0.09),
	})
	if err != nil {
		t.Fatalf("SampleToolsetsFromDistribution: %v", err)
	}
	want := []string{"image_gen", "web", "terminal", "moa"}
	if !reflect.DeepEqual(sample.Toolsets, want) {
		t.Fatalf("sampled toolsets = %v, want %v", sample.Toolsets, want)
	}
}

func TestToolsetDistributionFacadeDegradedCases(t *testing.T) {
	unknown, err := SampleToolsetsFromDistribution("does_not_exist", ToolsetDistributionSampleOptions{Random: sequenceRandom(0)})
	if !errors.Is(err, ErrUnknownToolsetDistribution) {
		t.Fatalf("unknown distribution error = %v, want ErrUnknownToolsetDistribution", err)
	}
	assertDistributionIssue(t, unknown.Issues, ToolsetDistributionIssueUnknownDistribution, "does_not_exist")

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

func distributionWeights(distribution ToolsetDistribution) map[string]int {
	weights := make(map[string]int, len(distribution.Toolsets))
	for _, entry := range distribution.Toolsets {
		weights[entry.Toolset] = entry.Probability
	}
	return weights
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
