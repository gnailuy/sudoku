package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/generator"
	"github.com/gnailuy/sudoku/solver"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Batch-generate puzzles and store them in the database",
	Long: `Generate puzzles at the specified difficulty using best-effort generation.
Each puzzle is classified by actual difficulty and stored in the database.
A report is printed showing how many puzzles were generated, stored, and
their difficulty breakdown.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGenerate(cmd)
	},
}

func init() {
	generateCmd.Flags().IntP("count", "n", 100, "Number of puzzles to generate")
	generateCmd.Flags().StringP("difficulty", "d", "hard", "Target difficulty: easy, medium, hard, expert, evil")
	generateCmd.Flags().DurationP("timeout", "t", 30*time.Second, "Time limit per puzzle generation attempt")
	generateCmd.Flags().Int("rounds", 10, "Max generation rounds per puzzle")
	generateCmd.Flags().IntP("workers", "w", 1, "Number of parallel workers")
	generateCmd.Flags().String("db", "", "Database path (default: $XDG_DATA_HOME/sudoku/puzzles.db)")

	rootCmd.AddCommand(generateCmd)
}

// generateReport holds the results of a batch generation run.
type generateReport struct {
	attempted  int
	generated  int
	stored     int
	duplicates int
	timedOut   int
	matched    int
	byLevel    map[string]int
	duration   time.Duration
}

type puzzleGenerator func(level string, timeout time.Duration, rounds int) generator.GenerationResult

func runGenerate(cmd *cobra.Command) error {
	count, _ := cmd.Flags().GetInt("count")
	level, _ := cmd.Flags().GetString("difficulty")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	rounds, _ := cmd.Flags().GetInt("rounds")
	workers, _ := cmd.Flags().GetInt("workers")
	dbPath, _ := cmd.Flags().GetString("db")

	if count <= 0 {
		return fmt.Errorf("count must be positive, got %d", count)
	}
	if workers <= 0 {
		workers = 1
	}

	// Validate difficulty level.
	_ = parseDifficulty(level)

	if dbPath == "" {
		dbPath = defaultDBPath()
	}

	// Ensure DB directory exists.
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}

	// Open the database.
	puzzleDB, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer puzzleDB.Close()

	fmt.Printf("Generating %d puzzles (target: %s, workers: %d, timeout: %s/puzzle, rounds: %d/puzzle)...\n",
		count, level, workers, timeout, rounds)

	startTime := time.Now()

	// Generate puzzles.
	report := batchGenerate(puzzleDB, count, level, timeout, rounds, workers)
	report.duration = time.Since(startTime)

	// Print report.
	printGenerateReport(report)

	return nil
}

func batchGenerate(puzzleDB *db.DB, count int, level string, timeout time.Duration, rounds int, workers int) generateReport {
	return batchGenerateWith(puzzleDB, count, level, timeout, rounds, workers, generateOnePuzzle)
}

func batchGenerateWith(puzzleDB *db.DB, count int, level string, timeout time.Duration, rounds int, workers int, generate puzzleGenerator) generateReport {
	report := generateReport{
		byLevel: make(map[string]int),
	}

	if workers <= 1 {
		// Single-threaded generation.
		for i := 0; i < count; i++ {
			result := generate(level, timeout, rounds)
			report.attempted++
			if result.TimedOut {
				report.timedOut++
			}
			if result.RoundsUsed == 0 {
				continue
			}
			report.generated++
			if result.Matched {
				report.matched++
			}
			report.byLevel[result.Classification.Difficulty]++
			stored := storePuzzle(puzzleDB, result)
			if stored {
				report.stored++
			} else {
				report.duplicates++
			}

			// Progress indicator every 10 puzzles.
			if (i+1)%10 == 0 || i+1 == count {
				fmt.Printf("\r  Progress: %d/%d attempted, %d generated, %d stored, %d timed out",
					report.attempted, count, report.generated, report.stored, report.timedOut)
			}
		}
		fmt.Println() // newline after progress
	} else {
		// Parallel generation.
		var (
			mu         sync.Mutex
			generated  int64
			stored     int64
			duplicates int64
			timedOut   int64
			matched    int64
			byLevel    = make(map[string]int)
		)

		sem := make(chan struct{}, workers)
		var wg sync.WaitGroup

		for i := 0; i < count; i++ {
			wg.Add(1)
			sem <- struct{}{} // acquire worker slot

			go func() {
				defer wg.Done()
				defer func() { <-sem }() // release worker slot

				result := generate(level, timeout, rounds)
				if result.TimedOut {
					atomic.AddInt64(&timedOut, 1)
				}
				if result.RoundsUsed == 0 {
					return
				}
				wasStored := storePuzzle(puzzleDB, result)

				atomic.AddInt64(&generated, 1)
				if result.Matched {
					atomic.AddInt64(&matched, 1)
				}
				if wasStored {
					atomic.AddInt64(&stored, 1)
				} else {
					atomic.AddInt64(&duplicates, 1)
				}

				mu.Lock()
				byLevel[result.Classification.Difficulty]++
				mu.Unlock()

				g := atomic.LoadInt64(&generated)
				if g%10 == 0 || int(g) == count {
					s := atomic.LoadInt64(&stored)
					d := atomic.LoadInt64(&duplicates)
					fmt.Printf("\r  Progress: %d/%d generated, %d stored, %d duplicates",
						g, count, s, d)
				}
			}()
		}

		wg.Wait()
		fmt.Println() // newline after progress

		report.attempted = count
		report.generated = int(generated)
		report.stored = int(stored)
		report.duplicates = int(duplicates)
		report.timedOut = int(timedOut)
		report.matched = int(matched)
		report.byLevel = byLevel
		return report
	}

	return report
}

func generateOnePuzzle(level string, timeout time.Duration, rounds int) generator.GenerationResult {
	// Each worker gets its own solver store for thread safety.
	store := solver.NewStore()
	difficulty := parseDifficultyQuiet(level)

	opts := generator.NewBestEffortOptions(store, difficulty)
	opts.MaxRounds = rounds
	opts.MaxDurationMs = timeout.Milliseconds()
	if timeout > 0 && opts.MaxDurationMs == 0 {
		opts.MaxDurationMs = 1
	}

	return generator.GenerateBestEffort(opts)
}

func storePuzzle(puzzleDB *db.DB, result generator.GenerationResult) bool {
	store := solver.NewStore()
	board := result.Puzzle

	// Normalize the puzzle for storage.
	solvedBoard := board.Copy()
	store.GetDefaultSolver().Solve(&solvedBoard)
	if !solvedBoard.IsSolved() {
		return false
	}

	normalizedSolved := solvedBoard.Copy()
	normalizedSolved.Normalize()

	var digitMap [10]int
	for col := 0; col < 9; col++ {
		original := solvedBoard.Get(core.NewPosition(0, col))
		normalized := normalizedSolved.Get(core.NewPosition(0, col))
		digitMap[original] = normalized
	}

	normalizedPuzzle := core.NewEmptyBoard()
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			pos := core.NewPosition(row, col)
			val := board.Get(pos)
			if val != 0 {
				_ = normalizedPuzzle.Set(pos, digitMap[val])
			}
		}
	}

	puzzleStr := normalizedPuzzle.ToString()
	classification := solver.ClassifyPuzzle(store, normalizedPuzzle)
	if classification.Outcome != solver.ClassificationSolved {
		return false
	}

	inserted, err := puzzleDB.InsertPuzzle(db.Puzzle{
		Puzzle:       puzzleStr,
		Difficulty:   classification.Difficulty,
		Score:        classification.Score,
		MaxTechnique: classification.MaxTechnique,
		Source:       "generated",
	})
	if err != nil {
		return false
	}
	return inserted
}

func printGenerateReport(report generateReport) {
	fmt.Println()
	fmt.Println("=== Generation Report ===")
	fmt.Printf("Attempted: %d\n", report.attempted)
	fmt.Printf("Generated: %d\n", report.generated)
	fmt.Printf("Stored (new): %d\n", report.stored)
	fmt.Printf("Duplicates: %d\n", report.duplicates)
	fmt.Printf("Timed out: %d\n", report.timedOut)
	fmt.Printf("Matched target: %d\n", report.matched)
	fmt.Printf("Duration: %s\n", report.duration.Round(time.Millisecond))
	fmt.Println()
	fmt.Println("By difficulty:")

	for _, level := range []string{"easy", "medium", "hard", "expert", "evil"} {
		if count, ok := report.byLevel[level]; ok && count > 0 {
			fmt.Printf("  %-8s %d\n", capitalize(level)+":", count)
		}
	}
}

// parseDifficultyQuiet is like parseDifficulty but doesn't exit on error.
func parseDifficultyQuiet(level string) generator.Difficulty {
	switch level {
	case "easy":
		return generator.NewEasyDifficulty()
	case "medium":
		return generator.NewMediumDifficulty()
	case "hard":
		return generator.NewHardDifficulty()
	case "expert":
		return generator.NewExpertDifficulty()
	case "evil":
		return generator.NewEvilDifficulty()
	default:
		return generator.NewHardDifficulty()
	}
}
