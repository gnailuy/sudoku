package db

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrIdentityAlreadyLinked = errors.New("external identity is already linked to another user")

// User is one application account. Email and display name are mutable profile
// attributes and are never used as identity keys.
type User struct {
	ID           string
	ProfileEmail string
	DisplayName  string
}

// ExternalIdentity binds one provider issuer/subject pair to an application user.
type ExternalIdentity struct {
	Issuer       string
	Subject      string
	UserID       string
	ProfileEmail string
	DisplayName  string
}

// WebSession is the durable server-side half of an opaque browser session.
// VerifierDigest stores only the digest of the browser's random verifier.
type WebSession struct {
	ID                string
	UserID            string
	VerifierDigest    []byte
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	RotatedFromID     string
	RevokedAt         *time.Time
}

// WebSessionRotation contains the durable inputs for one completed external
// login. CandidateUserID is used only when the provider identity is new.
type WebSessionRotation struct {
	Identity               ExternalIdentity
	CandidateUserID        string
	Session                WebSession
	PreviousVerifierDigest []byte
	Now                    time.Time
}

// AccountGame is durable game state authorized through its owning user.
type AccountGame struct {
	ID               string
	UserID           string
	BasePuzzleID     string
	PlayRunID        string
	EngineState      []byte
	Revision         int64
	ActualDifficulty string
	UpdatedAt        time.Time
}

