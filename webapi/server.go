package webapi

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/playrun"
	"github.com/gnailuy/sudoku/recovery"
)

const (
	MaxRequestBytes  = 1 << 20
	SessionMediaType = "application/vnd.sudoku.session+json"
)

type CreateGame func(kind, value string) (game.Game, error)
type CreateTrackedGame func(kind, value string) (game.Game, *playrun.Tracker, error)

type persistentSession struct {
	Version  int             `json:"version"`
	Revision int64           `json:"revision"`
	Game     json.RawMessage `json:"game"`
}

type entry struct {
	mu        sync.Mutex
	game      game.Game
	revision  int64
	updatedAt time.Time
	recovered bool
	tracker   *playrun.Tracker
}

type Registry struct {
	mu             sync.RWMutex
	entries        map[string]*entry
	store          recovery.Store
	options        game.Options
	trackerFactory func(game.Game) *playrun.Tracker
}

func NewRegistry(store recovery.Store, options game.Options) (*Registry, error) {
	r := &Registry{entries: make(map[string]*entry), store: store, options: options}
	records, err := store.Discover(func(data []byte) error {
		_, _, err := decodePersistent(data, options)
		return err
	})
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		g, revision, err := decodePersistent(record.Session, options)
		if err != nil {
			continue
		}
		r.entries[record.ID] = &entry{game: g, revision: revision, updatedAt: record.UpdatedAt, recovered: true}
	}
	return r, nil
}

func decodePersistent(data []byte, options game.Options) (game.Game, int64, error) {
	var doc persistentSession
	if err := decodeStrict(data, &doc); err != nil {
		return game.Game{}, 0, err
	}
	if doc.Version != 1 || doc.Revision < 0 || len(doc.Game) == 0 {
		return game.Game{}, 0, errors.New("invalid API session record")
	}
	g, err := game.Restore(doc.Game, options)
	return g, doc.Revision, err
}

// SetTrackerFactory attaches one tracker to each recovered or newly imported play run.
func (r *Registry) SetTrackerFactory(factory func(game.Game) *playrun.Tracker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trackerFactory = factory
	for _, entry := range r.entries {
		entry.mu.Lock()
		if factory == nil {
			entry.tracker = nil
		} else {
			entry.tracker = factory(entry.game)
		}
		entry.mu.Unlock()
	}
}

func (r *Registry) persist(id string, e *entry) error {
	data, err := e.game.Serialize()
	if err != nil {
		return err
	}
	doc, err := json.Marshal(persistentSession{Version: 1, Revision: e.revision, Game: data})
	if err != nil {
		return err
	}
	if err := r.store.Write(id, "api", doc); err != nil {
		return err
	}
	e.updatedAt = time.Now().UTC()
	return nil
}

func (r *Registry) add(g game.Game) (string, *entry, error) {
	id, err := recovery.NewID()
	if err != nil {
		return "", nil, err
	}
	e := &entry{game: g, updatedAt: time.Now().UTC()}
	if err := r.persist(id, e); err != nil {
		return "", nil, err
	}
	r.mu.Lock()
	if r.trackerFactory != nil {
		e.tracker = r.trackerFactory(g)
	}
	r.entries[id] = e
	r.mu.Unlock()
	return id, e, nil
}

func (r *Registry) get(id string) (*entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[id]
	return e, ok
}

