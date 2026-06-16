package gormescmd

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/spf13/cobra"
)

func TestCLIContractRegistryOwnsLiveCobraSetupAndSlashSurfaces(t *testing.T) {
	registry, err := gormescli.NewRegistry(gormescli.DefaultContracts())
	if err != nil {
		t.Fatalf("NewRegistry(DefaultContracts()) error = %v", err)
	}

	cobraPaths := collectVisibleCommandPathStrings(newRootCommandWithRuntime(rootRuntime{}))
	manifest, err := registry.CommandManifest(cobraPaths)
	if err != nil {
		t.Fatalf("live Cobra command paths must all have module owners: %v", err)
	}
	if len(manifest) != len(cobraPaths) {
		t.Fatalf("manifest entries = %d, Cobra paths = %d", len(manifest), len(cobraPaths))
	}
	assertManifestOwner(t, manifest, "profile", "profiles")
	assertManifestOwner(t, manifest, "profile use", "profiles")
	assertManifestOwner(t, manifest, "gateway status", "gateway")
	assertManifestOwner(t, manifest, "auth status", "providers")

	if err := setupRegistry.ValidateContracts(registry); err != nil {
		t.Fatalf("setup sections must all have matching module owners: %v", err)
	}

	slashNames := make([]string, 0, len(cli.CommandRegistry))
	for _, policy := range cli.CommandRegistry {
		slashNames = append(slashNames, policy.Name)
	}
	if err := registry.ValidateSlashCommands(slashNames); err != nil {
		t.Fatalf("slash commands must all have module owners: %v", err)
	}
}

func collectVisibleCommandPathStrings(root *cobra.Command) []string {
	paths := collectVisibleCommandPaths(root, nil)
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, strings.Join(path, " "))
	}
	return out
}

func assertManifestOwner(t *testing.T, manifest []gormescli.CommandManifestEntry, path, module string) {
	t.Helper()
	for _, entry := range manifest {
		if entry.Path == path {
			if entry.Module != module {
				t.Fatalf("manifest owner for %q = %q, want %q", path, entry.Module, module)
			}
			return
		}
	}
	t.Fatalf("manifest missing command path %q", path)
}
