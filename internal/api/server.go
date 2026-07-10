// Package api implements the VetScribe HTTP server: the JSON API and the
// server-rendered frontend, served from one static binary. The HTML page is
// rendered by html/template and the vanilla JS/CSS assets are embedded via
// embed.FS — there is no separate frontend build step or toolchain.
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// maxRequestBody caps the size of any request body the server will read. M1's
// upload endpoint will make this configurable; the default guards every route.
const maxRequestBody = 32 << 20 // 32 MiB

// BuildInfo is compile-time metadata surfaced on the health endpoint and the page.
type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// ServerOptions configures a Server.
type ServerOptions struct {
	BuildInfo BuildInfo
	Logger    *slog.Logger
}

// Server holds handler dependencies.
type Server struct {
	build  BuildInfo
	logger *slog.Logger
	tmpl   *template.Template
}

// NewServer constructs a Server. It parses the embedded templates once; a parse
// failure is a programming error (bad embed glob) and panics at startup.
func NewServer(opts ServerOptions) *Server {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	tmpl, err := template.ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		panic(fmt.Sprintf("api: parse embedded templates: %v", err))
	}
	return &Server{build: opts.BuildInfo, logger: logger, tmpl: tmpl}
}

// Handler returns the fully wired HTTP handler with the middleware chain applied.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)

	sub, err := fs.Sub(staticAssets, "assets")
	if err != nil {
		// Impossible with a valid embed glob; fail loudly rather than silently 404.
		panic(fmt.Sprintf("api: sub-fs for assets: %v", err))
	}
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(sub)))

	// Catch-all: renders the index page and provides the SPA/deep-link fallback.
	mux.HandleFunc("GET /", s.handleIndex)

	return s.recoverMiddleware(
		s.logMiddleware(
			s.securityHeaders(
				bodyLimitMiddleware(mux),
			),
		),
	)
}

type healthResponse struct {
	Status string    `json:"status"`
	Build  BuildInfo `json:"build"`
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Build: s.build})
}

type indexData struct {
	Build BuildInfo
}

// handleIndex renders the server-side page. Unmatched non-/api GET paths fall
// through here (SPA deep-link fallback); unknown /api paths return 404.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "index.html.tmpl", indexData{Build: s.build}); err != nil {
		s.logger.Error("render index", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// --- middleware ----------------------------------------------------------------

// bodyLimitMiddleware caps every request body via http.MaxBytesReader.
func bodyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		}
		next.ServeHTTP(w, r)
	})
}

// securityHeaders sets conservative, framework-free security response headers.
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self'")
		next.ServeHTTP(w, r)
	})
}

// statusRecorder captures the status code for access logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// logMiddleware emits one structured access log per request, tagged with a
// generated request id that is also returned to the client.
func (s *Server) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := newRequestID()
		w.Header().Set("X-Request-Id", reqID)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		s.logger.Info(
			"request",
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

// recoverMiddleware turns a panic into a 500 without leaking internals.
func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error(
					"panic recovered",
					"error", fmt.Sprint(rec),
					"method", r.Method,
					"path", r.URL.Path,
				)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is fatal-adjacent; fall back to a time-based id.
		return fmt.Sprintf("t%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
