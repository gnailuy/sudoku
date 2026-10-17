package webapi

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/gnailuy/sudoku/game"
)

const knownPuzzle = "..3.2.6..9..3.5..1..18.64....81.29..7.......8..67.82....26.95..8..2.3..9..5.1.3.."

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

func TestCleanWebHandlerRemovesLegacySessionNamespace(t *testing.T) {
	handler := NewHandler(NewServer(), nil)
	for _, path := range []string{"/api/v1/sessions", "/api/v1/sessions/legacy", "/api/v1/sessions/import"} {
		record := httptest.NewRecorder()
		handler.ServeHTTP(record, httptest.NewRequest(http.MethodGet, path, nil))
		if record.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d body=%s", path, record.Code, record.Body.String())
		}
	}
}

func TestSetNotesTranslatesWithoutChangingCandidateSemantics(t *testing.T) {
	var request ActionRequest
	if err := request.FromSetNotesAction(SetNotesAction{
		ExpectedRevision: 7,
		Row:              2,
		Column:           3,
		Values:           []int{1, 4, 9},
	}); err != nil {
		t.Fatal(err)
	}
	action, revision, err := translateAction(request)
	if err != nil {
		t.Fatal(err)
	}
	notes, ok := action.(game.SetNotes)
	if !ok || revision != 7 || notes.Position.Row != 1 || notes.Position.Column != 2 || !slices.Equal(notes.Values, []int{1, 4, 9}) {
		t.Fatalf("translated action = %#v revision=%d", action, revision)
	}
}

func TestHealthRemainsAvailable(t *testing.T) {
	record := httptest.NewRecorder()
	NewHandler(NewServer(), nil).ServeHTTP(record, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if record.Code != http.StatusOK || record.Body.String() != "{\"status\":\"healthy\"}\n" {
		t.Fatalf("health status=%d body=%s", record.Code, record.Body.String())
	}
}

func TestConfiguredOriginIsExact(t *testing.T) {
	handler := NewHandler(NewServer(), []string{"https://client.example"})
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Origin", "https://denied.example")
	record := httptest.NewRecorder()
	handler.ServeHTTP(record, request)
	if record.Code != http.StatusForbidden {
		t.Fatalf("denied origin status=%d", record.Code)
	}
}
