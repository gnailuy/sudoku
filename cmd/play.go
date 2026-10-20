package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gnailuy/sudoku/cli"
	"github.com/spf13/cobra"
)

func runPlay(cmd *cobra.Command) error {
	input, _ := cmd.Flags().GetString("input")
	level, _ := cmd.Flags().GetString("level")
	resume, _ := cmd.Flags().GetString("resume")
	dbPath, _ := cmd.Flags().GetString("db")
	fromDB, _ := cmd.Flags().GetBool("from-db")
	newGame, _, tracker, err := createTrackedSession(sessionRequest{input: input, level: level, resume: resume, dbPath: dbPath, fromDB: fromDB}, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctrl := cli.NewTrackedControllerWithContext(ctx, &newGame, tracker)
	ctrl.Play()
	return nil
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
