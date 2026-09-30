package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/replenisher"
	"github.com/spf13/cobra"
)

func newReplenishCommand() *cobra.Command {
	var path, level, state string
	var count, classifications int
	var randomSeed int64
	command := &cobra.Command{
		Use:   "replenish",
		Short: "Build and atomically publish exact-grade catalog additions",
		Long: `Build an offline Expert or Evil batch from exact-grade catalog seeds.
Search checkpoints after every classification and resumes from --state. The
live catalog changes only after the complete batch validates and publishes in
one transaction.`,
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			catalog, err := db.Open(resolveDatabasePath(path))
			if err != nil {
				return err
			}
			defer catalog.Close()
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			result, err := replenisher.Run(ctx, catalog, replenisher.NewBuilder(solverStore, level), replenisher.Config{
				Difficulty: level, Count: count, RandomSeed: randomSeed,
				MaxClassifications: classifications, StatePath: state,
			})
			fmt.Fprintf(command.OutOrStdout(), "Classifications: %d\nAccepted: %d/%d\nState: %s\n", result.Classifications, result.Accepted, count, state)
			if err != nil {
				return err
			}
			if result.Published {
				fmt.Fprintln(command.OutOrStdout(), "Published: yes")
			} else {
				fmt.Fprintln(command.OutOrStdout(), "Published: already complete")
			}
			return nil
		},
	}
	command.Flags().StringVar(&path, "db", "", "Puzzle database path (defaults to the XDG data directory)")
	command.Flags().StringVarP(&level, "level", "l", "", "Exact seed and candidate grade: expert or evil")
	command.Flags().IntVarP(&count, "count", "n", 1, "Number of exact-grade additions to publish atomically")
	command.Flags().Int64Var(&randomSeed, "seed", 1, "Deterministic search seed")
	command.Flags().IntVar(&classifications, "classifications", 100, "Maximum candidate classifications")
	command.Flags().StringVar(&state, "state", "", "Durable resumable state file")
	_ = command.MarkFlagRequired("level")
	_ = command.MarkFlagRequired("state")
	return command
}