func (db *DB) CreateUser(user User) error {
	if user.ID == "" {
		return errors.New("user identity is required")
	}
	_, err := db.conn.Exec(`INSERT INTO users (user_id, profile_email, display_name) VALUES (?, ?, ?)`, user.ID, user.ProfileEmail, user.DisplayName)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// LinkExternalIdentity never uses profile email for account matching. A
// provider identity already linked to the same user is updated idempotently;
// one linked to another user is rejected.
func (db *DB) LinkExternalIdentity(identity ExternalIdentity) error {
	if identity.Issuer == "" || identity.Subject == "" || identity.UserID == "" {
		return errors.New("issuer, subject, and user identity are required")
	}
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin identity link: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var existingUserID string
	err = tx.QueryRow(`SELECT user_id FROM external_identities WHERE issuer = ? AND subject = ?`, identity.Issuer, identity.Subject).Scan(&existingUserID)
	switch {
	case err == nil && existingUserID != identity.UserID:
		return ErrIdentityAlreadyLinked
	case err == nil:
		_, err = tx.Exec(`UPDATE external_identities SET profile_email = ?, display_name = ?, updated_at = CURRENT_TIMESTAMP WHERE issuer = ? AND subject = ?`, identity.ProfileEmail, identity.DisplayName, identity.Issuer, identity.Subject)
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.Exec(`INSERT INTO external_identities (issuer, subject, user_id, profile_email, display_name) VALUES (?, ?, ?, ?, ?)`, identity.Issuer, identity.Subject, identity.UserID, identity.ProfileEmail, identity.DisplayName)
	}
	if err != nil {
		return fmt.Errorf("link external identity: %w", err)
	}
	if _, err = tx.Exec(`UPDATE users SET profile_email = ?, display_name = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`, identity.ProfileEmail, identity.DisplayName, identity.UserID); err != nil {
		return fmt.Errorf("refresh user profile: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit identity link: %w", err)
	}
	return nil
}

func (db *DB) UserByExternalIdentity(issuer, subject string) (*User, error) {
	var user User
	err := db.conn.QueryRow(`SELECT u.user_id, u.profile_email, u.display_name
		FROM external_identities i JOIN users u ON u.user_id = i.user_id
		WHERE i.issuer = ? AND i.subject = ?`, issuer, subject).Scan(&user.ID, &user.ProfileEmail, &user.DisplayName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by external identity: %w", err)
	}
	return &user, nil
}

func (db *DB) CreateWebSession(session WebSession) error {
	if session.ID == "" || session.UserID == "" || len(session.VerifierDigest) == 0 {
		return errors.New("session identity, user, and verifier digest are required")
	}
	if !session.IdleExpiresAt.Before(session.AbsoluteExpiresAt) {
		return errors.New("idle expiry must precede absolute expiry")
	}
	var rotatedFrom any
	if session.RotatedFromID != "" {
		rotatedFrom = session.RotatedFromID
	}
	_, err := db.conn.Exec(`INSERT INTO web_sessions
		(web_session_id, user_id, verifier_digest, idle_expires_at, absolute_expires_at, rotated_from_id)
		VALUES (?, ?, ?, ?, ?, ?)`, session.ID, session.UserID, session.VerifierDigest, session.IdleExpiresAt.UTC(), session.AbsoluteExpiresAt.UTC(), rotatedFrom)
	if err != nil {
		return fmt.Errorf("create web session: %w", err)
	}
	return nil
}

// ActiveWebSession resolves only an unrevoked, unexpired verifier digest.
func (db *DB) ActiveWebSession(verifierDigest []byte, now time.Time) (*WebSession, error) {
	var session WebSession
	var rotatedFrom sql.NullString
	err := db.conn.QueryRow(`SELECT web_session_id, user_id, verifier_digest, idle_expires_at, absolute_expires_at, rotated_from_id
		FROM web_sessions WHERE verifier_digest = ? AND revoked_at IS NULL
		AND idle_expires_at > ? AND absolute_expires_at > ?`, verifierDigest, now.UTC(), now.UTC()).Scan(
		&session.ID, &session.UserID, &session.VerifierDigest, &session.IdleExpiresAt, &session.AbsoluteExpiresAt, &rotatedFrom,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active web session: %w", err)
	}
	if !bytes.Equal(session.VerifierDigest, verifierDigest) {
		return nil, nil
	}
	session.RotatedFromID = rotatedFrom.String
	return &session, nil
}

// RefreshActiveWebSession advances last-seen and idle expiry without extending
// the absolute lifetime. Invalid, revoked, and expired digests resolve as absent.
func (db *DB) RefreshActiveWebSession(verifierDigest []byte, now time.Time, idleLifetime time.Duration) (*WebSession, error) {
	if len(verifierDigest) == 0 || now.IsZero() || idleLifetime <= 0 {
		return nil, errors.New("verifier digest, current time, and idle lifetime are required")
	}
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin web session refresh: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var session WebSession
	var rotatedFrom sql.NullString
	err = tx.QueryRow(`SELECT web_session_id, user_id, verifier_digest, idle_expires_at, absolute_expires_at, rotated_from_id
		FROM web_sessions WHERE verifier_digest = ? AND revoked_at IS NULL
		AND idle_expires_at > ? AND absolute_expires_at > ?`, verifierDigest, now.UTC(), now.UTC()).Scan(
		&session.ID, &session.UserID, &session.VerifierDigest, &session.IdleExpiresAt, &session.AbsoluteExpiresAt, &rotatedFrom,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active web session for refresh: %w", err)
	}
	if !bytes.Equal(session.VerifierDigest, verifierDigest) {
		return nil, nil
	}
	session.RotatedFromID = rotatedFrom.String
	session.IdleExpiresAt = now.UTC().Add(idleLifetime)
	if session.IdleExpiresAt.After(session.AbsoluteExpiresAt) {
		session.IdleExpiresAt = session.AbsoluteExpiresAt
	}
	if _, err = tx.Exec(`UPDATE web_sessions SET last_seen_at = ?, idle_expires_at = ? WHERE web_session_id = ? AND revoked_at IS NULL`, now.UTC(), session.IdleExpiresAt, session.ID); err != nil {
		return nil, fmt.Errorf("refresh active web session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit web session refresh: %w", err)
	}
	return &session, nil
}

// RotateWebSessionForIdentity atomically resolves or creates the account,
// refreshes mutable profile attributes, revokes the presented prior session,
// and creates the replacement session. Profile email is never an identity key.
func (db *DB) RotateWebSessionForIdentity(rotation WebSessionRotation) (User, WebSession, error) {
	if rotation.Identity.Issuer == "" || rotation.Identity.Subject == "" || rotation.CandidateUserID == "" {
		return User{}, WebSession{}, errors.New("issuer, subject, and candidate user identity are required")
	}
	if rotation.Session.ID == "" || len(rotation.Session.VerifierDigest) == 0 {
		return User{}, WebSession{}, errors.New("session identity and verifier digest are required")
	}
	if rotation.Now.IsZero() || !rotation.Session.IdleExpiresAt.After(rotation.Now) || !rotation.Session.IdleExpiresAt.Before(rotation.Session.AbsoluteExpiresAt) {
		return User{}, WebSession{}, errors.New("valid session time bounds are required")
	}
	tx, err := db.conn.Begin()
	if err != nil {
		return User{}, WebSession{}, fmt.Errorf("begin web session rotation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var user User
	err = tx.QueryRow(`SELECT u.user_id, u.profile_email, u.display_name
		FROM external_identities i JOIN users u ON u.user_id = i.user_id
		WHERE i.issuer = ? AND i.subject = ?`, rotation.Identity.Issuer, rotation.Identity.Subject).Scan(&user.ID, &user.ProfileEmail, &user.DisplayName)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		user = User{ID: rotation.CandidateUserID, ProfileEmail: rotation.Identity.ProfileEmail, DisplayName: rotation.Identity.DisplayName}
		if _, err = tx.Exec(`INSERT INTO users (user_id, profile_email, display_name) VALUES (?, ?, ?)`, user.ID, user.ProfileEmail, user.DisplayName); err != nil {
			return User{}, WebSession{}, fmt.Errorf("create login user: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO external_identities (issuer, subject, user_id, profile_email, display_name) VALUES (?, ?, ?, ?, ?)`, rotation.Identity.Issuer, rotation.Identity.Subject, user.ID, user.ProfileEmail, user.DisplayName); err != nil {
			return User{}, WebSession{}, fmt.Errorf("create login identity: %w", err)
		}
	case err != nil:
		return User{}, WebSession{}, fmt.Errorf("resolve login identity: %w", err)
	default:
		user.ProfileEmail = rotation.Identity.ProfileEmail
		user.DisplayName = rotation.Identity.DisplayName
		if _, err = tx.Exec(`UPDATE external_identities SET profile_email = ?, display_name = ?, updated_at = ? WHERE issuer = ? AND subject = ?`, user.ProfileEmail, user.DisplayName, rotation.Now.UTC(), rotation.Identity.Issuer, rotation.Identity.Subject); err != nil {
			return User{}, WebSession{}, fmt.Errorf("refresh login identity: %w", err)
		}
		if _, err = tx.Exec(`UPDATE users SET profile_email = ?, display_name = ?, updated_at = ? WHERE user_id = ?`, user.ProfileEmail, user.DisplayName, rotation.Now.UTC(), user.ID); err != nil {
			return User{}, WebSession{}, fmt.Errorf("refresh login user: %w", err)
		}
	}

	var previousSessionID string
	if len(rotation.PreviousVerifierDigest) > 0 {
		err = tx.QueryRow(`SELECT web_session_id FROM web_sessions WHERE verifier_digest = ? AND revoked_at IS NULL`, rotation.PreviousVerifierDigest).Scan(&previousSessionID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return User{}, WebSession{}, fmt.Errorf("resolve previous web session: %w", err)
		}
		if err == nil {
			if _, err = tx.Exec(`UPDATE web_sessions SET revoked_at = ? WHERE web_session_id = ? AND revoked_at IS NULL`, rotation.Now.UTC(), previousSessionID); err != nil {
				return User{}, WebSession{}, fmt.Errorf("revoke previous web session: %w", err)
			}
		}
	}

	rotation.Session.UserID = user.ID
	rotation.Session.RotatedFromID = previousSessionID
	var rotatedFrom any
	if previousSessionID != "" {
		rotatedFrom = previousSessionID
	}
	if _, err = tx.Exec(`INSERT INTO web_sessions
		(web_session_id, user_id, verifier_digest, idle_expires_at, absolute_expires_at, rotated_from_id)
		VALUES (?, ?, ?, ?, ?, ?)`, rotation.Session.ID, rotation.Session.UserID, rotation.Session.VerifierDigest,
		rotation.Session.IdleExpiresAt.UTC(), rotation.Session.AbsoluteExpiresAt.UTC(), rotatedFrom); err != nil {
		return User{}, WebSession{}, fmt.Errorf("create rotated web session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return User{}, WebSession{}, fmt.Errorf("commit web session rotation: %w", err)
	}
	return user, rotation.Session, nil
}

func (db *DB) RevokeWebSession(userID, sessionID string) (bool, error) {
	result, err := db.conn.Exec(`UPDATE web_sessions SET revoked_at = CURRENT_TIMESTAMP
		WHERE web_session_id = ? AND user_id = ? AND revoked_at IS NULL`, sessionID, userID)
	if err != nil {
		return false, fmt.Errorf("revoke web session: %w", err)
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (db *DB) RevokeAllWebSessions(userID string) (int64, error) {
	result, err := db.conn.Exec(`UPDATE web_sessions SET revoked_at = CURRENT_TIMESTAMP WHERE user_id = ? AND revoked_at IS NULL`, userID)
	if err != nil {
		return 0, fmt.Errorf("revoke all web sessions: %w", err)
	}
	return result.RowsAffected()
}

func (db *DB) CreateAccountGame(accountGame AccountGame) error {
	if accountGame.ID == "" || accountGame.UserID == "" || accountGame.BasePuzzleID == "" || accountGame.PlayRunID == "" || len(accountGame.EngineState) == 0 || accountGame.ActualDifficulty == "" {
		return errors.New("account game identity, owner, puzzle, play run, state, and difficulty are required")
	}
	if accountGame.Revision < 0 {
		return errors.New("account game revision must be non-negative")
	}
	_, err := db.conn.Exec(`INSERT INTO account_games
		(account_game_id, user_id, base_puzzle_id, play_run_id, engine_state, revision, actual_difficulty)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, accountGame.ID, accountGame.UserID, accountGame.BasePuzzleID, accountGame.PlayRunID, accountGame.EngineState, accountGame.Revision, accountGame.ActualDifficulty)
	if err != nil {
		return fmt.Errorf("create account game: %w", err)
	}
	return nil
}

// AccountGameByID includes the owner predicate in the query so callers cannot
// distinguish another user's game from an absent game.
func (db *DB) AccountGameByID(userID, accountGameID string) (*AccountGame, error) {
	var accountGame AccountGame
	err := db.conn.QueryRow(`SELECT account_game_id, user_id, base_puzzle_id, play_run_id, engine_state, revision, actual_difficulty, updated_at
		FROM account_games WHERE account_game_id = ? AND user_id = ?`, accountGameID, userID).Scan(
		&accountGame.ID, &accountGame.UserID, &accountGame.BasePuzzleID, &accountGame.PlayRunID, &accountGame.EngineState, &accountGame.Revision, &accountGame.ActualDifficulty, &accountGame.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get account game: %w", err)
	}
	return &accountGame, nil
}

// AccountGamesByUser returns bounded game summaries in most-recently-updated
// order without exposing another owner's rows.
func (db *DB) AccountGamesByUser(userID string, limit int) ([]AccountGame, error) {
	if userID == "" || limit < 1 || limit > 100 {
		return nil, errors.New("user identity and a limit from 1 through 100 are required")
	}
	rows, err := db.conn.Query(`SELECT account_game_id, user_id, base_puzzle_id, play_run_id, revision, actual_difficulty, updated_at
		FROM account_games WHERE user_id = ? ORDER BY updated_at DESC, account_game_id LIMIT ?`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list account games: %w", err)
	}
	defer rows.Close()
	games := make([]AccountGame, 0)
	for rows.Next() {
		var game AccountGame
		if err := rows.Scan(&game.ID, &game.UserID, &game.BasePuzzleID, &game.PlayRunID, &game.Revision, &game.ActualDifficulty, &game.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan account game: %w", err)
		}
		games = append(games, game)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list account games: %w", err)
	}
	return games, nil
}

// UpdateAccountGame uses optimistic revision matching and owner-scoped lookup.
func (db *DB) UpdateAccountGame(userID, accountGameID string, expectedRevision int64, engineState []byte) (bool, error) {
	if expectedRevision < 0 || len(engineState) == 0 {
		return false, errors.New("expected revision and engine state are required")
	}
	result, err := db.conn.Exec(`UPDATE account_games SET engine_state = ?, revision = revision + 1, updated_at = CURRENT_TIMESTAMP
		WHERE account_game_id = ? AND user_id = ? AND revision = ?`, engineState, accountGameID, userID, expectedRevision)
	if err != nil {
		return false, fmt.Errorf("update account game: %w", err)
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (db *DB) DeleteAccountGame(userID, accountGameID string) (bool, error) {
	result, err := db.conn.Exec(`DELETE FROM account_games WHERE account_game_id = ? AND user_id = ?`, accountGameID, userID)
	if err != nil {
		return false, fmt.Errorf("delete account game: %w", err)
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

// DeleteUser removes identities, sessions, and owned games through foreign-key
// cascades while preserving catalog puzzles and play-run history.
func (db *DB) DeleteUser(userID string) (bool, error) {
	result, err := db.conn.Exec(`DELETE FROM users WHERE user_id = ?`, userID)
	if err != nil {
		return false, fmt.Errorf("delete user: %w", err)
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}
