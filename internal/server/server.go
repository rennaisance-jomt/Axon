package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rennaisance-jomt/axon/internal/config"
	"github.com/rennaisance-jomt/axon/internal/storage"
)

// Server represents the Axon HTTP server
type Server struct {
	app    *fiber.App
	cfg    *config.Config
	mu     sync.RWMutex
	start  time.Time
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
	// Create Fiber app
	s.app = fiber.New(fiber.Config{
		ReadTimeout:  s.cfg.Server.ReadTimeout,
		WriteTimeout: s.cfg.Server.WriteTimeout,
		AppName:      "Axon",
	})

	// Middleware
	s.app.Use(recover.New())
	s.app.Use(logger.New())
	s.app.Use(cors.New())

	// Initialize storage
	db, err := storage.New(s.cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	s.app.Storage().Set("db", db)

	// Routes
	s.setupRoutes()

	// Start listener
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	
	// Check if port is available
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	ln.Close()

	return s.app.Listener(ln)
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
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
	sessions.Get("/", s.handleListSessions)
	sessions.Post("/", s.handleCreateSession)
	sessions.Get("/:id", s.handleGetSession)
	sessions.Delete("/:id", s.handleDeleteSession)

	// Session actions
	sessions.Post("/:id/navigate", s.handleNavigate)
	sessions.Post("/:id/snapshot", s.handleSnapshot)
	sessions.Post("/:id/act", s.handleAct)
	sessions.Get("/:id/status", s.handleStatus)
	sessions.Post("/:id/screenshot", s.handleScreenshot)
	sessions.Post("/:id/wait", s.handleWait)
	sessions.Get("/:id/cookies", s.handleGetCookies)
	sessions.Post("/:id/cookies", s.handleSetCookies)

	// Audit
	api.Get("/audit", s.handleAudit)
}

// Health check handler
func (s *Server) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"version": "1.0.0",
		"uptime":  s.UpTime().String(),
	})
}

// Session handlers
func (s *Server) handleListSessions(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{"sessions": []interface{}{}})
}

func (s *Server) handleCreateSession(c *fiber.Ctx) error {
	// TODO: Implement
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"session_id": "example",
		"status":     "created",
	})
}

func (s *Server) handleGetSession(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{
		"session_id": c.Params("id"),
		"status":     "active",
	})
}

func (s *Server) handleDeleteSession(c *fiber.Ctx) error {
	// TODO: Implement
	return c.SendStatus(http.StatusNoContent)
}

func (s *Server) handleNavigate(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{
		"success": true,
		"url":     "https://example.com",
	})
}

func (s *Server) handleSnapshot(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{
		"content": "PAGE: example.com\n\n[e1] Example (link)",
	})
}

func (s *Server) handleAct(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{
		"success": true,
		"result":  "Clicked element",
	})
}

func (s *Server) handleStatus(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{
		"url":        "https://example.com",
		"auth_state": "unknown",
	})
}

func (s *Server) handleScreenshot(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{
		"path": "/screenshots/screenshot.png",
	})
}

func (s *Server) handleWait(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{
		"success": true,
		"matched": false,
	})
}

func (s *Server) handleGetCookies(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{"cookies": []interface{}{}})
}

func (s *Server) handleSetCookies(c *fiber.Ctx) error {
	// TODO: Implement
	return c.SendStatus(http.StatusNoContent)
}

func (s *Server) handleAudit(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{"logs": []interface{}{}})
}
