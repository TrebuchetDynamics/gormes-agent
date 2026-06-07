package benchmarks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type BenchmarkOptions struct {
	Root             string
	Binary           string
	Now              func() time.Time
	GitCommit        func(string) (string, error)
	RuntimeBenchmark func(RuntimeBenchmarkOptions) (RuntimeBenchmarkResult, error)
}

type RuntimeBenchmarkOptions struct {
	Root    string
	Binary  string
	Timeout time.Duration
}

type RuntimeBenchmarkResult struct {
	PeakRSSKB int64
	Elapsed   time.Duration
}

var supportedBenchmarkPlatforms = []string{
	"linux/amd64",
	"linux/arm64",
	"darwin/amd64",
	"darwin/arm64",
	"windows/amd64",
	"windows/arm64",
	"android/arm64",
}

const benchmarkGoVersionFloor = "1.26+"

const (
	benchmarkPreWASICommit              = "2e54a1a60"
	benchmarkPreWASISizeBytes     int64 = 29311138
	benchmarkPostHTTPSTTCommit          = "d42a77042"
	benchmarkPostHTTPSTTSizeBytes       = 44261163
	benchmarkRuntimeTimeout             = 20 * time.Second
)

var benchmarkRuntimeArgs = []string{"doctor", "--offline", "--json"}

var benchmarkMirrorPaths = []string{
	filepath.Join("webpages", "docs", "data", "benchmarks.json"),
	filepath.Join("webpages", "landing", "src", "data", "benchmarks.json"),
	filepath.Join("webpages", "landing", "legacy", "go-renderer", "internal", "site", "data", "benchmarks.json"),
}

