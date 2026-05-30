package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
	"gopkg.in/yaml.v3"
)

const ProfileDistributionManifestFile = "distribution.yaml"

var ErrProfileDistributionInvalid = errors.New("profile distribution invalid")

type ProfileDistributionEnvRequirement struct {
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required"`
	Default     *string `json:"default,omitempty"`
}

type ProfileDistributionManifest struct {
	Name              string                              `json:"name"`
	Version           string                              `json:"version"`
	Description       string                              `json:"description,omitempty"`
	HermesRequires    string                              `json:"hermes_requires,omitempty"`
	Author            string                              `json:"author,omitempty"`
	License           string                              `json:"license,omitempty"`
	EnvRequires       []ProfileDistributionEnvRequirement `json:"env_requires,omitempty"`
	DistributionOwned []string                            `json:"distribution_owned,omitempty"`
	Source            string                              `json:"source,omitempty"`
	InstalledAt       string                              `json:"installed_at,omitempty"`
}

func (m ProfileDistributionManifest) Summary() string {
	if !textvalue.IsNonBlank(m.Name) {
		return ""
	}
	version := strings.TrimSpace(m.Version)
	if version == "" {
		version = "?"
	}
	return strings.TrimSpace(m.Name) + "@" + version
}

type rawProfileDistributionManifest struct {
	Name              string                          `yaml:"name"`
	Version           string                          `yaml:"version"`
	Description       string                          `yaml:"description"`
	HermesRequires    string                          `yaml:"hermes_requires"`
	Author            string                          `yaml:"author"`
	License           string                          `yaml:"license"`
	EnvRequires       []rawDistributionEnvRequirement `yaml:"env_requires"`
	DistributionOwned []string                        `yaml:"distribution_owned"`
	Source            string                          `yaml:"source"`
	InstalledAt       string                          `yaml:"installed_at"`
}

type rawDistributionEnvRequirement struct {
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	Required    *bool   `yaml:"required"`
	Default     *string `yaml:"default"`
}

func ReadProfileDistributionManifest(profileRoot string) (ProfileDistributionManifest, bool, error) {
	path := filepath.Join(profileRoot, ProfileDistributionManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ProfileDistributionManifest{}, false, nil
		}
		return ProfileDistributionManifest{}, false, fmt.Errorf("read profile distribution manifest: %w", err)
	}

	var raw rawProfileDistributionManifest
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return ProfileDistributionManifest{}, true, fmt.Errorf("%w: decode %s: %v", ErrProfileDistributionInvalid, ProfileDistributionManifestFile, err)
	}

	manifest, err := normalizeProfileDistributionManifest(raw)
	if err != nil {
		return ProfileDistributionManifest{}, true, err
	}
	return manifest, true, nil
}

func normalizeProfileDistributionManifest(raw rawProfileDistributionManifest) (ProfileDistributionManifest, error) {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return ProfileDistributionManifest{}, fmt.Errorf("%w: %s missing name", ErrProfileDistributionInvalid, ProfileDistributionManifestFile)
	}
	version := strings.TrimSpace(raw.Version)
	if version == "" {
		version = "0.1.0"
	}

	env := make([]ProfileDistributionEnvRequirement, 0, len(raw.EnvRequires))
	for _, req := range raw.EnvRequires {
		reqName := strings.TrimSpace(req.Name)
		if reqName == "" {
			return ProfileDistributionManifest{}, fmt.Errorf("%w: env_requires entry missing name", ErrProfileDistributionInvalid)
		}
		required := true
		if req.Required != nil {
			required = *req.Required
		}
		env = append(env, ProfileDistributionEnvRequirement{
			Name:        reqName,
			Description: strings.TrimSpace(req.Description),
			Required:    required,
			Default:     req.Default,
		})
	}

	owned := make([]string, 0, len(raw.DistributionOwned))
	for _, entry := range raw.DistributionOwned {
		trimmed := strings.Trim(strings.TrimSpace(entry), "/")
		if trimmed == "" {
			continue
		}
		owned = append(owned, trimmed)
	}

	return ProfileDistributionManifest{
		Name:              name,
		Version:           version,
		Description:       strings.TrimSpace(raw.Description),
		HermesRequires:    strings.TrimSpace(raw.HermesRequires),
		Author:            strings.TrimSpace(raw.Author),
		License:           strings.TrimSpace(raw.License),
		EnvRequires:       env,
		DistributionOwned: owned,
		Source:            strings.TrimSpace(raw.Source),
		InstalledAt:       strings.TrimSpace(raw.InstalledAt),
	}, nil
}
