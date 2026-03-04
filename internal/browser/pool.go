package browser

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/rennaisance-jomt/axon/internal/config"
	"github.com/rennaisance-jomt/axon/pkg/logger"
)

// ContextStatus represents the health status of a browser context
type ContextStatus string

const (
	ContextStatusHealthy   ContextStatus = "healthy"
	ContextStatusUnhealthy ContextStatus = "unhealthy"
	ContextStatusClosed    ContextStatus = "closed"
)

// BrowserContext represents an isolated incognito browser context
type BrowserContext struct {
	ID         string
	Context    *rod.Browser
	Status     ContextStatus
	CreatedAt  time.Time
	LastUsed   time.Time
	mu         sync.RWMutex
}

// Pool manages a single browser daemon with isolated contexts
type Pool struct {
	cfg           *config.BrowserConfig
	mu            sync.RWMutex
	browser       *rod.Browser
	launcher      *launcher.Launcher
	contexts      map[string]*BrowserContext
	availableCh   chan *BrowserContext
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	activeCount   int32
	maxContexts   int
}

// NewPool creates a new browser pool with a single daemon and context pooling
func NewPool(cfg *config.BrowserConfig) (*Pool, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set defaults if not configured
	if cfg.MaxSessionLife <= 0 {
		cfg.MaxSessionLife = 30 * time.Minute
	}
	if cfg.MaxMemoryMB <= 0 {
		cfg.MaxMemoryMB = 512
	}
	if cfg.HealthCheckInterval <= 0 {
		cfg.HealthCheckInterval = 60 * time.Second
	}

	pool := &Pool{
		cfg:         cfg,
		ctx:         ctx,
		cancel:      cancel,
		contexts:    make(map[string]*BrowserContext),
		availableCh: make(chan *BrowserContext, 100),
		maxContexts: cfg.PoolSize * 10, // Allow 10x contexts vs old worker model
	}

	if pool.maxContexts <= 0 {
		pool.maxContexts = 50 // default max contexts
	}

	// Initialize the single browser daemon
	if err := pool.initBrowser(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to initialize browser daemon: %w", err)
	}

	// Start health check monitor
	pool.startHealthMonitor()

	logger.Success("Browser pool initialized with single daemon (max %d contexts)", pool.maxContexts)

	return pool, nil
}

// initBrowser creates the single browser daemon
func (p *Pool) initBrowser() error {
	// Set up launcher with performance optimizations
	l := launcher.New().
		Leakless(false).
		Set("--no-first-run").
		Set("--no-default-browser-check").
		Set("--disable-background-timer-throttling").
		Set("--disable-renderer-backgrounding").
		Set("--disable-backgrounding-occluded-windows")

	if p.cfg.LaunchOptions["--no-sandbox"] == true {
		l.NoSandbox(true)
	}

	if p.cfg.LaunchOptions["--headless"] == true || p.cfg.LaunchOptions["headless"] == true {
		l.Headless(true)
	}

	// Use custom binary if specified
	if p.cfg.BinaryPath != "" {
		l.Bin(p.cfg.BinaryPath)
	}

	// Launch browser
	u, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	p.launcher = l

	// Connect to browser
	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		l.Cleanup()
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	p.browser = browser
	logger.Success("Browser daemon connected and ready")

	return nil
}

// Acquire returns an available browser context (creates new if needed)
func (p *Pool) Acquire() (*BrowserContext, error) {
	select {
	case <-p.ctx.Done():
		return nil, fmt.Errorf("pool is closed")
	default:
	}

	// Try to get from available channel (non-blocking)
	select {
	case ctx := <-p.availableCh:
		if ctx != nil && p.isContextHealthy(ctx) {
			ctx.mu.Lock()
			ctx.LastUsed = time.Now()
			ctx.mu.Unlock()
			atomic.AddInt32(&p.activeCount, 1)
			return ctx, nil
		}
	default:
		// No available context, will create new
	}

	// Create new context if under limit
	p.mu.Lock()
	currentCount := len(p.contexts)
	p.mu.Unlock()

	if currentCount >= p.maxContexts {
		// Wait for an available context with timeout
		select {
		case ctx := <-p.availableCh:
			if ctx != nil && p.isContextHealthy(ctx) {
				ctx.mu.Lock()
				ctx.LastUsed = time.Now()
				ctx.mu.Unlock()
				atomic.AddInt32(&p.activeCount, 1)
				return ctx, nil
			}
		case <-time.After(5 * time.Second):
			return nil, fmt.Errorf("pool at max capacity (%d contexts) and timeout waiting", p.maxContexts)
		case <-p.ctx.Done():
			return nil, fmt.Errorf("pool is closed")
		}
	}

	// Create new context
	return p.createContext()
}

