// Package api implements the HTTP API server for agrev.
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Server is the agrev HTTP API server.
type Server struct {
	addr   string
	mux    *http.ServeMux
	server *http.Server
}

// New creates a new API server.
func New(addr string) *Server {
	s := &Server{addr: addr}
	s.mux = http.NewServeMux()
	s.registerRoutes()
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("POST /api/analyze", s.handleAnalyze)
	s.mux.HandleFunc("POST /api/parse", s.handleParse)
	s.mux.HandleFunc("POST /api/summary", s.handleSummary)
	s.mux.HandleFunc("GET /api/ws", s.handleWebSocket)
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	log.Printf("agrev API server listening on %s", s.addr)
	return s.server.ListenAndServe()
}

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// readJSON decodes a JSON request body into v.
func readJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("empty request body")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}
