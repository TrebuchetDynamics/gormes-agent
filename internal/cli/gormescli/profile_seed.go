package gormescli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/cli/profileseed"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type ProfileSeedSeams struct {
	CreateProfile   func(name string, cloneAll bool) (cli.ProfileCreateResult, error)
	BuildProvenance func() BuildProvenance
}

type profileSeedDraftReportJSON struct {
	Build  BuildProvenance   `json:"build"`
	Action string            `json:"action"`
	Status string            `json:"status"`
	Draft  profileseed.Draft `json:"draft"`
}

type profileSeedApplyReportJSON struct {
	Build          BuildProvenance   `json:"build"`
	Action         string            `json:"action"`
	Status         string            `json:"status"`
	Applied        bool              `json:"applied"`
	ProfileID      string            `json:"profile_id"`
	Root           string            `json:"root"`
	WorkspaceCount int               `json:"workspace_count"`
	Draft          profileseed.Draft `json:"draft"`
}

func NewProfileSeedCommand(seams ProfileSeedSeams) *cobra.Command {
	var asJSON bool
	var dryRun bool
	var apply bool
	var workspaces []string
	cmd := &cobra.Command{
		Use:          "seed <natural-language seed>",
		Short:        "Create a validated Gormes profile draft from a short natural-language seed",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileSeedCommand(cmd, seams, args[0], asJSON, dryRun, apply, workspaces)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON with redacted paths")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "return the validated draft without mutating profile files")
	cmd.Flags().BoolVar(&apply, "apply", false, "create the target profile and persist confirmed profile-seed metadata")
	cmd.Flags().StringArrayVar(&workspaces, "workspace", nil, "explicitly confirmed workspace path to grant on apply; repeatable")
	return cmd
}

func defaultProfileSeedDraftOptions() profileseed.DraftOptions {
	cfg, err := config.Load(nil)
	if err != nil {
		return profileseed.DraftOptions{}
	}
	return profileseed.DraftOptions{
		Provider: strings.TrimSpace(cfg.Hermes.Provider),
		Model:    strings.TrimSpace(cfg.Hermes.Model),
	}
}

func runProfileSeedCommand(cmd *cobra.Command, seams ProfileSeedSeams, rawSeed string, asJSON, dryRun, apply bool, workspaces []string) error {
	if apply && dryRun {
		return fmt.Errorf("gormes profile seed: choose either --dry-run or --apply")
	}
	if !apply {
		dryRun = true
	}
	draftOptions := defaultProfileSeedDraftOptions()
	if dryRun {
		draft, err := profileseed.NewDraft(rawSeed, draftOptions)
		if err != nil {
			return fmt.Errorf("gormes profile seed: %w", err)
		}
		return emitProfileSeedDraft(cmd, seams, draft, asJSON)
	}
	result, err := profileseed.Apply(rawSeed, profileseed.ApplyOptions{
		CreateProfile:       seams.CreateProfile,
		ConfirmedWorkspaces: workspaces,
		DraftOptions:        draftOptions,
	})
	if err != nil {
		return fmt.Errorf("gormes profile seed --apply: %w", err)
	}
	return emitProfileSeedApplied(cmd, seams, result, asJSON)
}

func emitProfileSeedDraft(cmd *cobra.Command, seams ProfileSeedSeams, draft profileseed.Draft, asJSON bool) error {
	if asJSON {
		body, err := json.MarshalIndent(profileSeedDraftReportJSON{
			Build:  profileSeedBuildProvenance(seams),
			Action: "profile_seed_draft",
			Status: "draft",
			Draft:  draft,
		}, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "profile seed draft: %s\n", draft.ProfileID)
	fmt.Fprintf(cmd.OutOrStdout(), "display_name: %s\n", draft.DisplayName)
	fmt.Fprintf(cmd.OutOrStdout(), "generation_source: %s\n", draft.GenerationSource)
	fmt.Fprintln(cmd.OutOrStdout(), "workspace_policy: explicit confirmation required before any path is granted")
	return nil
}

func emitProfileSeedApplied(cmd *cobra.Command, seams ProfileSeedSeams, result profileseed.ApplyResult, asJSON bool) error {
	redactedRoot := redactProfileSeedPath(result.Root)
	if asJSON {
		body, err := json.MarshalIndent(profileSeedApplyReportJSON{
			Build:          profileSeedBuildProvenance(seams),
			Action:         "profile_seed_applied",
			Status:         "applied",
			Applied:        result.Applied,
			ProfileID:      result.ProfileID,
			Root:           redactedRoot,
			WorkspaceCount: result.WorkspaceCount,
			Draft:          result.Draft,
		}, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "created profile from seed: %s\n", result.ProfileID)
	fmt.Fprintf(cmd.OutOrStdout(), "root: %s\n", redactedRoot)
	fmt.Fprintf(cmd.OutOrStdout(), "workspace_count: %d\n", result.WorkspaceCount)
	return nil
}

func profileSeedBuildProvenance(seams ProfileSeedSeams) BuildProvenance {
	if seams.BuildProvenance == nil {
		return BuildProvenance{}
	}
	return seams.BuildProvenance()
}

func redactProfileSeedPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	base := filepath.Base(filepath.Clean(path))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "..."
	}
	return ".../" + base
}