// createContext creates a new incognito browser context
func (p *Pool) createContext() (*BrowserContext, error) {
	p.mu.RLock()
	browser := p.browser
	p.mu.RUnlock()

	if browser == nil {
		return nil, fmt.Errorf("browser daemon not available")
	}

	// Create incognito context
	incognito, err := browser.Incognito()
	if err != nil {
		return nil, fmt.Errorf("failed to create incognito context: %w", err)
	}

	ctx := &BrowserContext{
		ID:        fmt.Sprintf("ctx-%d", time.Now().UnixNano()),
		Context:   incognito,
		Status:    ContextStatusHealthy,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	p.mu.Lock()
	p.contexts[ctx.ID] = ctx
	p.mu.Unlock()

	atomic.AddInt32(&p.activeCount, 1)

	logger.Debug("Created new browser context: %s (total: %d)", ctx.ID, len(p.contexts))

	return ctx, nil
}

// Release returns a context to the pool
func (p *Pool) Release(ctx *BrowserContext) {
	if ctx == nil {
		return
	}

	atomic.AddInt32(&p.activeCount, -1)

	ctx.mu.Lock()
	if ctx.Status != ContextStatusHealthy {
		ctx.mu.Unlock()
		// Don't return unhealthy contexts to pool
		go p.destroyContext(ctx)
		return
	}
	ctx.LastUsed = time.Now()
	ctx.mu.Unlock()

	// Return to available pool
	select {
	case p.availableCh <- ctx:
	default:
		// Channel full, destroy context
		go p.destroyContext(ctx)
	}
}

// isContextHealthy checks if a context is healthy
func (p *Pool) isContextHealthy(ctx *BrowserContext) bool {
	ctx.mu.RLock()
	status := ctx.Status
	createdAt := ctx.CreatedAt
	ctx.mu.RUnlock()

	if status != ContextStatusHealthy {
		return false
	}

	// Check if context has exceeded max lifetime
	if time.Since(createdAt) > p.cfg.MaxSessionLife {
		return false
	}

	return true
}

// destroyContext permanently destroys a context
func (p *Pool) destroyContext(ctx *BrowserContext) {
	ctx.mu.Lock()
	if ctx.Status == ContextStatusClosed {
		ctx.mu.Unlock()
		return
	}
	ctx.Status = ContextStatusClosed
	browserCtx := ctx.Context
	ctx.mu.Unlock()

	if browserCtx != nil {
		browserCtx.Close()
	}

	p.mu.Lock()
	delete(p.contexts, ctx.ID)
	p.mu.Unlock()

	logger.Debug("Destroyed browser context: %s", ctx.ID)
}

// startHealthMonitor starts the background health monitoring
func (p *Pool) startHealthMonitor() {
	// Context health monitor
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(p.cfg.HealthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				p.checkAllContexts()
			}
		}
	}()

	// Idle context cleanup
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				p.cleanupIdleContexts()
			}
		}
	}()
}

// checkAllContexts checks the health of all contexts
func (p *Pool) checkAllContexts() {
	p.mu.RLock()
	contexts := make([]*BrowserContext, 0, len(p.contexts))
	for _, ctx := range p.contexts {
		contexts = append(contexts, ctx)
	}
	p.mu.RUnlock()

	for _, ctx := range contexts {
		select {
		case <-p.ctx.Done():
			return
		default:
			p.checkContextHealth(ctx)
		}
	}
}

