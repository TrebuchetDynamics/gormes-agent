package repoctl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type ReadmeOptions struct {
	Root string
}

func UpdateReadme(opts ReadmeOptions) error {
	if opts.Root == "" {
		return fmt.Errorf("repo root is required")
	}
	benchPath := filepath.Join(opts.Root, "benchmarks.json")
	raw, err := os.ReadFile(benchPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "update-readme: benchmarks.json not found; skipping")
			return nil
		}
		return err
	}
	var data struct {
		Binary struct {
			SizeMB       json.RawMessage `json:"size_mb"`
			LastMeasured string          `json:"last_measured"`
		} `json:"binary"`
		Code struct {
			TestCount    json.RawMessage `json:"test_count"`
			GoFiles      json.RawMessage `json:"go_files"`
			GoLines      json.RawMessage `json:"go_lines"`
			Dependencies json.RawMessage `json:"dependencies"`
		} `json:"code"`
		STT struct {
			WASIWhisper struct {
				LastMeasured string `json:"last_measured"`
				Models       []struct {
					Name           string          `json:"name"`
					RealtimeFactor json.RawMessage `json:"realtime_factor"`
				} `json:"models"`
			} `json:"wasi_whisper"`
		} `json:"stt"`
		Runtime struct {
			OfflineDoctor struct {
				Status       string          `json:"status"`
				LastMeasured string          `json:"last_measured"`
				PeakRSSMB    json.RawMessage `json:"peak_rss_mb"`
			} `json:"offline_doctor"`
		} `json:"runtime"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}

	readmePath := filepath.Join(opts.Root, "README.md")
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}
	content := string(readme)

	re := regexp.MustCompile(`~[0-9.]+ MB`)
	if sizeMB, err := benchmarkSizeMB(data.Binary.SizeMB); err == nil && sizeMB != "" {
		content = re.ReplaceAllString(content, "~"+sizeMB+" MB")
	}

	if testCount, err := intVal(data.Code.TestCount); err == nil && testCount > 0 {
		re := regexp.MustCompile(`[0-9,]+\+ tests`)
		content = re.ReplaceAllString(content, fmt.Sprintf("%d+ tests", testCount))
	}

	if goFiles, err := intVal(data.Code.GoFiles); err == nil && goFiles > 0 {
		re := regexp.MustCompile(`[0-9,]+\+ Go source files`)
		content = re.ReplaceAllString(content, fmt.Sprintf("%d+ Go source files", goFiles))
	}

	if goLines, err := intVal(data.Code.GoLines); err == nil && goLines > 0 {
		re := regexp.MustCompile(`[0-9,]+ lines of Go`)
		content = re.ReplaceAllString(content, fmt.Sprintf("%d lines of Go", goLines))
	}

	if deps, err := intVal(data.Code.Dependencies); err == nil && deps > 0 {
		re := regexp.MustCompile(`[0-9,]+ dependencies`)
		content = re.ReplaceAllString(content, fmt.Sprintf("%d dependencies", deps))
	}

	release := readReleaseMetadata(opts.Root)
	if release.Tag != "" {
		content = updateReleaseMetadata(content, release, data.Binary.SizeMB, data.Binary.LastMeasured)
	}
	if summary := sttBenchmarkSummary(data.STT.WASIWhisper.LastMeasured, data.STT.WASIWhisper.Models); summary != "" {
		content = updateSTTBenchmarkSummary(content, summary)
	}
	if summary := runtimeBenchmarkSummary(data.Runtime.OfflineDoctor.Status, data.Runtime.OfflineDoctor.LastMeasured, data.Runtime.OfflineDoctor.PeakRSSMB); summary != "" {
		content = updateRuntimeBenchmarkSummary(content, summary)
	}

	return os.WriteFile(readmePath, []byte(content), 0o644)
}

type releaseMetadata struct {
	Tag string
	URL string
}

func readReleaseMetadata(root string) releaseMetadata {
	raw, err := os.ReadFile(filepath.Join(root, "webpages", "landing", "src", "data", "release.json"))
	if err == nil {
		var data struct {
			Version string `json:"version"`
			Tag     string `json:"tag"`
			URL     string `json:"url"`
		}
		if json.Unmarshal(raw, &data) == nil {
			tag := data.Tag
			if tag == "" && data.Version != "" {
				tag = "v" + data.Version
			}
			if tag != "" {
				url := data.URL
				if url == "" {
					url = "https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/" + tag
				}
				return releaseMetadata{Tag: tag, URL: url}
			}
		}
	}

	raw, err = os.ReadFile(filepath.Join(root, "cmd", "gormes", "version.go"))
	if err != nil {
		return releaseMetadata{}
	}
	match := regexp.MustCompile(`var\s+Version\s*=\s*"([^"]+)"`).FindStringSubmatch(string(raw))
	if len(match) != 2 {
		return releaseMetadata{}
	}
	tag := "v" + match[1]
	return releaseMetadata{
		Tag: tag,
		URL: "https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/" + tag,
	}
}

