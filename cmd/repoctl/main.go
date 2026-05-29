// Command repoctl wraps the existing internal/progress/repoctl package as a standalone
// binary so operators and CI can update repo metadata without any autonomous
// loop executable.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/repoctl"
)

const usage = "usage: repoctl [--repo-root <path>] {benchmark record|readme update|progress seed <fleet|missing-all>|hermes-source-pairs validate|hermes-source-pairs sync-sha|hermes-source-pairs report|hermes-contract-inventory}"

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

func run(stdout, stderr io.Writer, args []string) error {
	args, root, err := resolveRepoRoot(args)
	if err != nil {
		return err
	}
	switch {
	case len(args) >= 1 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help"):
		_, err := fmt.Fprintln(stdout, usage)
		return err
	case len(args) == 2 && args[0] == "benchmark" && args[1] == "record":
		return repoctl.RecordBenchmark(repoctl.BenchmarkOptions{
			Root:   root,
			Binary: os.Getenv("BINARY_PATH"),
		})
	case len(args) == 2 && args[0] == "readme" && args[1] == "update":
		return repoctl.UpdateReadme(repoctl.ReadmeOptions{Root: root})
	case len(args) == 3 && args[0] == "progress" && args[1] == "seed":
		result, err := repoctl.SeedProgressRows(repoctl.ProgressSeedOptions{Root: root, Set: args[2]})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "repoctl: progress seed %s added=%d skipped=%d total_items=%d\n",
			result.Set, result.Added, result.Skipped, result.TotalItems)
		return err
	case len(args) >= 2 && args[0] == "hermes-source-pairs" && args[1] == "validate":
		opts, err := parseSourcePairOptions(root, args[2:])
		if err != nil {
			return err
		}
		result, err := repoctl.ValidateSourcePairs(opts)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "repoctl: hermes-source-pairs validate ok pairs=%d hermes_sha=%s\n",
			len(result.Manifest.Pairs), result.Manifest.HermesSHA)
		return err
	case len(args) >= 2 && args[0] == "hermes-source-pairs" && args[1] == "sync-sha":
		opts, err := parseSourcePairOptions(root, args[2:])
		if err != nil {
			return err
		}
		result, err := repoctl.SyncSourcePairsSHA(opts)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "repoctl: hermes-source-pairs sync-sha ok pairs=%d hermes_sha=%s changed=%d demoted=%d\n",
			len(result.Manifest.Pairs), result.Manifest.HermesSHA, len(result.ChangedHermesFiles), len(result.DemotedCovered))
		return err
	case len(args) >= 2 && args[0] == "hermes-source-pairs" && args[1] == "report":
		opts, err := parseSourcePairOptions(root, args[2:])
		if err != nil {
			return err
		}
		if err := repoctl.WriteSourcePairsReport(opts); err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, "repoctl: hermes-source-pairs report updated")
		return err
	case len(args) >= 1 && args[0] == "hermes-contract-inventory":
		opts, err := parseHermesContractInventoryOptions(root, args[1:])
		if err != nil {
			return err
		}
		result, err := repoctl.WriteHermesContractInventory(opts)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "repoctl: hermes-contract-inventory report updated json=%s markdown=%s hermes_sha=%s strict_failures=%d\n",
			result.JSONPath, result.MarkdownPath, result.Report.HermesSHA, result.Report.Summary.StrictFailures)
		if err != nil {
			return err
		}
		if opts.Strict && !result.Report.OK {
			return fmt.Errorf("repoctl: hermes-contract-inventory strict failed strict_failures=%d", result.Report.Summary.StrictFailures)
		}
		return nil
	default:
		return fmt.Errorf("%w\n%s", errParse, usage)
	}
}

func parseSourcePairOptions(root string, args []string) (repoctl.SourcePairOptions, error) {
	opts := repoctl.SourcePairOptions{
		Root:                root,
		RequireHighPriority: true,
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--allow-unmapped-high-priority":
			opts.RequireHighPriority = false
		case "--manifest":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%w: --manifest requires a value\n%s", errParse, usage)
			}
			opts.ManifestPath = args[i+1]
			i++
		case "--report":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%w: --report requires a value\n%s", errParse, usage)
			}
			opts.ReportPath = args[i+1]
			i++
		case "--hermes-src":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%w: --hermes-src requires a value\n%s", errParse, usage)
			}
			opts.HermesSrc = args[i+1]
			i++
		case "--hermes-sha":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%w: --hermes-sha requires a value\n%s", errParse, usage)
			}
			opts.CurrentHermesSHA = args[i+1]
			i++
		default:
			return opts, fmt.Errorf("%w: unknown hermes-source-pairs option %s\n%s", errParse, args[i], usage)
		}
	}
	return opts, nil
}

func parseHermesContractInventoryOptions(root string, args []string) (repoctl.HermesContractInventoryOptions, error) {
	opts := repoctl.HermesContractInventoryOptions{Root: root}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--progress":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%w: --progress requires a value\n%s", errParse, usage)
			}
			opts.ProgressPath = args[i+1]
			i++
		case "--hermes-src":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%w: --hermes-src requires a value\n%s", errParse, usage)
			}
			opts.HermesSrc = args[i+1]
			i++
		case "--source-pairs":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%w: --source-pairs requires a value\n%s", errParse, usage)
			}
			opts.SourcePairsPath = args[i+1]
			i++
		case "--json-out":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%w: --json-out requires a value\n%s", errParse, usage)
			}
			opts.JSONPath = args[i+1]
			i++
		case "--markdown-out":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%w: --markdown-out requires a value\n%s", errParse, usage)
			}
			opts.MarkdownPath = args[i+1]
			i++
		case "--hermes-sha":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%w: --hermes-sha requires a value\n%s", errParse, usage)
			}
			opts.CurrentHermesSHA = args[i+1]
			i++
		case "--strict":
			opts.Strict = true
		default:
			return opts, fmt.Errorf("%w: unknown hermes-contract-inventory option %s\n%s", errParse, args[i], usage)
		}
	}
	return opts, nil
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
