// Package accountgame applies authenticated, owner-scoped operations to
// durable account games without exposing user identifiers as client input.
package accountgame

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/game"
)

var (
	ErrInvalidConfiguration          = errors.New("invalid account game configuration")
	ErrAuthenticatedIdentityRequired = errors.New("authenticated user identity is required")
	ErrNotFound                      = errors.New("account game not found")
	ErrRevisionConflict              = errors.New("account game revision conflict")
)

type store interface {
	CreateAccountGame(db.AccountGame) error
	AccountGameByID(string, string) (*db.AccountGame, error)
	AccountGamesByUser(string, int) ([]db.AccountGame, error)
	UpdateAccountGame(string, string, int64, []byte, string) (bool, error)
	UpdateAccountGameElapsed(string, string, int64) (bool, error)
	DeleteAccountGame(string, string) (bool, error)
	DeleteAllAccountGames(string) (int64, error)
}

// Config binds owner-scoped persistence to the engine restoration contract.
type Config struct {
	Store   store
	Options game.Options
	Random  io.Reader
}

// Service authorizes every operation with the authenticated user's identity.
type Service struct {
	store   store
	options game.Options
	random  io.Reader
}

// State is one authorized durable game and its detached engine snapshot.
type State struct {
	Game     db.AccountGame
	Snapshot game.Snapshot
}

// Transition is one successfully persisted engine action.
type Transition struct {
	State
	Result game.Result
}

// RevisionConflictError reports the durable revision a client must reload.
type RevisionConflictError struct {
	CurrentRevision int64
}

func (err *RevisionConflictError) Error() string {
	return fmt.Sprintf("%s: current revision is %d", ErrRevisionConflict, err.CurrentRevision)
}

func (err *RevisionConflictError) Unwrap() error { return ErrRevisionConflict }

func New(config Config) (*Service, error) {
	if config.Store == nil {
		return nil, ErrInvalidConfiguration
	}
	random := config.Random
	if random == nil {
		random = rand.Reader
	}
	return &Service{store: config.Store, options: config.Options, random: random}, nil
}

// Create persists one already-linked play run as an owner-scoped game.
func (service *Service) Create(userID string, current game.Game, basePuzzleID, playRunID, actualDifficulty string) (State, error) {
	if userID == "" {
		return State{}, ErrAuthenticatedIdentityRequired
	}
	if basePuzzleID == "" || playRunID == "" || actualDifficulty == "" {
		return State{}, ErrInvalidConfiguration
	}
	identifier := make([]byte, 16)
	if _, err := io.ReadFull(service.random, identifier); err != nil {
		return State{}, fmt.Errorf("create account game identity: %w", err)
	}
	serialized, err := current.Serialize()
	if err != nil {
		return State{}, fmt.Errorf("serialize account game: %w", err)
	}
	record := db.AccountGame{
		ID: "ag_" + hex.EncodeToString(identifier), UserID: userID,
		BasePuzzleID: basePuzzleID, PlayRunID: playRunID, EngineState: serialized,
		ActualDifficulty: actualDifficulty, Status: string(current.Snapshot().Status),
	}
	if err := service.store.CreateAccountGame(record); err != nil {
		return State{}, err
	}
	return State{Game: record, Snapshot: current.Snapshot()}, nil
}

// List returns bounded summaries for only the authenticated owner.
func (service *Service) List(userID string, limit int) ([]db.AccountGame, error) {
	if userID == "" {
		return nil, ErrAuthenticatedIdentityRequired
	}
	return service.store.AccountGamesByUser(userID, limit)
}

// Get restores one owner-scoped game. Another owner's identifier is
// indistinguishable from an absent identifier.
func (service *Service) Get(userID, accountGameID string) (State, error) {
	if userID == "" {
		return State{}, ErrAuthenticatedIdentityRequired
	}
	if accountGameID == "" {
		return State{}, ErrNotFound
	}
	record, err := service.store.AccountGameByID(userID, accountGameID)
	if err != nil {
		return State{}, err
	}
	if record == nil {
		return State{}, ErrNotFound
	}
	current, err := game.Restore(record.EngineState, service.options)
	if err != nil {
		return State{}, fmt.Errorf("restore account game: %w", err)
	}
	return State{Game: *record, Snapshot: current.Snapshot()}, nil
}

// Apply restores, mutates, serializes, and atomically persists one engine
// action under the owner's current optimistic revision.
func (service *Service) Apply(userID, accountGameID string, expectedRevision int64, action game.Action) (Transition, error) {
	state, err := service.Get(userID, accountGameID)
	if err != nil {
		return Transition{}, err
	}
	if expectedRevision != state.Game.Revision {
		return Transition{}, &RevisionConflictError{CurrentRevision: state.Game.Revision}
	}
	current, err := game.Restore(state.Game.EngineState, service.options)
	if err != nil {
		return Transition{}, fmt.Errorf("restore account game: %w", err)
	}
	result, err := current.Apply(action)
	if err != nil {
		return Transition{}, err
	}
	serialized, err := current.Serialize()
	if err != nil {
		return Transition{}, fmt.Errorf("serialize account game: %w", err)
	}
	updated, err := service.store.UpdateAccountGame(userID, accountGameID, expectedRevision, serialized, string(current.Snapshot().Status))
	if err != nil {
		return Transition{}, err
	}
	if !updated {
		latest, lookupErr := service.store.AccountGameByID(userID, accountGameID)
		if lookupErr != nil {
			return Transition{}, lookupErr
		}
		if latest == nil {
			return Transition{}, ErrNotFound
		}
		return Transition{}, &RevisionConflictError{CurrentRevision: latest.Revision}
	}
	state.Game.EngineState = serialized
	state.Game.Revision++
	state.Game.Status = string(current.Snapshot().Status)
	state.Snapshot = current.Snapshot()
	return Transition{State: state, Result: result}, nil
}

// UpdateElapsed persists presentation time independently from gameplay revisions.
func (service *Service) UpdateElapsed(userID, accountGameID string, elapsedSeconds int64) (State, error) {
	if userID == "" {
		return State{}, ErrAuthenticatedIdentityRequired
	}
	if accountGameID == "" || elapsedSeconds < 0 {
		return State{}, ErrNotFound
	}
	updated, err := service.store.UpdateAccountGameElapsed(userID, accountGameID, elapsedSeconds)
	if err != nil {
		return State{}, err
	}
	if !updated {
		return State{}, ErrNotFound
	}
	return service.Get(userID, accountGameID)
}

// Delete removes one owner-scoped game. Another owner's identifier is
// indistinguishable from an absent identifier.
func (service *Service) Delete(userID, accountGameID string) (bool, error) {
	if userID == "" {
		return false, ErrAuthenticatedIdentityRequired
	}
	if accountGameID == "" {
		return false, ErrNotFound
	}
	deleted, err := service.store.DeleteAccountGame(userID, accountGameID)
	if err != nil {
		return false, err
	}
	if !deleted {
		return false, ErrNotFound
	}
	return true, nil
}

// DeleteAll removes every game owned by one authenticated account.
func (service *Service) DeleteAll(userID string) (int64, error) {
	if userID == "" {
		return 0, ErrAuthenticatedIdentityRequired
	}
	return service.store.DeleteAllAccountGames(userID)
}
