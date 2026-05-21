package repoctl

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	sourcePairsManifestRel       = "webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json"
	sourcePairsLegacyManifestRel = "docs/content/building-gormes/architecture_plan/hermes-source-pairs.json"
	sourcePairsReportRel         = "webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.md"
	sourcePairsLegacyReportRel   = "docs/content/building-gormes/architecture_plan/hermes-source-pairs.md"
)

var highPriorityHermesSources = []string{
	"hermes_cli/default_soul.py",
	"hermes_cli/config.py",
	"hermes_cli/profiles.py",
	"agent/prompt_builder.py",
	"hermes_cli/main.py",
	"hermes_cli/commands.py",
	"hermes_cli/auth_commands.py",
	"gateway/run.py",
	"tools/skills_tool.py",
	"tools/skill_manager_tool.py",
	"tools/skills_sync.py",
	"run_agent.py",
	"cli.py",
}

type SourcePairOptions struct {
	Root                string
	ManifestPath        string
	ReportPath          string
	HermesSrc           string
	CurrentHermesSHA    string
	RequireHighPriority bool
}

type SourcePairsManifest struct {
	SchemaVersion string       `json:"schema_version"`
	HermesSHA     string       `json:"hermes_sha"`
	Pairs         []SourcePair `json:"pairs"`
}

type SourcePair struct {
	HermesFile           string   `json:"hermes_file"`
	GormesTargets        []string `json:"gormes_targets,omitempty"`
	Status               string   `json:"status"`
	Contract             string   `json:"contract"`
	Tests                []string `json:"tests,omitempty"`
	ProgressRows         []string `json:"progress_rows,omitempty"`
	UpstreamTests        []string `json:"upstream_tests,omitempty"`
	LastCheckedHermesSHA string   `json:"last_checked_hermes_sha"`
	Notes                string   `json:"notes,omitempty"`
}

type SourcePairsValidation struct {
	Manifest         SourcePairsManifest
	CurrentHermesSHA string
	Counts           map[string]int
	UnmappedHigh     []string
}

type SourcePairsSyncResult struct {
	Manifest           SourcePairsManifest
	CurrentHermesSHA   string
	ChangedHermesFiles []string
	DemotedCovered     []string
}

func ValidateSourcePairs(opts SourcePairOptions) (SourcePairsValidation, error) {
	manifest, err := loadSourcePairsManifest(opts)
	if err != nil {
		return SourcePairsValidation{}, err
	}
	root := sourcePairRoot(opts)
	hermesSrc := sourcePairHermesSrc(root, opts)
	currentSHA := opts.CurrentHermesSHA
	if currentSHA == "" && sourcePairIsGitCheckout(hermesSrc) {
		currentSHA, _ = sourcePairGitSHA(hermesSrc)
	}

	var errs []error
	if manifest.SchemaVersion == "" {
		errs = append(errs, errors.New("source-pairs: schema_version is required"))
	}
	if manifest.HermesSHA == "" {
		errs = append(errs, errors.New("source-pairs: hermes_sha is required"))
	}
	if currentSHA != "" && manifest.HermesSHA != "" && !shaMatches(currentSHA, manifest.HermesSHA) {
		errs = append(errs, fmt.Errorf("source-pairs: stale Hermes SHA manifest=%s current=%s", manifest.HermesSHA, currentSHA))
	}
	if len(manifest.Pairs) == 0 {
		errs = append(errs, errors.New("source-pairs: pairs must not be empty"))
	}

	seen := map[string]bool{}
	counts := map[string]int{}
	for i, pair := range manifest.Pairs {
		prefix := fmt.Sprintf("source-pairs: pair[%d] %s:", i, pair.HermesFile)
		if pair.HermesFile == "" {
			errs = append(errs, fmt.Errorf("%s hermes_file is required", prefix))
			continue
		}
		if seen[pair.HermesFile] {
			errs = append(errs, fmt.Errorf("%s duplicate hermes_file", prefix))
		}
		seen[pair.HermesFile] = true
		counts[pair.Status]++

		if !validSourcePairStatus(pair.Status) {
			errs = append(errs, fmt.Errorf("%s invalid status %q", prefix, pair.Status))
		}
		if strings.TrimSpace(pair.Contract) == "" {
			errs = append(errs, fmt.Errorf("%s contract is required", prefix))
		}
		if strings.TrimSpace(pair.LastCheckedHermesSHA) == "" {
			errs = append(errs, fmt.Errorf("%s last_checked_hermes_sha is required", prefix))
		}
		if currentSHA != "" && pair.LastCheckedHermesSHA != "" && !shaMatches(currentSHA, pair.LastCheckedHermesSHA) {
			errs = append(errs, fmt.Errorf("%s stale Hermes SHA row=%s current=%s", prefix, pair.LastCheckedHermesSHA, currentSHA))
		}
		if !sourcePairPathExists(filepath.Join(hermesSrc, filepath.FromSlash(pair.HermesFile))) {
			errs = append(errs, fmt.Errorf("%s missing Hermes file", prefix))
		}
		for _, test := range pair.UpstreamTests {
			if !sourcePairPathExists(filepath.Join(hermesSrc, filepath.FromSlash(test))) {
				errs = append(errs, fmt.Errorf("%s missing upstream test %q", prefix, test))
			}
		}
		for _, target := range pair.GormesTargets {
			if !sourcePairPathExists(filepath.Join(root, filepath.FromSlash(target))) {
				errs = append(errs, fmt.Errorf("%s missing Gormes target %q", prefix, target))
			}
		}
		if pair.Status == "covered" {
			if len(pair.GormesTargets) == 0 {
				errs = append(errs, fmt.Errorf("%s covered row requires gormes_targets", prefix))
			}
			if len(pair.Tests) == 0 {
				errs = append(errs, fmt.Errorf("%s covered row requires tests", prefix))
			}
		}
	}

	var unmappedHigh []string
	if opts.RequireHighPriority {
		for _, source := range highPriorityHermesSources {
			if !sourcePairPathExists(filepath.Join(hermesSrc, filepath.FromSlash(source))) {
				continue
			}
			if !seen[source] {
				unmappedHigh = append(unmappedHigh, source)
				errs = append(errs, fmt.Errorf("source-pairs: high-priority Hermes file is unmapped: %s", source))
			}
		}
	}

	sortSourcePairs(manifest.Pairs)
	validation := SourcePairsValidation{
		Manifest:         manifest,
		CurrentHermesSHA: currentSHA,
		Counts:           counts,
		UnmappedHigh:     unmappedHigh,
	}
	return validation, errors.Join(errs...)
}

