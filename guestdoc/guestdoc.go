// Package guestdoc seals stateless guest game documents for browser storage.
package guestdoc

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/sessionfile"
)

const (
	FormatVersion = 1
	maxTokenSize  = 2 * sessionfile.MaxSize
)

// ErrorCode identifies a guest-document failure without exposing sealed data.
type ErrorCode string

const (
	ErrorMalformed          ErrorCode = "malformed-guest-document"
	ErrorUnsupportedVersion ErrorCode = "unsupported-guest-document"
	ErrorUnknownKey         ErrorCode = "unknown-guest-key"
	ErrorAuthentication     ErrorCode = "invalid-guest-document"
	ErrorExpired            ErrorCode = "expired-guest-document"
	ErrorInvalidPayload     ErrorCode = "invalid-guest-payload"
	ErrorRevisionConflict   ErrorCode = "guest-revision-conflict"
)

// Document is the authenticated plaintext carried by one opaque guest token.
type Document struct {
	Version          int             `json:"version"`
	ID               string          `json:"id"`
	BasePuzzleID     string          `json:"base_puzzle_id"`
	PresentedPuzzle  string          `json:"presented_puzzle"`
	EngineState      json.RawMessage `json:"engine_state"`
	Revision         int64           `json:"revision"`
	ActualDifficulty string          `json:"actual_difficulty"`
	Status           string          `json:"status,omitempty"`
	IssuedAt         time.Time       `json:"issued_at"`
	ExpiresAt        time.Time       `json:"expires_at"`
}

// Error reports a stable guest-document failure code.
type Error struct {
	Code ErrorCode
	Err  error
}

func (err *Error) Error() string { return string(err.Code) }
func (err *Error) Unwrap() error { return err.Err }
func (err *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && err.Code == other.Code
}

// Config defines the active sealing key and bounded decrypt-only rotation keys.
type Config struct {
	ActiveKeyID string
	Keys        map[string][]byte
	Lifetime    time.Duration
	Random      io.Reader
	Now         func() time.Time
}

// Sealer owns authenticated encryption and guest-game transitions.
type Sealer struct {
	activeKeyID string
	keys        map[string][]byte
	lifetime    time.Duration
	random      io.Reader
	now         func() time.Time
}

