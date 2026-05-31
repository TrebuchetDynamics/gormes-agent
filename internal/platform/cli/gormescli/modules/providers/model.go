package providers

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func NewModelCommandWithSeams(seams ModelCommandSeams) *cobra.Command {
	chooseModel := normalizedModelChooser(seams)
	cmd := &cobra.Command{
		Use:          "model",
		Short:        "Interactively select the active model/provider",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			picker := cli.NewModelPicker(cli.ModelPickerOptions{
				IsTTY:            seams.IsTTY,
				LoadCurrent:      seams.LoadCurrent,
				ListProviders:    seams.ListProviders,
				ChooseProvider:   seams.ChooseProvider,
				ChooseModel:      chooseModel,
				PersistSelection: seams.PersistSelection,
			})
			selection, err := picker.Pick(cmd.Context())
			if err != nil {
				return fmt.Errorf("gormes model: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "model selection saved: provider=%s model=%s\n", selection.Provider, selection.Model)
			fmt.Fprintf(cmd.OutOrStdout(), "Provider auth was not changed. If credentials are missing, run: gormes auth add %s\n", selection.Provider)
			return nil
		},
	}
	return cmd
}