func RenderSourcePairsReport(opts SourcePairOptions) (string, error) {
	validation, err := ValidateSourcePairs(opts)
	if err != nil {
		return "", err
	}
	manifest := validation.Manifest
	var b strings.Builder
	b.WriteString("# Hermes Source Pairs\n\n")
	fmt.Fprintf(&b, "- Hermes SHA: `%s`\n", manifest.HermesSHA)
	if validation.CurrentHermesSHA != "" {
		fmt.Fprintf(&b, "- Current checkout SHA: `%s`\n", validation.CurrentHermesSHA)
	}
	fmt.Fprintf(&b, "- Source pairs: `%d`\n\n", len(manifest.Pairs))

	b.WriteString("## Status Counts\n\n")
	for _, status := range []string{"covered", "partial", "planned", "owned", "excluded"} {
		if n := validation.Counts[status]; n > 0 {
			fmt.Fprintf(&b, "- `%s`: %d\n", status, n)
		}
	}
	b.WriteString("\n## Source Pair Table\n\n")
	b.WriteString("| Hermes file | Status | Gormes targets | Tests | Progress rows | Contract |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, pair := range manifest.Pairs {
		fmt.Fprintf(&b, "| `%s` | `%s` | %s | %s | %s | %s |\n",
			mdCell(pair.HermesFile),
			mdCell(pair.Status),
			mdList(pair.GormesTargets),
			mdList(pair.Tests),
			mdList(pair.ProgressRows),
			mdCell(pair.Contract),
		)
	}
	return b.String(), nil
}

func WriteSourcePairsReport(opts SourcePairOptions) error {
	report, err := RenderSourcePairsReport(opts)
	if err != nil {
		return err
	}
	root := sourcePairRoot(opts)
	path := opts.ReportPath
	if path == "" {
		path = sourcePairDefaultPath(root, sourcePairsReportRel, sourcePairsLegacyReportRel)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(report), 0o644)
}

func SyncSourcePairsSHA(opts SourcePairOptions) (SourcePairsSyncResult, error) {
	manifest, err := loadSourcePairsManifest(opts)
	if err != nil {
		return SourcePairsSyncResult{}, err
	}
	root := sourcePairRoot(opts)
	hermesSrc := sourcePairHermesSrc(root, opts)
	currentSHA := opts.CurrentHermesSHA
	if currentSHA == "" {
		currentSHA, err = sourcePairGitSHA(hermesSrc)
		if err != nil {
			return SourcePairsSyncResult{}, fmt.Errorf("source-pairs: resolve Hermes SHA: %w", err)
		}
	}
	if strings.TrimSpace(currentSHA) == "" {
		return SourcePairsSyncResult{}, errors.New("source-pairs: current Hermes SHA is required")
	}

	changedSet := map[string]bool{}
	var changed []string
	if manifest.HermesSHA != "" && !shaMatches(currentSHA, manifest.HermesSHA) && sourcePairIsGitCheckout(hermesSrc) {
		changed, err = sourcePairChangedFiles(hermesSrc, manifest.HermesSHA, currentSHA)
		if err != nil {
			return SourcePairsSyncResult{}, err
		}
		for _, path := range changed {
			changedSet[path] = true
		}
	}

	var demoted []string
	oldSHA := manifest.HermesSHA
	manifest.HermesSHA = currentSHA
	for i := range manifest.Pairs {
		pair := &manifest.Pairs[i]
		if pair.Status == "covered" && changedSet[pair.HermesFile] {
			pair.Status = "partial"
			demoted = append(demoted, pair.HermesFile)
			pair.Notes = appendSourcePairNote(pair.Notes, fmt.Sprintf("Phase 0 update: upstream source changed between %s and %s; coverage requires review.", shortSHA(oldSHA), shortSHA(currentSHA)))
		}
		pair.LastCheckedHermesSHA = currentSHA
	}
	sortSourcePairs(manifest.Pairs)

	if err := writeSourcePairsManifest(opts, manifest); err != nil {
		return SourcePairsSyncResult{}, err
	}
	validation, err := ValidateSourcePairs(SourcePairOptions{
		Root:                opts.Root,
		ManifestPath:        opts.ManifestPath,
		ReportPath:          opts.ReportPath,
		HermesSrc:           opts.HermesSrc,
		CurrentHermesSHA:    currentSHA,
		RequireHighPriority: opts.RequireHighPriority,
	})
	if err != nil {
		return SourcePairsSyncResult{}, err
	}
	return SourcePairsSyncResult{
		Manifest:           validation.Manifest,
		CurrentHermesSHA:   currentSHA,
		ChangedHermesFiles: changed,
		DemotedCovered:     demoted,
	}, nil
}

