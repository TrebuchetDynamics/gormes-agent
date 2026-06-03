package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// BuildProvenance is the shared build metadata embedded in secrets JSON output.
type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

// Options supplies CLI-specific seams while keeping secrets behavior in this package.
type Options struct {
	BuildProvenance func() BuildProvenance
}

func (o Options) buildProvenance() BuildProvenance {
	if o.BuildProvenance != nil {
		return o.BuildProvenance()
	}
	return BuildProvenance{}
}

// NewCommand builds the secrets command tree.
func NewCommand(options Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "secrets",
		Short:        "Apply, audit, configure, and reload SecretRef-backed runtime secrets",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(newApplyCommand(options))
	cmd.AddCommand(newAuditCommand(options))
	cmd.AddCommand(newConfigureCommand(options))
	cmd.AddCommand(newReloadCommand(options))
	return cmd
}

func newApplyCommand(options Options) *cobra.Command {
	var planPath string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "apply --plan <file>",
		Short: "Resolve a generated SecretRef plan into the runtime snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			planFile, err := LoadPlanFile(planPath)
			if err != nil {
				return err
			}
			controller, err := NewRuntimeController(planFile)
			if err != nil {
				return err
			}
			result, err := controller.Apply(cmd.Context(), planFile.Plan())
			if err == nil {
				err = WriteSnapshotFile(result.Snapshot)
			}
			writeApplyResult(cmd, result, jsonOut, options)
			return err
		},
	}
	cmd.Flags().StringVar(&planPath, "plan", "", "JSON plan containing SecretRef targets")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("plan")
	return cmd
}

func newAuditCommand(options Options) *cobra.Command {
	var planPath string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "audit --plan <file>",
		Short: "Audit plaintext secrets, unresolved refs, and snapshot precedence drift",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(planPath) == "" {
				const msg = `secrets audit: required flag "--plan <file>" not set`
				if jsonOut {
					return emitJSONInputError(cmd, "missing_flag", msg, options)
				}
				return fmt.Errorf("%s", msg)
			}
			planFile, err := LoadPlanFile(planPath)
			if err != nil {
				return err
			}
			previous, err := ReadSnapshotFile()
			if err != nil {
				return err
			}
			result := toolspkg.AuditSecrets(cmd.Context(), toolspkg.SecretsAuditRequest{
				Resolver:         NewConfigSecretResolver(planFile.Secrets),
				Plan:             planFile.Plan(),
				PreviousSnapshot: &previous,
			})
			writeAuditResult(cmd, result, jsonOut, options)
			if !result.OK {
				return newExitCodeError(1, errors.New("secrets audit found findings"))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&planPath, "plan", "", "JSON plan containing SecretRef targets")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	return cmd
}

func newConfigureCommand(options Options) *cobra.Command {
	var source string
	var provider string
	var id string
	var optional bool
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "configure <path>",
		Short: "Build and preflight a typed SecretRef mapping for one config path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if provider == "" {
				provider = config.DefaultSecretProviderAlias
			}
			result, err := toolspkg.ConfigureSecretRef(cmd.Context(), toolspkg.SecretsConfigureRequest{
				Resolver: NewConfigSecretResolver(config.SecretsCfg{}),
				Path:     args[0],
				Required: !optional,
				Ref: toolspkg.SecretRef{
					Source:   source,
					Provider: provider,
					ID:       id,
				},
			})
			writeConfigureResult(cmd, result, jsonOut, options)
			return err
		},
	}
	cmd.Flags().StringVar(&source, "source", "env", "SecretRef source: env or file")
	cmd.Flags().StringVar(&provider, "provider", config.DefaultSecretProviderAlias, "SecretRef provider alias")
	cmd.Flags().StringVar(&id, "id", "", "SecretRef id, such as an environment variable name")
	cmd.Flags().BoolVar(&optional, "optional", false, "allow preflight failure for optional refs")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newReloadCommand(options Options) *cobra.Command {
	var planPath string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "reload --plan <file>",
		Short: "Atomically re-resolve SecretRefs and keep the last-good snapshot on failure",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			planFile, err := LoadPlanFile(planPath)
			if err != nil {
				return err
			}
			controller, err := NewRuntimeController(planFile)
			if err != nil {
				return err
			}
			result, err := controller.Reload(cmd.Context(), planFile.Plan())
			if err == nil {
				err = WriteSnapshotFile(result.Snapshot)
			}
			writeApplyResult(cmd, result, jsonOut, options)
			return err
		},
	}
	cmd.Flags().StringVar(&planPath, "plan", "", "JSON plan containing SecretRef targets")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("plan")
	return cmd
}

