package webapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gnailuy/sudoku/accountauth"
	"github.com/gnailuy/sudoku/accountgame"
	"github.com/gnailuy/sudoku/db"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/guestclaim"
	"github.com/gnailuy/sudoku/guestdoc"
	"github.com/gnailuy/sudoku/oidcauth"
)

const accountGameLimit = 100

type AccountStore interface {
	UserByID(string) (*db.User, error)
	RevokeAllWebSessions(string) (int64, error)
	DeleteUser(string) (bool, error)
}

type CreateAccountGameFunc func(kind, value string) (game.Game, string, string, Difficulty, error)
type CreateGuestGameFunc func(kind, value string) (game.Game, string, Difficulty, error)

type AccountRuntime struct {
	Documents   *guestdoc.Sealer
	OIDC        *oidcauth.Manager
	Auth        *accountauth.Service
	Games       *accountgame.Service
	Claims      *guestclaim.Service
	Store       AccountStore
	CreateGuest CreateGuestGameFunc
	CreateOwned CreateAccountGameFunc
	Options     game.Options
	CookiePath  string
	CSRFKey     []byte
}

func (runtime *AccountRuntime) valid() bool {
	return runtime != nil && runtime.Documents != nil && runtime.OIDC != nil && runtime.Auth != nil && runtime.Games != nil && runtime.Claims != nil && runtime.Store != nil && runtime.CreateGuest != nil && runtime.CreateOwned != nil && runtime.CookiePath != "" && len(runtime.CSRFKey) >= 32
}

func (s *Server) SetAccountRuntime(runtime *AccountRuntime) { s.accounts = runtime }

func sourceValue(source CreateSessionRequest_Source) (string, string, error) {
	kind, err := source.Discriminator()
	if err != nil {
		return "", "", err
	}
	switch kind {
	case "difficulty":
		value, err := source.AsDifficultySource()
		return kind, string(value.Difficulty), err
	case "puzzle":
		value, err := source.AsPuzzleSource()
		return kind, value.Puzzle, err
	default:
		return "", "", errors.New("unknown source kind")
	}
}

func unavailable() Error {
	return apiError(ErrorCodeAccountUnavailable, "account routes are not configured")
}
func accountUnauthorized() Error {
	return apiError(ErrorCodeAccountUnauthorized, "valid application session required")
}
func csrfFailed() Error      { return apiError(ErrorCodeCsrfFailed, "request proof is invalid") }
func accountNotFound() Error { return apiError(ErrorCodeAccountGameNotFound, "account game not found") }

func (runtime *AccountRuntime) authenticate(verifier string) (*db.WebSession, error) {
	return runtime.Auth.Authenticate(verifier)
}

func (runtime *AccountRuntime) csrfToken(verifier string) string {
	mac := hmac.New(sha256.New, runtime.CSRFKey)
	_, _ = mac.Write([]byte("sudoku-account-csrf:v1:" + verifier))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (runtime *AccountRuntime) validCSRF(verifier, supplied string) bool {
	expected := runtime.csrfToken(verifier)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(supplied)) == 1
}

func (runtime *AccountRuntime) sessionCookie(verifier string, expires time.Time) string {
	cookie := http.Cookie{Name: "sudoku_session", Value: verifier, Path: runtime.CookiePath, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: int(time.Until(expires).Seconds())}
	return cookie.String()
}

func (runtime *AccountRuntime) clearCookie() string {
	return (&http.Cookie{Name: "sudoku_session", Path: runtime.CookiePath, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: time.Unix(1, 0), MaxAge: -1}).String()
}

func accountGame(state accountgame.State) AccountGame {
	return AccountGame{Id: state.Game.ID, Revision: state.Game.Revision, ActualDifficulty: Difficulty(state.Game.ActualDifficulty), ElapsedSeconds: state.Game.ElapsedSeconds, Snapshot: apiSnapshot(state.Snapshot)}
}

