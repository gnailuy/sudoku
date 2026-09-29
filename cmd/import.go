package cmd

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	importCmd.Flags().String("format", "plain", "Input format: plain or sudoku-exchange")
	importCmd.Flags().String("sha256", "", "Expected lowercase SHA-256 of the complete input file")
	importCmd.Flags().IntP("workers", "w", 1, "Parallel canonicalization/classification workers")
	importCmd.Flags().String("db", "", "Database path (default: $XDG_DATA_HOME/sudoku/puzzles.db)")
	_ = importCmd.MarkFlagRequired("file")

	rootCmd.AddCommand(importCmd)
}

// importReport holds the results of a batch import run.
type importReport struct {
	total            int
	valid            int
	invalid          int
	stored           int
	duplicates       int
	strategyUnsolved int
	databaseErrors   int
	byLevel          map[string]int
	duration         time.Duration
}

func runImport(cmd *cobra.Command) error {
	filePath, _ := cmd.Flags().GetString("file")
	source, _ := cmd.Flags().GetString("source")
	inputFormat, _ := cmd.Flags().GetString("format")
	expectedSHA256, _ := cmd.Flags().GetString("sha256")
	workers, _ := cmd.Flags().GetInt("workers")
	dbPath, _ := cmd.Flags().GetString("db")

	if workers < 1 {
		return fmt.Errorf("workers must be positive")
	}

	if inputFormat != "plain" && inputFormat != "sudoku-exchange" {
		return fmt.Errorf("unsupported import format %q", inputFormat)
	}
	if inputFormat == "sudoku-exchange" && expectedSHA256 == "" {
		return fmt.Errorf("sudoku-exchange imports require --sha256 to pin the source file")
	}
	if expectedSHA256 != "" {
		actual, err := fileSHA256(filePath)
		if err != nil {
			return err
		}
		if actual != expectedSHA256 {
			return fmt.Errorf("input SHA-256 is %s, want %s", actual, expectedSHA256)
		}
	}

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
	report := importReport{byLevel: make(map[string]int)}

	var records []importRecord
	scanner := bufio.NewScanner(file)
	for lineNum := 1; scanner.Scan(); lineNum++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		records = append(records, importRecord{lineNumber: lineNum, line: line})
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	report.total = len(records)

	jobs := make(chan importRecord)
	results := make(chan importResult, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			store := solver.NewStore()
			for record := range jobs {
				results <- prepareImportRecord(store, record, inputFormat)
			}
		}()
	}
	go func() {
		for _, record := range records {
			jobs <- record
		}
		close(jobs)
		group.Wait()
		close(results)
	}()

	processed := 0
	for result := range results {
		processed++
		if result.err != nil {
			report.invalid++
			fmt.Fprintf(os.Stderr, "  Line %d: %v (skipped)\n", result.lineNumber, result.err)
			continue
		}
		report.valid++
		if result.classification.Outcome != solver.ClassificationSolved {
			report.strategyUnsolved++
			continue
		}

		inserted, err := puzzleDB.InsertPuzzle(db.Puzzle{
			Puzzle:       result.puzzle,
			Difficulty:   result.classification.Difficulty,
			Score:        result.classification.Score,
			MaxTechnique: result.classification.MaxTechnique,
			Source:       source,
			SourceRef:    result.sourceRef,
		})
		if err != nil {
			report.databaseErrors++
			fmt.Fprintf(os.Stderr, "  Line %d: database error: %v (skipped)\n", result.lineNumber, err)
			continue
		}

		report.byLevel[result.classification.Difficulty]++
		if inserted {
			report.stored++
		} else {
			report.duplicates++
		}
		if processed%100 == 0 || processed == report.total {
			fmt.Printf("\r  Progress: %d/%d processed, %d stored, %d duplicates, %d strategy-unsolved, %d invalid",
				processed, report.total, report.stored, report.duplicates, report.strategyUnsolved, report.invalid)
		}
	}

	if report.total >= 100 {
		fmt.Println()
	}

	report.duration = time.Since(startTime)
	printImportReport(report)

	return nil
}

type importRecord struct {
	lineNumber int
	line       string
}

type importResult struct {
	lineNumber     int
	puzzle         string
	sourceRef      string
	classification solver.Classification
	err            error
}

func prepareImportRecord(store solver.Store, record importRecord, inputFormat string) importResult {
	result := importResult{lineNumber: record.lineNumber}
	puzzleText, sourceRef, err := parseImportRecord(record.line, inputFormat)
	if err != nil {
		result.err = err
		return result
	}
	puzzle := normalizePuzzleInput(puzzleText)
	if !core.IsValidSudokuString(puzzle) {
		result.err = fmt.Errorf("invalid puzzle string")
		return result
	}
	board := core.NewEmptyBoard()
	board.FromString(puzzle)
	if !board.IsValid() {
		result.err = fmt.Errorf("invalid board")
		return result
	}
	// The hash-pinned Sudoku Exchange source guarantees unique solutions.
	// Generic files retain an independent uniqueness check.
	if inputFormat == "plain" && store.GetDefaultSolver().CountSolutions(&board) == 0 {
		result.err = fmt.Errorf("unsolvable puzzle")
		return result
	}
	result.puzzle = normalizePuzzleForDB(store, board)
	result.sourceRef = sourceRef
	canonicalBoard := core.NewEmptyBoard()
	canonicalBoard.FromString(result.puzzle)
	result.classification = solver.ClassifyPuzzle(store, canonicalBoard)
	return result
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

// normalizePuzzleForDB canonicalizes a puzzle for catalog identity.
func normalizePuzzleForDB(_ solver.Store, board core.Board) string {
	return core.CanonicalPuzzle(board.ToString())
}

func parseImportRecord(line, inputFormat string) (puzzle, sourceRef string, err error) {
	if inputFormat == "plain" {
		return line, "", nil
	}
	fields := strings.Fields(line)
	if len(fields) != 3 || len(fields[0]) != 12 || len(fields[1]) != 81 {
		return "", "", fmt.Errorf("invalid sudoku-exchange record")
	}
	for _, character := range fields[0] {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return "", "", fmt.Errorf("invalid sudoku-exchange source hash")
		}
	}
	if _, parseErr := strconv.ParseFloat(fields[2], 64); parseErr != nil {
		return "", "", fmt.Errorf("invalid sudoku-exchange rating")
	}
	return fields[1], "sha1:" + fields[0], nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for SHA-256: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func printImportReport(report importReport) {
	fmt.Println()
	fmt.Println("=== Import Report ===")
	fmt.Printf("Total lines: %d\n", report.total)
	fmt.Printf("Valid: %d\n", report.valid)
	fmt.Printf("Invalid (skipped): %d\n", report.invalid)
	fmt.Printf("Strategy-unsolved (skipped): %d\n", report.strategyUnsolved)
	fmt.Printf("Storage failures (skipped): %d\n", report.databaseErrors)
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
