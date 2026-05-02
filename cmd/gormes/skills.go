package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/spf13/cobra"
)

func newSkillsCommand() *cobra.Command {
	return cli.NewSkillsCommand(cli.SkillsCommandDeps{
		ListInstalledSkills: func(opts skills.ListOptions, disabled map[string]struct{}) []skills.SkillRow {
			cfg, err := config.Load(nil)
			if err != nil {
				return nil
			}
			return skills.ListInstalledSkillsFromRoots(cfg.SkillsRoot(), skills.BundledRoot(), opts, disabled)
		},
		DisabledSkills: func(string) map[string]struct{} { return nil },
		URLInstall: cli.SkillsURLInstallDeps{
			Fetcher: httpSkillFetcher{client: &http.Client{Timeout: 30 * time.Second}},
			Store:   configSkillStore{},
		},
	})
}

type httpSkillFetcher struct {
	client *http.Client
}

func (f httpSkillFetcher) Fetch(ctx context.Context, rawURL string) ([]byte, error) {
	client := f.client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch skill: HTTP %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, int64(skills.DefaultMaxDocumentBytes)+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(body) > skills.DefaultMaxDocumentBytes {
		return nil, fmt.Errorf("skill document too large: %d > %d bytes", len(body), skills.DefaultMaxDocumentBytes)
	}
	return body, nil
}

type configSkillStore struct{}

func (configSkillStore) ActiveDir() string {
	cfg, err := config.Load(nil)
	if err != nil {
		return filepath.Join(config.GormesHome(), "skills", "active")
	}
	return filepath.Join(cfg.SkillsRoot(), "active")
}

func (s configSkillStore) WriteSkill(_ context.Context, dir string, file string, body []byte) (string, error) {
	if file != "SKILL.md" {
		return "", fmt.Errorf("unsupported skill file %q", file)
	}
	activeDir := s.ActiveDir()
	target := filepath.Join(dir, file)
	if !pathWithin(activeDir, target) {
		return "", fmt.Errorf("skill install target escapes active skills root")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, ".SKILL.md-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, target); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return target, nil
}

func pathWithin(root, target string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
