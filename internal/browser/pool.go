package browser

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/rennaisance-jomt/axon/internal/config"
)

// Pool manages browser instances
type Pool struct {
	cfg         *config.BrowserConfig
	mu          sync.RWMutex
	rootBrowser *rod.Browser
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewPool creates a new browser pool
func NewPool(cfg *config.BrowserConfig) (*Pool, error) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &Pool{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	// Launch single root daemon
	browser, err := pool.launchBrowser()
	if err != nil {
		return nil, fmt.Errorf("failed to launch root browser: %w", err)
	}
	pool.rootBrowser = browser

	return pool, nil
}

func (p *Pool) launchBrowser() (*rod.Browser, error) {
	// Set up launcher
	l := launcher.New().Leakless(false)
	
	if p.cfg.LaunchOptions["--no-sandbox"] == true {
		l.NoSandbox(true)
	}

	// Use custom binary if specified
	if p.cfg.BinaryPath != "" {
		l.Bin(p.cfg.BinaryPath)
	}

	// Create browser
	u := l.MustLaunch()

	browser := rod.New().ControlURL(u)

	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	return browser, nil
}

// Acquire returns the single root browser daemon
func (p *Pool) Acquire() (*rod.Browser, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	select {
	case <-p.ctx.Done():
		return nil, fmt.Errorf("pool is closed")
	default:
	}

	if p.rootBrowser == nil {
		return nil, fmt.Errorf("root browser not initialized")
	}

	return p.rootBrowser, nil
}

// Release is a no-op as session isolation is now handled via Incognito contexts
func (p *Pool) Release(browser *rod.Browser) {
	// No-op
}

// Close closes the single root browser daemon
func (p *Pool) Close() error {
	p.cancel()
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.rootBrowser != nil {
		err := p.rootBrowser.Close()
		p.rootBrowser = nil
		return err
	}
	return nil
}

// Stats returns pool statistics (always 1 active daemon)
func (p *Pool) Stats() (active, idle int) {
	return 1, 0
}