type envelope struct {
	Version    int    `json:"version"`
	KeyID      string `json:"key_id"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

// Transition is the complete authoritative result of one accepted guest action.
type Transition struct {
	Token    string
	Document Document
	Snapshot game.Snapshot
	Result   game.Result
}

// New validates key material without retaining caller-owned byte slices.
func New(config Config) (*Sealer, error) {
	if !validKeyID(config.ActiveKeyID) {
		return nil, errors.New("active guest key identifier is invalid")
	}
	if config.Lifetime <= 0 {
		return nil, errors.New("guest document lifetime must be positive")
	}
	keys := make(map[string][]byte, len(config.Keys))
	for keyID, key := range config.Keys {
		if !validKeyID(keyID) || len(key) != 32 {
			return nil, errors.New("guest keys require bounded identifiers and 32-byte AES-256 material")
		}
		keys[keyID] = bytes.Clone(key)
	}
	if _, ok := keys[config.ActiveKeyID]; !ok {
		return nil, errors.New("active guest key is missing")
	}
	random := config.Random
	if random == nil {
		random = rand.Reader
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return &Sealer{activeKeyID: config.ActiveKeyID, keys: keys, lifetime: config.Lifetime, random: random, now: now}, nil
}

// Create serializes one game into a fresh revision-zero guest document.
func (sealer *Sealer) Create(current *game.Game, basePuzzleID, actualDifficulty string) (string, Document, error) {
	if current == nil {
		return "", Document{}, payloadError(errors.New("game is required"))
	}
	state, err := current.Serialize()
	if err != nil {
		return "", Document{}, payloadError(err)
	}
	idBytes := make([]byte, 16)
	if _, err := io.ReadFull(sealer.random, idBytes); err != nil {
		return "", Document{}, fmt.Errorf("create guest document identifier: %w", err)
	}
	now := sealer.now().UTC()
	presented := current.ProblemBoard()
	document := Document{
		Version:          FormatVersion,
		ID:               hex.EncodeToString(idBytes),
		BasePuzzleID:     basePuzzleID,
		PresentedPuzzle:  presented.ToString(),
		EngineState:      state,
		Revision:         0,
		ActualDifficulty: actualDifficulty,
		Status:           string(current.Snapshot().Status),
		IssuedAt:         now,
		ExpiresAt:        now.Add(sealer.lifetime),
	}
	token, err := sealer.Seal(document)
	return token, document, err
}

// Seal authenticates and encrypts a validated guest document with the active key.
func (sealer *Sealer) Seal(document Document) (string, error) {
	if err := validateDocument(document, sealer.now().UTC(), sealer.lifetime, false); err != nil {
		return "", err
	}
	plaintext, err := json.Marshal(document)
	if err != nil {
		return "", payloadError(err)
	}
	if int64(len(plaintext)) > sessionfile.MaxSize {
		return "", payloadError(sessionfile.ErrTooLarge)
	}
	block, err := aes.NewCipher(sealer.keys[sealer.activeKeyID])
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(sealer.random, nonce); err != nil {
		return "", fmt.Errorf("create guest document nonce: %w", err)
	}
	sealed := aead.Seal(nil, nonce, plaintext, additionalData(FormatVersion, sealer.activeKeyID))
	encoded, err := json.Marshal(envelope{Version: FormatVersion, KeyID: sealer.activeKeyID, Nonce: nonce, Ciphertext: sealed})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

// Open authenticates, decrypts, and validates one opaque guest token.
func (sealer *Sealer) Open(token string) (Document, error) {
	if token == "" || int64(len(token)) > maxTokenSize {
		return Document{}, malformedError(nil)
	}
	encoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return Document{}, malformedError(err)
	}
	var wrapped envelope
	if err := decodeStrict(encoded, &wrapped); err != nil {
		return Document{}, malformedError(err)
	}
	if wrapped.Version != FormatVersion {
		return Document{}, &Error{Code: ErrorUnsupportedVersion}
	}
	key, ok := sealer.keys[wrapped.KeyID]
	if !ok {
		return Document{}, &Error{Code: ErrorUnknownKey}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return Document{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return Document{}, err
	}
	if len(wrapped.Nonce) != aead.NonceSize() || len(wrapped.Ciphertext) < aead.Overhead() {
		return Document{}, malformedError(nil)
	}
	plaintext, err := aead.Open(nil, wrapped.Nonce, wrapped.Ciphertext, additionalData(wrapped.Version, wrapped.KeyID))
	if err != nil {
		return Document{}, &Error{Code: ErrorAuthentication, Err: err}
	}
	var document Document
	if err := decodeStrict(plaintext, &document); err != nil {
		return Document{}, payloadError(err)
	}
	if err := validateDocument(document, sealer.now().UTC(), sealer.lifetime, true); err != nil {
		return Document{}, err
	}
	return document, nil
}

// Apply restores a guest game, applies one revision-checked action, and returns
// a replacement token encrypted with the active key.
func (sealer *Sealer) Apply(token string, expectedRevision int64, action game.Action, options game.Options) (Transition, error) {
	document, err := sealer.Open(token)
	if err != nil {
		return Transition{}, err
	}
	if expectedRevision != document.Revision {
		return Transition{}, &Error{Code: ErrorRevisionConflict}
	}
	current, err := game.Restore(document.EngineState, options)
	if err != nil {
		return Transition{}, payloadError(err)
	}
	result, err := current.Apply(action)
	if err != nil {
		return Transition{}, err
	}
	state, err := current.Serialize()
	if err != nil {
		return Transition{}, payloadError(err)
	}
	document.EngineState = state
	document.Status = string(current.Snapshot().Status)
	document.Revision++
	replacement, err := sealer.Seal(document)
	if err != nil {
		return Transition{}, err
	}
	return Transition{Token: replacement, Document: document, Snapshot: current.Snapshot(), Result: result}, nil
}

func validateDocument(document Document, now time.Time, maxLifetime time.Duration, enforceExpiry bool) error {
	if document.Version != FormatVersion {
		return &Error{Code: ErrorUnsupportedVersion}
	}
	lifetime := document.ExpiresAt.Sub(document.IssuedAt)
	if !validHex(document.ID, 16) || !validBasePuzzleID(document.BasePuzzleID) || !core.IsValidSudokuString(document.PresentedPuzzle) || document.Revision < 0 || !validDifficulty(document.ActualDifficulty) || document.Status != "" && !validGameStatus(document.Status) || document.IssuedAt.IsZero() || lifetime <= 0 || lifetime > maxLifetime || document.IssuedAt.After(now.Add(5*time.Minute)) || int64(len(document.EngineState)) > sessionfile.MaxSize {
		return payloadError(nil)
	}
	if enforceExpiry && !now.Before(document.ExpiresAt) {
		return &Error{Code: ErrorExpired}
	}
	var state struct {
		Version int    `json:"version"`
		Puzzle  string `json:"puzzle"`
	}
	if err := json.Unmarshal(document.EngineState, &state); err != nil || state.Version != game.StateVersion || state.Puzzle != document.PresentedPuzzle {
		return payloadError(err)
	}
	return nil
}

func validKeyID(value string) bool {
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	for _, character := range value {
		if !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '-' || character == '_') {
			return false
		}
	}
	return true
}

func validHex(value string, byteCount int) bool {
	if len(value) != byteCount*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == byteCount
}

func validBasePuzzleID(value string) bool {
	return strings.HasPrefix(value, "bp_") && validHex(strings.TrimPrefix(value, "bp_"), 32)
}

func validGameStatus(value string) bool {
	return value == string(game.StatusInProgress) || value == string(game.StatusInvalid) || value == string(game.StatusSolved)
}

func validDifficulty(value string) bool {
	switch strings.ToLower(value) {
	case "easy", "medium", "hard", "expert", "evil":
		return value == strings.ToLower(value)
	default:
		return false
	}
}

func additionalData(version int, keyID string) []byte {
	return []byte(fmt.Sprintf("sudoku-guest-document:%d:%s", version, keyID))
}

func decodeStrict(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func malformedError(err error) *Error { return &Error{Code: ErrorMalformed, Err: err} }
func payloadError(err error) *Error   { return &Error{Code: ErrorInvalidPayload, Err: err} }
