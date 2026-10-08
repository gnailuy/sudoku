package cmd

import (
	"fmt"

	"github.com/gnailuy/sudoku/difficultyaudit"
	"github.com/spf13/cobra"
)

func newDifficultyAuditCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "audit",
		Short: "Create an immutable full-catalog difficulty evidence artifact",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			manifest, _ := command.Flags().GetString("manifest")
			output, _ := command.Flags().GetString("output")
			commit, _ := command.Flags().GetString("repository-commit")
			workers, _ := command.Flags().GetInt("workers")
			result, err := difficultyaudit.Run(difficultyaudit.Options{ManifestPath: manifest, OutputDir: output, RepositoryCommit: commit, Workers: workers}, solverStore)
			if err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "Audited %d puzzles.\nManifest SHA-256: %s\nResults: %s\n", result.Report.Total, result.Report.ManifestHash, output)
			return nil
		},
	}
	command.Flags().String("manifest", "", "Path to immutable version 2 analysis manifest")
	command.Flags().String("output", "", "New directory for evidence.jsonl and reports")
	command.Flags().String("repository-commit", "", "Exact source commit used for classification")
	command.Flags().Int("workers", 1, "Parallel classification workers")
	_ = command.MarkFlagRequired("manifest")
	_ = command.MarkFlagRequired("output")
	_ = command.MarkFlagRequired("repository-commit")
	return command
}