// PlanFile is the JSON plan consumed by secrets commands.
type PlanFile struct {
	Targets []toolspkg.SecretTarget `json:"targets"`
	Secrets config.SecretsCfg       `json:"secrets,omitempty"`
}

func (p PlanFile) Plan() toolspkg.SecretsPlan {
	return toolspkg.SecretsPlan{Targets: p.Targets}
}

func LoadPlanFile(path string) (PlanFile, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return PlanFile{}, errors.New("gormes secrets: --plan is required")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return PlanFile{}, fmt.Errorf("gormes secrets: read plan: %w", err)
	}
	var plan PlanFile
	if err := json.Unmarshal(body, &plan); err != nil {
		return PlanFile{}, fmt.Errorf("gormes secrets: parse plan: %w", err)
	}
	return plan, nil
}

func NewRuntimeController(plan PlanFile) (*toolspkg.SecretsRuntimeController, error) {
	snapshot, err := ReadSnapshotFile()
	if err != nil {
		return nil, err
	}
	return toolspkg.NewSecretsRuntimeController(toolspkg.SecretsRuntimeControllerConfig{
		Resolver:        NewConfigSecretResolver(plan.Secrets),
		InitialSnapshot: &snapshot,
	}), nil
}

type ConfigSecretResolver struct {
	resolver *config.SecretResolver
}

func NewConfigSecretResolver(secrets config.SecretsCfg) ConfigSecretResolver {
	return ConfigSecretResolver{resolver: config.NewSecretResolver(config.SecretResolverConfig{Secrets: secrets})}
}

func (r ConfigSecretResolver) ResolveSecretString(ref toolspkg.SecretRef) (string, toolspkg.SecretRefEvidence, error) {
	value, evidence, err := r.resolver.ResolveString(config.SecretRef{
		Source:   config.SecretRefSource(ref.Source),
		Provider: ref.Provider,
		ID:       ref.ID,
	})
	return value, toolspkg.SecretRefEvidence{
		Code:     evidence.Code,
		Source:   evidence.Source,
		Provider: evidence.Provider,
		ID:       evidence.ID,
		Redacted: evidence.Redacted,
	}, err
}

func ReadSnapshotFile() (toolspkg.SecretsRuntimeSnapshot, error) {
	path := SnapshotPath()
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return toolspkg.SecretsRuntimeSnapshot{Entries: map[string]toolspkg.SecretsRuntimeEntry{}, Redacted: true}, nil
	}
	if err != nil {
		return toolspkg.SecretsRuntimeSnapshot{}, fmt.Errorf("gormes secrets: read snapshot: %w", err)
	}
	var snapshot toolspkg.SecretsRuntimeSnapshot
	if err := json.Unmarshal(body, &snapshot); err != nil {
		return toolspkg.SecretsRuntimeSnapshot{}, fmt.Errorf("gormes secrets: parse snapshot: %w", err)
	}
	if snapshot.Entries == nil {
		snapshot.Entries = map[string]toolspkg.SecretsRuntimeEntry{}
	}
	snapshot.Redacted = true
	return snapshot, nil
}

