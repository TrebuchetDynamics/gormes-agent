package main

import (
	"fmt"
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/internal/repoctl"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if len(args) == 2 && args[0] == "benchmark" && args[1] == "record" {
		return repoctl.RecordBenchmark(repoctl.BenchmarkOptions{Root: root})
	}
	if len(args) == 2 && args[0] == "progress" && args[1] == "sync" {
		return repoctl.SyncProgress(repoctl.ProgressOptions{Root: root})
	}
	if len(args) == 2 && args[0] == "readme" && args[1] == "update" {
		return repoctl.UpdateReadme(repoctl.ReadmeOptions{Root: root})
	}
	return fmt.Errorf("usage: repoctl benchmark record | progress sync | readme update")
}
