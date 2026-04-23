package plugins

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Source string

const (
	SourceBundled Source = "bundled"
	SourceUser    Source = "user"
	SourceProject Source = "project"
)

type State string

const (
	StateAvailable State = "available"
	StateDisabled  State = "disabled"
	StateEnabled   State = "enabled"
	StateSelected  State = "selected"
)

type CatalogEntry struct {
	Manifest   Manifest
	Name       string
	Kind       Kind
	Source     Source
	State      State
	MissingEnv []string
}

func (e CatalogEntry) Details() string {
	if len(e.MissingEnv) == 0 {
		return "ready"
	}
	return "missing env: " + strings.Join(e.MissingEnv, ", ")
}

type DiscoveryOptions struct {
	BundledRoots    []string
	UserRoots       []string
	ProjectRoots    []string
	EnableProject   bool
	DisabledGeneral []string
	MemoryProvider  string
	ContextEngine   string
}

func DiscoverCatalog(opts DiscoveryOptions) ([]CatalogEntry, error) {
	disabled := normalizeSet(opts.DisabledGeneral)
	selectedMemory := normalizeName(opts.MemoryProvider)
	selectedContext := normalizeName(opts.ContextEngine)

	var out []CatalogEntry
	seen := make(map[string]struct{})
	for _, spec := range []struct {
		roots   []string
		source  Source
		enabled bool
	}{
		{roots: opts.BundledRoots, source: SourceBundled, enabled: true},
		{roots: opts.UserRoots, source: SourceUser, enabled: true},
		{roots: opts.ProjectRoots, source: SourceProject, enabled: opts.EnableProject},
	} {
		if !spec.enabled {
			continue
		}
		for _, root := range cleanRoots(spec.roots) {
			for _, candidate := range []struct {
				path string
				kind Kind
			}{
				{path: root, kind: KindGeneral},
				{path: filepath.Join(root, "memory"), kind: KindMemoryProvider},
				{path: filepath.Join(root, "context_engine"), kind: KindContextEngine},
			} {
				manifests, err := Discover(candidate.path, candidate.kind)
				if err != nil {
					return nil, err
				}
				for _, manifest := range manifests {
					key := catalogKey(manifest.Kind, manifest.Name)
					if _, exists := seen[key]; exists {
						continue
					}
					seen[key] = struct{}{}

					entry := CatalogEntry{
						Manifest:   manifest,
						Name:       manifest.Name,
						Kind:       manifest.Kind,
						Source:     spec.source,
						State:      stateFor(manifest.Kind, manifest.Name, disabled, selectedMemory, selectedContext),
						MissingEnv: missingEnv(manifest.RequiresEnv),
					}
					out = append(out, entry)
				}
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Source < out[j].Source
	})
	return out, nil
}

func stateFor(kind Kind, name string, disabled map[string]struct{}, selectedMemory, selectedContext string) State {
	switch kind {
	case KindMemoryProvider:
		if normalizeName(name) == selectedMemory {
			return StateSelected
		}
		return StateAvailable
	case KindContextEngine:
		if normalizeName(name) == selectedContext {
			return StateSelected
		}
		return StateAvailable
	default:
		if _, ok := disabled[normalizeName(name)]; ok {
			return StateDisabled
		}
		return StateEnabled
	}
}

func missingEnv(reqs []EnvRequirement) []string {
	if len(reqs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(reqs))
	out := make([]string, 0, len(reqs))
	for _, req := range reqs {
		name := strings.TrimSpace(req.Name)
		if name == "" {
			continue
		}
		if _, ok := os.LookupEnv(name); ok {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func cleanRoots(roots []string) []string {
	if len(roots) == 0 {
		return nil
	}
	out := make([]string, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		root = filepath.Clean(root)
		if _, exists := seen[root]; exists {
			continue
		}
		seen[root] = struct{}{}
		out = append(out, root)
	}
	return out
}

func normalizeSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalized := normalizeName(value); normalized != "" {
			out[normalized] = struct{}{}
		}
	}
	return out
}

func normalizeName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func catalogKey(kind Kind, name string) string {
	return string(kind) + ":" + normalizeName(name)
}