// checkContextHealth performs a lightweight health check on a context
func (p *Pool) checkContextHealth(ctx *BrowserContext) {
	ctx.mu.RLock()
	if ctx.Status != ContextStatusHealthy {
		ctx.mu.RUnlock()
		return
	}
	browserCtx := ctx.Context
	ctx.mu.RUnlock()

	if browserCtx == nil {
		p.markUnhealthy(ctx)
		return
	}

	// Sprint 27.5: Use Version() instead of Page() for health check to avoid CDP congestion.
	// Version() is a very lightweight call that confirms the connection is alive.
	_, err := browserCtx.Version()
	if err != nil {
		logger.Warn("Context %s health check failed (Connection lost): %v", ctx.ID, err)
		p.markUnhealthy(ctx)
		return
	}
}

// markUnhealthy marks a context as unhealthy
func (p *Pool) markUnhealthy(ctx *BrowserContext) {
	ctx.mu.Lock()
	ctx.Status = ContextStatusUnhealthy
	ctx.mu.Unlock()

	logger.Warn("Context %s marked unhealthy, will be destroyed", ctx.ID)
	go p.destroyContext(ctx)
}

// cleanupIdleContexts removes contexts that have been idle too long
func (p *Pool) cleanupIdleContexts() {
	p.mu.RLock()
	contexts := make([]*BrowserContext, 0, len(p.contexts))
	for _, ctx := range p.contexts {
		contexts = append(contexts, ctx)
	}
	p.mu.RUnlock()

	idleTimeout := 3 * time.Minute

	for _, ctx := range contexts {
		ctx.mu.RLock()
		lastUsed := ctx.LastUsed
		status := ctx.Status
		ctx.mu.RUnlock()

		if status == ContextStatusHealthy && time.Since(lastUsed) > idleTimeout {
			logger.Debug("Cleaning up idle context: %s", ctx.ID)
			go p.destroyContext(ctx)
		}
	}
}

// Close closes the browser pool and all contexts
func (p *Pool) Close() error {
	p.mu.Lock()
	select {
	case <-p.ctx.Done():
		p.mu.Unlock()
		return nil
	default:
	}
	p.cancel()
	p.mu.Unlock()

	p.wg.Wait()

	// Close all contexts
	p.mu.Lock()
	contexts := make([]*BrowserContext, 0, len(p.contexts))
	for _, ctx := range p.contexts {
		contexts = append(contexts, ctx)
	}
	p.mu.Unlock()

	for _, ctx := range contexts {
		ctx.mu.Lock()
		if ctx.Context != nil && ctx.Status != ContextStatusClosed {
			ctx.Context.Close()
		}
		ctx.Status = ContextStatusClosed
		ctx.mu.Unlock()
	}

	// Close browser daemon
	p.mu.Lock()
	if p.browser != nil {
		p.browser.Close()
		p.browser = nil
	}
	if p.launcher != nil {
		p.launcher.Cleanup()
		p.launcher = nil
	}
	p.mu.Unlock()

	logger.Success("Browser pool closed")
	return nil
}

// Stats returns pool statistics
func (p *Pool) Stats() (active, idle int) {
	active = int(atomic.LoadInt32(&p.activeCount))
	
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	total := len(p.contexts)
	idle = total - active
	if idle < 0 {
		idle = 0
	}

	return active, idle
}

// GetContextCount returns the number of contexts in the pool
func (p *Pool) GetContextCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.contexts)
}

// GetContext returns a context by ID (for backward compatibility with session.go)
func (p *Pool) GetContext(id string) (*BrowserContext, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ctx, exists := p.contexts[id]
	if !exists {
		return nil, fmt.Errorf("context %s not found", id)
	}
	return ctx, nil
}

// GetWorker is an alias for GetContext for backward compatibility
// Deprecated: Use GetContext instead
func (p *Pool) GetWorker(id string) (*BrowserContext, error) {
	return p.GetContext(id)
}

// GetWorkerCount is an alias for GetContextCount for backward compatibility
// Deprecated: Use GetContextCount instead
func (p *Pool) GetWorkerCount() int {
	return p.GetContextCount()
}
