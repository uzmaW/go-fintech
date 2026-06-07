package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

type Server struct {
	server   *http.Server
	ready    atomic.Bool
	healthy  atomic.Bool
	started  time.Time
	checkers []Checker
}

type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

type status struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Uptime    string            `json:"uptime"`
	Ready     bool              `json:"ready"`
	Checks    map[string]string `json:"checks,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

func New(serviceName, addr string, checkers ...Checker) *Server {
	s := &Server{
		started:  time.Now(),
		checkers: checkers,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleLiveness)
	mux.HandleFunc("/readyz", s.handleReadiness)
	mux.HandleFunc("/livez", s.handleLiveness)

	s.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.healthy.Store(true)

	return s
}

func (s *Server) SetReady(v bool) {
	s.ready.Store(v)
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.healthy.Store(false)
	s.ready.Store(false)
	return s.server.Shutdown(ctx)
}

func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !s.healthy.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(status{Status: "DOWN"})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status{Status: "UP", Timestamp: time.Now().UTC()})
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !s.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(status{Status: "NOT_READY", Ready: false, Timestamp: time.Now().UTC()})
		return
	}

	checks := make(map[string]string)
	allOK := true
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	for _, c := range s.checkers {
		if err := c.Check(ctx); err != nil {
			checks[c.Name()] = "FAIL: " + err.Error()
			allOK = false
		} else {
			checks[c.Name()] = "OK"
		}
	}

	resp := status{
		Service:   "fintech",
		Uptime:    time.Since(s.started).Truncate(time.Second).String(),
		Ready:     allOK,
		Checks:    checks,
		Timestamp: time.Now().UTC(),
	}

	if !allOK {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(resp)
}