func (s *Server) CreateGuestGame(_ context.Context, request CreateGuestGameRequestObject) (CreateGuestGameResponseObject, error) {
	if !s.accounts.valid() {
		return CreateGuestGame503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	if request.Body == nil {
		return CreateGuestGame400JSONResponse{BadRequestJSONResponse(apiError(ErrorCodeInvalidRequest, "request body is required"))}, nil
	}
	kind, value, err := sourceValue(request.Body.Source)
	if err != nil {
		return CreateGuestGame422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeInvalidRequest, "invalid game source"))}, nil
	}
	current, baseID, difficulty, err := s.accounts.CreateGuest(kind, value)
	if err != nil {
		return CreateGuestGame422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeInvalidSession, err.Error()))}, nil
	}
	token, document, err := s.accounts.Documents.Create(&current, baseID, string(difficulty))
	if err != nil {
		return CreateGuestGame500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to seal guest game"))}, nil
	}
	return CreateGuestGame201JSONResponse(GuestGame{Document: token, Revision: document.Revision, ActualDifficulty: difficulty, Snapshot: apiSnapshot(current.Snapshot())}), nil
}

func (s *Server) ApplyGuestAction(_ context.Context, request ApplyGuestActionRequestObject) (ApplyGuestActionResponseObject, error) {
	if !s.accounts.valid() {
		return ApplyGuestAction503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	if request.Body == nil {
		return ApplyGuestAction400JSONResponse{BadRequestJSONResponse(apiError(ErrorCodeInvalidRequest, "request body is required"))}, nil
	}
	action, expected, err := translateAction(request.Body.Action)
	if err != nil {
		return ApplyGuestAction422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeInvalidAction, err.Error()))}, nil
	}
	transition, err := s.accounts.Documents.Apply(request.Body.Document, expected, action, s.accounts.Options)
	if err != nil {
		var documentError *guestdoc.Error
		if errors.As(err, &documentError) && documentError.Code == guestdoc.ErrorRevisionConflict {
			return ApplyGuestAction409JSONResponse{GuestRevisionConflictJSONResponse(apiError(ErrorCodeGuestRevisionConflict, "expected_revision does not match the guest document"))}, nil
		}
		var engineError *game.EngineError
		if errors.As(err, &engineError) {
			return ApplyGuestAction422JSONResponse{UnprocessableEntityJSONResponse(engineAPIError(engineError))}, nil
		}
		return ApplyGuestAction422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeInvalidSession, "guest document is invalid or expired"))}, nil
	}
	return ApplyGuestAction200JSONResponse(GuestActionResponse{Document: transition.Token, Revision: transition.Document.Revision, ActualDifficulty: Difficulty(transition.Document.ActualDifficulty), Snapshot: apiSnapshot(transition.Snapshot), Result: apiResult(transition.Result)}), nil
}