func (r *Registry) delete(id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[id]
	if !ok {
		return false, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := r.store.Delete(id); err != nil {
		return true, err
	}
	delete(r.entries, id)
	return true, nil
}

type Server struct {
	registry          *Registry
	createGame        CreateGame
	createTrackedGame CreateTrackedGame
}

func NewServer(registry *Registry, createGame CreateGame) *Server {
	return &Server{registry: registry, createGame: createGame}
}

func NewTrackedServer(registry *Registry, createGame CreateTrackedGame) *Server {
	return &Server{registry: registry, createTrackedGame: createGame}
}

func (s *Server) ListSessions(context.Context, ListSessionsRequestObject) (ListSessionsResponseObject, error) {
	s.registry.mu.RLock()
	items := make([]SessionSummary, 0, len(s.registry.entries))
	for id, e := range s.registry.entries {
		e.mu.Lock()
		items = append(items, SessionSummary{Id: id, Revision: e.revision, Status: apiStatus(e.game.Snapshot().Status), UpdatedAt: e.updatedAt, Recovered: e.recovered})
		e.mu.Unlock()
	}
	s.registry.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return ListSessions200JSONResponse{Sessions: items}, nil
}

func (s *Server) CreateSession(_ context.Context, request CreateSessionRequestObject) (CreateSessionResponseObject, error) {
	if request.Body == nil {
		return CreateSession400JSONResponse{BadRequestJSONResponse(apiError(ErrorCodeInvalidRequest, "request body is required"))}, nil
	}
	kind, err := request.Body.Source.Discriminator()
	if err != nil {
		return CreateSession400JSONResponse{BadRequestJSONResponse(apiError(ErrorCodeInvalidJson, "invalid source"))}, nil
	}
	var value string
	switch kind {
	case "difficulty":
		v, parseErr := request.Body.Source.AsDifficultySource()
		if parseErr != nil {
			err = parseErr
		} else {
			value = string(v.Difficulty)
		}
	case "puzzle":
		v, parseErr := request.Body.Source.AsPuzzleSource()
		if parseErr != nil {
			err = parseErr
		} else {
			value = v.Puzzle
		}
	default:
		err = errors.New("unknown source kind")
	}
	if err != nil {
		return CreateSession422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeInvalidRequest, "invalid session source"))}, nil
	}
	var g game.Game
	var tracker *playrun.Tracker
	if s.createTrackedGame != nil {
		g, tracker, err = s.createTrackedGame(kind, value)
	} else {
		g, err = s.createGame(kind, value)
	}
	if err != nil {
		return CreateSession422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeInvalidSession, err.Error()))}, nil
	}
	id, e, err := s.registry.add(g)
	if e != nil && tracker != nil {
		e.mu.Lock()
		e.tracker = tracker
		e.mu.Unlock()
	}
	if err != nil {
		return CreateSession500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodePersistenceFailed, "unable to persist session"))}, nil
	}
	return CreateSession201JSONResponse{Body: apiSession(id, e), Headers: CreateSession201ResponseHeaders{Location: "/api/v1/sessions/" + id}}, nil
}

func (s *Server) ImportSession(context.Context, ImportSessionRequestObject) (ImportSessionResponseObject, error) {
	return ImportSession400JSONResponse{BadRequestJSONResponse(apiError(ErrorCodeInvalidRequest, "import is handled by the raw session adapter"))}, nil
}

func (s *Server) DeleteSession(_ context.Context, request DeleteSessionRequestObject) (DeleteSessionResponseObject, error) {
	found, err := s.registry.delete(request.SessionId)
	if !found {
		return DeleteSession404JSONResponse{NotFoundJSONResponse(apiError(ErrorCodeSessionNotFound, "session not found"))}, nil
	}
	if err != nil {
		return DeleteSession500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodePersistenceFailed, "unable to delete session"))}, nil
	}
	return DeleteSession204Response{}, nil
}

func (s *Server) GetSession(_ context.Context, request GetSessionRequestObject) (GetSessionResponseObject, error) {
	e, ok := s.registry.get(request.SessionId)
	if !ok {
		return GetSession404JSONResponse{NotFoundJSONResponse(apiError(ErrorCodeSessionNotFound, "session not found"))}, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return GetSession200JSONResponse(apiSession(request.SessionId, e)), nil
}

func (s *Server) ApplyAction(_ context.Context, request ApplyActionRequestObject) (ApplyActionResponseObject, error) {
	e, ok := s.registry.get(request.SessionId)
	if !ok {
		return ApplyAction404JSONResponse{NotFoundJSONResponse(apiError(ErrorCodeSessionNotFound, "session not found"))}, nil
	}
	if request.Body == nil {
		return ApplyAction400JSONResponse{BadRequestJSONResponse(apiError(ErrorCodeInvalidRequest, "request body is required"))}, nil
	}
	action, expected, err := translateAction(*request.Body)
	if err != nil {
		return ApplyAction422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeInvalidAction, err.Error()))}, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if expected != e.revision {
		conflict := RevisionConflict{CurrentRevision: e.revision, Snapshot: apiSnapshot(e.game.Snapshot())}
		conflict.Error.Code = "revision-conflict"
		conflict.Error.Message = "expected_revision does not match current revision"
		return ApplyAction409JSONResponse(conflict), nil
	}
	before, serializeErr := e.game.Serialize()
	if serializeErr != nil {
		return ApplyAction500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to snapshot session"))}, nil
	}
	beforeStatus := e.game.Snapshot().Status
	result, err := e.game.Apply(action)
	if err != nil {
		var engineErr *game.EngineError
		if errors.As(err, &engineErr) {
			return ApplyAction422JSONResponse{UnprocessableEntityJSONResponse(engineAPIError(engineErr))}, nil
		}
		return ApplyAction500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "unable to apply action"))}, nil
	}
	e.revision++
	if err := s.registry.persist(request.SessionId, e); err != nil {
		if restored, restoreErr := game.Restore(before, s.registry.options); restoreErr == nil {
			e.game = restored
			e.revision--
		}
		return ApplyAction500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodePersistenceFailed, "unable to persist action"))}, nil
	}
	var warnings *[]string
	if e.tracker != nil {
		e.tracker.Observe(beforeStatus, result)
		if warning := e.tracker.TakeWarning(); warning != nil {
			items := []string{warning.Error()}
			warnings = &items
		}
	}
	return ApplyAction200JSONResponse(ActionResponse{Revision: e.revision, Snapshot: apiSnapshot(e.game.Snapshot()), Result: apiResult(result), Warnings: warnings}), nil
}

