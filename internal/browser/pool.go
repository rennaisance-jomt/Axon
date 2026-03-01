package browser

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"github.com/rennaisance-jomt/axon/pkg/logger"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rennaisance-jomt/axon/internal/config"
)

// WorkerStatus represents the health status of a browser worker
type WorkerStatus string

const (
	WorkerStatusHealthy   WorkerStatus = "healthy"
	WorkerStatusUnhealthy WorkerStatus = "unhealthy"
	WorkerStatusRotating  WorkerStatus = "rotating"
	WorkerStatusClosed    WorkerStatus = "closed"
)

// Worker represents a single browser daemon process in the pool
type Worker struct {
	ID         string
	Browser    *rod.Browser
	Status     WorkerStatus
	Sessions   int32 // atomic counter
	CreatedAt time.Time
	LastHealth time.Time
	mu        sync.RWMutex
}

// Pool manages multiple browser worker processes
type Pool struct {
	cfg           *config.BrowserConfig
	mu            sync.RWMutex
	workers       map[string]*Worker
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	healthCheckCh chan string // channel for health check signals
}

// NewPool creates a new browser pool with multiple workers
func NewPool(cfg *config.BrowserConfig) (*Pool, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set defaults if not configured
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 5
	}
	if cfg.MaxSessionLife <= 0 {
		cfg.MaxSessionLife = 30 * time.Minute
	}
	if cfg.MaxMemoryMB <= 0 {
		cfg.MaxMemoryMB = 512
	}
	if cfg.HealthCheckInterval <= 0 {
		cfg.HealthCheckInterval = 30 * time.Second
	}

	pool := &Pool{
		cfg:           cfg,
		ctx:           ctx,
		cancel:        cancel,
		workers:       make(map[string]*Worker),
		healthCheckCh: make(chan string, 10),
	}

	// Initialize the worker pool
	if err := pool.initWorkers(cfg.PoolSize); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to initialize workers: %w", err)
	}

	// Start health check monitor
	pool.startHealthMonitor()

	return pool, nil
}

// initWorkers creates the initial set of browser workers
func (p *Pool) initWorkers(count int) error {
	for i := 0; i < count; i++ {
		worker, err := p.createWorker()
		if err != nil {
			logger.Warn("Failed to create browser worker %d: %v", i, err)
			continue
		}
		p.workers[worker.ID] = worker
	}

	if len(p.workers) == 0 {
		return fmt.Errorf("failed to create any workers")
	}

	return nil
}

// createWorker creates a new browser worker
func (p *Pool) createWorker() (*Worker, error) {
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

	worker := &Worker{
		ID:         fmt.Sprintf("worker-%d", time.Now().UnixNano()),
		Browser:    browser,
		Status:     WorkerStatusHealthy,
		Sessions:   0,
		CreatedAt:  time.Now(),
		LastHealth: time.Now(),
	}

	return worker, nil
}

// Acquire returns an available browser worker
func (p *Pool) Acquire() (*Worker, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	select {
	case <-p.ctx.Done():
		return nil, fmt.Errorf("pool is closed")
	default:
	}

	// Find the healthiest worker with lowest session count
	var bestWorker *Worker
	var minSessions int32 = int32(^int32(0)) // Max int32

	for _, worker := range p.workers {
		if worker.Status != WorkerStatusHealthy {
			continue
		}

		sessions := atomic.LoadInt32(&worker.Sessions)
		if sessions < minSessions {
			minSessions = sessions
			bestWorker = worker
		}
	}

	if bestWorker == nil {
		// All workers are unhealthy, try to find any available
		for _, worker := range p.workers {
			if worker.Status != WorkerStatusClosed {
				bestWorker = worker
				break
			}
		}
		if bestWorker == nil {
			return nil, fmt.Errorf("no available workers")
		}
	}

	// Increment session count
	atomic.AddInt32(&bestWorker.Sessions, 1)

	return bestWorker, nil
}

// Release decrements the session count for a worker
func (p *Pool) Release(worker *Worker) {
	if worker == nil {
		return
	}
	atomic.AddInt32(&worker.Sessions, -1)
	
	// Trigger health check for this worker
	select {
	case p.healthCheckCh <- worker.ID:
	default:
	}
}

// markUnhealthy marks a worker as unhealthy and triggers rotation
func (p *Pool) markUnhealthy(workerID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	worker, exists := p.workers[workerID]
	if !exists {
		return
	}

	worker.mu.Lock()
	worker.Status = WorkerStatusUnhealthy
	worker.mu.Unlock()

	logger.Warn("Browser worker %s is unhealthy, initiating rotation", workerID)

	// Start rotation in background
	go p.rotateWorker(workerID)
}

// rotateWorker replaces an unhealthy worker with a new one
func (p *Pool) rotateWorker(workerID string) {
	p.mu.Lock()
	
	worker, exists := p.workers[workerID]
	if !exists {
		p.mu.Unlock()
		return
	}

	worker.mu.Lock()
	worker.Status = WorkerStatusRotating
	worker.mu.Unlock()
	p.mu.Unlock()

	logger.System("Rotating browser worker %s", workerID)

	// Wait for sessions to drain (max 30 seconds)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			logger.Warn("Force rotating browser worker %s after timeout", workerID)
			goto forceRotate
		case <-ticker.C:
			sessions := atomic.LoadInt32(&worker.Sessions)
			if sessions == 0 {
				goto rotate
			}
		}
	}

