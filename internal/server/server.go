package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rennaisance-jomt/axon/internal/browser"
	"github.com/rennaisance-jomt/axon/internal/config"
	"github.com/rennaisance-jomt/axon/internal/storage"
)

// Server represents the Axon HTTP server
type Server struct {
	app      *fiber.App
	cfg      *config.Config
	handlers *Handlers
	pool     *browser.Pool
	db       *storage.DB
	mu       sync.RWMutex
	start    time.Time
}

// New creates a new Server instance
func New(cfg *config.Config) *Server {
	return &Server{
		cfg:   cfg,
		start: time.Now(),
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Initialize browser pool
	pool, err := browser.NewPool(&s.cfg.Browser)
	if err != nil {
		return fmt.Errorf("failed to initialize browser pool: %w", err)
	}
	s.pool = pool

	// Initialize storage
	db, err := storage.New(s.cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	s.db = db

	// Initialize handlers
	s.handlers = NewHandlers(pool, db, s.cfg)

	// Create Fiber app
	s.app = fiber.New(fiber.Config{
		ReadTimeout:  s.cfg.Server.ReadTimeout,
		WriteTimeout: s.cfg.Server.WriteTimeout,
		AppName:      "Axon",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		},
	})

	// Middleware
	s.app.Use(recover.New())
	s.app.Use(logger.New())
	s.app.Use(cors.New())

	// routes
	s.setupRoutes()

	// Start listener
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	return s.app.Listen(addr)
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	if s.pool != nil {
		s.pool.Close()
	}
	if s.app != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.app.ShutdownWithContext(ctx)
	}
	return nil
}

// UpTime returns server uptime
func (s *Server) UpTime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.start)
}

func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/health", s.handleHealth)

	// API v1
	api := s.app.Group("/api/v1")

	// Sessions
	sessions := api.Group("/sessions")
	sessions.Get("", s.handlers.handleListSessions)
	sessions.Post("", s.handlers.handleCreateSession)
	sessions.Get("/:id", s.handlers.handleGetSession)
	sessions.Delete("/:id", s.handlers.handleDeleteSession)

	// Session actions
	sessions.Post("/:id/navigate", s.handlers.handleNavigate)
	sessions.Post("/:id/snapshot", s.handlers.handleSnapshot)
	sessions.Post("/:id/act", s.handlers.handleAct)
	sessions.Get("/:id/status", s.handlers.handleStatus)
	sessions.Post("/:id/screenshot", s.handlers.handleScreenshot)
	sessions.Post("/:id/resize", s.handlers.handleResize)
	sessions.Post("/:id/wait", s.handlers.handleWait)
	sessions.Get("/:id/cookies", s.handlers.handleGetCookies)
	sessions.Post("/:id/cookies", s.handlers.handleSetCookies)

	// Audit
	api.Get("/audit", s.handlers.handleAudit)
}

// Health check handler
func (s *Server) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"version": "1.0.0",
		"uptime":  s.UpTime().String(),
	})
}