func (s *Server) ExportSession(context.Context, ExportSessionRequestObject) (ExportSessionResponseObject, error) {
	return ExportSession500JSONResponse{InternalErrorJSONResponse(apiError(ErrorCodeInternalError, "export is handled by the raw session adapter"))}, nil
}

func (s *Server) PreviewHint(_ context.Context, request PreviewHintRequestObject) (PreviewHintResponseObject, error) {
	e, ok := s.registry.get(request.SessionId)
	if !ok {
		return PreviewHint404JSONResponse{NotFoundJSONResponse(apiError(ErrorCodeSessionNotFound, "session not found"))}, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	h := e.game.Hint()
	if h == nil {
		return PreviewHint422JSONResponse{UnprocessableEntityJSONResponse(apiError(ErrorCodeNoHint, "no hint is available"))}, nil
	}
	return PreviewHint200JSONResponse(HintPreview{Revision: e.revision, Hint: Hint{Row: h.Cell.Position.Row + 1, Column: h.Cell.Position.Column + 1, Value: h.Cell.Value, Technique: h.Technique, Reason: h.Reason}}), nil
}

func (*Server) GetHealth(context.Context, GetHealthRequestObject) (GetHealthResponseObject, error) {
	return GetHealth200JSONResponse{Status: "healthy"}, nil
}

func (s *Server) RawImport(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBytes+1))
	if err != nil {
		writeError(w, 400, ErrorCodeInvalidRequest, "unable to read request")
		return
	}
	if len(data) > MaxRequestBytes {
		writeError(w, 413, ErrorCodePayloadTooLarge, "request exceeds 1 MiB")
		return
	}
	g, err := game.Restore(data, s.registry.options)
	if err != nil {
		writeError(w, 422, ErrorCodeInvalidSession, err.Error())
		return
	}
	id, e, err := s.registry.add(g)
	if err != nil {
		writeError(w, 500, ErrorCodePersistenceFailed, "unable to persist session")
		return
	}
	w.Header().Set("Location", "/api/v1/sessions/"+id)
	writeJSON(w, 201, apiSession(id, e))
}

func (s *Server) RawExport(w http.ResponseWriter, _ *http.Request, id string) {
	e, ok := s.registry.get(id)
	if !ok {
		writeError(w, 404, ErrorCodeSessionNotFound, "session not found")
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	data, err := e.game.Serialize()
	if err != nil {
		writeError(w, 500, ErrorCodeInternalError, "unable to serialize session")
		return
	}
	w.Header().Set("Content-Type", SessionMediaType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"sudoku-%s.json\"", id))
	w.WriteHeader(200)
	_, _ = w.Write(data)
}

func NewHandler(server *Server, token string, origins []string) http.Handler {
	strict := NewStrictHandlerWithOptions(server, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeError(w, 400, ErrorCodeInvalidJson, "invalid JSON request")
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeError(w, 500, ErrorCodeInternalError, "internal server error")
		},
	})
	generated := Handler(strict)
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/sessions/import" {
			server.RawImport(w, r)
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/sessions/") && strings.HasSuffix(r.URL.Path, "/export") {
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/"), "/export")
			server.RawExport(w, r, id)
			return
		}
		generated.ServeHTTP(w, r)
	}), token, allowed)
}

