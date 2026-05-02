package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

var (
	uninstallDryRun         bool
	uninstallYes            bool
	uninstallKeepConfig     bool
	uninstallKeepCredential bool
)

func newUninstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove Gormes artifacts from this system",
		Long:  "Enumerates every artifact Gormes wrote. Dry-run by default; use --yes to confirm.",
		RunE:  runUninstall,
	}
	cmd.Flags().BoolVar(&uninstallDryRun, "dry-run", true, "List artifacts without deleting")
	cmd.Flags().BoolVar(&uninstallYes, "yes", false, "Confirm destructive removal")
	cmd.Flags().BoolVar(&uninstallKeepConfig, "keep-config", false, "Preserve config files")
	cmd.Flags().BoolVar(&uninstallKeepCredential, "keep-credentials", false, "Preserve credential pool")
	return cmd
}

type artifactGroup struct {
	Name  string
	Paths []string
}

func runUninstall(_ *cobra.Command, _ []string) error {
	home := config.GormesHome()
	groups := collectArtifacts(home)
	if uninstallKeepConfig {
		groups = removeGroup(groups, "config")
	}
	if uninstallKeepCredential {
		groups = removeGroup(groups, "credentials")
	}
	if uninstallDryRun || !uninstallYes {
		return printDryRun(groups)
	}
	return executeUninstall(groups)
}

func collectArtifacts(home string) []artifactGroup {
	return []artifactGroup{
		{Name: "config", Paths: sortedExisting(config.ConfigPath(), config.YAMLConfigPath(),
			filepath.Join(home, ".env"), filepath.Join(home, "config.toml"), filepath.Join(home, "config.yaml"))},
		{Name: "credentials", Paths: sortedExisting(filepath.Join(home, "auth.json"))},
		{Name: "sessions", Paths: sortedExisting(config.SessionDBPath(), config.SessionIndexMirrorPath())},
		{Name: "gateway-state", Paths: sortedExisting(filepath.Join(home, "gateway_state.json"),
			filepath.Join(home, "gateway-locks"), filepath.Join(home, "gateway.pid"),
			filepath.Join(home, "channel_directory_sources.json"))},
		{Name: "memory", Paths: sortedExisting(config.MemoryDBPath(), filepath.Join(home, "memory"))},
		{Name: "logs", Paths: sortedExisting(config.LogPath(), config.CrashLogDir())},
		{Name: "cron", Paths: sortedExisting(filepath.Join(home, "CRON.md"))},
		{Name: "mcp-oauth", Paths: sortedExisting(filepath.Join(home, "mcp_oauth.json"))},
	}
}

func sortedExisting(paths ...string) []string {
	existing := make([]string, 0, len(paths))
	seen := make(map[string]bool)
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		entry := p
		if info.IsDir() {
			entry += "/"
		}
		existing = append(existing, entry)
	}
	sort.Strings(existing)
	return existing
}

func removeGroup(groups []artifactGroup, name string) []artifactGroup {
	out := make([]artifactGroup, 0, len(groups))
	for _, g := range groups {
		if g.Name != name {
			out = append(out, g)
		}
	}
	return out
}

func printDryRun(groups []artifactGroup) error {
	total := 0
	for _, g := range groups {
		total += len(g.Paths)
	}
	fmt.Printf("uninstall dry-run: %d artifact(s)\n\n", total)
	for _, g := range groups {
		fmt.Printf("[%s]\n", g.Name)
		for _, p := range g.Paths {
			fmt.Printf("  %s\n", p)
		}
		fmt.Println()
	}
	if total == 0 {
		fmt.Println("No Gormes artifacts found.")
	}
	return nil
}

func executeUninstall(groups []artifactGroup) error {
	for _, g := range groups {
		fmt.Printf("removing [%s]...\n", g.Name)
		for _, p := range g.Paths {
			clean := strings.TrimSuffix(p, "/")
			if err := os.RemoveAll(clean); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not remove %s: %v\n", clean, err)
			}
		}
	}
	fmt.Println("\nuninstall complete.")
	return nil
}