func RecordBenchmark(opts BenchmarkOptions) error {
	if opts.Root == "" {
		opts.Root = "."
	}
	if opts.Binary == "" {
		opts.Binary = filepath.Join(opts.Root, "bin", "gormes")
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.GitCommit == nil {
		opts.GitCommit = gitCommit
	}

	info, err := os.Stat(opts.Binary)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return nil
	}

	benchPath := filepath.Join(opts.Root, "benchmarks.json")
	bench := map[string]any{}
	if raw, err := os.ReadFile(benchPath); err == nil {
		if err := json.Unmarshal(raw, &bench); err != nil {
			return err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		bench = defaultBenchmarkSkeleton()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	commit, err := opts.GitCommit(opts.Root)
	if err != nil {
		return err
	}

	date := opts.Now().Format("2006-01-02")
	sizeMB := fmt.Sprintf("%.1f", float64(info.Size())/1048576)
	historySizeMB, err := strconv.ParseFloat(sizeMB, 64)
	if err != nil {
		return err
	}

	binary, _ := bench["binary"].(map[string]any)
	if binary == nil {
		binary = map[string]any{}
	}
	binary["size_bytes"] = info.Size()
	binary["size_mb"] = sizeMB
	binary["commit"] = commit
	binary["last_measured"] = date
	binary["go_version"] = benchmarkGoVersionFloor
	bench["binary"] = binary

	code := countCodeMetrics(opts.Root)
	bench["code"] = code
	updateSTTBenchmarkBinaryMetadata(bench, info.Size())
	updateRuntimeBenchmarkMetadata(bench, opts, date, commit)

	properties, _ := bench["properties"].(map[string]any)
	if properties == nil {
		properties = map[string]any{}
	}
	if _, ok := properties["cgo"]; !ok {
		properties["cgo"] = false
	}
	if _, ok := properties["dependencies"]; !ok {
		properties["dependencies"] = "zero (no dynamic library deps)"
	}
	properties["platforms"] = append([]string(nil), supportedBenchmarkPlatforms...)
	bench["properties"] = properties

	history, _ := bench["history"].([]any)
	if len(history) == 0 || historyDate(history[0]) != date {
		entry := map[string]any{
			"date":       date,
			"size_bytes": info.Size(),
			"size_mb":    historySizeMB,
			"commit":     commit,
			"phase":      currentPhase(opts.Root, history),
		}
		history = append([]any{entry}, history...)
	}
	bench["history"] = history

	raw, err := json.MarshalIndent(bench, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(benchPath, raw, 0o644); err != nil {
		return err
	}

	for _, rel := range benchmarkMirrorPaths {
		target := filepath.Join(opts.Root, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, raw, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func updateRuntimeBenchmarkMetadata(bench map[string]any, opts BenchmarkOptions, date, commit string) {
	runtimeBench, _ := bench["runtime"].(map[string]any)
	if runtimeBench == nil {
		runtimeBench = map[string]any{}
	}

	command := benchmarkCommandLabel(opts.Root, opts.Binary, benchmarkRuntimeArgs)
	entry := map[string]any{
		"command":       command,
		"last_measured": date,
		"commit":        commit,
		"goos":          stdruntime.GOOS,
		"goarch":        stdruntime.GOARCH,
		"isolated_home": true,
	}

	run := opts.RuntimeBenchmark
	if run == nil {
		run = runRuntimeBenchmark
	}
	result, err := run(RuntimeBenchmarkOptions{
		Root:    opts.Root,
		Binary:  opts.Binary,
		Timeout: benchmarkRuntimeTimeout,
	})
	if err != nil {
		entry["status"] = "skipped"
		entry["error"] = trimBenchmarkError(err)
		runtimeBench["offline_doctor"] = entry
		bench["runtime"] = runtimeBench
		return
	}

	entry["status"] = "measured"
	entry["peak_rss_kb"] = result.PeakRSSKB
	entry["peak_rss_mb"] = benchmarkRSSMegabytes(result.PeakRSSKB)
	entry["elapsed_ms"] = result.Elapsed.Milliseconds()
	runtimeBench["offline_doctor"] = entry
	bench["runtime"] = runtimeBench
}

func runRuntimeBenchmark(opts RuntimeBenchmarkOptions) (RuntimeBenchmarkResult, error) {
	if stdruntime.GOOS == "windows" {
		return RuntimeBenchmarkResult{}, fmt.Errorf("peak RSS benchmark unsupported on windows")
	}
	if opts.Binary == "" {
		return RuntimeBenchmarkResult{}, fmt.Errorf("binary path is required")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = benchmarkRuntimeTimeout
	}

	home, err := os.MkdirTemp("", "gormes-benchmark-home-*")
	if err != nil {
		return RuntimeBenchmarkResult{}, err
	}
	defer os.RemoveAll(home)

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, opts.Binary, benchmarkRuntimeArgs...)
	cmd.Dir = opts.Root
	cmd.Env = append(os.Environ(),
		"GORMES_HOME="+home,
		"NO_COLOR=1",
		"TERM=dumb",
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	start := time.Now()
	err = cmd.Run()
	elapsed := time.Since(start)
	if ctx.Err() == context.DeadlineExceeded {
		return RuntimeBenchmarkResult{}, fmt.Errorf("runtime benchmark timed out after %s", opts.Timeout)
	}
	if err != nil {
		return RuntimeBenchmarkResult{}, fmt.Errorf("runtime benchmark command failed: %w", err)
	}
	if cmd.ProcessState == nil {
		return RuntimeBenchmarkResult{}, fmt.Errorf("runtime benchmark process state unavailable")
	}
	usage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage)
	if !ok || usage == nil {
		return RuntimeBenchmarkResult{}, fmt.Errorf("runtime benchmark rusage unavailable")
	}
	peakRSSKB := maxRSSKilobytes(usage.Maxrss)
	if peakRSSKB <= 0 {
		return RuntimeBenchmarkResult{}, fmt.Errorf("runtime benchmark peak RSS unavailable")
	}
	return RuntimeBenchmarkResult{
		PeakRSSKB: peakRSSKB,
		Elapsed:   elapsed,
	}, nil
}

func maxRSSKilobytes(maxRSS int64) int64 {
	switch stdruntime.GOOS {
	case "darwin", "ios":
		return maxRSS / 1024
	default:
		return maxRSS
	}
}

func benchmarkRSSMegabytes(kilobytes int64) float64 {
	return math.Round(float64(kilobytes)/102.4) / 10
}

func benchmarkCommandLabel(root, binary string, args []string) string {
	label := binary
	if root != "" {
		if rel, err := filepath.Rel(root, binary); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			label = rel
		}
	}
	if label == "" {
		label = filepath.Base(binary)
	}
	parts := append([]string{filepath.ToSlash(label)}, args...)
	return strings.Join(parts, " ")
}

func trimBenchmarkError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return msg
}

func updateSTTBenchmarkBinaryMetadata(bench map[string]any, currentSizeBytes int64) {
	if currentSizeBytes <= 0 {
		return
	}
	stt, _ := bench["stt"].(map[string]any)
	if stt == nil {
		stt = map[string]any{}
	}
	wasi, _ := stt["wasi_whisper"].(map[string]any)
	if wasi == nil {
		wasi = map[string]any{}
	}
	wasi["current_binary_size_bytes"] = currentSizeBytes
	wasi["current_binary_size_mb"] = benchmarkMegabytes(currentSizeBytes)
	wasi["binary_size_delta_mb_vs_pre_wasi_baseline"] = benchmarkMegabytes(currentSizeBytes - benchmarkPreWASISizeBytes)
	wasi["pre_wasi_baseline"] = map[string]any{
		"commit":     benchmarkPreWASICommit,
		"size_bytes": benchmarkPreWASISizeBytes,
		"size_mb":    benchmarkMegabytes(benchmarkPreWASISizeBytes),
	}
	wasi["post_http_stt_baseline"] = map[string]any{
		"commit":     benchmarkPostHTTPSTTCommit,
		"size_bytes": benchmarkPostHTTPSTTSizeBytes,
		"size_mb":    benchmarkMegabytes(benchmarkPostHTTPSTTSizeBytes),
	}
	if _, ok := wasi["status"]; !ok {
		if models, _ := wasi["models"].([]any); len(models) > 0 {
			wasi["status"] = "measured"
		} else {
			wasi["status"] = "pending_measurement"
		}
	}
	stt["wasi_whisper"] = wasi
	bench["stt"] = stt
}

func benchmarkMegabytes(sizeBytes int64) float64 {
	return math.Round(float64(sizeBytes)/104857.6) / 10
}

func gitCommit(root string) (string, error) {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func defaultBenchmarkSkeleton() map[string]any {
	return map[string]any{
		"binary": map[string]any{
			"name":        "gormes",
			"path":        "bin/gormes",
			"build_flags": `CGO_ENABLED=0 -trimpath -ldflags="-s -w"`,
			"linker":      "static",
			"stripped":    true,
			"go_version":  benchmarkGoVersionFloor,
		},
		"code": map[string]any{
			"test_count":   0,
			"go_files":     0,
			"go_lines":     0,
			"dependencies": 0,
		},
		"properties": map[string]any{
			"cgo":          false,
			"dependencies": "zero (no dynamic library deps)",
			"platforms":    append([]string(nil), supportedBenchmarkPlatforms...),
		},
		"history": []any{},
	}
}

func historyDate(entry any) string {
	fields, ok := entry.(map[string]any)
	if !ok {
		return ""
	}
	date, _ := fields["date"].(string)
	return date
}

func currentPhase(root string, history []any) string {
	if phase := phaseFromArchPlan(filepath.Join(root, "webpages", "docs", "ARCH_PLAN.md")); phase != "" {
		return phase
	}
	if phase := phaseFromProgress(filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "progress.json")); phase != "" {
		return phase
	}
	for _, entry := range history {
		fields, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		phase, _ := fields["phase"].(string)
		if phase != "" {
			return phase
		}
	}
	return "unknown"
}

func phaseFromArchPlan(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "## Phase") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "## Phase"))
		if rest == "" {
			continue
		}
		return "Phase " + rest
	}
	return ""
}

