package profileseed

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

var (
	ErrEmptySeed  = errors.New("profile_seed_empty")
	ErrUnsafeSeed = errors.New("profile_seed_unsafe")
)

type DraftOptions struct {
	Provider      string
	Model         string
	ProviderDraft *Draft
}

type Draft struct {
	ProfileID                string                    `json:"profile_id"`
	DisplayName              string                    `json:"display_name"`
	Instructions             string                    `json:"instructions"`
	ProviderModelState       ProviderModelState        `json:"provider_model_state"`
	WorkspaceRootSuggestions []WorkspaceRootSuggestion `json:"workspace_root_suggestions"`
	ToolPolicy               ToolPolicy                `json:"tool_policy"`
	VoiceProfileMetadata     VoiceProfileMetadata      `json:"voice_profile_metadata"`
	GenerationSource         string                    `json:"generation_source"`
	Evidence                 []string                  `json:"evidence,omitempty"`
}

type ProviderModelState struct {
	Status   string   `json:"status"`
	Provider string   `json:"provider,omitempty"`
	Model    string   `json:"model,omitempty"`
	Evidence []string `json:"evidence,omitempty"`
}

type WorkspaceRootSuggestion struct {
	Label                string `json:"label"`
	Purpose              string `json:"purpose"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
}

type ToolPolicy struct {
	Mode             string   `json:"mode"`
	Allowed          []string `json:"allowed,omitempty"`
	RequiresApproval []string `json:"requires_approval,omitempty"`
}

type VoiceProfileMetadata struct {
	Status         string `json:"status"`
	LanguagePolicy string `json:"language_policy"`
	STTProvider    string `json:"stt_provider,omitempty"`
	TTSProvider    string `json:"tts_provider,omitempty"`
	FallbackVoice  string `json:"fallback_voice,omitempty"`
}

type ApplyOptions struct {
	CreateProfile       func(name string, cloneAll bool) (cli.ProfileCreateResult, error)
	ConfirmedWorkspaces []string
	Now                 time.Time
	DraftOptions        DraftOptions
}

type ApplyResult struct {
	Draft
	Applied        bool     `json:"applied"`
	Root           string   `json:"root,omitempty"`
	ConfigPath     string   `json:"config_path,omitempty"`
	ManifestPath   string   `json:"manifest_path,omitempty"`
	WorkspaceCount int      `json:"workspace_count"`
	Workspaces     []string `json:"workspaces,omitempty"`
}

type manifest struct {
	Draft                   Draft    `json:"draft"`
	AppliedAt               string   `json:"applied_at"`
	WorkspacePolicy         string   `json:"workspace_policy"`
	ConfirmedWorkspaceCount int      `json:"confirmed_workspace_count"`
	ConfirmedWorkspaces     []string `json:"confirmed_workspaces,omitempty"`
}

var secretLikeSeedPattern = regexp.MustCompile(`(?i)\b(api[_-]?key|authorization|bearer|password|secret|token)\s*[:=]\s*([^\s]+)`)

func NewDraft(seed string, opts DraftOptions) (Draft, error) {
	seed = normalizeSeed(seed)
	if seed == "" {
		return Draft{}, ErrEmptySeed
	}
	if secretLikeSeedPattern.MatchString(seed) {
		return Draft{}, fmt.Errorf("%w: seed contains credential-like material", ErrUnsafeSeed)
	}
	provider := safeEvidenceString(opts.Provider, 64)
	model := safeEvidenceString(opts.Model, 96)
	if opts.ProviderDraft != nil && provider != "" && model != "" {
		draft := *opts.ProviderDraft
		if err := normalizeProviderDraft(&draft, seed); err != nil {
			return Draft{}, err
		}
		draft.GenerationSource = "model"
		draft.ProviderModelState = ProviderModelState{
			Status:   "configured",
			Provider: provider,
			Model:    model,
			Evidence: []string{"provider_draft_validated"},
		}
		draft.Evidence = appendMissing(draft.Evidence, "provider_draft_validated")
		return draft, nil
	}
	profileID := profileIDFromSeed(seed)
	displayName := displayNameFromProfileID(profileID)
	return Draft{
		ProfileID:    profileID,
		DisplayName:  displayName,
		Instructions: templateInstructions(displayName, seed),
		ProviderModelState: ProviderModelState{
			Status:   providerStatus(provider, model),
			Provider: provider,
			Model:    model,
			Evidence: providerEvidence(provider, model),
		},
		WorkspaceRootSuggestions: []WorkspaceRootSuggestion{{
			Label:                workspaceLabel(seed),
			Purpose:              "Operator-confirmed workspace for " + safeEvidenceString(seed, 96),
			RequiresConfirmation: true,
		}},
		ToolPolicy: ToolPolicy{
			Mode:             "safe",
			Allowed:          []string{"read", "search", "list"},
			RequiresApproval: []string{"write_files", "run_commands", "network", "secrets"},
		},
		VoiceProfileMetadata: VoiceProfileMetadata{
			Status:         "draft",
			LanguagePolicy: "match_user_language",
			STTProvider:    "device_or_profile_default",
			TTSProvider:    "profile_default",
			FallbackVoice:  "text_only",
		},
		GenerationSource: "template",
		Evidence:         []string{"template_fallback", "workspace_confirmation_required"},
	}, nil
}

func Apply(seed string, opts ApplyOptions) (ApplyResult, error) {
	draft, err := NewDraft(seed, opts.DraftOptions)
	if err != nil {
		return ApplyResult{}, err
	}
	create := opts.CreateProfile
	if create == nil {
		create = defaultCreateProfile
	}
	created, err := create(draft.ProfileID, false)
	if err != nil {
		return ApplyResult{}, err
	}
	root := strings.TrimSpace(created.Root)
	if root == "" {
		return ApplyResult{}, fmt.Errorf("profile seed apply: empty profile root")
	}
	workspaces, err := canonicalWorkspaces(opts.ConfirmedWorkspaces)
	if err != nil {
		return ApplyResult{}, err
	}
	configPath := filepath.Join(root, "config.toml")
	if err := config.WriteTOMLValue(configPath, "hermes.provider", draft.ProviderModelState.Provider); err != nil {
		return ApplyResult{}, err
	}
	if strings.TrimSpace(draft.ProviderModelState.Model) != "" {
		if err := config.WriteTOMLValue(configPath, "hermes.model", draft.ProviderModelState.Model); err != nil {
			return ApplyResult{}, err
		}
	}
	if len(workspaces) > 0 {
		if err := config.WriteTOMLValue(configPath, "agents.defaults.workspaces", strings.Join(workspaces, ",")); err != nil {
			return ApplyResult{}, err
		}
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	manifestPath := filepath.Join(root, "profile_seed.json")
	if err := writeManifest(manifestPath, manifest{
		Draft:                   draft,
		AppliedAt:               now.UTC().Format(time.RFC3339),
		WorkspacePolicy:         "explicit_confirmation_required",
		ConfirmedWorkspaceCount: len(workspaces),
		ConfirmedWorkspaces:     workspaces,
	}); err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{
		Draft:          draft,
		Applied:        true,
		Root:           root,
		ConfigPath:     configPath,
		ManifestPath:   manifestPath,
		WorkspaceCount: len(workspaces),
		Workspaces:     workspaces,
	}, nil
}

func defaultCreateProfile(name string, cloneAll bool) (cli.ProfileCreateResult, error) {
	if name == "default" {
		return cli.ProfileCreateResult{}, cli.ErrProfileCreateDefaultReserved
	}
	baseHome := config.GormesBaseHome()
	sourceRoot := ""
	if cloneAll {
		var err error
		sourceRoot, err = cli.ResolveProfileRuntimeRoot(baseHome, "default")
		if err != nil {
			return cli.ProfileCreateResult{}, err
		}
	}
	return cli.CreateProfile(cli.ProfileCreateOptions{
		Name:       name,
		TargetRoot: filepath.Join(baseHome, "profiles", name),
		SourceRoot: sourceRoot,
		CloneAll:   cloneAll,
	})
}

func normalizeProviderDraft(draft *Draft, seed string) error {
	if draft == nil {
		return nil
	}
	if strings.TrimSpace(draft.ProfileID) == "" {
		draft.ProfileID = profileIDFromSeed(seed)
	} else {
		draft.ProfileID = profileIDFromSeed(draft.ProfileID)
	}
	if err := cli.ValidateProfileName(draft.ProfileID); err != nil {
		return fmt.Errorf("profile seed provider draft profile_id: %w", err)
	}
	if strings.TrimSpace(draft.DisplayName) == "" {
		draft.DisplayName = displayNameFromProfileID(draft.ProfileID)
	} else {
		draft.DisplayName = safeEvidenceString(draft.DisplayName, 96)
	}
	if strings.TrimSpace(draft.Instructions) == "" {
		draft.Instructions = templateInstructions(draft.DisplayName, seed)
	} else if secretLikeSeedPattern.MatchString(draft.Instructions) {
		return fmt.Errorf("%w: provider instructions contain credential-like material", ErrUnsafeSeed)
	} else {
		draft.Instructions = safeEvidenceString(draft.Instructions, 512)
	}
	for i := range draft.WorkspaceRootSuggestions {
		draft.WorkspaceRootSuggestions[i].Label = safeEvidenceString(draft.WorkspaceRootSuggestions[i].Label, 96)
		draft.WorkspaceRootSuggestions[i].Purpose = safeEvidenceString(draft.WorkspaceRootSuggestions[i].Purpose, 160)
		draft.WorkspaceRootSuggestions[i].RequiresConfirmation = true
	}
	if len(draft.WorkspaceRootSuggestions) == 0 {
		draft.WorkspaceRootSuggestions = []WorkspaceRootSuggestion{{Label: workspaceLabel(seed), Purpose: "Operator-confirmed workspace", RequiresConfirmation: true}}
	}
	if draft.ToolPolicy.Mode == "" {
		draft.ToolPolicy = ToolPolicy{Mode: "safe", Allowed: []string{"read", "search", "list"}, RequiresApproval: []string{"write_files", "run_commands", "network", "secrets"}}
	}
	if draft.VoiceProfileMetadata.Status == "" {
		draft.VoiceProfileMetadata = VoiceProfileMetadata{Status: "draft", LanguagePolicy: "match_user_language", FallbackVoice: "text_only"}
	}
	return nil
}

func canonicalWorkspaces(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if strings.Contains(value, "\x00") {
			return nil, fmt.Errorf("workspace path contains NUL")
		}
		abs, err := filepath.Abs(value)
		if err != nil {
			return nil, fmt.Errorf("workspace path %q: %w", value, err)
		}
		abs = filepath.Clean(abs)
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}
	return out, nil
}

func writeManifest(path string, payload manifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("profile seed manifest mkdir: %w", err)
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("profile seed manifest write: %w", err)
	}
	return nil
}

func normalizeSeed(seed string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(seed)), " ")
}

func profileIDFromSeed(seed string) string {
	seed = strings.ToLower(normalizeSeed(seed))
	parts := strings.Fields(seed)
	var b strings.Builder
	lastDash := false
	for _, part := range parts {
		if profileSeedStopWord(part) {
			continue
		}
		for _, r := range part {
			switch {
			case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
				b.WriteRune(r)
				lastDash = false
			case r == '-' || r == '_' || unicode.IsSpace(r):
				if b.Len() > 0 && !lastDash {
					b.WriteByte('-')
					lastDash = true
				}
			}
			if b.Len() >= 64 {
				break
			}
		}
		if b.Len() >= 64 {
			break
		}
		if b.Len() > 0 && !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	id := strings.Trim(b.String(), "-")
	if id == "" {
		id = "profile"
	}
	if len(id) > 64 {
		id = strings.TrimRight(id[:64], "-")
	}
	if err := cli.ValidateProfileName(id); err != nil {
		id = "profile-" + id
		if len(id) > 64 {
			id = id[:64]
		}
		id = strings.TrimRight(id, "-")
	}
	if err := cli.ValidateProfileName(id); err != nil {
		id = "profile"
	}
	return id
}

func profileSeedStopWord(value string) bool {
	switch strings.Trim(value, "-_.,:;") {
	case "a", "an", "and", "for", "in", "on", "the", "to":
		return true
	default:
		return false
	}
}

func displayNameFromProfileID(id string) string {
	parts := strings.FieldsFunc(id, func(r rune) bool { return r == '-' || r == '_' })
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

func templateInstructions(displayName, seed string) string {
	return fmt.Sprintf("You are the %s profile. Focus on %s. Ask before changing workspace roots, credentials, providers, or files.", safeEvidenceString(displayName, 96), safeEvidenceString(seed, 160))
}

func workspaceLabel(seed string) string {
	seed = safeEvidenceString(seed, 80)
	if seed == "" {
		return "workspace"
	}
	return seed + " workspace"
}

func providerStatus(provider, model string) string {
	if strings.TrimSpace(provider) != "" && strings.TrimSpace(model) != "" {
		return "configured"
	}
	return "unconfigured"
}

func providerEvidence(provider, model string) []string {
	if providerStatus(provider, model) == "configured" {
		return []string{"provider_configured", "template_validated"}
	}
	return []string{"provider_unconfigured", "template_fallback"}
}

func safeEvidenceString(raw string, maxRunes int) string {
	raw = normalizeSeed(raw)
	if raw == "" {
		return ""
	}
	raw = secretLikeSeedPattern.ReplaceAllString(raw, "$1=[redacted]")
	if maxRunes <= 0 {
		maxRunes = 120
	}
	runes := []rune(raw)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes-3]) + "..."
	}
	return raw
}

func appendMissing(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, addition := range additions {
		if _, ok := seen[addition]; ok {
			continue
		}
		values = append(values, addition)
		seen[addition] = struct{}{}
	}
	return values
}
