// Package guestclaim converts authenticated sealed guest games into durable,
// owner-scoped account games with exactly-once persistence.
package guestclaim

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/guestdoc"
)

const identifierBytes = 16

var ErrInvalidConfiguration = errors.New("invalid guest claim configuration")

type documentOpener interface {
	Open(string) (guestdoc.Document, error)
}

type claimStore interface {
	ClaimGuestGame(db.GuestClaim) (db.AccountGame, bool, error)
}

type Config struct {
	Documents documentOpener
	Store     claimStore
	Random    io.Reader
}

type Service struct {
	documents documentOpener
	store     claimStore
	random    io.Reader
}

type Result struct {
	Game    db.AccountGame
	Created bool
}

func New(config Config) (*Service, error) {
	if config.Documents == nil || config.Store == nil {
		return nil, ErrInvalidConfiguration
	}
	random := config.Random
	if random == nil {
		random = rand.Reader
	}
	return &Service{documents: config.Documents, store: config.Store, random: random}, nil
}

// Claim authenticates the sealed document before deriving its stable claim
// fingerprint and committing one owner-scoped account game.
func (service *Service) Claim(userID, token string) (Result, error) {
	if userID == "" {
		return Result{}, errors.New("authenticated user identity is required")
	}
	document, err := service.documents.Open(token)
	if err != nil {
		return Result{}, err
	}
	accountGameID, err := service.randomID("ag_")
	if err != nil {
		return Result{}, fmt.Errorf("create account game identity: %w", err)
	}
	playRunID, err := service.randomID("pr_")
	if err != nil {
		return Result{}, fmt.Errorf("create play run identity: %w", err)
	}
	fingerprint := sha256.Sum256([]byte("sudoku-guest-claim:v1:" + document.ID))
	game, created, err := service.store.ClaimGuestGame(db.GuestClaim{
		Fingerprint: fingerprint[:],
		AccountGame: db.AccountGame{
			ID: accountGameID, UserID: userID, BasePuzzleID: document.BasePuzzleID,
			PlayRunID: playRunID, EngineState: document.EngineState,
			Revision: document.Revision, ActualDifficulty: document.ActualDifficulty,
		},
		PresentedPuzzle: document.PresentedPuzzle,
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Game: game, Created: created}, nil
}

func (service *Service) randomID(prefix string) (string, error) {
	value := make([]byte, identifierBytes)
	if _, err := io.ReadFull(service.random, value); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(value), nil
}
