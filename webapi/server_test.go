package webapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gnailuy/sudoku/core"
	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/playrun"
	"github.com/gnailuy/sudoku/recovery"
	"github.com/gnailuy/sudoku/solver"
)

const knownPuzzle = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."

func testHandler(t *testing.T, token string, origins []string) (http.Handler, recovery.Store, game.Options) {
	t.Helper()
	store := solver.NewStore()
	options := game.NewDefaultOptions(store)
	options.StrategySolverKeys = store.GetAllStrategySolverKeys()
	recoveryStore := recovery.NewStore(filepath.Join(t.TempDir(), "recovery"))
	registry, err := NewRegistry(recoveryStore, options)
	if err != nil {
		t.Fatal(err)
	}
	factory := func(_, _ string) (game.Game, error) {
		board := core.NewEmptyBoard()
		board.FromString(knownPuzzle)
		return game.NewGame(board, options), nil
	}
	return NewHandler(NewServer(registry, factory), token, origins), recoveryStore, options
}

func request(t *testing.T, handler http.Handler, method, path, contentType, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	for key, value := range headers {
		r.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func createTestSession(t *testing.T, handler http.Handler, headers map[string]string) Session {
	t.Helper()
	w := request(t, handler, http.MethodPost, "/api/v1/sessions", "application/json", `{"source":{"kind":"puzzle","puzzle":"`+knownPuzzle+`"}}`, headers)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	var session Session
	if err := json.Unmarshal(w.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	if session.Id == "" || session.Revision != 0 || len(session.Snapshot.Values) != 9 {
		t.Fatalf("invalid session: %+v", session)
	}
	return session
}

func TestCreateSessionAcceptsCanonicalDifficultySource(t *testing.T) {
	handler, _, _ := testHandler(t, "", nil)
	w := request(t, handler, http.MethodPost, "/api/v1/sessions", "application/json", `{"source":{"kind":"difficulty","difficulty":"easy"}}`, nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSessionLifecycleRecoveryAndTransfer(t *testing.T) {
	handler, recoveryStore, options := testHandler(t, "", nil)
	session := createTestSession(t, handler, nil)
	actionPath := "/api/v1/sessions/" + session.Id + "/actions"
	w := request(t, handler, http.MethodPost, actionPath, "application/json", `{"kind":"set-value","expected_revision":0,"row":1,"column":1,"value":4}`, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("action status=%d body=%s", w.Code, w.Body.String())
	}
	var result ActionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Revision != 1 || result.Result.Action != "set-value" {
		t.Fatalf("unexpected action result: %+v", result)
	}
	w = request(t, handler, http.MethodPost, actionPath, "application/json", `{"kind":"clear-value","expected_revision":0,"row":1,"column":1}`, nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("stale revision status=%d body=%s", w.Code, w.Body.String())
	}

	exportPath := "/api/v1/sessions/" + session.Id + "/export"
	w = request(t, handler, http.MethodGet, exportPath, "", "", nil)
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != SessionMediaType {
		t.Fatalf("export status=%d type=%s", w.Code, w.Header().Get("Content-Type"))
	}
	exported := append([]byte(nil), w.Body.Bytes()...)
	w = request(t, handler, http.MethodPost, "/api/v1/sessions/import", SessionMediaType, string(exported), nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("import status=%d body=%s", w.Code, w.Body.String())
	}

	recovered, err := NewRegistry(recoveryStore, options)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(recovered, func(string, string) (game.Game, error) { panic("not called") })
	w = request(t, NewHandler(server, "", nil), http.MethodGet, "/api/v1/sessions/"+session.Id, "", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("recover status=%d body=%s", w.Code, w.Body.String())
	}
	var got Session
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Revision != 1 {
		t.Fatalf("unexpected recovered session: %+v", got)
	}

	w = request(t, handler, http.MethodDelete, "/api/v1/sessions/"+session.Id, "", "", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d", w.Code)
	}
	w = request(t, handler, http.MethodGet, "/api/v1/sessions/"+session.Id, "", "", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("deleted get status=%d", w.Code)
	}
}

func TestSecurityCORSAndRequestValidation(t *testing.T) {
	handler, _, _ := testHandler(t, "secret", []string{"https://client.example"})
	if w := request(t, handler, http.MethodGet, "/healthz", "", "", nil); w.Code != http.StatusOK {
		t.Fatalf("health status=%d", w.Code)
	}
	if w := request(t, handler, http.MethodGet, "/api/v1/sessions", "", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("auth status=%d", w.Code)
	}
	auth := map[string]string{"Authorization": "Bearer secret"}
	badOrigin := map[string]string{"Authorization": "Bearer secret", "Origin": "https://evil.example"}
	if w := request(t, handler, http.MethodGet, "/api/v1/sessions", "", "", badOrigin); w.Code != http.StatusForbidden {
		t.Fatalf("origin status=%d", w.Code)
	}
	preflight := map[string]string{"Origin": "https://client.example", "Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "authorization, content-type"}
	if w := request(t, handler, http.MethodOptions, "/api/v1/sessions", "", "", preflight); w.Code != http.StatusNoContent {
		t.Fatalf("preflight status=%d", w.Code)
	}
	if w := request(t, handler, http.MethodPost, "/api/v1/sessions", "text/plain", `{}`, auth); w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("media status=%d", w.Code)
	}
	if w := request(t, handler, http.MethodPost, "/api/v1/sessions", "application/json", `{"source":{"kind":"puzzle","puzzle":"x","extra":1}}`, auth); w.Code != http.StatusBadRequest {
		t.Fatalf("strict status=%d body=%s", w.Code, w.Body.String())
	}
	large := strings.Repeat("x", MaxRequestBytes+1)
	if w := request(t, handler, http.MethodPost, "/api/v1/sessions", "application/json", large, auth); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large status=%d", w.Code)
	}
}

func TestConcurrentRevisionConflict(t *testing.T) {
	handler, _, _ := testHandler(t, "", nil)
	session := createTestSession(t, handler, nil)
	path := "/api/v1/sessions/" + session.Id + "/actions"
	bodies := []string{`{"kind":"set-value","expected_revision":0,"row":1,"column":1,"value":4}`, `{"kind":"set-value","expected_revision":0,"row":1,"column":2,"value":8}`}
	codes := make(chan int, 2)
	var wg sync.WaitGroup
	for _, body := range bodies {
		wg.Add(1)
		go func(body string) {
			defer wg.Done()
			codes <- request(t, handler, http.MethodPost, path, "application/json", body, nil).Code
		}(body)
	}
	wg.Wait()
	close(codes)
	counts := map[int]int{}
	for code := range codes {
		counts[code]++
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusConflict] != 1 {
		t.Fatalf("statuses: %v", counts)
	}
}

func TestImportRejectsOversizedPayload(t *testing.T) {
	handler, _, _ := testHandler(t, "", nil)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/import", io.NopCloser(bytes.NewReader(make([]byte, MaxRequestBytes+1))))
	r.Header.Set("Content-Type", SessionMediaType)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestAuthenticationRequiresExactBearerCredential(t *testing.T) {
	handler, _, _ := testHandler(t, "secret", nil)
	tests := []struct {
		name   string
		header string
		want   int
	}{
		{name: "missing", want: http.StatusUnauthorized},
		{name: "bare token", header: "secret", want: http.StatusUnauthorized},
		{name: "wrong scheme", header: "Basic secret", want: http.StatusUnauthorized},
		{name: "wrong token", header: "Bearer other", want: http.StatusUnauthorized},
		{name: "valid", header: "Bearer secret", want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.header != "" {
				headers["Authorization"] = tt.header
			}
			w := request(t, handler, http.MethodGet, "/api/v1/sessions", "", "", headers)
			if w.Code != tt.want {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if w.Code == http.StatusUnauthorized && w.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf("missing bearer challenge")
			}
		})
	}
}

func TestCORSRequiresExactOriginAndBoundedPreflight(t *testing.T) {
	handler, _, _ := testHandler(t, "", []string{"https://client.example:8443"})
	allowed := map[string]string{"Origin": "https://client.example:8443", "Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "content-type"}
	if w := request(t, handler, http.MethodOptions, "/api/v1/sessions", "", "", allowed); w.Code != http.StatusNoContent {
		t.Fatalf("allowed preflight status=%d body=%s", w.Code, w.Body.String())
	}
	for name, headers := range map[string]map[string]string{
		"different port": {"Origin": "https://client.example", "Access-Control-Request-Method": "POST"},
		"null origin":    {"Origin": "null", "Access-Control-Request-Method": "POST"},
		"wrong method":   {"Origin": "https://client.example:8443", "Access-Control-Request-Method": "PATCH"},
		"wrong header":   {"Origin": "https://client.example:8443", "Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "x-secret"},
	} {
		t.Run(name, func(t *testing.T) {
			w := request(t, handler, http.MethodOptions, "/api/v1/sessions", "", "", headers)
			if w.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if w.Header().Get("Access-Control-Allow-Origin") == "*" {
				t.Fatal("wildcard origin must never be emitted")
			}
		})
	}
}

func TestMalformedRequestsReturnBoundedStableErrors(t *testing.T) {
	handler, _, _ := testHandler(t, "", nil)
	tests := []struct {
		name, body string
	}{
		{name: "malformed", body: `{`},
		{name: "trailing JSON", body: `{"source":{"kind":"puzzle","puzzle":"` + knownPuzzle + `"}} {}`},
		{name: "missing source", body: `{}`},
		{name: "unknown source", body: `{"source":{"kind":"file","value":"x"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := request(t, handler, http.MethodPost, "/api/v1/sessions", "application/json", tt.body, nil)
			if w.Code != http.StatusBadRequest || w.Body.Len() > 1024 {
				t.Fatalf("status=%d bytes=%d body=%s", w.Code, w.Body.Len(), w.Body.String())
			}
			var envelope Error
			if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil || envelope.Error.Code == "" {
				t.Fatalf("invalid error envelope: %v body=%s", err, w.Body.String())
			}
			if strings.Contains(w.Body.String(), knownPuzzle) {
				t.Fatalf("error leaked request data: %s", w.Body.String())
			}
		})
	}
}

func TestConcurrentSessionsAdvanceIndependently(t *testing.T) {
	handler, _, _ := testHandler(t, "", nil)
	first := createTestSession(t, handler, nil)
	second := createTestSession(t, handler, nil)
	ids := []string{first.Id, second.Id}
	codes := make(chan int, len(ids))
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			codes <- request(t, handler, http.MethodPost, "/api/v1/sessions/"+id+"/actions", "application/json", `{"kind":"set-value","expected_revision":0,"row":1,"column":1,"value":4}`, nil).Code
		}(id)
	}
	wg.Wait()
	close(codes)
	for code := range codes {
		if code != http.StatusOK {
			t.Fatalf("status=%d", code)
		}
	}
	for _, id := range ids {
		w := request(t, handler, http.MethodGet, "/api/v1/sessions/"+id, "", "", nil)
		var session Session
		if err := json.Unmarshal(w.Body.Bytes(), &session); err != nil || session.Revision != 1 {
			t.Fatalf("session %s did not advance independently: %+v err=%v", id, session, err)
		}
	}
}

type completionRecorder struct {
	calls int
	err   error
}

func (recorder *completionRecorder) RecordCompletion(string) (bool, error) {
	recorder.calls++
	return true, recorder.err
}

func TestTrackedServerRecordsPlayerCompletion(t *testing.T) {
	store := solver.NewStore()
	options := game.NewDefaultOptions(store)
	recoveryStore := recovery.NewStore(filepath.Join(t.TempDir(), "recovery"))
	registry, err := NewRegistry(recoveryStore, options)
	if err != nil {
		t.Fatal(err)
	}
	recorder := &completionRecorder{err: errors.New("database locked")}
	factory := func(_, _ string) (game.Game, *playrun.Tracker, error) {
		board := core.NewEmptyBoard()
		board.FromString(".23456789456789123789123456214365897365897214897214365531642978642978531978531642")
		current := game.NewGame(board, options)
		return current, playrun.New("normalized", recorder), nil
	}
	handler := NewHandler(NewTrackedServer(registry, factory), "", nil)
	session := createTestSession(t, handler, nil)
	path := "/api/v1/sessions/" + session.Id + "/actions"
	response := request(t, handler, http.MethodPost, path, "application/json", `{"kind":"set-value","expected_revision":0,"row":1,"column":1,"value":1}`, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("completion status=%d body=%s", response.Code, response.Body.String())
	}
	var actionResponse ActionResponse
	if err := json.Unmarshal(response.Body.Bytes(), &actionResponse); err != nil {
		t.Fatal(err)
	}
	if actionResponse.Warnings == nil || len(*actionResponse.Warnings) != 1 || !strings.Contains((*actionResponse.Warnings)[0], "database locked") {
		t.Fatalf("warnings = %#v", actionResponse.Warnings)
	}
	if recorder.calls != 1 {
		t.Fatalf("completion calls = %d, want 1", recorder.calls)
	}
}
