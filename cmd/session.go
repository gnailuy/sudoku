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

func newCompletionTracker(current game.Game, path string, existingRunID ...string) *playrun.Tracker {
	if path == "" {
		path, _ = defaultDBPath()
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
	return current, resumePath, newCompletionTracker(current, request.dbPath), nil
}
