// Command progress is a thin wrapper around internal/progress/ctl: it
// validates the canonical progress.json and regenerates the markered docs
// the skill-driven planning/building workflow reads from.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/ctl"
)

const usage = "usage: progress [--repo-root <path>] {validate [--format text|json]|write [--dry-run]|compact|split <dir>|emit|list --module <module>|next-work [--repo-only]}"

var errParse = errors.New("parse error")

func main() {
	if err := run(os.Stdout, os.Stderr, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if errors.Is(err, errParse) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

type progressCommand struct {
	Run func(stdout io.Writer, root string, args []string) error
}

var progressCommands = map[string]progressCommand{
	"validate": {Run: func(stdout io.Writer, root string, args []string) error {
		format, err := parseFormat(args)
		if err != nil {
			return err
		}
		return progressctl.Validate(stdout, root, format)
	}},
	"write": {Run: func(stdout io.Writer, root string, args []string) error {
		opts, err := parseWriteOptions(args)
		if err != nil {
			return err
		}
		return progressctl.WriteWithOptions(stdout, root, opts)
	}},
	"compact": {Run: func(stdout io.Writer, root string, args []string) error {
		if err := requireNoArgs(args); err != nil {
			return err
		}
		return progressctl.Compact(stdout, root)
	}},
	"split": {Run: func(stdout io.Writer, root string, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("%w\n%s", errParse, usage)
		}
		return progressctl.Split(stdout, root, args[0])
	}},
	"emit": {Run: func(stdout io.Writer, root string, args []string) error {
		if err := requireNoArgs(args); err != nil {
			return err
		}
		return progressctl.Emit(stdout, root)
	}},
	"list": {Run: func(stdout io.Writer, root string, args []string) error {
		opts, err := parseListOptions(args)
		if err != nil {
			return err
		}
		return progressctl.List(stdout, root, opts)
	}},
	"next-work": {Run: func(stdout io.Writer, root string, args []string) error {
		opts, err := parseNextWorkOptions(args)
		if err != nil {
			return err
		}
		return progressctl.NextWorkWithOptions(stdout, root, opts)
	}},
}

func run(stdout, stderr io.Writer, args []string) error {
	args, root, err := resolveRepoRoot(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("%w\n%s", errParse, usage)
	}
	if isHelp(args[0]) {
		_, err := fmt.Fprintln(stdout, usage)
		return err
	}
	cmd, ok := progressCommands[args[0]]
	if !ok {
		return fmt.Errorf("%w\n%s", errParse, usage)
	}
	return cmd.Run(stdout, root, args[1:])
}

func isHelp(arg string) bool {
	return arg == "--help" || arg == "-h" || arg == "help"
}

func requireNoArgs(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("%w\n%s", errParse, usage)
	}
	return nil
}

func resolveRepoRoot(args []string) ([]string, string, error) {
	out := make([]string, 0, len(args))
	root := os.Getenv("REPO_ROOT")
	for i := 0; i < len(args); i++ {
		if args[i] == "--repo-root" {
			if i+1 >= len(args) {
				return nil, "", fmt.Errorf("%w: --repo-root requires a value\n%s", errParse, usage)
			}
			root = args[i+1]
			i++
			continue
		}
		out = append(out, args[i])
	}
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", err
		}
		root = cwd
	}
	return out, root, nil
}

func parseFormat(args []string) (string, error) {
	format := "text"
	for i := 0; i < len(args); i++ {
		if args[i] == "--format" {
			if i+1 >= len(args) {
				return "", fmt.Errorf("%w: --format requires a value\n%s", errParse, usage)
			}
			switch args[i+1] {
			case "text", "json":
				format = args[i+1]
			default:
				return "", fmt.Errorf("%w: --format must be text or json (got %q)\n%s",
					errParse, args[i+1], usage)
			}
			i++
			continue
		}
		return "", fmt.Errorf("%w: unexpected argument %q\n%s", errParse, args[i], usage)
	}
	return format, nil
}

func parseWriteOptions(args []string) (progressctl.WriteOptions, error) {
	var opts progressctl.WriteOptions
	for _, arg := range args {
		switch arg {
		case "--dry-run":
			opts.DryRun = true
		default:
			return opts, fmt.Errorf("%w: unexpected argument %q\n%s", errParse, arg, usage)
		}
	}
	return opts, nil
}

func parseNextWorkOptions(args []string) (progressctl.NextWorkOptions, error) {
	var opts progressctl.NextWorkOptions
	for _, arg := range args {
		switch arg {
		case "--repo-only":
			opts.RepoOnly = true
		default:
			return opts, fmt.Errorf("%w: unexpected argument %q\n%s", errParse, arg, usage)
		}
	}
	return opts, nil
}

func parseListOptions(args []string) (progressctl.ListOptions, error) {
	var opts progressctl.ListOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--module":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%w: --module requires a value\n%s", errParse, usage)
			}
			opts.Module = args[i+1]
			i++
		default:
			return opts, fmt.Errorf("%w: unexpected argument %q\n%s", errParse, args[i], usage)
		}
	}
	if opts.Module == "" {
		return opts, fmt.Errorf("%w: list requires --module <module>\n%s", errParse, usage)
	}
	return opts, nil
}
