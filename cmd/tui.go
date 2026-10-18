package cmd

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/recovery"
	front "github.com/gnailuy/sudoku/tui"
	"github.com/spf13/cobra"
)

func newTUICommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "tui",
		Short: "Play in the full-screen Bubble Tea interface",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			input, _ := command.Flags().GetString("input")
			level, _ := command.Flags().GetString("level")
			resume, _ := command.Flags().GetString("resume")
			noAutosave, _ := command.Flags().GetBool("no-autosave")
			dbPath, _ := command.Flags().GetString("db")
			if dbPath == "" {
				catalog, resolvedDBPath, err := openCatalogDatabase("")
				if err != nil {
					return err
				}
				_ = catalog.Close()
				dbPath = resolvedDBPath
			}

			var recoveryOptions front.RecoveryOptions
			if !noAutosave {
				directory, locationErr := recovery.DefaultDirectory()
				if locationErr != nil {
					recoveryOptions.Warning = "Autosave unavailable: " + locationErr.Error()
				} else {
					store := recovery.NewStore(directory)
					recoveryOptions.Store = &store
					plainStartup := input == "" && resume == "" && !command.Flags().Changed("level")
					if plainStartup {
						options := game.NewDefaultOptions(solverStore)
						options.StrategySolverKeys = solverStore.GetAllStrategySolverKeys()
						records, discoverErr := store.Discover(func(data []byte) error {
							_, restoreErr := game.Restore(data, options)
							return restoreErr
						})
						if discoverErr != nil {
							recoveryOptions.Warning = "Recovery unavailable: " + discoverErr.Error()
						} else {
							for _, record := range records {
								restored, restoreErr := game.Restore(record.Session, options)
								if restoreErr == nil {
									recoveryOptions.Choices = append(recoveryOptions.Choices, front.RecoveryChoice{Record: record, Game: restored, Tracker: newCompletionTracker(restored, dbPath)})
								}
							}
						}
					}
				}
			}

			current, resumePath, tracker, err := createTrackedSession(sessionRequest{input: input, level: level, resume: resume, dbPath: dbPath}, command.OutOrStdout(), command.ErrOrStderr())
			if err != nil {
				return err
			}
			program := tea.NewProgram(front.NewTrackedModelWithRecovery(current, resumePath, recoveryOptions, tracker), tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
			_, err = program.Run()
			return err
		},
	}
	command.Flags().StringP("input", "i", "", "Specify a Sudoku problem string to play")
	command.Flags().StringP("level", "l", "hard", "Difficulty level: easy, medium, hard, expert, evil")
	command.Flags().String("resume", "", "Resume a saved game session")
	command.Flags().Bool("no-autosave", false, "Disable TUI background autosave and recovery")
	command.Flags().String("db", "", "Puzzle database path (defaults to the managed application data directory)")
	command.MarkFlagsMutuallyExclusive("resume", "input")
	command.MarkFlagsMutuallyExclusive("resume", "level")
	return command
}