func (s *Server) BeginGoogleLogin(_ context.Context, request BeginGoogleLoginRequestObject) (BeginGoogleLoginResponseObject, error) {
	if !s.accounts.valid() {
		return BeginGoogleLogin503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	start, err := s.accounts.OIDC.Begin(request.Params.ReturnTo)
	if err != nil {
		return BeginGoogleLogin400JSONResponse{BadRequestJSONResponse(apiError(ErrorCodeInvalidRequest, "invalid return destination"))}, nil
	}
	return BeginGoogleLogin303Response{Headers: BeginGoogleLogin303ResponseHeaders{Location: start.AuthorizationURL}}, nil
}

func (s *Server) CompleteGoogleLogin(ctx context.Context, request CompleteGoogleLoginRequestObject) (CompleteGoogleLoginResponseObject, error) {
	if !s.accounts.valid() {
		return CompleteGoogleLogin503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	current := ""
	if request.Params.SudokuSession != nil {
		current = string(*request.Params.SudokuSession)
	}
	login, err := s.accounts.Auth.Complete(ctx, request.Params.State, request.Params.Code, current)
	if err != nil {
		return CompleteGoogleLogin400JSONResponse{BadRequestJSONResponse(apiError(ErrorCodeInvalidRequest, "login callback is invalid or expired"))}, nil
	}
	return CompleteGoogleLogin303Response{Headers: CompleteGoogleLogin303ResponseHeaders{Location: login.ReturnTo, SetCookie: s.accounts.sessionCookie(login.Verifier, login.AbsoluteExpiresAt)}}, nil
}

func (s *Server) GetCurrentAccount(_ context.Context, request GetCurrentAccountRequestObject) (GetCurrentAccountResponseObject, error) {
	if !s.accounts.valid() {
		return GetCurrentAccount503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	if request.Params.SudokuSession == nil {
		return GetCurrentAccount401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	verifier := string(*request.Params.SudokuSession)
	session, err := s.accounts.authenticate(verifier)
	if err != nil {
		return GetCurrentAccount500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to authenticate session"))}, nil
	}
	if session == nil {
		return GetCurrentAccount401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	user, err := s.accounts.Store.UserByID(session.UserID)
	if err != nil {
		return GetCurrentAccount500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to load account"))}, nil
	}
	if user == nil {
		return GetCurrentAccount401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	return GetCurrentAccount200JSONResponse(Account{Email: user.ProfileEmail, DisplayName: user.DisplayName, CsrfToken: s.accounts.csrfToken(verifier)}), nil
}

func (s *Server) LogoutAccount(_ context.Context, request LogoutAccountRequestObject) (LogoutAccountResponseObject, error) {
	if !s.accounts.valid() {
		return LogoutAccount503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	verifier := string(request.Params.SudokuSession)
	if !s.accounts.validCSRF(verifier, string(request.Params.XSudokuCSRF)) {
		return LogoutAccount403JSONResponse{ForbiddenJSONResponse(csrfFailed())}, nil
	}
	ok, err := s.accounts.Auth.Logout(verifier)
	if err != nil {
		return LogoutAccount500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to revoke session"))}, nil
	}
	if !ok {
		return LogoutAccount401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	return LogoutAccount204Response{Headers: LogoutAccount204ResponseHeaders{SetCookie: s.accounts.clearCookie()}}, nil
}

func (s *Server) DeleteAllAccountGames(_ context.Context, request DeleteAllAccountGamesRequestObject) (DeleteAllAccountGamesResponseObject, error) {
	if !s.accounts.valid() {
		return DeleteAllAccountGames503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	verifier := string(request.Params.SudokuSession)
	session, err := s.accounts.authenticate(verifier)
	if err != nil {
		return DeleteAllAccountGames500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to authenticate session"))}, nil
	}
	if session == nil {
		return DeleteAllAccountGames401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	if !s.accounts.validCSRF(verifier, string(request.Params.XSudokuCSRF)) {
		return DeleteAllAccountGames403JSONResponse{ForbiddenJSONResponse(csrfFailed())}, nil
	}
	if _, err := s.accounts.Games.DeleteAll(session.UserID); err != nil {
		return DeleteAllAccountGames500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to delete account games"))}, nil
	}
	return DeleteAllAccountGames204Response{}, nil
}

func (s *Server) ListAccountGames(_ context.Context, request ListAccountGamesRequestObject) (ListAccountGamesResponseObject, error) {
	if !s.accounts.valid() {
		return ListAccountGames503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	session, err := s.accounts.authenticate(string(request.Params.SudokuSession))
	if err != nil {
		return ListAccountGames500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to authenticate session"))}, nil
	}
	if session == nil {
		return ListAccountGames401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	rows, err := s.accounts.Games.List(session.UserID, accountGameLimit)
	if err != nil {
		return ListAccountGames500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to list account games"))}, nil
	}
	items := make([]AccountGameSummary, len(rows))
	for i, row := range rows {
		items[i] = AccountGameSummary{Id: row.ID, Revision: row.Revision, ActualDifficulty: Difficulty(row.ActualDifficulty), Status: GameStatus(row.Status), ElapsedSeconds: row.ElapsedSeconds, UpdatedAt: row.UpdatedAt}
	}
	return ListAccountGames200JSONResponse(AccountGameList{Games: items}), nil
}

func (s *Server) CreateAccountGame(_ context.Context, request CreateAccountGameRequestObject) (CreateAccountGameResponseObject, error) {
	if !s.accounts.valid() {
		return CreateAccountGame503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	verifier := string(request.Params.SudokuSession)
	session, err := s.accounts.authenticate(verifier)
	if err != nil {
		return CreateAccountGame500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to authenticate session"))}, nil
	}
	if session == nil {
		return CreateAccountGame401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	if !s.accounts.validCSRF(verifier, string(request.Params.XSudokuCSRF)) {
		return CreateAccountGame403JSONResponse{ForbiddenJSONResponse(csrfFailed())}, nil
	}
	if request.Body == nil {
		return CreateAccountGame400JSONResponse{BadRequestJSONResponse(apiError(ErrorCodeInvalidRequest, "request body is required"))}, nil
	}
	kind, value, err := sourceValue(request.Body.Source)
	if err != nil {
		return CreateAccountGame422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeInvalidRequest, "invalid game source"))}, nil
	}
	current, baseID, runID, difficulty, err := s.accounts.CreateOwned(kind, value)
	if err != nil {
		return CreateAccountGame422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeInvalidSession, err.Error()))}, nil
	}
	state, err := s.accounts.Games.Create(session.UserID, current, baseID, runID, string(difficulty))
	if err != nil {
		return CreateAccountGame500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodePersistenceFailed, "unable to persist account game"))}, nil
	}
	body := accountGame(state)
	return CreateAccountGame201JSONResponse{Body: body, Headers: CreateAccountGame201ResponseHeaders{Location: "/api/v1/account/games/" + body.Id}}, nil
}

func (s *Server) ClaimGuestGame(_ context.Context, request ClaimGuestGameRequestObject) (ClaimGuestGameResponseObject, error) {
	if !s.accounts.valid() {
		return ClaimGuestGame503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	verifier := string(request.Params.SudokuSession)
	session, err := s.accounts.authenticate(verifier)
	if err != nil {
		return ClaimGuestGame500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to authenticate session"))}, nil
	}
	if session == nil {
		return ClaimGuestGame401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	if !s.accounts.validCSRF(verifier, string(request.Params.XSudokuCSRF)) {
		return ClaimGuestGame403JSONResponse{ForbiddenJSONResponse(csrfFailed())}, nil
	}
	if request.Body == nil {
		return ClaimGuestGame400JSONResponse{BadRequestJSONResponse(apiError(ErrorCodeInvalidRequest, "request body is required"))}, nil
	}
	result, err := s.accounts.Claims.Claim(session.UserID, request.Body.Document)
	if errors.Is(err, db.ErrGuestAlreadyClaimed) {
		return ClaimGuestGame409JSONResponse{ClaimConflictJSONResponse(apiError(ErrorCodeGuestClaimConflict, "guest game was already claimed"))}, nil
	}
	if err != nil {
		return ClaimGuestGame422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeInvalidSession, "guest document is invalid or expired"))}, nil
	}
	state, err := s.accounts.Games.Get(session.UserID, result.Game.ID)
	if err != nil {
		return ClaimGuestGame500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to load claimed game"))}, nil
	}
	body := accountGame(state)
	if result.Created {
		return ClaimGuestGame201JSONResponse(body), nil
	}
	return ClaimGuestGame200JSONResponse(body), nil
}

func (s *Server) GetAccountGame(_ context.Context, request GetAccountGameRequestObject) (GetAccountGameResponseObject, error) {
	if !s.accounts.valid() {
		return GetAccountGame503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	session, err := s.accounts.authenticate(string(request.Params.SudokuSession))
	if err != nil {
		return GetAccountGame500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to authenticate session"))}, nil
	}
	if session == nil {
		return GetAccountGame401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	state, err := s.accounts.Games.Get(session.UserID, request.AccountGameId)
	if errors.Is(err, accountgame.ErrNotFound) {
		return GetAccountGame404JSONResponse{AccountGameNotFoundJSONResponse(accountNotFound())}, nil
	}
	if err != nil {
		return GetAccountGame500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to load account game"))}, nil
	}
	return GetAccountGame200JSONResponse(accountGame(state)), nil
}

func (s *Server) DeleteAccountGame(_ context.Context, request DeleteAccountGameRequestObject) (DeleteAccountGameResponseObject, error) {
	if !s.accounts.valid() {
		return DeleteAccountGame503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	verifier := string(request.Params.SudokuSession)
	session, err := s.accounts.authenticate(verifier)
	if err != nil {
		return DeleteAccountGame500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to authenticate session"))}, nil
	}
	if session == nil {
		return DeleteAccountGame401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	if !s.accounts.validCSRF(verifier, string(request.Params.XSudokuCSRF)) {
		return DeleteAccountGame403JSONResponse{ForbiddenJSONResponse(csrfFailed())}, nil
	}
	_, err = s.accounts.Games.Delete(session.UserID, request.AccountGameId)
	if errors.Is(err, accountgame.ErrNotFound) {
		return DeleteAccountGame404JSONResponse{AccountGameNotFoundJSONResponse(accountNotFound())}, nil
	}
	if err != nil {
		return DeleteAccountGame500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to delete account game"))}, nil
	}
	return DeleteAccountGame204Response{}, nil
}

func (s *Server) UpdateAccountGamePresentation(_ context.Context, request UpdateAccountGamePresentationRequestObject) (UpdateAccountGamePresentationResponseObject, error) {
	if !s.accounts.valid() {
		return UpdateAccountGamePresentation503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	verifier := string(request.Params.SudokuSession)
	session, err := s.accounts.authenticate(verifier)
	if err != nil {
		return UpdateAccountGamePresentation500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to authenticate session"))}, nil
	}
	if session == nil {
		return UpdateAccountGamePresentation401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	if !s.accounts.validCSRF(verifier, string(request.Params.XSudokuCSRF)) {
		return UpdateAccountGamePresentation403JSONResponse{ForbiddenJSONResponse(csrfFailed())}, nil
	}
	if request.Body == nil {
		return UpdateAccountGamePresentation400JSONResponse{BadRequestJSONResponse(apiError(ErrorCodeInvalidRequest, "request body is required"))}, nil
	}
	if request.Body.ElapsedSeconds < 0 {
		return UpdateAccountGamePresentation422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeInvalidRequest, "elapsed_seconds must be non-negative"))}, nil
	}
	state, err := s.accounts.Games.UpdateElapsed(session.UserID, request.AccountGameId, request.Body.ElapsedSeconds)
	if errors.Is(err, accountgame.ErrNotFound) {
		return UpdateAccountGamePresentation404JSONResponse{AccountGameNotFoundJSONResponse(accountNotFound())}, nil
	}
	if err != nil {
		return UpdateAccountGamePresentation500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodePersistenceFailed, "unable to persist account game time"))}, nil
	}
	return UpdateAccountGamePresentation200JSONResponse(accountGame(state)), nil
}

func (s *Server) ApplyAccountGameAction(_ context.Context, request ApplyAccountGameActionRequestObject) (ApplyAccountGameActionResponseObject, error) {
	if !s.accounts.valid() {
		return ApplyAccountGameAction503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	verifier := string(request.Params.SudokuSession)
	session, err := s.accounts.authenticate(verifier)
	if err != nil {
		return ApplyAccountGameAction500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to authenticate session"))}, nil
	}
	if session == nil {
		return ApplyAccountGameAction401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	if !s.accounts.validCSRF(verifier, string(request.Params.XSudokuCSRF)) {
		return ApplyAccountGameAction403JSONResponse{ForbiddenJSONResponse(csrfFailed())}, nil
	}
	if request.Body == nil {
		return ApplyAccountGameAction400JSONResponse{BadRequestJSONResponse(apiError(ErrorCodeInvalidRequest, "request body is required"))}, nil
	}
	action, expected, err := translateAction(*request.Body)
	if err != nil {
		return ApplyAccountGameAction422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeInvalidAction, err.Error()))}, nil
	}
	transition, err := s.accounts.Games.Apply(session.UserID, request.AccountGameId, expected, action)
	var conflict *accountgame.RevisionConflictError
	if errors.As(err, &conflict) {
		return ApplyAccountGameAction409JSONResponse{AccountRevisionConflictJSONResponse(AccountRevisionConflict{Error: ErrorDetail{Code: ErrorCodeRevisionConflict, Message: conflict.Error()}, CurrentRevision: conflict.CurrentRevision})}, nil
	}
	if errors.Is(err, accountgame.ErrNotFound) {
		return ApplyAccountGameAction404JSONResponse{AccountGameNotFoundJSONResponse(accountNotFound())}, nil
	}
	var engineError *game.EngineError
	if errors.As(err, &engineError) {
		return ApplyAccountGameAction422JSONResponse{UnprocessableEntityJSONResponse(engineAPIError(engineError))}, nil
	}
	if err != nil {
		return ApplyAccountGameAction500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodePersistenceFailed, "unable to persist account action"))}, nil
	}
	return ApplyAccountGameAction200JSONResponse(AccountActionResponse{Game: accountGame(transition.State), Result: apiResult(transition.Result)}), nil
}

