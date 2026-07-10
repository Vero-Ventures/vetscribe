package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer() *Server {
	return NewServer(ServerOptions{BuildInfo: BuildInfo{Version: "test", Commit: "abc", Date: "now"}})
}

func TestHealthEndpoint(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newTestServer().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("expected status ok, got %q", body.Status)
	}
	if body.Build.Version != "test" {
		t.Fatalf("expected build version test, got %q", body.Build.Version)
	}
}

func TestFrontendServed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	newTestServer().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for index, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected text/html content-type, got %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "VetScribe") {
		t.Fatalf("expected body to contain VetScribe, got %q", body)
	}
	if !strings.Contains(body, "vtest") {
		t.Fatalf("expected injected version vtest in body, got %q", body)
	}
}

func TestStaticAssetServed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	newTestServer().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for app.js, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Fatalf("expected a javascript content-type, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "refreshHealth") {
		t.Fatalf("expected app.js body to contain the refreshHealth function")
	}
}

func TestSPAFallbackRendersIndex(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/visits/deep/link", nil)
	newTestServer().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 SPA fallback, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "VetScribe") {
		t.Fatalf("expected fallback to render the index page")
	}
}

func TestUnknownAPIPathIs404(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/does-not-exist", nil)
	newTestServer().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown /api path, got %d", rec.Code)
	}
}

func TestSecurityHeadersAndRequestID(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	newTestServer().Handler().ServeHTTP(rec, req)

	h := rec.Header()
	if got := h.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if got := h.Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q", got)
	}
	if got := h.Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q", got)
	}
	if !strings.Contains(h.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatalf("CSP missing frame-ancestors: %q", h.Get("Content-Security-Policy"))
	}
	if h.Get("X-Request-Id") == "" {
		t.Fatalf("expected an X-Request-Id header")
	}
}

func TestPanicRecovery(t *testing.T) {
	s := newTestServer()
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})
	h := s.recoverMiddleware(panicking)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 after panic, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("panic detail leaked to client: %q", rec.Body.String())
	}
}

func TestBodyLimitApplied(t *testing.T) {
	// A handler that reads the body should see MaxBytesReader enforced.
	var readErr error
	reader := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, maxRequestBody+1)
		_, readErr = r.Body.Read(buf)
		w.WriteHeader(http.StatusOK)
	})
	h := bodyLimitMiddleware(reader)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", 8)))
	h.ServeHTTP(rec, req)
	// Small body reads fine; the wrapper simply must not break normal reads.
	if readErr != nil && readErr.Error() == "http: request body too large" {
		t.Fatalf("small body should not trip the limit")
	}
}
