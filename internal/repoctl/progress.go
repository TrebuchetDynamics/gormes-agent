package repoctl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ProgressOptions struct {
	Root string
	Now  func() time.Time
}

func SyncProgress(opts ProgressOptions) error {
	if opts.Root == "" {
		return fmt.Errorf("repo root is required")
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	docsProgress := filepath.Join(opts.Root, "docs", "data", "progress.json")
	raw, err := os.ReadFile(docsProgress)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(os.Stdout, "record-progress: progress.json not found; skipping")
			return nil
		}
		return err
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	meta, _ := data["meta"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
		data["meta"] = meta
	}
	meta["last_updated"] = opts.Now().UTC().Format(time.DateOnly)
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.WriteFile(docsProgress, out, 0o644); err != nil {
		return err
	}
	archProgress := filepath.Join(opts.Root, "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	siteProgress := filepath.Join(opts.Root, "www.gormes.ai", "internal", "site", "data", "progress.json")
	archRaw, err := os.ReadFile(archProgress)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(siteProgress), 0o755); err != nil {
		return err
	}
	return os.WriteFile(siteProgress, archRaw, 0o644)
}
