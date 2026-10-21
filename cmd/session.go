package cmd

import (
	"fmt"
	"io"

	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/playrun"
	"github.com/gnailuy/sudoku/recovery"
	playsession "github.com/gnailuy/sudoku/session"
)

type sessionRequest struct {
	input  string
	level  string
	resume string
	dbPath string
	fromDB bool
}

func createSession(request sessionRequest, output, errorOutput io.Writer) (game.Game, string, error) {
	if request.input == "" && request.resume == "" && !request.fromDB {
		if request.level == "hard" || request.level == "expert" || request.level == "evil" {
			fmt.Fprintf(output, "Selecting an exact %s puzzle from the catalog...\n", capitalize(request.level))
		} else {
			fmt.Fprintf(output, "Generating a random %s Sudoku problem...\n", capitalize(request.level))
		}
	}
	service, err := playsession.New(playsession.Config{Solver: &solverStore, OpenCatalog: func(path string) (*db.DB, error) {
		catalog, _, err := openCatalogDatabase(path)
		return catalog, err
	}})
	if err != nil {
		return game.Game{}, "", err
	}
	result, err := service.Create(playsession.Request{
		Puzzle: request.input, Difficulty: request.level, ResumePath: request.resume,
		DatabasePath: request.dbPath, FromDatabase: request.fromDB,
	})
	if err != nil {
		return game.Game{}, "", err
	}
	for _, notice := range result.Notices {
		switch notice.Kind {
		case playsession.NoticeMultipleSolutions:
			fmt.Fprintf(errorOutput, "The input has more than one solution: %s\n", request.input)
		case playsession.NoticeCatalogFallback:
			fmt.Fprintf(output, "Exact-grade catalog unavailable (%v). Falling back to bounded generation.\n", notice.Cause)
		case playsession.NoticeDifficultyChanged:
			fmt.Fprintf(output, "Requested difficulty: %s. Generated puzzle difficulty: %s. Enjoy!\n", capitalize(notice.RequestedDifficulty), capitalize(notice.ActualDifficulty))
		case playsession.NoticePersistenceFailed:
			fmt.Fprintf(errorOutput, "Warning: unable to %s: %v\n", notice.Operation, notice.Cause)
		}
	}
	return result.Game, result.ResumePath, nil
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

func newCompletionTracker(current game.Game, path string, existingRunID ...string) (*playrun.Tracker, error) {
	key := normalizePuzzleForDB(solverStore, current.ProblemBoard())
	if path == "" {
		resolved, err := defaultDBPath()
		if err != nil {
			return playrun.New(key, completionRecorder{}), fmt.Errorf("resolve play-run database: %w", err)
		}
		path = resolved
	}
	runID := ""
	if len(existingRunID) > 0 {
		runID = existingRunID[0]
	}
	if runID == "" {
		generated, err := recovery.NewID()
		if err != nil {
			return playrun.New(key, completionRecorder{path: path}), fmt.Errorf("create play-run identity: %w", err)
		}
		puzzleDB, err := db.Open(path)
		if err != nil {
			return playrun.New(key, completionRecorder{path: path}), fmt.Errorf("open play-run database: %w", err)
		}
		presented := current.ProblemBoard()
		run := db.PlayRun{ID: generated, BasePuzzleID: db.BasePuzzleID(key), PresentedPuzzle: presented.ToString()}
		if err := puzzleDB.InsertPlayRun(run); err != nil {
			_ = puzzleDB.Close()
			return playrun.New(key, completionRecorder{path: path}), fmt.Errorf("create play run: %w", err)
		}
		runID = generated
		if err := puzzleDB.Close(); err != nil {
			return playrun.NewLinked(key, runID, completionRecorder{path: path, runID: runID}), fmt.Errorf("close play-run database: %w", err)
		}
	}
	if runID == "" {
		return playrun.New(key, completionRecorder{path: path}), nil
	}
	return playrun.NewLinked(key, runID, completionRecorder{path: path, runID: runID}), nil
}

func createTrackedSession(request sessionRequest, output, errorOutput io.Writer) (game.Game, string, *playrun.Tracker, error) {
	if request.dbPath == "" {
		catalog, resolvedDBPath, err := openCatalogDatabase("")
		if err != nil {
			return game.Game{}, "", nil, fmt.Errorf("open puzzle database: %w", err)
		}
		_ = catalog.Close()
		request.dbPath = resolvedDBPath
	}
	current, resumePath, err := createSession(request, output, errorOutput)
	if err != nil {
		return game.Game{}, "", nil, err
	}
	tracker, trackingErr := newCompletionTracker(current, request.dbPath)
	if trackingErr != nil {
		fmt.Fprintf(errorOutput, "Warning: completion tracking setup failed: %v\n", trackingErr)
	}
	return current, resumePath, tracker, nil
}
