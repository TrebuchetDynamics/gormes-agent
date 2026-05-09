package main

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

func newSecretsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "secrets",
		Short:        "Apply, audit, configure, and reload SecretRef-backed runtime secrets",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(newSecretsApplyCommand())
	cmd.AddCommand(newSecretsAuditCommand())
	cmd.AddCommand(newSecretsConfigureCommand())
	cmd.AddCommand(newSecretsReloadCommand())
	return cmd
}

func newSecretsApplyCommand() *cobra.Command {
	var planPath string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "apply --plan <file>",
		Short: "Resolve a generated SecretRef plan into the runtime snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			planFile, err := loadSecretsPlanFile(planPath)
			if err != nil {
				return err
			}
			controller, err := newCLISecretsRuntimeController(planFile)
			if err != nil {
				return err
			}
			result, err := controller.Apply(cmd.Context(), planFile.Plan())
			if err == nil {
				err = writeSecretsSnapshotFile(result.Snapshot)
			}
			writeSecretsApplyResult(cmd, result, jsonOut)
			return err
		},
	}
	cmd.Flags().StringVar(&planPath, "plan", "", "JSON plan containing SecretRef targets")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("plan")
	return cmd
}

func newSecretsAuditCommand() *cobra.Command {
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
					return emitJSONInputError(cmd, "missing_flag", msg)
				}
				return fmt.Errorf("%s", msg)
			}
			planFile, err := loadSecretsPlanFile(planPath)
			if err != nil {
				return err
			}
			previous, err := readSecretsSnapshotFile()
			if err != nil {
				return err
			}
			result := toolspkg.AuditSecrets(cmd.Context(), toolspkg.SecretsAuditRequest{
				Resolver:         newConfigSecretResolver(planFile.Secrets),
				Plan:             planFile.Plan(),
				PreviousSnapshot: &previous,
			})
			writeSecretsAuditResult(cmd, result, jsonOut)
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

func newSecretsConfigureCommand() *cobra.Command {
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
				Resolver: newConfigSecretResolver(config.SecretsCfg{}),
				Path:     args[0],
				Required: !optional,
				Ref: toolspkg.SecretRef{
					Source:   source,
					Provider: provider,
					ID:       id,
				},
			})
			writeSecretsConfigureResult(cmd, result, jsonOut)
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

func newSecretsReloadCommand() *cobra.Command {
	var planPath string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "reload --plan <file>",
		Short: "Atomically re-resolve SecretRefs and keep the last-good snapshot on failure",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			planFile, err := loadSecretsPlanFile(planPath)
			if err != nil {
				return err
			}
			controller, err := newCLISecretsRuntimeController(planFile)
			if err != nil {
				return err
			}
			result, err := controller.Reload(cmd.Context(), planFile.Plan())
			if err == nil {
				err = writeSecretsSnapshotFile(result.Snapshot)
			}
			writeSecretsApplyResult(cmd, result, jsonOut)
			return err
		},
	}
	cmd.Flags().StringVar(&planPath, "plan", "", "JSON plan containing SecretRef targets")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("plan")
	return cmd
}

func newCLISecretsRuntimeController(plan secretsPlanFile) (*toolspkg.SecretsRuntimeController, error) {
	snapshot, err := readSecretsSnapshotFile()
	if err != nil {
		return nil, err
	}
	return toolspkg.NewSecretsRuntimeController(toolspkg.SecretsRuntimeControllerConfig{
		Resolver:        newConfigSecretResolver(plan.Secrets),
		InitialSnapshot: &snapshot,
	}), nil
}

type secretsPlanFile struct {
	Targets []toolspkg.SecretTarget `json:"targets"`
	Secrets config.SecretsCfg       `json:"secrets,omitempty"`
}

func (p secretsPlanFile) Plan() toolspkg.SecretsPlan {
	return toolspkg.SecretsPlan{Targets: p.Targets}
}

