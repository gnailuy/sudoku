package cmd

import (
	"fmt"

	"github.com/gnailuy/sudoku/generationexperiment"
	"github.com/spf13/cobra"
)

func newExperimentCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "experiment",
		Short: "Run isolated development experiments",
		Args:  cobra.NoArgs,
	}
	command.AddCommand(newGenerationExperimentCommand())
	return command
}

func newGenerationExperimentCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "generation",
		Short: "Compare exact-grade generation arms",
		Long: `Run the manifest-bound generate-from-scratch baseline and trace-guided
mutation candidate with equal budgets. Results are append-only and resumable,
and the command never opens or mutates the live puzzle database.`,
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			manifest, _ := command.Flags().GetString("manifest")
			output, _ := command.Flags().GetString("output")
			result, err := generationexperiment.Run(manifest, output, generationexperiment.NewExecutor(solverStore))
			if err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "Observed %d/%d jobs (%d new).\nManifest SHA-256: %s\nResults: %s\n", result.Report.Observed, result.Report.Total, result.Appended, result.ManifestHash, output)
			return nil
		},
	}
	command.Flags().String("manifest", "", "Path to immutable generation experiment manifest")
	command.Flags().String("output", "", "Directory for observations, checkpoint, and report")
	_ = command.MarkFlagRequired("manifest")
	_ = command.MarkFlagRequired("output")
	return command
}
