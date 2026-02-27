package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/rennaisance-jomt/axon/internal/config"
)

// Pool manages browser instances
type Pool struct {
	cfg        *config.BrowserConfig
	mu         sync.RWMutex
	available  chan *rod.Browser
	inUse      map[*rod.Browser]bool
	browsers   []*rod.Browser
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewPool creates a new browser pool
func NewPool(cfg *config.BrowserConfig) (*Pool, error) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &Pool{
		cfg:       cfg,
		available: make(chan *rod.Browser, cfg.PoolSize),
		inUse:     make(map[*rod.Browser]bool),
		ctx:       ctx,
	}

	// Pre-launch browsers
	for i := 0; i < cfg.PoolSize; i++ {
		browser, err := pool.launchBrowser()
		if err != nil {
			return nil, fmt.Errorf("failed to launch browser %d: %w", i, err)
		}
		pool.browsers = append(pool.browsers, browser)
		pool.available <- browser
	}

	return pool, nil
}

func (p *Pool) launchBrowser() (*rod.Browser, error) {
	// Set up launcher
	launchOpts := []launcher.Option{
		launcher.NoSandbox(p.cfg.LaunchOptions["--no-sandbox"] == true),
	}

	// Use custom binary if specified
	if p.cfg.BinaryPath != "" {
		launchOpts = append(launchOpts, launcher.Bin(p.cfg.BinaryPath))
	}

	// Create browser
	u := launcher.New().
		Set(p.cfg.LaunchOptions).
		MustLaunch()

	browser := rod.New().Client(rod.NewClient(u))

	// Set timeouts
	browser = browser.DefaultTimeout(30 * time.Second)

	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	return browser, nil
}

// Acquire gets a browser from the pool
func (p *Pool) Acquire() (*rod.Browser, error) {
	select {
	case browser := <-p.available:
		p.mu.Lock()
		p.inUse[browser] = true
		p.mu.Unlock()
		return browser, nil
	case <-p.ctx.Done():
		return nil, fmt.Errorf("pool is closed")
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for browser")
	}
}

// Release returns a browser to the pool
func (p *Pool) Release(browser *rod.Browser) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.inUse[browser]; ok {
		delete(p.inUse, browser)
		select {
		case p.available <- browser:
		default:
			// Pool is full, close the browser
			browser.Close()
		}
	}
}

// Close closes all browsers in the pool
func (p *Pool) Close() error {
	p.cancel()
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for _, browser := range p.browsers {
		if err := browser.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing browsers: %v", errs)
	}
	return nil
}

// Stats returns pool statistics
func (p *Pool) Stats() (active, idle int) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.inUse), len(p.available)
}