func middleware(next http.Handler, token string, origins map[string]struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if _, ok := origins[origin]; !ok {
				writeError(w, 403, ErrorCodeForbiddenOrigin, "origin is not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			if r.Method == http.MethodOptions {
				if !validPreflight(r) {
					writeError(w, 403, ErrorCodeForbiddenOrigin, "preflight request is not allowed")
					return
				}
				w.WriteHeader(204)
				return
			}
		}
		if r.URL.Path != "/healthz" && token != "" {
			value, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || value == "" || subtle.ConstantTimeCompare([]byte(value), []byte(token)) != 1 {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeError(w, 401, ErrorCodeUnauthorized, "valid bearer token required")
				return
			}
		}
		if r.Method == http.MethodPost {
			want := "application/json"
			if r.URL.Path == "/api/v1/sessions/import" {
				want = SessionMediaType
			}
			got := strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0])
			if got != want {
				writeError(w, 415, ErrorCodeUnsupportedMediaType, "unsupported content type")
				return
			}
			data, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBytes+1))
			if err != nil {
				writeError(w, 400, ErrorCodeInvalidRequest, "unable to read request")
				return
			}
			if len(data) > MaxRequestBytes {
				writeError(w, 413, ErrorCodePayloadTooLarge, "request exceeds 1 MiB")
				return
			}
			if want == "application/json" {
				if err := validateJSONRequest(r.URL.Path, data); err != nil {
					writeError(w, 400, ErrorCodeInvalidJson, err.Error())
					return
				}
			}
			r.Body = io.NopCloser(bytes.NewReader(data))
		}
		next.ServeHTTP(w, r)
	})
}

func validPreflight(r *http.Request) bool {
	switch r.Header.Get("Access-Control-Request-Method") {
	case http.MethodGet, http.MethodPost, http.MethodDelete:
	default:
		return false
	}
	for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
		header = strings.TrimSpace(header)
		if header == "" {
			continue
		}
		if !strings.EqualFold(header, "Authorization") && !strings.EqualFold(header, "Content-Type") {
			return false
		}
	}
	return true
}

