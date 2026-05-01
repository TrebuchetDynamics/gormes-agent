package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type PermissionManifest struct {
	Version   string           `yaml:"version"`
	Scopes    []DirectoryScope `yaml:"scopes,omitempty"`
	Approvals []ApprovalRecord `yaml:"approvals,omitempty"`
}

type DirectoryScope struct {
	Path       string `yaml:"path"`
	AllowRead  bool   `yaml:"allow_read"`
	AllowWrite bool   `yaml:"allow_write"`
}

type ApprovalRecord struct {
	CommandPattern string `yaml:"command_pattern"`
	Approved       bool   `yaml:"approved"`
	Mode           string `yaml:"mode"`
}

func DefaultPermissionManifestPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(configDir, "gormes", "permissions.yaml")
}

func LoadPermissionManifest(path string) (*PermissionManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &PermissionManifest{}, nil
		}
		return nil, fmt.Errorf("load permission manifest: %w", err)
	}
	var m PermissionManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse permission manifest: %w", err)
	}
	return &m, nil
}

func (m *PermissionManifest) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func CheckPathScope(path, operation string, scopes []DirectoryScope) error {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	resolved = filepath.Clean(resolved)

	for _, scope := range scopes {
		scopeResolved, err := filepath.Abs(scope.Path)
		if err != nil {
			continue
		}
		scopeResolved = filepath.Clean(scopeResolved)
		if strings.HasPrefix(resolved, scopeResolved+string(filepath.Separator)) || resolved == scopeResolved {
			switch operation {
			case "read":
				if scope.AllowRead {
					return nil
				}
			case "write":
				if scope.AllowWrite {
					return nil
				}
			}
		}
	}
	return fmt.Errorf("path %q is outside allowed %s scopes", path, operation)
}

func CheckCwdOnly(workdir, cwd string) error {
	workdirAbs, err := filepath.Abs(workdir)
	if err != nil {
		return fmt.Errorf("resolve workdir: %w", err)
	}
	workdirAbs = filepath.Clean(workdirAbs)
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return fmt.Errorf("resolve cwd: %w", err)
	}
	cwdAbs = filepath.Clean(cwdAbs)
	if workdirAbs == cwdAbs || strings.HasPrefix(workdirAbs, cwdAbs+string(filepath.Separator)) {
		return nil
	}
	return fmt.Errorf("workdir %q escapes cwd %q", workdir, cwd)
}

func FindApprovalRecord(records []ApprovalRecord, cmd string) (ApprovalRecord, bool) {
	for _, r := range records {
		if strings.Contains(cmd, r.CommandPattern) {
			return r, true
		}
	}
	return ApprovalRecord{}, false
}

var (
	approvalManifestOnce sync.Once
	approvalManifest     *PermissionManifest
	approvalManifestErr  error
	approvalManifestPath string
)

func loadApprovalManifestLazy() {
	approvalManifestPath = DefaultPermissionManifestPath()
	approvalManifest, approvalManifestErr = LoadPermissionManifest(approvalManifestPath)
}

func GetApprovalManifest() (*PermissionManifest, error) {
	approvalManifestOnce.Do(loadApprovalManifestLazy)
	return approvalManifest, approvalManifestErr
}

func ResetApprovalManifestCache() {
	approvalManifestOnce = sync.Once{}
}
