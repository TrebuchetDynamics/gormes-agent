package repoctl

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

//go:embed rowpacks/*.json
var rowPackFS embed.FS

type ProgressSeedOptions struct {
	Root         string
	ProgressPath string
	Set          string
}

type ProgressSeedResult struct {
	Set          string
	Added        int
	Skipped      int
	AddedNames   []string
	SkippedNames []string
	TotalItems   int
}

type progressRowPack struct {
	Name string            `json:"name"`
	Rows []progressRowSeed `json:"rows"`
}

type progressRowSeed struct {
	Phase    string        `json:"phase"`
	Subphase string        `json:"subphase"`
	Item     progress.Item `json:"item"`
}

func SeedProgressRows(opts ProgressSeedOptions) (ProgressSeedResult, error) {
	if opts.Root == "" {
		opts.Root = "."
	}
	opts.Set = strings.TrimSpace(opts.Set)
	if opts.Set == "" {
		opts.Set = "fleet"
	}
	pack, err := loadProgressRowPack(opts.Set)
	if err != nil {
		return ProgressSeedResult{}, err
	}
	path := opts.ProgressPath
	if path == "" {
		path = filepath.Join(opts.Root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	}
	doc, err := progress.Load(path)
	if err != nil {
		return ProgressSeedResult{}, err
	}

	result := ProgressSeedResult{Set: pack.Name}
	for _, row := range pack.Rows {
		phase, ok := doc.Phases[row.Phase]
		if !ok {
			return result, fmt.Errorf("progress seed %s: phase %s not found", pack.Name, row.Phase)
		}
		subphase, ok := phase.Subphases[row.Subphase]
		if !ok {
			return result, fmt.Errorf("progress seed %s: subphase %s not found in phase %s", pack.Name, row.Subphase, row.Phase)
		}
		if progressSubphaseHasItem(subphase, row.Item.Name) {
			result.Skipped++
			result.SkippedNames = append(result.SkippedNames, row.Item.Name)
			continue
		}
		if len(subphase.Items) == 0 {
			subphase.Status = ""
		}
		subphase.Items = append(subphase.Items, row.Item)
		phase.Subphases[row.Subphase] = subphase
		doc.Phases[row.Phase] = phase
		result.Added++
		result.AddedNames = append(result.AddedNames, row.Item.Name)
	}

	if result.Added > 0 {
		if err := progress.Validate(doc); err != nil {
			return result, err
		}
		if err := progress.SaveProgress(path, doc); err != nil {
			return result, err
		}
	}
	result.TotalItems = doc.Stats().Items.Total
	return result, nil
}

func loadProgressRowPack(set string) (progressRowPack, error) {
	name := strings.TrimSpace(set)
	raw, err := rowPackFS.ReadFile(path.Join("rowpacks", name+".json"))
	if err != nil {
		if _, ok := err.(*fs.PathError); ok {
			return progressRowPack{}, fmt.Errorf("unknown progress row seed set %q (available: %s)", set, strings.Join(progressRowPackNames(), ", "))
		}
		return progressRowPack{}, err
	}
	var pack progressRowPack
	if err := json.Unmarshal(raw, &pack); err != nil {
		return progressRowPack{}, fmt.Errorf("parse progress row pack %s: %w", name, err)
	}
	if pack.Name == "" {
		pack.Name = name
	}
	return pack, nil
}

func progressRowPackNames() []string {
	matches, err := fs.Glob(rowPackFS, "rowpacks/*.json")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		base := path.Base(match)
		names = append(names, strings.TrimSuffix(base, path.Ext(base)))
	}
	sort.Strings(names)
	return names
}

func progressSubphaseHasItem(subphase progress.Subphase, name string) bool {
	for _, item := range subphase.Items {
		if item.Name == name {
			return true
		}
	}
	return false
}
