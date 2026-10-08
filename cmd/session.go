package cmd

import (
	"fmt"
	"io"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/generator"
	"github.com/gnailuy/sudoku/playrun"
	"github.com/gnailuy/sudoku/recovery"
	"github.com/gnailuy/sudoku/sessionfile"
	"github.com/gnailuy/sudoku/solver"
)

type sessionRequest struct {
	input  string
	level  string
	resume string
	dbPath string
	fromDB bool
}

func createSession(request sessionRequest, output, errorOutput io.Writer) (game.Game, string, error) {
	options := game.NewDefaultOptions(solverStore)
	options.StrategySolverKeys = solverStore.GetAllStrategySolverKeys()
	if request.resume != "" {
		data, err := sessionfile.Read(request.resume)
		if err != nil {
			return game.Game{}, "", fmt.Errorf("unable to read saved session: %w", err)
		}
		restored, err := game.Restore(data, options)
		if err != nil {
			return game.Game{}, "", fmt.Errorf("unable to resume saved session: %w", err)
		}
		return restored, request.resume, nil
	}

	var problem core.Board
	keys := solverStore.GetAllStrategySolverKeys()
	if request.input != "" {
		parsed, err := generator.GenerateSudokuProblemFromString(request.input)
		if err != nil {
			return game.Game{}, "", fmt.Errorf("the input is not a valid Sudoku problem: %s", request.input)
		}
		count := solverStore.GetDefaultSolver().CountSolutions(parsed)
		if count == 0 {
			return game.Game{}, "", fmt.Errorf("the input is not a solvable Sudoku problem: %s", request.input)
		}
		if count > 1 {
			fmt.Fprintf(errorOutput, "The input has %d solutions: %s\n", count, request.input)
		}
		problem = *parsed
		dbPath := request.dbPath
		if dbPath == "" {
			dbPath = defaultDBPath()
		}
		_, _ = autoStoreTo(solverStore, problem, "input", dbPath)
	} else {
		difficulty, err := difficultyForLevel(request.level)
		if err != nil {
			return game.Game{}, "", err
		}
		dbPath := request.dbPath
		if dbPath == "" {
			dbPath = defaultDBPath()
		}
		if request.fromDB {
			problem, keys, err = acquireFromDB(solverStore, difficulty, request.level, dbPath)
		} else if catalogFirstLevel(request.level) {
			fmt.Fprintf(output, "Selecting an exact %s puzzle from the catalog...\n", capitalize(request.level))
			problem, keys, err = acquireFromDB(solverStore, difficulty, request.level, dbPath)
			if err != nil {
				if request.level == "evil" {
					return game.Game{}, "", fmt.Errorf("Evil cohort selection failed: %w", err)
				}
				fmt.Fprintf(output, "Exact-grade catalog unavailable (%v). Falling back to bounded generation.\n", err)
				problem, keys, err = generateWithFallbackTo(output, solverStore, difficulty, request.level, dbPath)
			}
		} else {
			fmt.Fprintf(output, "Generating a random %s Sudoku problem...\n", capitalize(request.level))
			problem, keys, err = generateWithFallbackTo(output, solverStore, difficulty, request.level, dbPath)
		}
		if err != nil {
			return game.Game{}, "", err
		}
	}
	options.StrategySolverKeys = keys
	return game.NewGame(problem, options), "", nil
}

func catalogFirstLevel(level string) bool {
	switch level {
	case "hard", "expert", "evil":
		return true
	default:
		return false
	}
}

func acquireFromDB(store solver.Store, difficulty generator.Difficulty, levelName, dbPath string) (core.Board, []string, error) {
	puzzleDB, err := db.Open(dbPath)
	if err != nil {
		return core.Board{}, nil, fmt.Errorf("open puzzle database: %w", err)
	}
	defer puzzleDB.Close()
	record, err := puzzleDB.AcquireForPlay(levelName)
	if err != nil {
		return core.Board{}, nil, err
	}
	if record == nil {
		return core.Board{}, nil, fmt.Errorf("no %s puzzle is available in the database", levelName)
	}
	board := core.NewEmptyBoard()
	board.FromString(record.Puzzle)
	board.Randomize()
	keys := difficulty.AllowedSolverKeys()
	if len(keys) == 0 {
		keys = store.GetAllStrategySolverKeys()
	}
	return board, keys, nil
}

func difficultyForLevel(level string) (generator.Difficulty, error) {
	switch level {
	case "easy":
		return generator.NewEasyDifficulty(), nil
	case "medium":
		return generator.NewMediumDifficulty(), nil
	case "hard":
		return generator.NewHardDifficulty(), nil
	case "expert":
		return generator.NewExpertDifficulty(), nil
	case "evil":
		return generator.NewEvilDifficulty(), nil
	default:
		return generator.Difficulty{}, fmt.Errorf("invalid difficulty level: %s. Options: easy, medium, hard, expert, evil", level)
	}
}

type completionRecorder struct {
	path  string
	runID string
}

func (recorder completionRecorder) RecordCompletion(puzzle string) (bool, error) {
	puzzleDB, err := db.Open(recorder.path)
	if err != nil {
		return false, err
	}
	defer puzzleDB.Close()
	if recorder.runID != "" {
		return puzzleDB.CompletePlayRun(recorder.runID)
	}
	return puzzleDB.RecordCompletion(puzzle)
}

func newCompletionTracker(current game.Game, path string, existingRunID ...string) *playrun.Tracker {
	if path == "" {
		path = defaultDBPath()
	}
	key := normalizePuzzleForDB(solverStore, current.ProblemBoard())
	runID := ""
	if len(existingRunID) > 0 {
		runID = existingRunID[0]
	}
	if runID == "" {
		generated, err := recovery.NewID()
		if err == nil {
			puzzleDB, openErr := db.Open(path)
			if openErr == nil {
				presented := current.ProblemBoard()
				run := db.PlayRun{ID: generated, BasePuzzleID: db.BasePuzzleID(key), PresentedPuzzle: presented.ToString()}
				if insertErr := puzzleDB.InsertPlayRun(run); insertErr == nil {
					runID = generated
				}
				_ = puzzleDB.Close()
			}
		}
	}
	if runID == "" {
		return playrun.New(key, completionRecorder{path: path})
	}
	return playrun.NewLinked(key, runID, completionRecorder{path: path, runID: runID})
}

func createTrackedSession(request sessionRequest, output, errorOutput io.Writer) (game.Game, string, *playrun.Tracker, error) {
	current, resumePath, err := createSession(request, output, errorOutput)
	if err != nil {
		return game.Game{}, "", nil, err
	}
	return current, resumePath, newCompletionTracker(current, request.dbPath), nil
}
