package parity

import (
	"reflect"
	"testing"
)

func TestFacadeSamplesDistributionWithManifestValidator(t *testing.T) {
	sample, err := SampleToolsetsFromDistribution("terminal_only", ToolsetDistributionSampleOptions{Random: sequenceRandom(0.99, 0.99)})
	if err != nil {
		t.Fatalf("SampleToolsetsFromDistribution: %v", err)
	}
	if got, want := sample.Toolsets, []string{"terminal", "file"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sampled toolsets = %v, want %v", got, want)
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
