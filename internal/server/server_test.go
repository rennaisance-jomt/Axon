package server

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rennaisance-jomt/axon/internal/config"
)

func TestServer_Health(t *testing.T) {
	// Skip if running in restricted environment (e.g. CI without Chrome)
	if os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("Skipping server test in CI")
	}

	cfg := config.DefaultConfig()
	cfg.Browser.Headless = true
	cfg.Browser.PoolSize = 1 
	
	dir, err := os.MkdirTemp("", "axon-db")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	cfg.Storage.Path = dir

	server := New(cfg)
	
	// Create fibers without starting true HTTP daemon immediately 
	// Start() actually spins up Badger and Rod, then we can test app instance.
	// Since Start() runs app.Listen() in a blocking way normally if not handled, 
	// we will just test the handlers manually or mock out part of the start.
	// We'll mimic setupRoutes manually for health.

	// In real env, Start() blocks. Let's just create Fiber and setup routes manually for unit test
	// to avoid blocking or heavy Rod startup for a simple health check.
	server.app = fiberSetupForTest(server)
	defer server.Stop()

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := server.app.Test(req)
	
	if err != nil {
		t.Fatalf("Error making health request: %v", err)
	}
	
	if resp.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
	
	body, _ := io.ReadAll(resp.Body)
	var health map[string]string
	json.Unmarshal(body, &health)
	
	if health["status"] != "ok" {
		t.Fatalf("Expected health status ok, got %s", health["status"])
	}
}

func fiberSetupForTest(s *Server) *fiber.App {
	app := fiber.New()
	s.app = app
	// just register health 
	app.Get("/health", s.handleHealth)
	return app
}