func loadSourcePairsManifest(opts SourcePairOptions) (SourcePairsManifest, error) {
	path := sourcePairManifestPath(opts)
	raw, err := os.ReadFile(path)
	if err != nil {
		return SourcePairsManifest{}, fmt.Errorf("source-pairs: read manifest: %w", err)
	}
	var manifest SourcePairsManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return SourcePairsManifest{}, fmt.Errorf("source-pairs: parse manifest: %w", err)
	}
	return manifest, nil
}

func writeSourcePairsManifest(opts SourcePairOptions, manifest SourcePairsManifest) error {
	path := sourcePairManifestPath(opts)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("source-pairs: marshal manifest: %w", err)
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func sourcePairManifestPath(opts SourcePairOptions) string {
	root := sourcePairRoot(opts)
	if opts.ManifestPath != "" {
		return opts.ManifestPath
	}
	return sourcePairDefaultPath(root, sourcePairsManifestRel, sourcePairsLegacyManifestRel)
}

func sourcePairDefaultPath(root, primaryRel, legacyRel string) string {
	primary := filepath.Join(root, primaryRel)
	if sourcePairPathExists(primary) {
		return primary
	}
	legacy := filepath.Join(root, legacyRel)
	if sourcePairPathExists(legacy) {
		return legacy
	}
	return primary
}

func sortSourcePairs(pairs []SourcePair) {
	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].HermesFile < pairs[j].HermesFile
	})
}

func validSourcePairStatus(status string) bool {
	switch status {
	case "covered", "partial", "planned", "owned", "excluded":
		return true
	default:
		return false
	}
}

func sourcePairRoot(opts SourcePairOptions) string {
	if opts.Root != "" {
		return opts.Root
	}
	return "."
}

func sourcePairHermesSrc(root string, opts SourcePairOptions) string {
	if opts.HermesSrc != "" {
		return opts.HermesSrc
	}
	return filepath.Join(root, "hermes-agent")
}

func sourcePairGitSHA(dir string) (string, error) {
	if !sourcePairIsGitCheckout(dir) {
		return "", fmt.Errorf("%s is not a git checkout root", dir)
	}
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func sourcePairChangedFiles(dir, oldSHA, currentSHA string) ([]string, error) {
	out, err := exec.Command("git", "-C", dir, "diff", "--name-only", oldSHA+".."+currentSHA, "--").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("source-pairs: list changed Hermes files %s..%s: %w\n%s", oldSHA, currentSHA, err, out)
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, filepath.ToSlash(line))
		}
	}
	sort.Strings(files)
	return files, nil
}

func sourcePairIsGitCheckout(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func sourcePairPathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func shaMatches(current, recorded string) bool {
	current = strings.TrimSpace(current)
	recorded = strings.TrimSpace(recorded)
	if current == "" || recorded == "" {
		return true
	}
	return current == recorded || strings.HasPrefix(current, recorded) || strings.HasPrefix(recorded, current)
}

func shortSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) <= 12 {
		return sha
	}
	return sha[:12]
}

func appendSourcePairNote(note, addition string) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return addition
	}
	if strings.Contains(note, addition) {
		return note
	}
	return note + " " + addition
}

func mdList(items []string) string {
	if len(items) == 0 {
		return "`none`"
	}
	escaped := make([]string, 0, len(items))
	for _, item := range items {
		escaped = append(escaped, "`"+mdCell(item)+"`")
	}
	return strings.Join(escaped, "<br>")
}

func mdCell(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '|':
			b.WriteString(`\|`)
		case '[':
			b.WriteString("&#91;")
		case ']':
			b.WriteString("&#93;")
		case '\n', '\r', '\t':
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
