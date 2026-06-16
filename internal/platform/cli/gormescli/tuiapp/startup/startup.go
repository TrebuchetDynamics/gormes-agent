package startup

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// CommandBoolFlag resolves a bool flag from cmd, inherited flags, or the root command.
func CommandBoolFlag(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}
	if flags := cmd.Flags(); flags != nil && flags.Lookup(name) != nil {
		value, _ := flags.GetBool(name)
		return value
	}
	if flags := cmd.PersistentFlags(); flags != nil && flags.Lookup(name) != nil {
		value, _ := flags.GetBool(name)
		return value
	}
	if flags := cmd.InheritedFlags(); flags != nil && flags.Lookup(name) != nil {
		value, _ := flags.GetBool(name)
		return value
	}
	if root := cmd.Root(); root != nil && root != cmd {
		if flags := root.Flags(); flags != nil && flags.Lookup(name) != nil {
			value, _ := flags.GetBool(name)
			return value
		}
		if flags := root.PersistentFlags(); flags != nil && flags.Lookup(name) != nil {
			value, _ := flags.GetBool(name)
			return value
		}
	}
	return false
}

// CommandStringFlag resolves a string flag from cmd, inherited flags, or the root command.
func CommandStringFlag(cmd *cobra.Command, name string) string {
	if cmd == nil {
		return ""
	}
	if flags := cmd.Flags(); flags != nil && flags.Lookup(name) != nil {
		value, _ := flags.GetString(name)
		return value
	}
	if flags := cmd.PersistentFlags(); flags != nil && flags.Lookup(name) != nil {
		value, _ := flags.GetString(name)
		return value
	}
	if flags := cmd.InheritedFlags(); flags != nil && flags.Lookup(name) != nil {
		value, _ := flags.GetString(name)
		return value
	}
	if root := cmd.Root(); root != nil && root != cmd {
		if flags := root.Flags(); flags != nil && flags.Lookup(name) != nil {
			value, _ := flags.GetString(name)
			return value
		}
		if flags := root.PersistentFlags(); flags != nil && flags.Lookup(name) != nil {
			value, _ := flags.GetString(name)
			return value
		}
	}
	return ""
}

// CommandStringArrayFlag resolves a string-array flag from cmd, inherited flags, or the root command.
func CommandStringArrayFlag(cmd *cobra.Command, name string) []string {
	if cmd == nil {
		return nil
	}
	if flags := cmd.Flags(); flags != nil && flags.Lookup(name) != nil {
		values, _ := flags.GetStringArray(name)
		return values
	}
	if flags := cmd.PersistentFlags(); flags != nil && flags.Lookup(name) != nil {
		values, _ := flags.GetStringArray(name)
		return values
	}
	if flags := cmd.InheritedFlags(); flags != nil && flags.Lookup(name) != nil {
		values, _ := flags.GetStringArray(name)
		return values
	}
	if root := cmd.Root(); root != nil && root != cmd {
		if flags := root.Flags(); flags != nil && flags.Lookup(name) != nil {
			values, _ := flags.GetStringArray(name)
			return values
		}
		if flags := root.PersistentFlags(); flags != nil && flags.Lookup(name) != nil {
			values, _ := flags.GetStringArray(name)
			return values
		}
	}
	return nil
}

// ForcedSkillNames normalizes the root CLI skill allowlist.
func ForcedSkillNames(cmd *cobra.Command) []string {
	raw := CommandStringArrayFlag(cmd, "skills")
	if len(raw) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		for _, part := range strings.Split(value, ",") {
			name := strings.TrimSpace(part)
			if name == "" {
				continue
			}
			key := strings.ToLower(name)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}

// ApplyProviderStartupFlags applies invocation-only provider overrides to cfg.
func ApplyProviderStartupFlags(cfg *config.Config, endpointFlag, apiKeyFlag string) {
	if endpoint := strings.TrimSpace(endpointFlag); endpoint != "" {
		cfg.Hermes.Endpoint = endpoint
	}
	if apiKey := strings.TrimSpace(apiKeyFlag); apiKey != "" {
		cfg.Hermes.APIKey = apiKey
	}
}

// ResolveStaticStartupInference normalizes known model aliases without making
// provider network calls during startup.
func ResolveStaticStartupInference(resolution config.InferenceResolution) config.InferenceResolution {
	if resolution.Model == "" {
		return resolution
	}
	metadata := llm.LookupModelMetadata(llm.ModelRegistryQuery{
		Provider: resolution.Provider,
		Model:    resolution.Model,
	})
	if !metadata.Found {
		return resolution
	}
	resolution.Model = metadata.Model
	if resolution.Provider == "" {
		resolution.Provider = metadata.Provider
		resolution.ProviderAutoDetectRequired = false
	}
	return resolution
}
