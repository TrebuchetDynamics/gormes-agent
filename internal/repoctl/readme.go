package repoctl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
			SizeMB json.RawMessage `json:"size_mb"`
		} `json:"binary"`
		Code struct {
			TestCount    json.RawMessage `json:"test_count"`
			GoFiles      json.RawMessage `json:"go_files"`
			GoLines      json.RawMessage `json:"go_lines"`
			Dependencies json.RawMessage `json:"dependencies"`
		} `json:"code"`
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

	return os.WriteFile(readmePath, []byte(content), 0o644)
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
