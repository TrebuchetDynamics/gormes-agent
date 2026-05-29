package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/core/agenttemplate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

// ProfileContextScaffoldOptions names the profile root that should receive the
// editable context files Gormes exposes to operators. BaseHome is the global
// Gormes metadata home (normally ~/.gormes); ProfileName resolves below
// BaseHome/profiles/<name>. TargetRoot may be supplied by tests or migrations
// that already resolved the profile root.
type ProfileContextScaffoldOptions struct {
	BaseHome    string
	ProfileName string
	DisplayName string
	TargetRoot  string
	Force       bool
	DryRun      bool
}

// ProfileContextScaffoldResult reports which profile root was considered and
// the per-file template actions. Callers use this as operator evidence for
// setup, migration, and dry-run flows.
type ProfileContextScaffoldResult struct {
	ProfileName string
	Root        string
	Templates   agenttemplate.WriteResult
	MemoryDB    agenttemplate.FileResult
}

// MaterializeMainProfileContextScaffold materializes the built-in main profile
// root and seeds its editable context files. Existing files are preserved unless
// Force is set. Creating profiles/main is intentionally explicit because its
// existence changes profile-rooted runtime path probing.
const profileContextDefaultProfileName = "main"

func MaterializeMainProfileContextScaffold(opts ProfileContextScaffoldOptions) (ProfileContextScaffoldResult, error) {
	opts.ProfileName = profileContextDefaultProfileName
	return ApplyProfileContextScaffold(opts)
}

// ApplyProfileContextScaffold seeds the default Gormes context templates into a
// profile root. It is the single module that owns profile-local context file
// placement; template content and parity evidence remain in internal/agenttemplate.
func ApplyProfileContextScaffold(opts ProfileContextScaffoldOptions) (ProfileContextScaffoldResult, error) {
	name := strings.TrimSpace(opts.ProfileName)
	if name == "" {
		name = profileContextDefaultProfileName
	}
	if err := ValidateProfileName(name); err != nil {
		return ProfileContextScaffoldResult{}, err
	}

	root := strings.TrimSpace(opts.TargetRoot)
	if root == "" {
		contract, err := NewProfileStorageContract(opts.BaseHome)
		if err != nil {
			return ProfileContextScaffoldResult{}, err
		}
		root, err = contract.ProfileRoot(name)
		if err != nil {
			return ProfileContextScaffoldResult{}, err
		}
	}
	root = filepath.Clean(root)
	if !opts.DryRun {
		if err := os.MkdirAll(root, 0o700); err != nil {
			return ProfileContextScaffoldResult{}, fmt.Errorf("profile context scaffold root: %w", err)
		}
	}

	templates, err := agenttemplate.ApplyTemplates(agenttemplate.WriteOptions{
		TargetDir: root,
		Force:     opts.Force,
		DryRun:    opts.DryRun,
	}, profileContextTemplates(name, opts.DisplayName))
	if err != nil {
		return ProfileContextScaffoldResult{}, fmt.Errorf("profile context scaffold templates: %w", err)
	}
	memoryDB, err := bootstrapProfileMemoryDB(root, opts.DryRun)
	if err != nil {
		return ProfileContextScaffoldResult{}, err
	}
	return ProfileContextScaffoldResult{ProfileName: name, Root: root, Templates: templates, MemoryDB: memoryDB}, nil
}

func bootstrapProfileMemoryDB(root string, dryRun bool) (agenttemplate.FileResult, error) {
	path := filepath.Join(root, "memory.db")
	exists, err := profileScaffoldFileExists(path)
	if err != nil {
		return agenttemplate.FileResult{}, fmt.Errorf("profile memory db stat: %w", err)
	}
	if exists {
		action := agenttemplate.ActionSkip
		if dryRun {
			action = agenttemplate.ActionWouldSkip
		}
		return agenttemplate.FileResult{Path: "memory.db", Action: action}, nil
	}
	if dryRun {
		return agenttemplate.FileResult{Path: "memory.db", Action: agenttemplate.ActionWouldCreate}, nil
	}
	store, err := memory.OpenSqlite(path, 1, nil)
	if err != nil {
		return agenttemplate.FileResult{}, fmt.Errorf("profile memory db bootstrap: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.Close(ctx); err != nil {
		return agenttemplate.FileResult{}, fmt.Errorf("profile memory db close: %w", err)
	}
	return agenttemplate.FileResult{Path: "memory.db", Action: agenttemplate.ActionCreate}, nil
}

func profileScaffoldFileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func profileContextTemplates(profileName, displayName string) []agenttemplate.FileTemplate {
	files := agenttemplate.DefaultFiles()
	name := profileContextDisplayName(profileName, displayName)
	profileBlock := fmt.Sprintf("\n## Gormes Profile\n\n- Profile ID: `%s`\n- Agent name: %s\n", profileName, name)
	for i := range files {
		switch files[i].Path {
		case "SOUL.md":
			files[i].Content += profileBlock
		case "IDENTITY.md":
			files[i].Content = strings.Replace(files[i].Content, "- Name: Gorm\n", fmt.Sprintf("- Name: %s\n- Profile ID: `%s`\n", name, profileName), 1)
		}
	}
	return files
}

func profileContextDisplayName(profileName, displayName string) string {
	if name := strings.TrimSpace(displayName); name != "" {
		return name
	}
	if profileName == "" || profileName == profileContextDefaultProfileName {
		return "Gorm"
	}
	parts := strings.FieldsFunc(profileName, func(r rune) bool { return r == '-' || r == '_' })
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		parts[i] = string(runes)
	}
	if len(parts) == 0 {
		return profileName
	}
	return strings.Join(parts, " ")
}
