package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

func runServerMode(cfg *config.Config) {
	// Create and start server
	srv := server.New(cfg)

	// Handle shutdown gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println()
		logger.Warn("Shutdown signal received")
		logger.System("Stopping server services...")
		if err := srv.Stop(); err != nil {
			logger.Error("Error during shutdown: %v", err)
		}
		logger.Success("Axon stopped gracefully")
		os.Exit(0)
	}()

	if err := srv.Start(); err != nil {
		logger.Error("Critical server error: %v", err)
		os.Exit(1)
	}
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
