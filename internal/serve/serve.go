// Package serve provides an HTTP server wrapping the bosun lifecycle,
// enabling containerized operation with health checks and status queries.
package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/chronick/bosun/internal/config"
	"github.com/chronick/bosun/internal/lifecycle"
)

// Status represents the current state of the bosun serve instance.
type Status struct {
	State     string    `json:"state"`      // "idle", "running", "stopped"
	TaskID    string    `json:"task_id"`    // current task (empty if idle)
	AgentName string    `json:"agent_name"`
	StartedAt time.Time `json:"started_at"`
	Uptime    string    `json:"uptime"`
}

// Server is the bosun HTTP server wrapping the lifecycle loop.
type Server struct {
	cfg    *config.Config
	server *http.Server

	mu       sync.Mutex
	loopCtx  context.Context
	loopStop context.CancelFunc
	running  bool
	started  time.Time
}

// New creates a serve.Server that auto-starts the lifecycle loop.
func New(cfg *config.Config) *Server {
	s := &Server{
		cfg:     cfg,
		started: time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("POST /start", s.handleStart)
	mux.HandleFunc("POST /stop", s.handleStop)

	s.server = &http.Server{
		Addr:    cfg.ServeAddr,
		Handler: mux,
	}

	return s
}

// Run starts the HTTP server and lifecycle loop, blocking until shutdown.
func (s *Server) Run(ctx context.Context) error {
	// Start lifecycle loop automatically
	s.startLoop()

	// Start HTTP server
	ln, err := net.Listen("tcp", s.cfg.ServeAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	log.Printf("bosun serve: listening on %s", s.cfg.ServeAddr)

	go s.server.Serve(ln)

	// Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigCh:
		log.Printf("bosun serve: received %s, shutting down", sig)
	case <-ctx.Done():
		log.Println("bosun serve: context cancelled")
	}

	// Stop lifecycle loop
	s.stopLoop()

	// Shutdown HTTP server
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.server.Shutdown(shutCtx)
}

func (s *Server) startLoop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.loopCtx = ctx
	s.loopStop = cancel
	s.running = true

	go func() {
		err := lifecycle.Run(s.cfg)
		if err != nil {
			log.Printf("bosun serve: lifecycle exited: %v", err)
		}

		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
}

func (s *Server) stopLoop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.loopStop == nil {
		return
	}

	s.loopStop()
	s.running = false
}

// --- HTTP handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()

	status := "ok"
	if !running {
		status = "degraded"
	}
	json.NewEncoder(w).Encode(map[string]string{"status": status})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()

	state := "stopped"
	if running {
		state = "running"
	}

	st := Status{
		State:     state,
		AgentName: s.cfg.AgentName,
		StartedAt: s.started,
		Uptime:    time.Since(s.started).Truncate(time.Second).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(st)
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()

	if running {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "already_running"})
		return
	}

	s.startLoop()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()

	if !running {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "already_stopped"})
		return
	}

	s.stopLoop()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}