func (s *Server) RevokeAccountSessions(_ context.Context, request RevokeAccountSessionsRequestObject) (RevokeAccountSessionsResponseObject, error) {
	if !s.accounts.valid() {
		return RevokeAccountSessions503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	verifier := string(request.Params.SudokuSession)
	session, err := s.accounts.authenticate(verifier)
	if err != nil {
		return RevokeAccountSessions500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to authenticate session"))}, nil
	}
	if session == nil {
		return RevokeAccountSessions401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	if !s.accounts.validCSRF(verifier, string(request.Params.XSudokuCSRF)) {
		return RevokeAccountSessions403JSONResponse{ForbiddenJSONResponse(csrfFailed())}, nil
	}
	if _, err := s.accounts.Store.RevokeAllWebSessions(session.UserID); err != nil {
		return RevokeAccountSessions500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to revoke sessions"))}, nil
	}
	return RevokeAccountSessions204Response{Headers: RevokeAccountSessions204ResponseHeaders{SetCookie: s.accounts.clearCookie()}}, nil
}

func (s *Server) DeleteCurrentAccount(_ context.Context, request DeleteCurrentAccountRequestObject) (DeleteCurrentAccountResponseObject, error) {
	if !s.accounts.valid() {
		return DeleteCurrentAccount503JSONResponse{UnavailableJSONResponse(unavailable())}, nil
	}
	verifier := string(request.Params.SudokuSession)
	session, err := s.accounts.authenticate(verifier)
	if err != nil {
		return DeleteCurrentAccount500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to authenticate session"))}, nil
	}
	if session == nil {
		return DeleteCurrentAccount401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	if !s.accounts.validCSRF(verifier, string(request.Params.XSudokuCSRF)) {
		return DeleteCurrentAccount403JSONResponse{ForbiddenJSONResponse(csrfFailed())}, nil
	}
	deleted, err := s.accounts.Store.DeleteUser(session.UserID)
	if err != nil {
		return DeleteCurrentAccount500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to delete account"))}, nil
	}
	if !deleted {
		return DeleteCurrentAccount401JSONResponse{AccountUnauthorizedJSONResponse(accountUnauthorized())}, nil
	}
	return DeleteCurrentAccount204Response{Headers: DeleteCurrentAccount204ResponseHeaders{SetCookie: s.accounts.clearCookie()}}, nil
}

func (runtime *AccountRuntime) String() string {
	if runtime == nil {
		return "accounts disabled"
	}
	return fmt.Sprintf("accounts cookie path %s", runtime.CookiePath)
}
