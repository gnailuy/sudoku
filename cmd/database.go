package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gnailuy/sudoku/db"
	"github.com/spf13/cobra"
)

func newDatabaseCommand() *cobra.Command {
	command := &cobra.Command{Use: "db", Short: "Inspect and manage puzzle history"}
	command.AddCommand(newDatabaseStatsCommand(), newDatabaseResetCommand(), newDatabaseRebuildCommand())
	return command
}

func validDatabaseLevel(level string) error {
	if level == "" {
		return nil
	}
	_, err := difficultyForLevel(level)
	return err
}

func resolveDatabasePath(path string) string {
	if path == "" {
		return defaultDBPath()
	}
	return path
}

func displayTime(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func newDatabaseStatsCommand() *cobra.Command {
	var path, level string
	command := &cobra.Command{
		Use:   "stats",
		Short: "Show acquisition and completion statistics",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if err := validDatabaseLevel(level); err != nil {
				return err
			}
			puzzleDB, err := db.Open(resolveDatabasePath(path))
			if err != nil {
				return err
			}
			defer puzzleDB.Close()
			rows, err := puzzleDB.PlayStatistics(level)
			if err != nil {
				return err
			}
			fmt.Fprintln(command.OutOrStdout(), "LEVEL    STORED  NEVER-SELECTED  SELECTED  ACQUISITIONS  COMPLETED  COMPLETIONS  LATEST-SELECTION  LATEST-COMPLETION")
			for _, row := range rows {
				fmt.Fprintf(command.OutOrStdout(), "%-8s %-7d %-15d %-9d %-13d %-10d %-12d %-17s %s\n", row.Level, row.Stored, row.NeverSelected, row.Selected, row.Acquisitions, row.Completed, row.Completions, displayTime(row.LatestSelection), displayTime(row.LatestCompletion))
			}
			return nil
		},
	}
	command.Flags().StringVar(&path, "db", "", "Puzzle database path (defaults to the XDG data directory)")
	command.Flags().StringVarP(&level, "level", "l", "", "Filter by difficulty level")
	return command
}

func newDatabaseResetCommand() *cobra.Command {
	var path, level, history string
	var yes bool
	command := &cobra.Command{
		Use:   "reset-history",
		Short: "Reset acquisition and/or completion history",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if history != "acquisition" && history != "completion" && history != "all" {
				return fmt.Errorf("--history must be acquisition, completion, or all")
			}
			if err := validDatabaseLevel(level); err != nil {
				return err
			}
			resolved := resolveDatabasePath(path)
			puzzleDB, err := db.Open(resolved)
			if err != nil {
				return err
			}
			defer puzzleDB.Close()
			preview, err := puzzleDB.PreviewHistoryReset(level)
			if err != nil {
				return err
			}
			filter := "all levels"
			if level != "" {
				filter = level
			}
			acquisitions, completions := 0, 0
			if history == "acquisition" || history == "all" {
				acquisitions = preview.Acquisitions
			}
			if history == "completion" || history == "all" {
				completions = preview.Completions
			}
			fmt.Fprintf(command.OutOrStdout(), "Database: %s\nHistory: %s\nLevel: %s\nAffected puzzles: %d\nAcquisitions to clear: %d\nCompletions to clear: %d\n", resolved, history, filter, preview.Rows, acquisitions, completions)
			if preview.Rows == 0 {
				return nil
			}
			if !yes {
				inputFile, interactive := command.InOrStdin().(*os.File)
				if !interactive {
					return fmt.Errorf("confirmation requires --yes in non-interactive use")
				}
				info, statErr := inputFile.Stat()
				if statErr != nil || info.Mode()&os.ModeCharDevice == 0 {
					return fmt.Errorf("confirmation requires --yes in non-interactive use")
				}
				fmt.Fprint(command.OutOrStdout(), "Type yes to continue: ")
				answer, readErr := bufio.NewReader(command.InOrStdin()).ReadString('\n')
				if readErr != nil && strings.TrimSpace(answer) == "" {
					return fmt.Errorf("confirmation requires --yes in non-interactive use")
				}
				if strings.TrimSpace(answer) != "yes" {
					fmt.Fprintln(command.OutOrStdout(), "Cancelled; no history was changed.")
					return nil
				}
			}
			if err := puzzleDB.ResetHistory(history, level); err != nil {
				return err
			}
			fmt.Fprintln(command.OutOrStdout(), "History reset complete.")
			return nil
		},
	}
	command.Flags().StringVar(&history, "history", "", "History to reset: acquisition, completion, or all")
	command.Flags().StringVar(&path, "db", "", "Puzzle database path (defaults to the XDG data directory)")
	command.Flags().StringVarP(&level, "level", "l", "", "Filter by difficulty level")
	command.Flags().BoolVar(&yes, "yes", false, "Confirm reset without an interactive prompt")
	_ = command.MarkFlagRequired("history")
	return command
}

func newDatabaseRebuildCommand() *cobra.Command {
	var path string
	var yes bool
	command := &cobra.Command{
		Use:   "rebuild",
		Short: "Destructively rebuild the development database",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			resolved := resolveDatabasePath(path)
			fmt.Fprintf(command.OutOrStdout(), "Database: %s\nAll catalog, play-run, account, identity, and web-session rows will be deleted.\n", resolved)
			if !yes {
				inputFile, interactive := command.InOrStdin().(*os.File)
				if !interactive {
					return fmt.Errorf("destructive rebuild requires --yes in non-interactive use")
				}
				info, statErr := inputFile.Stat()
				if statErr != nil || info.Mode()&os.ModeCharDevice == 0 {
					return fmt.Errorf("destructive rebuild requires --yes in non-interactive use")
				}
				fmt.Fprint(command.OutOrStdout(), "Type rebuild to continue: ")
				answer, readErr := bufio.NewReader(command.InOrStdin()).ReadString('\n')
				if readErr != nil && strings.TrimSpace(answer) == "" {
					return fmt.Errorf("destructive rebuild requires --yes in non-interactive use")
				}
				if strings.TrimSpace(answer) != "rebuild" {
					fmt.Fprintln(command.OutOrStdout(), "Cancelled; the catalog was not changed.")
					return nil
				}
			}
			if err := db.Rebuild(resolved); err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "Database rebuilt at schema version %d.\n", db.SchemaVersion)
			return nil
		},
	}
	command.Flags().StringVar(&path, "db", "", "Puzzle database path (defaults to the XDG data directory)")
	command.Flags().BoolVar(&yes, "yes", false, "Confirm destructive rebuild without an interactive prompt")
	return command
}
