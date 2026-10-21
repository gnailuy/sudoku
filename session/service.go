// Package session creates playable games from saved sessions, puzzle text, the
// catalog, or bounded generation without coupling the work to a frontend.
package session

import (
	"errors"
	"fmt"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/generator"
	"github.com/gnailuy/sudoku/sessionfile"
	"github.com/gnailuy/sudoku/solver"
)

var ErrInvalidConfiguration = errors.New("invalid session service configuration")

var errCatalogExcluded = errors.New("puzzle is excluded from the reusable catalog")

// Source identifies the successful puzzle/session source without prescribing
// frontend wording.
type Source string

const (
	SourceSavedSession      Source = "saved-session"
	SourcePuzzleInput       Source = "puzzle-input"
	SourceCatalog           Source = "catalog"
	SourceGenerated         Source = "generated"
	SourceGeneratedFallback Source = "generated-fallback"
)

// NoticeKind describes non-fatal creation facts a frontend may present.
type NoticeKind string

const (
	NoticeMultipleSolutions NoticeKind = "multiple-solutions"
	NoticeCatalogFallback   NoticeKind = "catalog-fallback"
	NoticeDifficultyChanged NoticeKind = "difficulty-changed"
	NoticePersistenceFailed NoticeKind = "persistence-failed"
)

// Notice is structured presentation input rather than rendered text.
type Notice struct {
	Kind                NoticeKind
	RequestedDifficulty string
	ActualDifficulty    string
	Operation           string
	Cause               error
}

// Request selects exactly one creation path. OpenCatalog interprets
// DatabasePath, allowing the composition root to own managed-path policy.
type Request struct {
	Puzzle       string
	Difficulty   string
	ResumePath   string
	DatabasePath string
	FromDatabase bool
}

// Result contains the created game and structured facts needed by clients.
type Result struct {
	Game                game.Game
	ResumePath          string
	Source              Source
	RequestedDifficulty string
	ActualDifficulty    string
	Notices             []Notice
}

// Config binds creation to the solver registry used by the application.
type Config struct {
	Solver      *solver.Store
	OpenCatalog func(string) (*db.DB, error)
}

// Service owns frontend-neutral puzzle acquisition and game construction.
type Service struct {
	solver      solver.Store
	openCatalog func(string) (*db.DB, error)
}

func New(config Config) (*Service, error) {
	if config.Solver == nil || config.OpenCatalog == nil {
		return nil, ErrInvalidConfiguration
	}
	return &Service{solver: *config.Solver, openCatalog: config.OpenCatalog}, nil
}

// Create restores or acquires one playable game.
func (service *Service) Create(request Request) (Result, error) {
	options := game.NewDefaultOptions(service.solver)
	options.StrategySolverKeys = service.solver.GetAllStrategySolverKeys()
	if request.ResumePath != "" {
		data, err := sessionfile.Read(request.ResumePath)
		if err != nil {
			return Result{}, fmt.Errorf("unable to read saved session: %w", err)
		}
		restored, err := game.Restore(data, options)
		if err != nil {
			return Result{}, fmt.Errorf("unable to resume saved session: %w", err)
		}
		return Result{Game: restored, ResumePath: request.ResumePath, Source: SourceSavedSession}, nil
	}
	var parsedPuzzle core.Board
	if request.Puzzle != "" {
		var err error
		parsedPuzzle, err = core.ParseBoard(request.Puzzle)
		if err != nil {
			return Result{}, fmt.Errorf("the input is not a valid Sudoku problem: %s", request.Puzzle)
		}
	} else if _, err := generator.DifficultyForName(request.Difficulty); err != nil {
		return Result{}, err
	}
	catalog, catalogErr := service.openCatalog(request.DatabasePath)
	if catalog != nil {
		defer catalog.Close()
	}
	if request.Puzzle != "" {
		return service.createFromPuzzle(request, parsedPuzzle, options, catalog)
	}
	if catalogErr != nil && request.FromDatabase {
		return Result{}, fmt.Errorf("open puzzle database: %w", catalogErr)
	}
	if catalogErr != nil && request.Difficulty == "evil" {
		return Result{}, fmt.Errorf("Evil cohort selection failed: open puzzle database: %w", catalogErr)
	}
	return service.createFromDifficulty(request, options, catalog, catalogErr)
}

func (service *Service) createFromPuzzle(request Request, problem core.Board, options game.Options, catalog *db.DB) (Result, error) {
	count := service.solver.GetDefaultSolver().CountSolutionsUpTo(&problem, 2)
	if count == 0 {
		return Result{}, fmt.Errorf("the input is not a solvable Sudoku problem: %s", request.Puzzle)
	}
	result := Result{Source: SourcePuzzleInput}
	if count > 1 {
		result.Notices = append(result.Notices, Notice{Kind: NoticeMultipleSolutions})
	}
	if _, err := service.store(catalog, problem, "input"); err != nil && !errors.Is(err, errCatalogExcluded) {
		result.Notices = append(result.Notices, Notice{Kind: NoticePersistenceFailed, Operation: "store the reusable puzzle", Cause: err})
	}
	result.Game = game.NewGame(problem, options)
	return result, nil
}