func updateReleaseMetadata(content string, release releaseMetadata, sizeRaw json.RawMessage, measured string) string {
	latest := regexp.MustCompile(`Latest public release: \[[^\]]+\]\([^)]+\)\.`)
	content = latest.ReplaceAllString(content, fmt.Sprintf("Latest public release: [%s](%s).", release.Tag, release.URL))

	devHead := regexp.MustCompile("Current `development` head after `[^`]+`")
	content = devHead.ReplaceAllString(content, fmt.Sprintf("Current `development` head after `%s`", release.Tag))

	size, err := benchmarkSizeMB(sizeRaw)
	if err != nil || size == "" || measured == "" {
		return content
	}

	summary := fmt.Sprintf("Release %s publishes static Go binaries for Linux, macOS, Windows, and Termux/Android across the supported release matrix. The current benchmark mirror reports a Linux build at ~%s MB (`benchmarks.json`, %s).",
		release.Tag, size, measured)
	releaseSummary := regexp.MustCompile("Release v[0-9][^\\n]+?\\. The current benchmark mirror reports a Linux build at ~[0-9.]+ MB \\(`benchmarks\\.json`, [0-9-]+\\)\\.")
	return releaseSummary.ReplaceAllString(content, summary)
}

func sttBenchmarkSummary(measured string, models []struct {
	Name           string          `json:"name"`
	RealtimeFactor json.RawMessage `json:"realtime_factor"`
}) string {
	if measured == "" {
		return ""
	}
	for _, model := range models {
		if model.Name != "ggml-tiny.en" {
			continue
		}
		factor, err := benchmarkFloat(model.RealtimeFactor)
		if err != nil || factor <= 0 {
			return ""
		}
		return fmt.Sprintf("WASI Whisper tiny.en runs at %sx realtime (`benchmarks.json`, %s).", formatBenchmarkFloat(factor), measured)
	}
	return ""
}

func updateSTTBenchmarkSummary(content string, summary string) string {
	stale := regexp.MustCompile(`WASI Whisper tiny\.en (?:has not been benchmarked yet|runs at [0-9.]+x realtime \(` + "`" + `benchmarks\.json` + "`" + `, [0-9-]+\))\.`)
	if stale.MatchString(content) {
		return stale.ReplaceAllString(content, summary)
	}
	statusLine := regexp.MustCompile(`(The current Linux build measures ~[0-9.]+ MB \(` + "`" + `benchmarks\.json` + "`" + `\)\.)`)
	return statusLine.ReplaceAllString(content, "${1} "+summary)
}

func runtimeBenchmarkSummary(status, measured string, peakRaw json.RawMessage) string {
	if status != "measured" || measured == "" {
		return ""
	}
	peak, err := benchmarkFloat(peakRaw)
	if err != nil || peak <= 0 {
		return ""
	}
	return fmt.Sprintf("Offline doctor peaks at ~%s MB RSS (`benchmarks.json`, %s).", formatBenchmarkFloat(peak), measured)
}

func updateRuntimeBenchmarkSummary(content string, summary string) string {
	stale := regexp.MustCompile(`Offline doctor peaks at ~[0-9.]+ MB RSS \(` + "`" + `benchmarks\.json` + "`" + `, [0-9-]+\)\.`)
	if stale.MatchString(content) {
		return stale.ReplaceAllString(content, summary)
	}
	sttLine := regexp.MustCompile(`(WASI Whisper tiny\.en runs at [0-9.]+x realtime \(` + "`" + `benchmarks\.json` + "`" + `, [0-9-]+\)\.)`)
	if sttLine.MatchString(content) {
		return sttLine.ReplaceAllString(content, "${1} "+summary)
	}
	statusLine := regexp.MustCompile(`(The current Linux build measures ~[0-9.]+ MB \(` + "`" + `benchmarks\.json` + "`" + `\)\.)`)
	return statusLine.ReplaceAllString(content, "${1} "+summary)
}

func benchmarkSizeMB(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value, nil
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return strconv.FormatFloat(number, 'f', -1, 64), nil
	}
	return "", fmt.Errorf("benchmarks.json binary.size_mb has unsupported type")
}

func benchmarkFloat(raw json.RawMessage) (float64, error) {
	if len(raw) == 0 {
		return 0, nil
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return number, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strconv.ParseFloat(text, 64)
	}
	return 0, fmt.Errorf("benchmarks.json value has unsupported type")
}

func formatBenchmarkFloat(value float64) string {
	text := strconv.FormatFloat(value, 'f', 2, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	return text
}

func intVal(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return 0, nil
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, err
	}
	return int(n), nil
}
