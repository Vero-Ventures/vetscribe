// Package api implements the VetScribe HTTP server: the JSON API and the embedded
// frontend, served from one binary.
package api

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
)

// BuildInfo is compile-time metadata surfaced on the health endpoint.
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
}

// NewServer constructs a Server.
func NewServer(opts ServerOptions) *Server {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{build: opts.BuildInfo, logger: logger}
}

// Handler returns the fully wired HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.Handle("GET /", s.frontendHandler())
	return mux
}

type healthResponse struct {
	Status string    `json:"status"`
	Build  BuildInfo `json:"build"`
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Build: s.build})
}

// placeholderIndex is served when no built frontend is embedded (e.g. a fresh
// checkout before `pnpm --dir web build`). The real UI replaces it in CI and after
// a local frontend build.
const placeholderIndex = `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>VetScribe</title></head>` +
	`<body><h1>VetScribe</h1><p>Frontend build not present. Run the frontend build to serve the app.</p></body></html>`

func (s *Server) frontendHandler() http.Handler {
	sub, err := fs.Sub(frontendAssets, frontendRoot)
	if err != nil {
		s.logger.Error("frontend assets unavailable", "error", err)
		return s.placeholderHandler()
	}
	if _, statErr := fs.Stat(sub, "index.html"); statErr != nil {
		return s.placeholderHandler()
	}
	return http.FileServerFS(sub)
}

func (s *Server) placeholderHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(placeholderIndex))
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
