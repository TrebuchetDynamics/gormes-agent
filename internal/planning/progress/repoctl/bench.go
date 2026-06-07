package repoctl

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/repoctl/benchmarks"

type BenchmarkOptions = benchmarks.BenchmarkOptions
type RuntimeBenchmarkOptions = benchmarks.RuntimeBenchmarkOptions
type RuntimeBenchmarkResult = benchmarks.RuntimeBenchmarkResult

func RecordBenchmark(opts BenchmarkOptions) error {
	return benchmarks.RecordBenchmark(opts)
}