func loadSecretsPlanFile(path string) (secretsPlanFile, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return secretsPlanFile{}, errors.New("gormes secrets: --plan is required")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return secretsPlanFile{}, fmt.Errorf("gormes secrets: read plan: %w", err)
	}
	var plan secretsPlanFile
	if err := json.Unmarshal(body, &plan); err != nil {
		return secretsPlanFile{}, fmt.Errorf("gormes secrets: parse plan: %w", err)
	}
	return plan, nil
}

type configSecretResolver struct {
	resolver *config.SecretResolver
}

func newConfigSecretResolver(secrets config.SecretsCfg) configSecretResolver {
	return configSecretResolver{resolver: config.NewSecretResolver(config.SecretResolverConfig{Secrets: secrets})}
}

func (r configSecretResolver) ResolveSecretString(ref toolspkg.SecretRef) (string, toolspkg.SecretRefEvidence, error) {
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

func readSecretsSnapshotFile() (toolspkg.SecretsRuntimeSnapshot, error) {
	path := secretsSnapshotPath()
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

func writeSecretsSnapshotFile(snapshot toolspkg.SecretsRuntimeSnapshot) error {
	path := secretsSnapshotPath()
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

func secretsSnapshotPath() string {
	return filepath.Join(config.GormesHome(), "secrets-runtime.json")
}

func writeSecretsApplyResult(cmd *cobra.Command, result toolspkg.SecretsApplyResult, jsonOut bool) {
	if jsonOut {
		writeSecretsJSON(cmd, result)
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s generation=%d entries=%d redacted=%t\n", result.Code, result.Snapshot.Generation, len(result.Snapshot.Entries), result.Redacted)
	for _, finding := range result.Findings {
		fmt.Fprintf(cmd.OutOrStdout(), "%s path=%s severity=%s evidence=%s redacted=%t\n", finding.Code, finding.Path, finding.Severity, finding.Evidence.Code, finding.Redacted)
	}
}

func writeSecretsAuditResult(cmd *cobra.Command, result toolspkg.SecretsAuditResult, jsonOut bool) {
	if jsonOut {
		writeSecretsJSON(cmd, result)
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s ok=%t findings=%d redacted=%t\n", result.Code, result.OK, len(result.Findings), result.Redacted)
	for _, finding := range result.Findings {
		fmt.Fprintf(cmd.OutOrStdout(), "%s path=%s severity=%s evidence=%s redacted=%t\n", finding.Code, finding.Path, finding.Severity, finding.Evidence.Code, finding.Redacted)
	}
}

func writeSecretsConfigureResult(cmd *cobra.Command, result toolspkg.SecretsConfigureResult, jsonOut bool) {
	if jsonOut {
		writeSecretsJSON(cmd, result)
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s path=%s source=%s provider=%s id=%s preflight_ok=%t redacted=%t\n", result.Code, result.Target.Path, result.Target.Ref.Source, result.Target.Ref.Provider, result.Target.Ref.ID, result.PreflightOK, result.Redacted)
}

// writeSecretsJSON encodes a `secrets ... --json` payload with the
// shared `build` provenance block merged inline alongside the underlying
// result fields. Same convention as
// update --json / doctor --json / status --json / restore --list --json /
// auth status --json — captured secrets-runtime snapshots stay
// attributable to a specific binary. Inline merge (rather than nesting
// under a wrapper) keeps existing consumers parsing into the raw result
// struct working (Go's json decoder ignores the extra `build` key).
func writeSecretsJSON(cmd *cobra.Command, value any) {
	bodyBytes, err := json.Marshal(value)
	if err != nil {
		return
	}
	merged := map[string]json.RawMessage{}
	if jerr := json.Unmarshal(bodyBytes, &merged); jerr != nil {
		return
	}
	buildBytes, err := json.Marshal(newBuildProvenance())
	if err != nil {
		return
	}
	merged["build"] = buildBytes
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	_ = enc.Encode(merged)
}
