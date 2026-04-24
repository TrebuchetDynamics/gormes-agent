package repoctl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
			SizeMB string `json:"size_mb"`
		} `json:"binary"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	if data.Binary.SizeMB == "" {
		return fmt.Errorf("benchmarks.json missing binary.size_mb")
	}
	readmePath := filepath.Join(opts.Root, "README.md")
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}
	re := regexp.MustCompile(`~[0-9.]+ MB`)
	updated := re.ReplaceAllString(string(readme), "~"+data.Binary.SizeMB+" MB")
	return os.WriteFile(readmePath, []byte(updated), 0o644)
}
