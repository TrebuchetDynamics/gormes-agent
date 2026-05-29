package credentials

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	TokenVaultSourceRegistered = "registered"
	TokenVaultSourceConfig     = "config"
	defaultContainerHermesHome = "/root/.hermes"
)

type TokenVaultReason string

const (
	TokenVaultReasonEmptyPath     TokenVaultReason = "empty_path"
	TokenVaultReasonAbsolutePath  TokenVaultReason = "absolute_path"
	TokenVaultReasonTraversal     TokenVaultReason = "path_traversal"
	TokenVaultReasonMissing       TokenVaultReason = "missing"
	TokenVaultReasonUnreadable    TokenVaultReason = "unreadable"
	TokenVaultReasonSymlinkEscape TokenVaultReason = "symlink_escape"
)

type TokenVaultOptions struct {
	HermesHome          string
	ContainerHermesHome string
}

type CredentialFileMount struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Source        string `json:"source"`
}

type TokenVaultEvidence struct {
	Input         string           `json:"input"`
	ContainerPath string           `json:"container_path,omitempty"`
	HostPath      string           `json:"host_path,omitempty"`
	Reason        TokenVaultReason `json:"reason,omitempty"`
	Message       string           `json:"message"`
}

type TokenVaultError struct {
	Input  string
	Reason TokenVaultReason
	Err    error
}

func (e *TokenVaultError) Error() string {
	if e == nil {
		return "token vault: <nil>"
	}
	if e.Err != nil {
		return fmt.Sprintf("token vault: %s: %v", e.Reason, e.Err)
	}
	return fmt.Sprintf("token vault: %s", e.Reason)
}

func (e *TokenVaultError) Unwrap() error { return e.Err }

func (e *TokenVaultError) Evidence() TokenVaultEvidence {
	if e == nil {
		return TokenVaultEvidence{}
	}
	input := filepath.ToSlash(strings.TrimSpace(e.Input))
	return TokenVaultEvidence{
		Input:   input,
		Reason:  e.Reason,
		Message: tokenVaultReasonMessage(e.Reason),
	}
}

func AsTokenVaultError(err error, target **TokenVaultError) bool {
	return errors.As(err, target)
}

type TokenVault struct {
	hermesHome          string
	containerHermesHome string
	registered          map[string]CredentialFileMount
	configured          map[string]CredentialFileMount
}

type TokenVaultConfigResult struct {
	Mounts   []CredentialFileMount `json:"mounts"`
	Rejected []TokenVaultEvidence  `json:"rejected"`
}

func NewTokenVault(opts TokenVaultOptions) (*TokenVault, error) {
	home := strings.TrimSpace(opts.HermesHome)
	if home == "" {
		home = gormesHome()
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, fmt.Errorf("token vault home: %w", err)
	}
	realHome, err := filepath.EvalSymlinks(absHome)
	if err != nil {
		return nil, fmt.Errorf("token vault home: %w", err)
	}
	containerHome := strings.TrimRight(strings.TrimSpace(opts.ContainerHermesHome), "/")
	if containerHome == "" {
		containerHome = defaultContainerHermesHome
	}
	return &TokenVault{
		hermesHome:          realHome,
		containerHermesHome: containerHome,
		registered:          make(map[string]CredentialFileMount),
		configured:          make(map[string]CredentialFileMount),
	}, nil
}

func (v *TokenVault) RegisterCredentialFile(relPath string) (CredentialFileMount, error) {
	mount, err := v.resolveCredentialFile(relPath, TokenVaultSourceRegistered, false)
	if err != nil {
		return CredentialFileMount{}, err
	}
	v.registered[mount.ContainerPath] = mount
	return mount, nil
}

func (v *TokenVault) Mounts() []CredentialFileMount {
	merged := v.mergedMounts()
	out := make([]CredentialFileMount, 0, len(merged))
	for _, mount := range merged {
		out = append(out, mount)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ContainerPath < out[j].ContainerPath })
	return out
}

func (v *TokenVault) Clear() {
	v.registered = make(map[string]CredentialFileMount)
}

func (v *TokenVault) LoadConfigCredentialFiles(configPath string) (TokenVaultConfigResult, error) {
	paths, err := credentialFilesFromHermesConfig(configPath)
	if err != nil {
		return TokenVaultConfigResult{}, err
	}
	v.configured = make(map[string]CredentialFileMount)
	var rejected []TokenVaultEvidence
	for _, relPath := range paths {
		mount, err := v.resolveCredentialFile(relPath, TokenVaultSourceConfig, true)
		if err != nil {
			var vaultErr *TokenVaultError
			if errors.As(err, &vaultErr) {
				rejected = append(rejected, vaultErr.Evidence())
				continue
			}
			return TokenVaultConfigResult{}, err
		}
		v.configured[mount.ContainerPath] = mount
	}
	return TokenVaultConfigResult{Mounts: v.Mounts(), Rejected: rejected}, nil
}

