package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/rennaisance-jomt/axon/internal/browser"
	"github.com/rennaisance-jomt/axon/internal/config"
	"github.com/rennaisance-jomt/axon/internal/mcp"
	"github.com/rennaisance-jomt/axon/internal/server"
	"github.com/rennaisance-jomt/axon/internal/storage"
	"github.com/rennaisance-jomt/axon/pkg/logger"
)

var (
	version = "dev"
)

func main() {
	var (
		mcpMode     = flag.Bool("mcp", false, "Run in MCP (Model Context Protocol) mode")
		versionFlag = flag.Bool("version", false, "Print version and exit")
	)

	// Custom flag parsing to support subcommands
	if len(os.Args) > 1 && os.Args[1] == "run" {
		// Run UI mode
		cfg, err := config.Load()
		if err != nil {
			logger.Error("Failed to load config: %v", err)
			os.Exit(1)
		}
		runUIMode(cfg)
		return
	}

	flag.Parse()

	if *versionFlag {
		fmt.Printf("Axon %s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	if *mcpMode {
		runMCPMode(cfg)
	} else {
		runServerMode(cfg)
	}
}

func runUIMode(cfg *config.Config) {
	logger.Banner()
	logger.System("Starting Axon UI Mode (Dashboard & Server)...")

	// Create and start server
	srv := server.New(cfg)

	// Handle shutdown gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Wait 1 second for the server to bind port, then open HTML
	go func() {
		time.Sleep(1 * time.Second)
		cwd, _ := os.Getwd()
		htmlPath := filepath.Join(cwd, "dashboard_pure.html")
		
		if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
			logger.Error("Could not find dashboard_pure.html in current directory.")
			return
		}
		
		logger.System("Opening Dashboard UI directly in default browser...")
		openBrowser("file://" + htmlPath)
	}()

	// Wait for signal or error
	select {
	case <-sigCh:
		fmt.Println()
		logger.Warn("Shutdown signal received")
	case err := <-errCh:
		if err != nil {
			logger.Error("Critical server error: %v", err)
		}
	}

	logger.System("Stopping server services...")
	if err := srv.Stop(); err != nil {
		logger.Error("Error during shutdown: %v", err)
	}
	logger.Success("Axon UI stopped gracefully")
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

func runServerMode(cfg *config.Config) {
	// Create and start server
	srv := server.New(cfg)

	// Handle shutdown gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Wait for signal or error
	select {
	case <-sigCh:
		fmt.Println()
		logger.Warn("Shutdown signal received")
	case err := <-errCh:
		if err != nil {
			logger.Error("Critical server error: %v", err)
		}
	}

	logger.System("Stopping server services...")
	if err := srv.Stop(); err != nil {
		logger.Error("Error during shutdown: %v", err)
	}
	logger.Success("Axon stopped gracefully")
}

func runMCPMode(cfg *config.Config) {
	logger.Banner()
	logger.System("Starting Axon in MCP mode...")

	// Initialize dependencies
	pool, err := browser.NewPool(&cfg.Browser)
	if err != nil {
		logger.Error("Failed to initialize browser pool: %v", err)
		os.Exit(1)
	}
	defer pool.Close()

	db, err := storage.New(cfg.Storage.Path)
	if err != nil {
		logger.Error("Failed to initialize storage: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create and run MCP server
	mcpServer := mcp.NewMCPServer(pool, db, cfg)
	
	if err := mcpServer.Run(); err != nil {
		logger.Error("MCP server encountered a fatal error: %v", err)
		os.Exit(1)
	}
}