forceRotate:
rotate:
	// Close the old browser
	if worker.Browser != nil {
		worker.Browser.Close()
	}

	// Create new worker
	p.mu.Lock()
	newWorker, err := p.createWorker()
	if err != nil {
		logger.Error("Failed to create replacement browser worker: %v", err)
		// Keep the old worker in unhealthy state
		worker.mu.Lock()
		worker.Status = WorkerStatusUnhealthy
		worker.mu.Unlock()
		p.mu.Unlock()
		return
	}

	// Replace the worker
	delete(p.workers, workerID)
	p.workers[newWorker.ID] = newWorker
	p.mu.Unlock()

	logger.Success("Browser worker %s successfully rotated to %s", workerID, newWorker.ID)
}

// startHealthMonitor starts the background health monitoring
func (p *Pool) startHealthMonitor() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(p.cfg.HealthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-p.ctx.Done():
				return
			case workerID := <-p.healthCheckCh:
				p.checkWorkerHealth(workerID)
			case <-ticker.C:
				p.checkAllWorkers()
			}
		}
	}()

	// Start memory monitor
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(10 * time.Second) // Check memory every 10 seconds
		defer ticker.Stop()

		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				p.checkMemoryUsage()
			}
		}
	}()
}

// checkWorkerHealth checks the health of a specific worker
func (p *Pool) checkWorkerHealth(workerID string) {
	p.mu.RLock()
	worker, exists := p.workers[workerID]
	p.mu.RUnlock()

	if !exists {
		return
	}

	// Perform a simple health check
	// Try to get browser version as health check
	worker.mu.Lock()
	if worker.Browser == nil {
		worker.mu.Unlock()
		p.markUnhealthy(workerID)
		return
	}
	worker.mu.Unlock()

	// Simple check: try to create a page
	_, err := worker.Browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		logger.Warn("Browser worker %s health check failed: %v", workerID, err)
		p.markUnhealthy(workerID)
		return
	}

	worker.mu.Lock()
	worker.LastHealth = time.Now()
	worker.Status = WorkerStatusHealthy
	worker.mu.Unlock()
}

// checkAllWorkers checks the health of all workers
func (p *Pool) checkAllWorkers() {
	p.mu.RLock()
	workerIDs := make([]string, 0, len(p.workers))
	for id := range p.workers {
		workerIDs = append(workerIDs, id)
	}
	p.mu.RUnlock()

	for _, id := range workerIDs {
		select {
		case <-p.ctx.Done():
			return
		default:
			p.checkWorkerHealth(id)
		}
	}
}

// checkMemoryUsage monitors memory usage of workers
func (p *Pool) checkMemoryUsage() {
	p.mu.RLock()
	workers := make([]*Worker, 0, len(p.workers))
	for _, w := range p.workers {
		workers = append(workers, w)
	}
	p.mu.RUnlock()

	for _, worker := range workers {
		// Get memory usage via CDP
		worker.mu.Lock()
		browser := worker.Browser
		worker.mu.Unlock()

		if browser == nil {
			continue
		}

		// Use process memory metrics via Eval on a temporary page
		var memUsage int64
		page, err := browser.Page(proto.TargetCreateTarget{})
		if err == nil {
			if result, err := page.Eval(`() => { 
				if (performance.memory) {
					return performance.memory.usedJSHeapSize;
				}
				return 0;
			}`); err == nil {
				memUsage = int64(result.Value.Int())
			}
			page.Close()
		}

		memMB := memUsage / (1024 * 1024)
		if memMB > int64(p.cfg.MaxMemoryMB) {
			logger.Warn("Browser worker %s exceeded memory limit: %dMB > %dMB", 
				worker.ID, memMB, p.cfg.MaxMemoryMB)
			p.markUnhealthy(worker.ID)
		}
	}
}

// Close closes all browser workers in the pool
func (p *Pool) Close() error {
	p.cancel()
	p.wg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	for _, worker := range p.workers {
		if worker.Browser != nil {
			if err := worker.Browser.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		worker.mu.Lock()
		worker.Status = WorkerStatusClosed
		worker.mu.Unlock()
	}
	p.workers = make(map[string]*Worker)

	return firstErr
}

// Stats returns pool statistics
func (p *Pool) Stats() (active, idle int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	active = 0
	idle = 0

	for _, worker := range p.workers {
		worker.mu.RLock()
		status := worker.Status
		sessions := atomic.LoadInt32(&worker.Sessions)
		worker.mu.RUnlock()

		if status == WorkerStatusHealthy && sessions == 0 {
			idle++
		}
		if status == WorkerStatusHealthy {
			active++
		}
	}

	return active, idle
}

// GetWorkerCount returns the number of workers in the pool
func (p *Pool) GetWorkerCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.workers)
}

// GetWorker returns a worker by ID
func (p *Pool) GetWorker(id string) (*Worker, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	worker, exists := p.workers[id]
	if !exists {
		return nil, fmt.Errorf("worker %s not found", id)
	}
	return worker, nil
}