func (v *TokenVault) mergedMounts() map[string]CredentialFileMount {
	merged := make(map[string]CredentialFileMount, len(v.configured)+len(v.registered))
	for key, mount := range v.configured {
		merged[key] = mount
	}
	for key, mount := range v.registered {
		merged[key] = mount
	}
	return merged
}

func (v *TokenVault) resolveCredentialFile(relPath, source string, missingAsEvidence bool) (CredentialFileMount, error) {
	input := strings.TrimSpace(relPath)
	clean, err := safeRelativeCredentialPath(input)
	if err != nil {
		return CredentialFileMount{}, err
	}
	hostPath := filepath.Join(v.hermesHome, filepath.FromSlash(clean))
	realPath, err := filepath.EvalSymlinks(hostPath)
	if err != nil {
		if os.IsNotExist(err) {
			return CredentialFileMount{}, &TokenVaultError{Input: input, Reason: TokenVaultReasonMissing, Err: os.ErrNotExist}
		}
		return CredentialFileMount{}, &TokenVaultError{Input: input, Reason: TokenVaultReasonUnreadable, Err: err}
	}
	if err := validateWithinDir(v.hermesHome, realPath); err != nil {
		return CredentialFileMount{}, &TokenVaultError{Input: input, Reason: TokenVaultReasonSymlinkEscape, Err: err}
	}
	info, err := os.Stat(realPath)
	if err != nil {
		return CredentialFileMount{}, &TokenVaultError{Input: input, Reason: TokenVaultReasonUnreadable, Err: err}
	}
	if info.IsDir() {
		return CredentialFileMount{}, &TokenVaultError{Input: input, Reason: TokenVaultReasonUnreadable, Err: fmt.Errorf("credential file is a directory")}
	}
	return CredentialFileMount{
		HostPath:      realPath,
		ContainerPath: v.containerPath(clean),
		Source:        source,
	}, nil
}

func safeRelativeCredentialPath(input string) (string, error) {
	if input == "" {
		return "", &TokenVaultError{Input: input, Reason: TokenVaultReasonEmptyPath}
	}
	if filepath.IsAbs(input) || strings.HasPrefix(input, "/") {
		return "", &TokenVaultError{Input: input, Reason: TokenVaultReasonAbsolutePath}
	}
	slash := filepath.ToSlash(input)
	clean := filepath.Clean(filepath.FromSlash(slash))
	cleanSlash := filepath.ToSlash(clean)
	if cleanSlash == "." || cleanSlash == ".." || strings.HasPrefix(cleanSlash, "../") || strings.Contains(cleanSlash, "/../") {
		return "", &TokenVaultError{Input: input, Reason: TokenVaultReasonTraversal}
	}
	return cleanSlash, nil
}

func validateWithinDir(root, candidate string) error {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return err
	}
	if rel == "." || rel == "" || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." || filepath.IsAbs(rel) {
		return fmt.Errorf("resolved path escapes credential root")
	}
	return nil
}

func (v *TokenVault) containerPath(cleanRelPath string) string {
	parts := strings.Split(filepath.ToSlash(cleanRelPath), "/")
	return v.containerHermesHome + "/" + strings.Join(parts, "/")
}

type hermesTokenVaultConfigYAML struct {
	Terminal struct {
		CredentialFiles []string `yaml:"credential_files"`
	} `yaml:"terminal"`
}

func credentialFilesFromHermesConfig(configPath string) ([]string, error) {
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Hermes config credential_files: %w", err)
	}
	var cfg hermesTokenVaultConfigYAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("decode Hermes config credential_files: %w", err)
	}
	return cfg.Terminal.CredentialFiles, nil
}

func tokenVaultReasonMessage(reason TokenVaultReason) string {
	switch reason {
	case TokenVaultReasonEmptyPath:
		return "credential file path is empty"
	case TokenVaultReasonAbsolutePath:
		return "credential file path must be relative to the active credential root"
	case TokenVaultReasonTraversal:
		return "credential file path may not traverse outside the active Hermes profile"
	case TokenVaultReasonMissing:
		return "credential file is missing"
	case TokenVaultReasonUnreadable:
		return "credential file is unreadable"
	case TokenVaultReasonSymlinkEscape:
		return "credential file resolves outside the active credential root"
	default:
		return "credential file rejected"
	}
}
