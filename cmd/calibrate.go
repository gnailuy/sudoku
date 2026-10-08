package cmd

import (
	"fmt"

	"github.com/gnailuy/sudoku/calibration"
	"github.com/spf13/cobra"
)

func newCalibrateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "calibrate",
		Short: "Measure difficulty for an immutable puzzle corpus",
		Long: `Classify an ordered JSON corpus and persist append-only observations,
a resumable checkpoint, and deterministic JSON and Markdown reports.
Re-running with the same manifest and output directory resumes safely.`,
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			manifest, _ := command.Flags().GetString("manifest")
			output, _ := command.Flags().GetString("output")
			result, err := calibration.Run(manifest, output, solverStore)
			if err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "Measured %d/%d puzzles (%d new).\nManifest SHA-256: %s\nResults: %s\n", result.Report.Observed, result.Report.Total, result.Appended, result.ManifestHash, output)
			return nil
		},
	}
	command.Flags().String("manifest", "", "Path to immutable JSON corpus manifest")
	command.Flags().String("output", "", "Directory for observations, checkpoint, and reports")
	_ = command.MarkFlagRequired("manifest")
	_ = command.MarkFlagRequired("output")
	command.AddCommand(newPrepareCorpusCommand())
	command.AddCommand(newDifficultyAuditCommand())
	return command
}

func newPrepareCorpusCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "prepare",
		Short: "Normalize and validate a candidate corpus manifest",
		Long: `Normalize zero and whitespace notation, add SHA-256 content hashes,
reject duplicate content, validate provenance and split metadata, and write a
canonical immutable manifest for the calibration runner.`,
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			input, _ := command.Flags().GetString("input")
			output, _ := command.Flags().GetString("output")
			manifest, err := calibration.PrepareManifest(input, output)
			if err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "Prepared %d normalized puzzles: %s\n", len(manifest.Puzzles), output)
			return nil
		},
	}
	command.Flags().String("input", "", "Path to candidate version 2 JSON corpus")
	command.Flags().String("output", "", "Path for the canonical immutable manifest")
	_ = command.MarkFlagRequired("input")
	_ = command.MarkFlagRequired("output")
	return command
}