func translateAction(body ActionRequest) (game.Action, int64, error) {
	kind, err := body.Discriminator()
	if err != nil {
		return nil, 0, err
	}
	position := func(row, column int) (core.Position, error) {
		value, err := core.NewPositionFromInput(row, column)
		if err != nil {
			return core.Position{}, err
		}
		return *value, nil
	}
	validRevision := func(value int64) error {
		if value < 0 {
			return errors.New("expected_revision must be non-negative")
		}
		return nil
	}
	switch kind {
	case "set-value":
		v, e := body.AsSetValueAction()
		if e != nil {
			return nil, 0, e
		}
		p, e := position(v.Row, v.Column)
		if e != nil || v.Value < 1 || v.Value > 9 {
			return nil, 0, errors.New("invalid set-value action")
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.SetValue{Position: p, Value: v.Value}, v.ExpectedRevision, nil
	case "clear-value":
		v, e := body.AsClearValueAction()
		if e != nil {
			return nil, 0, e
		}
		p, e := position(v.Row, v.Column)
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.ClearValue{Position: p}, v.ExpectedRevision, nil
	case "set-notes":
		v, e := body.AsSetNotesAction()
		if e != nil {
			return nil, 0, e
		}
		p, e := position(v.Row, v.Column)
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.SetNotes{Position: p, Values: append([]int(nil), v.Values...)}, v.ExpectedRevision, nil
	case "reset":
		v, e := body.AsResetAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.Reset{}, v.ExpectedRevision, nil
	case "undo":
		v, e := body.AsUndoAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.Undo{}, v.ExpectedRevision, nil
	case "redo":
		v, e := body.AsRedoAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.Redo{}, v.ExpectedRevision, nil
	case "apply-hint":
		v, e := body.AsApplyHintAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.ApplyHint{}, v.ExpectedRevision, nil
	case "repair":
		v, e := body.AsRepairAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.Repair{}, v.ExpectedRevision, nil
	case "solve":
		v, e := body.AsSolveAction()
		if e != nil {
			return nil, 0, e
		}
		if e = validRevision(v.ExpectedRevision); e != nil {
			return nil, 0, e
		}
		return game.Solve{}, v.ExpectedRevision, nil
	default:
		return nil, 0, errors.New("unknown action kind")
	}
}

func apiSession(id string, e *entry) Session {
	return Session{Id: id, Revision: e.revision, Snapshot: apiSnapshot(e.game.Snapshot())}
}
func apiStatus(v game.Status) GameStatus { return GameStatus(v) }
func digits(set core.CandidateSet) []int { return set.Values() }
func apiSnapshot(v game.Snapshot) Snapshot {
	o := Snapshot{Status: apiStatus(v.Status), CanUndo: v.CanUndo, CanRedo: v.CanRedo, Givens: make(ValueGrid, 9), Values: make(ValueGrid, 9), Invalid: make(BooleanGrid, 9), Notes: make(DigitSetGrid, 9), Candidates: make(DigitSetGrid, 9)}
	for r := 0; r < 9; r++ {
		o.Givens[r] = make(ValueRow, 9)
		o.Values[r] = make(ValueRow, 9)
		o.Invalid[r] = make(BooleanRow, 9)
		o.Notes[r] = make(DigitSetRow, 9)
		o.Candidates[r] = make(DigitSetRow, 9)
		for c := 0; c < 9; c++ {
			o.Givens[r][c] = v.Givens[r][c]
			o.Values[r][c] = v.Values[r][c]
			o.Invalid[r][c] = v.Invalid[r][c]
			o.Notes[r][c] = digits(v.Notes[r][c])
			o.Candidates[r][c] = digits(v.Candidates[r][c])
		}
	}
	return o
}
func apiResult(v game.Result) ActionResult {
	o := ActionResult{Action: ActionKind(v.Action), Status: apiStatus(v.Status), CanUndo: v.CanUndo, CanRedo: v.CanRedo, Changes: make([]CellChange, len(v.Changes))}
	for i, c := range v.Changes {
		o.Changes[i] = CellChange{Row: c.Position.Row + 1, Column: c.Position.Column + 1, Before: c.Before, After: c.After, InvalidBefore: c.InvalidBefore, InvalidAfter: c.InvalidAfter, NotesBefore: digits(c.NotesBefore), NotesAfter: digits(c.NotesAfter)}
	}
	if v.Hint != nil {
		o.Hint = &Hint{Row: v.Hint.Position.Row + 1, Column: v.Hint.Position.Column + 1, Value: v.Hint.Value, Technique: v.Hint.Technique, Reason: v.Hint.Reason}
	}
	return o
}
func apiError(code ErrorCode, message string) Error {
	return Error{Error: ErrorDetail{Code: code, Message: message}}
}
func engineAPIError(e *game.EngineError) Error {
	out := apiError(ErrorCode(e.Code), e.Error())
	if e.Position != nil {
		out.Error.Cell = &CellPosition{Row: e.Position.Row + 1, Column: e.Position.Column + 1}
	}
	return out
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code ErrorCode, message string) {
	writeJSON(w, status, apiError(code, message))
}
func decodeStrict(data []byte, value any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}

func validateJSONRequest(path string, data []byte) error {
	var object map[string]json.RawMessage
	if err := decodeStrict(data, &object); err != nil {
		return err
	}
	allowed := map[string]bool{}
	switch {
	case path == "/api/v1/sessions":
		allowed["source"] = true
		raw, ok := object["source"]
		if !ok {
			return errors.New("source is required")
		}
		var source map[string]json.RawMessage
		if err := decodeStrict(raw, &source); err != nil {
			return err
		}
		var kind string
		if err := json.Unmarshal(source["kind"], &kind); err != nil {
			return errors.New("source kind is required")
		}
		if kind == "difficulty" {
			if err := requireKeys(source, "kind", "difficulty"); err != nil {
				return err
			}
		} else if kind == "puzzle" {
			if err := requireKeys(source, "kind", "puzzle"); err != nil {
				return err
			}
		} else {
			return errors.New("unknown source kind")
		}
	case strings.HasSuffix(path, "/actions"):
		var kind string
		if err := json.Unmarshal(object["kind"], &kind); err != nil {
			return errors.New("action kind is required")
		}
		keys := []string{"kind", "expected_revision"}
		switch kind {
		case "set-value":
			keys = append(keys, "row", "column", "value")
		case "set-notes":
			keys = append(keys, "row", "column", "values")
		case "clear-value":
			keys = append(keys, "row", "column")
		case "reset", "undo", "redo", "apply-hint", "repair", "solve":
		default:
			return errors.New("unknown action kind")
		}
		return requireKeys(object, keys...)
	default:
		return nil
	}
	for key := range object {
		if !allowed[key] {
			return fmt.Errorf("unknown field %q", key)
		}
	}
	return nil
}

func requireKeys(object map[string]json.RawMessage, keys ...string) error {
	allowed := make(map[string]bool, len(keys))
	for _, key := range keys {
		allowed[key] = true
		if _, ok := object[key]; !ok {
			return fmt.Errorf("field %q is required", key)
		}
	}
	for key := range object {
		if !allowed[key] {
			return fmt.Errorf("unknown field %q", key)
		}
	}
	return nil
}