func phaseFromProgress(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var progress struct {
		Phases map[string]struct {
			Name      string                          `json:"name"`
			Status    string                          `json:"status"`
			Subphases map[string]progressSubphaseJSON `json:"subphases"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(raw, &progress); err != nil {
		return ""
	}

	keys := make([]string, 0, len(progress.Phases))
	for key := range progress.Phases {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return phaseKeyLess(keys[i], keys[j])
	})

	for _, key := range keys {
		phase := progress.Phases[key]
		if phase.Name == "" {
			continue
		}
		if phaseIncomplete(phase.Status, phase.Subphases) {
			return phase.Name
		}
	}
	for i := len(keys) - 1; i >= 0; i-- {
		if name := progress.Phases[keys[i]].Name; name != "" {
			return name
		}
	}
	return ""
}

type progressSubphaseJSON struct {
	Status string `json:"status"`
	Items  []struct {
		Status string `json:"status"`
	} `json:"items"`
}

func phaseIncomplete(status string, subphases map[string]progressSubphaseJSON) bool {
	if len(subphases) == 0 {
		return status != "complete"
	}
	for _, subphase := range subphases {
		if len(subphase.Items) == 0 {
			if subphase.Status != "complete" {
				return true
			}
			continue
		}
		for _, item := range subphase.Items {
			if item.Status != "complete" {
				return true
			}
		}
	}
	return false
}

func phaseKeyLess(left, right string) bool {
	leftInt, leftErr := parsePhaseKey(left)
	rightInt, rightErr := parsePhaseKey(right)
	if leftErr == nil && rightErr == nil {
		return leftInt < rightInt
	}
	if leftErr == nil {
		return true
	}
	if rightErr == nil {
		return false
	}
	return left < right
}

func parsePhaseKey(key string) (int, error) {
	var n int
	_, err := fmt.Sscanf(key, "%d", &n)
	return n, err
}

func countCodeMetrics(root string) map[string]any {
	metrics := map[string]any{}

	testCount := 0
	cmd := exec.Command("go", "test", "./...", "-list", ".*")
	cmd.Dir = root
	out, _ := cmd.Output()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Test") {
			testCount++
		}
	}
	if testCount > 0 {
		metrics["test_count"] = testCount
	}

	goFiles := 0
	goLines := 0
	for _, dir := range []string{"internal", "cmd"} {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && (d.Name() == "testdata" || d.Name() == ".git") {
				return filepath.SkipDir
			}
			if strings.HasSuffix(d.Name(), ".go") && !strings.HasSuffix(d.Name(), "_test.go") {
				goFiles++
				if raw, err := os.ReadFile(path); err == nil {
					goLines += strings.Count(string(raw), "\n")
				}
			}
			return nil
		})
	}
	if goFiles > 0 {
		metrics["go_files"] = goFiles
	}
	if goLines > 0 {
		metrics["go_lines"] = goLines
	}

	depsRaw, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err == nil {
		deps := 0
		for _, line := range strings.Split(string(depsRaw), "\n") {
			if strings.TrimSpace(line) != "" {
				deps++
			}
		}
		if deps > 0 {
			metrics["dependencies"] = deps
		}
	}

	return metrics
}
