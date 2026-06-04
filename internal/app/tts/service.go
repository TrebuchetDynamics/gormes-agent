package tts

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	speechtts "github.com/TrebuchetDynamics/gormes-agent/internal/speech/tts"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tts",
		Short: "Inspect and manage local text-to-speech assets",
	}
	cmd.AddCommand(newPiperCommand())
	return cmd
}

func newPiperCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "piper",
		Short: "Manage local Piper neural TTS models",
	}
	cmd.AddCommand(newPiperListCommand(), newPiperVoicesCommand(), newPiperInstallCommand(), newPiperCleanCommand(), newPiperRepairCommand())
	return cmd
}

func newPiperListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List cached Piper models",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			cacheDir := speechtts.PiperCacheDir()
			if cacheDir == "" {
				return fmt.Errorf("Piper model cache directory is unavailable")
			}
			fmt.Fprintf(out, "Piper model cache: %s\n", cacheDir)
			statuses := speechtts.CachedPiperModelStatuses()
			usableCount := 0
			for _, status := range statuses {
				if status.Usable {
					usableCount++
				}
			}
			if len(statuses) == 0 {
				fmt.Fprintln(out, "No Piper models cached.")
				return nil
			}
			if usableCount == 0 {
				fmt.Fprintln(out, "No usable Piper models cached.")
			} else {
				fmt.Fprintln(out, "Usable Piper models:")
				for _, status := range statuses {
					if status.Usable {
						fmt.Fprintf(out, "- %s\n", status.Path)
					}
				}
			}
			unusableCount := len(statuses) - usableCount
			if unusableCount > 0 {
				fmt.Fprintln(out, "Unusable Piper models:")
				for _, status := range statuses {
					if !status.Usable {
						fmt.Fprintf(out, "- %s reason=%s\n", status.Path, status.Reason)
					}
				}
			}
			if selected := speechtts.DiscoverCachedPiperModel(""); selected != "" {
				fmt.Fprintf(out, "Selected default: %s\n", selected)
			}
			return nil
		},
	}
}

func newPiperVoicesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "voices",
		Short: "List built-in Piper voice shortcuts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			for _, voice := range speechtts.BuiltinPiperVoices() {
				fmt.Fprintf(out, "- %s language=%s quality=%s file=%s\n", voice.Name, voice.Language, voice.Quality, voice.FileName)
			}
			return nil
		},
	}
}

func newPiperCleanCommand() *cobra.Command {
	var unusable bool
	cmd := &cobra.Command{
		Use:   "clean --unusable",
		Short: "Remove unusable Piper .onnx files from the local cache",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !unusable {
				return fmt.Errorf("refusing to clean Piper cache without --unusable")
			}
			removed, err := speechtts.RemoveUnusablePiperModels()
			out := cmd.OutOrStdout()
			if len(removed) == 0 {
				fmt.Fprintln(out, "No unusable Piper models removed.")
			} else {
				for _, path := range removed {
					fmt.Fprintf(out, "Removed unusable Piper model: %s\n", path)
				}
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&unusable, "unusable", false, "remove cached .onnx files that fail Piper cache validation")
	return cmd
}

func newPiperRepairCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "repair",
		Short: "Fetch missing Piper sidecar metadata for cached known voices",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repaired, err := speechtts.RepairPiperSidecars(cmd.Context())
			out := cmd.OutOrStdout()
			if len(repaired) == 0 {
				fmt.Fprintln(out, "No Piper sidecars repaired.")
			} else {
				for _, path := range repaired {
					fmt.Fprintf(out, "Repaired Piper sidecar: %s\n", path)
				}
			}
			return err
		},
	}
}

func newPiperInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install <path-or-url|voice>",
		Short: "Copy, download, or install a named Piper .onnx model into the local cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			installed, err := speechtts.InstallPiperModel(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Installed Piper model: %s\n", filepath.Clean(installed))
			return nil
		},
	}
}