func (service *Service) createFromDifficulty(request Request, options game.Options, catalog *db.DB, catalogErr error) (Result, error) {
	difficulty, err := generator.DifficultyForName(request.Difficulty)
	if err != nil {
		return Result{}, err
	}
	result := Result{RequestedDifficulty: request.Difficulty}
	var problem core.Board
	var keys []string
	if request.FromDatabase {
		problem, keys, err = service.acquire(catalog, catalogErr, difficulty, request.Difficulty)
		result.Source = SourceCatalog
	} else if catalogFirst(request.Difficulty) {
		problem, keys, err = service.acquire(catalog, catalogErr, difficulty, request.Difficulty)
		result.Source = SourceCatalog
		if err != nil {
			if request.Difficulty == "evil" {
				return Result{}, fmt.Errorf("Evil cohort selection failed: %w", err)
			}
			result.Notices = append(result.Notices, Notice{Kind: NoticeCatalogFallback, RequestedDifficulty: request.Difficulty, Cause: err})
			var notices []Notice
			problem, keys, result.Source, result.ActualDifficulty, notices, err = service.generateWithFallback(catalog, catalogErr, difficulty, request.Difficulty)
			result.Notices = append(result.Notices, notices...)
		}
	} else {
		var notices []Notice
		problem, keys, result.Source, result.ActualDifficulty, notices, err = service.generateWithFallback(catalog, catalogErr, difficulty, request.Difficulty)
		result.Notices = append(result.Notices, notices...)
	}
	if err != nil {
		return Result{}, err
	}
	if result.ActualDifficulty != "" && result.ActualDifficulty != request.Difficulty {
		result.Notices = append(result.Notices, Notice{Kind: NoticeDifficultyChanged, RequestedDifficulty: request.Difficulty, ActualDifficulty: result.ActualDifficulty})
	}
	options.StrategySolverKeys = keys
	result.Game = game.NewGame(problem, options)
	return result, nil
}

func catalogFirst(level string) bool {
	return level == "hard" || level == "expert" || level == "evil"
}

func (service *Service) acquire(catalog *db.DB, catalogErr error, difficulty generator.Difficulty, level string) (core.Board, []string, error) {
	if catalog == nil {
		if catalogErr != nil {
			return core.Board{}, nil, fmt.Errorf("open puzzle database: %w", catalogErr)
		}
		return core.Board{}, nil, errors.New("puzzle database is unavailable")
	}
	record, err := catalog.AcquireForPlay(level)
	if err != nil {
		return core.Board{}, nil, err
	}
	if record == nil {
		return core.Board{}, nil, fmt.Errorf("no %s puzzle is available in the database", level)
	}
	board := core.NewEmptyBoard()
	board.FromString(record.Puzzle)
	board.Randomize()
	return board, service.allowedKeys(difficulty), nil
}

func (service *Service) generateWithFallback(catalog *db.DB, catalogErr error, difficulty generator.Difficulty, level string) (core.Board, []string, Source, string, []Notice, error) {
	options := generator.NewBestEffortOptions(service.solver, difficulty)
	generated := generator.GenerateBestEffort(options)
	storedPuzzle := ""
	var notices []Notice
	if generated.RoundsUsed > 0 {
		var storeErr error
		storedPuzzle, storeErr = service.store(catalog, generated.Puzzle, "generated")
		if storeErr != nil && !errors.Is(storeErr, errCatalogExcluded) {
			notices = append(notices, Notice{Kind: NoticePersistenceFailed, Operation: "store the generated puzzle", Cause: storeErr})
		}
	}
	if generated.Matched {
		if err := markPlayed(catalog, storedPuzzle); err != nil {
			notices = append(notices, Notice{Kind: NoticePersistenceFailed, Operation: "record puzzle acquisition", Cause: err})
		}
		return generated.Puzzle, service.allowedKeys(difficulty), SourceGenerated, generated.Classification.Difficulty, notices, nil
	}
	if board, keys, err := service.acquire(catalog, catalogErr, difficulty, level); err == nil {
		return board, keys, SourceCatalog, level, notices, nil
	}
	if generated.RoundsUsed == 0 {
		return core.Board{}, nil, "", "", notices, fmt.Errorf("generation timed out before a puzzle completed and no %s puzzle is available in the database", level)
	}
	if err := markPlayed(catalog, storedPuzzle); err != nil {
		notices = append(notices, Notice{Kind: NoticePersistenceFailed, Operation: "record puzzle acquisition", Cause: err})
	}
	return generated.Puzzle, service.allowedKeys(difficulty), SourceGeneratedFallback, generated.Classification.Difficulty, notices, nil
}

func (service *Service) allowedKeys(difficulty generator.Difficulty) []string {
	keys := difficulty.AllowedSolverKeys()
	if len(keys) == 0 {
		return service.solver.GetAllStrategySolverKeys()
	}
	return keys
}

func (service *Service) store(catalog *db.DB, board core.Board, source string) (string, error) {
	if catalog == nil {
		return "", errors.New("puzzle database is unavailable")
	}
	puzzle := core.CanonicalPuzzle(board.ToString())
	normalized, err := core.ParseBoard(puzzle)
	if err != nil {
		return "", err
	}
	classification := solver.ClassifyPuzzle(service.solver, normalized)
	if classification.Outcome != solver.ClassificationSolved {
		return "", errCatalogExcluded
	}
	_, err = catalog.InsertPuzzle(db.Puzzle{Puzzle: puzzle, Difficulty: classification.Difficulty, Score: classification.Score, MaxTechnique: classification.MaxTechnique, Source: source})
	return puzzle, err
}

func markPlayed(catalog *db.DB, puzzle string) error {
	if catalog == nil || puzzle == "" {
		return nil
	}
	record, err := catalog.MarkForPlay(puzzle)
	if err != nil {
		return err
	}
	if record == nil {
		return errors.New("stored puzzle is absent")
	}
	return nil
}