func WriteSnapshotFile(snapshot toolspkg.SecretsRuntimeSnapshot) error {
	path := SnapshotPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("gormes secrets: mkdir snapshot dir: %w", err)
	}
	snapshot.Redacted = true
	body, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("gormes secrets: marshal snapshot: %w", err)
	}
	body = append(body, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".secrets-runtime.*")
	if err != nil {
		return fmt.Errorf("gormes secrets: create snapshot temp: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("gormes secrets: write snapshot temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("gormes secrets: chmod snapshot temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("gormes secrets: close snapshot temp: %w", err)
	}
	if _, err := toolspkg.AtomicReplace(tmpName, path, toolspkg.AtomicReplaceOptions{FirstWriteMode: 0o600}); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("gormes secrets: replace snapshot: %w", err)
	}
	return nil
}

func SnapshotPath() string {
	return filepath.Join(config.GormesHome(), "secrets-runtime.json")
}

func writeApplyResult(cmd *cobra.Command, result toolspkg.SecretsApplyResult, jsonOut bool, options Options) {
	if jsonOut {
		writeJSON(cmd, result, options)
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s generation=%d entries=%d redacted=%t\n", result.Code, result.Snapshot.Generation, len(result.Snapshot.Entries), result.Redacted)
	for _, finding := range result.Findings {
		fmt.Fprintf(cmd.OutOrStdout(), "%s path=%s severity=%s evidence=%s redacted=%t\n", finding.Code, finding.Path, finding.Severity, finding.Evidence.Code, finding.Redacted)
	}
}

func writeAuditResult(cmd *cobra.Command, result toolspkg.SecretsAuditResult, jsonOut bool, options Options) {
	if jsonOut {
		writeJSON(cmd, result, options)
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s ok=%t findings=%d redacted=%t\n", result.Code, result.OK, len(result.Findings), result.Redacted)
	for _, finding := range result.Findings {
		fmt.Fprintf(cmd.OutOrStdout(), "%s path=%s severity=%s evidence=%s redacted=%t\n", finding.Code, finding.Path, finding.Severity, finding.Evidence.Code, finding.Redacted)
	}
}

func writeConfigureResult(cmd *cobra.Command, result toolspkg.SecretsConfigureResult, jsonOut bool, options Options) {
	if jsonOut {
		writeJSON(cmd, result, options)
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s path=%s source=%s provider=%s id=%s preflight_ok=%t redacted=%t\n", result.Code, result.Target.Path, result.Target.Ref.Source, result.Target.Ref.Provider, result.Target.Ref.ID, result.PreflightOK, result.Redacted)
}

func writeJSON(cmd *cobra.Command, value any, options Options) {
	bodyBytes, err := json.Marshal(value)
	if err != nil {
		return
	}
	merged := map[string]json.RawMessage{}
	if jerr := json.Unmarshal(bodyBytes, &merged); jerr != nil {
		return
	}
	buildBytes, err := json.Marshal(options.buildProvenance())
	if err != nil {
		return
	}
	merged["build"] = buildBytes
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	_ = enc.Encode(merged)
}

type jsonInputErrorReport struct {
	Build  BuildProvenance `json:"build"`
	Action string          `json:"action"`
	Error  string          `json:"error"`
}

func emitJSONInputError(cmd *cobra.Command, action, errMsg string, options Options) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(jsonInputErrorReport{
		Build:  options.buildProvenance(),
		Action: action,
		Error:  errMsg,
	})
	return newExitCodeError(1, fmt.Errorf("%s", errMsg))
}

type exitCodeError struct {
	code int
	err  error
}

func newExitCodeError(code int, err error) error {
	if err == nil {
		err = fmt.Errorf("exit code %d", code)
	}
	return exitCodeError{code: code, err: err}
}

func (e exitCodeError) Error() string { return e.err.Error() }
func (e exitCodeError) Unwrap() error { return e.err }
func (e exitCodeError) ExitCode() int { return e.code }
