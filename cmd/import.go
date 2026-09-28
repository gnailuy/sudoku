package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/solver"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import puzzles from a file into the database",
	Long: `Import puzzles from a text file (one 81-character puzzle string per line).
Each puzzle is normalized, classified by difficulty, deduplicated against
the existing database, and stored. A report is printed showing how many
puzzles were imported, stored, and their difficulty breakdown.

Supported formats:
  - One puzzle per line (81 chars, using 0 or . for empty cells)
  - Lines starting with # are treated as comments and skipped
  - Empty lines are skipped`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImport(cmd)
	},
}

func init() {
	importCmd.Flags().StringP("file", "f", "", "Path to the puzzle file (required)")
	importCmd.Flags().String("source", "imported", "Source label for imported puzzles")
	importCmd.Flags().String("db", "", "Database path (default: $XDG_DATA_HOME/sudoku/puzzles.db)")
	_ = importCmd.MarkFlagRequired("file")

	rootCmd.AddCommand(importCmd)
}

// importReport holds the results of a batch import run.
type importReport struct {
	total      int
	valid      int
	invalid    int
	stored     int
	duplicates int
	byLevel    map[string]int
	duration   time.Duration
}

func runImport(cmd *cobra.Command) error {
	filePath, _ := cmd.Flags().GetString("file")
	source, _ := cmd.Flags().GetString("source")
	dbPath, _ := cmd.Flags().GetString("db")

	if dbPath == "" {
		dbPath = defaultDBPath()
	}

	// Open the input file.
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

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

	fmt.Printf("Importing puzzles from: %s\n", filePath)

	startTime := time.Now()
	store := solver.NewStore()

	report := importReport{
		byLevel: make(map[string]int),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNum++

		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		report.total++

		// Validate the puzzle string.
		puzzleStr := normalizePuzzleInput(line)
		if !core.IsValidSudokuString(puzzleStr) {
			report.invalid++
			fmt.Fprintf(os.Stderr, "  Line %d: invalid puzzle string (skipped)\n", lineNum)
			continue
		}

		// Parse and validate the board.
		board := core.NewEmptyBoard()
		board.FromString(puzzleStr)

		if !board.IsValid() {
			report.invalid++
			fmt.Fprintf(os.Stderr, "  Line %d: invalid board (skipped)\n", lineNum)
			continue
		}

		// Verify the puzzle is solvable.
		solutionCount := store.GetDefaultSolver().CountSolutions(&board)
		if solutionCount == 0 {
			report.invalid++
			fmt.Fprintf(os.Stderr, "  Line %d: unsolvable puzzle (skipped)\n", lineNum)
			continue
		}

		report.valid++

		// Normalize and classify.
		normalizedStr := normalizePuzzleForDB(store, board)
		normalizedBoard := core.NewEmptyBoard()
		normalizedBoard.FromString(normalizedStr)
		classification := solver.ClassifyPuzzle(store, normalizedBoard)
		if classification.Outcome != solver.ClassificationSolved {
			report.invalid++
			fmt.Fprintf(os.Stderr, "  Line %d: strategy classifier could not solve puzzle (skipped)\n", lineNum)
			continue
		}

		// Store in DB.
		inserted, err := puzzleDB.InsertPuzzle(db.Puzzle{
			Puzzle:       normalizedStr,
			Difficulty:   classification.Difficulty,
			Score:        classification.Score,
			MaxTechnique: classification.MaxTechnique,
			Source:       source,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Line %d: database error: %v (skipped)\n", lineNum, err)
			continue
		}

		report.byLevel[classification.Difficulty]++
		if inserted {
			report.stored++
		} else {
			report.duplicates++
		}

		// Progress indicator every 100 puzzles.
		if report.total%100 == 0 {
			fmt.Printf("\r  Progress: %d processed, %d stored, %d duplicates, %d invalid",
				report.total, report.stored, report.duplicates, report.invalid)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	if report.total >= 100 {
		fmt.Println() // newline after progress
	}

	report.duration = time.Since(startTime)
	printImportReport(report)

	return nil
}

// normalizePuzzleInput converts common input formats to the standard format.
// Converts '0' to '.' for empty cells.
func normalizePuzzleInput(s string) string {
	// Handle lines that might have extra characters (spaces, etc.)
	var cleaned strings.Builder
	for _, ch := range s {
		if ch >= '1' && ch <= '9' {
			cleaned.WriteRune(ch)
		} else if ch == '0' || ch == '.' {
			cleaned.WriteByte('.')
		}
		// Skip any other characters (spaces, separators, etc.)
	}
	return cleaned.String()
}

// normalizePuzzleForDB normalizes a puzzle board for database storage.
func normalizePuzzleForDB(store solver.Store, board core.Board) string {
	solvedBoard := board.Copy()
	store.GetDefaultSolver().Solve(&solvedBoard)
	if !solvedBoard.IsSolved() {
		return board.ToString() // fallback
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

	return normalizedPuzzle.ToString()
}

func printImportReport(report importReport) {
	fmt.Println()
	fmt.Println("=== Import Report ===")
	fmt.Printf("Total lines: %d\n", report.total)
	fmt.Printf("Valid: %d\n", report.valid)
	fmt.Printf("Invalid (skipped): %d\n", report.invalid)
	fmt.Printf("Stored (new): %d\n", report.stored)
	fmt.Printf("Duplicates: %d\n", report.duplicates)
	fmt.Printf("Duration: %s\n", report.duration.Round(time.Millisecond))
	fmt.Println()
	fmt.Println("By difficulty:")

	for _, level := range []string{"easy", "medium", "hard", "expert", "evil"} {
		if count, ok := report.byLevel[level]; ok && count > 0 {
			fmt.Printf("  %-8s %d\n", capitalize(level)+":", count)
		}
	}
}
