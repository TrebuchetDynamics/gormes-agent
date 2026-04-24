package autoloop

import "context"

type RunOptions struct {
	Config Config
	Runner Runner
	DryRun bool
}

type RunSummary struct {
	Candidates int
	Selected   []Candidate
}

func RunOnce(ctx context.Context, opts RunOptions) (RunSummary, error) {
	candidates, err := NormalizeCandidates(opts.Config.ProgressJSON, CandidateOptions{ActiveFirst: true})
	if err != nil {
		return RunSummary{}, err
	}

	selected := candidates
	if opts.Config.MaxAgents > 0 && opts.Config.MaxAgents < len(selected) {
		selected = selected[:opts.Config.MaxAgents]
	}

	summary := RunSummary{
		Candidates: len(candidates),
		Selected:   append([]Candidate(nil), selected...),
	}
	if opts.DryRun {
		return summary, nil
	}

	runner := opts.Runner
	if runner == nil {
		runner = ExecRunner{}
	}

	argv, err := BuildBackendCommand(opts.Config.Backend, opts.Config.Mode)
	if err != nil {
		return RunSummary{}, err
	}

	for range selected {
		result := runner.Run(ctx, Command{
			Name: argv[0],
			Args: argv[1:],
			Dir:  opts.Config.RepoRoot,
		})
		if result.Err != nil {
			return RunSummary{}, result.Err
		}
	}

	return summary, nil
}
