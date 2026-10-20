package webapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestEveryBodyOperationUsesSharedIngressPolicy(t *testing.T) {
	handler := NewHandler(NewServer(), nil)
	operations := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/guest/games"},
		{http.MethodPost, "/api/v1/guest/games/actions"},
		{http.MethodPost, "/api/v1/guest/games/hint"},
		{http.MethodPost, "/api/v1/account/games"},
		{http.MethodPost, "/api/v1/account/games/claim"},
		{http.MethodPut, "/api/v1/account/games/example/presentation"},
		{http.MethodPost, "/api/v1/account/games/example/actions"},
	}

	for _, operation := range operations {
		name := operation.method + " " + operation.path
		t.Run(name+" content type", func(t *testing.T) {
			record := request(t, handler, operation.method, operation.path, "text/plain", `{}`, nil)
			if record.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("status=%d body=%s", record.Code, record.Body.String())
			}
		})
		t.Run(name+" body limit", func(t *testing.T) {
			body := strings.Repeat(" ", MaxRequestBytes+1)
			record := request(t, handler, operation.method, operation.path, "application/json", body, nil)
			if record.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status=%d body=%s", record.Code, record.Body.String())
			}
		})
	}
}
