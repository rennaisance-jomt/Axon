package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"github.com/rennaisance-jomt/axon/internal/browser"
	"github.com/rennaisance-jomt/axon/internal/config"
	"github.com/rennaisance-jomt/axon/internal/middleware"
	"github.com/rennaisance-jomt/axon/internal/storage"
	"github.com/rennaisance-jomt/axon/internal/telemetry"
	"github.com/rennaisance-jomt/axon/pkg/logger"
)

// Server represents the Axon HTTP server
type Server struct {
	app      *fiber.App
	cfg      *config.Config
	handlers *Handlers
	pool     *browser.Pool
	db       *storage.DB
	stats    *StatsCollector
	dashboard *DashboardHandler
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
func (s *Server) Start() (err error) {
	defer func() {
		if err != nil {
			logger.Warn("Server start failed, cleaning up...")
			_ = s.Stop()
		}
	}()

	logger.Banner()

	// Initialize telemetry if enabled
	if s.cfg.Telemetry.Enabled {
		logger.System("Initializing telemetry: %s", s.cfg.Telemetry.Provider)
		if err := telemetry.Init(&s.cfg.Telemetry); err != nil {
			logger.Warn("Telemetry initialization failed: %v", err)
		} else {
			logger.Success("Telemetry enabled: %s", s.cfg.Telemetry.Provider)
		}
	}

	// Initialize browser pool
	logger.System("Initializing browser pool (PoolSize: %d)", s.cfg.Browser.PoolSize)
	pool, err := browser.NewPool(&s.cfg.Browser)
	if err != nil {
		logger.Error("Failed to initialize browser pool: %v", err)
		return fmt.Errorf("failed to initialize browser pool: %w", err)
	}
	s.pool = pool
	logger.Success("Browser pool ready")

	// Initialize storage
	logger.System("Connecting to storage: %s", s.cfg.Storage.Path)
	db, err := storage.New(s.cfg.Storage.Path)
	if err != nil {
		logger.Error("Failed to initialize storage: %v", err)
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	s.db = db
	logger.Success("Storage connection established")

	// Initialize handlers
	s.handlers = NewHandlers(pool, db, s.cfg)

	// Initialize stats collector and dashboard
	s.stats = NewStatsCollector()
	s.dashboard = NewDashboardHandler(s.stats)
	s.handlers.SetStatsCollector(s.stats)

	// Create Fiber app
	s.app = fiber.New(fiber.Config{
		ReadTimeout:           s.cfg.Server.ReadTimeout,
		WriteTimeout:          s.cfg.Server.WriteTimeout,
		BodyLimit:             10 * 1024 * 1024, // 10MB for large snapshots
		AppName:               "Axon",
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		},
	})

	// Middleware
	s.app.Use(recover.New())
	// s.app.Use(logger.New()) // Replacing noisy fiber logger with custom targeted logs
	s.app.Use(cors.New())
	s.app.Use(middleware.RetryMiddleware(middleware.DefaultRetryConfig()))

	// routes
	s.setupRoutes()

	// Start listener
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	logger.Success("Axon server listening on %s", addr)
	logger.Info("Press Ctrl+C to stop the server")
	return s.app.Listen(addr)
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Shutdown telemetry
	telemetry.Shutdown()

	if s.pool != nil {
		s.pool.Close()
		s.pool = nil
	}
	if s.app != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := s.app.ShutdownWithContext(ctx)
		s.app = nil
		return err
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

	// Dashboard routes
	s.dashboard.RegisterRoutes(s.app)

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
	
	// Telemetry bridging
	sessions.Post("/:id/telemetry/llm", s.handlers.handleLLMTelemetry)
	
	// Sprint 27: Vision Overlay API (WebSocket Stream)
	sessions.Get("/:id/stream", websocket.New(s.handlers.handleStream))
	sessions.Get("/:id/replay", s.handlers.handleReplay)

	// Phase 2: Intent-based resolution
	sessions.Post("/:id/find_and_act", s.handlers.handleFindAndAct)

	// Agent integration (Backend LLM call)
	api.Post("/agent/chat", s.handlers.handleAgentChat)

	// Tabs (Multi-tasking)
	sessions.Get("/:id/tabs", s.handlers.handleListTabs)
	sessions.Post("/:id/tabs", s.handlers.handleCreateTab)
	sessions.Post("/:id/tabs/:target_id/activate", s.handlers.handleActivateTab)
	sessions.Delete("/:id/tabs/:target_id", s.handlers.handleCloseTab)

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

