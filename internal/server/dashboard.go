package server

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// StatsCollector collects and manages statistics
type StatsCollector struct {
	mu sync.RWMutex
	
	// System stats
	StartTime       time.Time
	TotalRequests   int64
	ActiveSessions  int
	TotalSessions   int64
	
	// Performance stats
	RequestLatencies []time.Duration
	AvgLatency       time.Duration
	P95Latency       time.Duration
	P99Latency       time.Duration
	
	// Token optimization stats
	TotalTokensSaved    int64
	AvgTokenReduction   float64
	
	// Error stats
	TotalErrors     int64
	RetryCount      int64
	SuccessRate     float64
	
	// Browser stats
	BrowserPoolSize     int
	ActiveContexts      int
	MemoryUsage         uint64
	
	// Snapshot stats
	SnapshotsTaken  int64
	AvgSnapshotTime time.Duration
	
	// Real-time clients
	wsClients map[*websocket.Conn]bool
	wsMu      sync.RWMutex
}

// NewStatsCollector creates a new stats collector
func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		StartTime:        time.Now(),
		RequestLatencies: make([]time.Duration, 0, 1000),
		wsClients:        make(map[*websocket.Conn]bool),
	}
}

// RecordRequest records a request
func (s *StatsCollector) RecordRequest(latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.TotalRequests++
	s.RequestLatencies = append(s.RequestLatencies, latency)
	
	// Keep only last 1000 latencies
	if len(s.RequestLatencies) > 1000 {
		s.RequestLatencies = s.RequestLatencies[len(s.RequestLatencies)-1000:]
	}
	
	s.calculateLatencies()
	s.broadcastUpdate()
}

// RecordSession records session creation
func (s *StatsCollector) RecordSession(active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if active {
		s.ActiveSessions++
		s.TotalSessions++
	} else {
		s.ActiveSessions--
	}
	
	s.broadcastUpdate()
}

// RecordError records an error
func (s *StatsCollector) RecordError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.TotalErrors++
	s.calculateSuccessRate()
	s.broadcastUpdate()
}

// RecordRetry records a retry
func (s *StatsCollector) RecordRetry() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.RetryCount++
	s.broadcastUpdate()
}

// RecordSnapshot records snapshot metrics
func (s *StatsCollector) RecordSnapshot(duration time.Duration, tokensSaved int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.SnapshotsTaken++
	
	// Calculate average snapshot time
	if s.AvgSnapshotTime == 0 {
		s.AvgSnapshotTime = duration
	} else {
		s.AvgSnapshotTime = (s.AvgSnapshotTime + duration) / 2
	}
	
	if tokensSaved > 0 {
		s.TotalTokensSaved += int64(tokensSaved)
	}
	
	s.broadcastUpdate()
}

// UpdateBrowserStats updates browser pool statistics
func (s *StatsCollector) UpdateBrowserStats(poolSize, activeContexts int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.BrowserPoolSize = poolSize
	s.ActiveContexts = activeContexts
	
	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	s.MemoryUsage = m.Alloc
	
	s.broadcastUpdate()
}

func (s *StatsCollector) calculateLatencies() {
	if len(s.RequestLatencies) == 0 {
		return
	}
	
	// Calculate average
	var total time.Duration
	for _, lat := range s.RequestLatencies {
		total += lat
	}
	s.AvgLatency = total / time.Duration(len(s.RequestLatencies))
	
	// Simple P95/P99 calculation (not exact but good enough)
	sorted := make([]time.Duration, len(s.RequestLatencies))
	copy(sorted, s.RequestLatencies)
	
	// Bubble sort for simplicity (small array)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	p95Idx := int(float64(len(sorted)) * 0.95)
	p99Idx := int(float64(len(sorted)) * 0.99)
	
	if p95Idx < len(sorted) {
		s.P95Latency = sorted[p95Idx]
	}
	if p99Idx < len(sorted) {
		s.P99Latency = sorted[p99Idx]
	}
}

func (s *StatsCollector) calculateSuccessRate() {
	total := s.TotalRequests + s.TotalErrors
	if total > 0 {
		s.SuccessRate = float64(s.TotalRequests) / float64(total) * 100
	}
}

// GetStats returns current statistics
func (s *StatsCollector) GetStats() StatsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return StatsSnapshot{
		Uptime:              time.Since(s.StartTime).String(),
		TotalRequests:       s.TotalRequests,
		ActiveSessions:      s.ActiveSessions,
		TotalSessions:       s.TotalSessions,
		AvgLatency:          s.AvgLatency.String(),
		P95Latency:          s.P95Latency.String(),
		P99Latency:          s.P99Latency.String(),
		TotalTokensSaved:    s.TotalTokensSaved,
		SuccessRate:         s.SuccessRate,
		TotalErrors:         s.TotalErrors,
		RetryCount:          s.RetryCount,
		BrowserPoolSize:     s.BrowserPoolSize,
		ActiveContexts:      s.ActiveContexts,
		MemoryUsage:         formatBytes(s.MemoryUsage),
		SnapshotsTaken:      s.SnapshotsTaken,
		AvgSnapshotTime:     s.AvgSnapshotTime.String(),
	}
}

// StatsSnapshot represents a snapshot of statistics
type StatsSnapshot struct {
	Uptime           string  `json:"uptime"`
	TotalRequests    int64   `json:"total_requests"`
	ActiveSessions   int     `json:"active_sessions"`
	TotalSessions    int64   `json:"total_sessions"`
	AvgLatency       string  `json:"avg_latency"`
	P95Latency       string  `json:"p95_latency"`
	P99Latency       string  `json:"p99_latency"`
	TotalTokensSaved int64   `json:"total_tokens_saved"`
	SuccessRate      float64 `json:"success_rate"`
	TotalErrors      int64   `json:"total_errors"`
	RetryCount       int64   `json:"retry_count"`
	BrowserPoolSize  int     `json:"browser_pool_size"`
	ActiveContexts   int     `json:"active_contexts"`
	MemoryUsage      string  `json:"memory_usage"`
	SnapshotsTaken   int64   `json:"snapshots_taken"`
	AvgSnapshotTime  string  `json:"avg_snapshot_time"`
}

// RegisterWSClient registers a WebSocket client
func (s *StatsCollector) RegisterWSClient(conn *websocket.Conn) {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	s.wsClients[conn] = true
}

// UnregisterWSClient unregisters a WebSocket client
func (s *StatsCollector) UnregisterWSClient(conn *websocket.Conn) {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	delete(s.wsClients, conn)
	conn.Close()
}

func (s *StatsCollector) broadcastUpdate() {
	s.wsMu.RLock()
	defer s.wsMu.RUnlock()
	
	stats := s.GetStats()
	data, err := json.Marshal(stats)
	if err != nil {
		return
	}
	
	for client := range s.wsClients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			// Client will be cleaned up on next write
		}
	}
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// DashboardHandler handles dashboard routes
type DashboardHandler struct {
	stats *StatsCollector
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(stats *StatsCollector) *DashboardHandler {
	return &DashboardHandler{stats: stats}
}

// RegisterRoutes registers dashboard routes
func (h *DashboardHandler) RegisterRoutes(app *fiber.App) {
	// Stats API endpoint
	app.Get("/api/stats", h.handleGetStats)
	
	// WebSocket endpoint for real-time updates
	app.Get("/ws/stats", websocket.New(h.handleWSStats))
	
	// Dashboard UI (simple HTML for now)
	app.Get("/dashboard", h.handleDashboardUI)
}

func (h *DashboardHandler) handleGetStats(c *fiber.Ctx) error {
	return c.JSON(h.stats.GetStats())
}

func (h *DashboardHandler) handleWSStats(c *websocket.Conn) {
	h.stats.RegisterWSClient(c)
	defer h.stats.UnregisterWSClient(c)
	
	// Send initial stats
	stats := h.stats.GetStats()
	data, _ := json.Marshal(stats)
	c.WriteMessage(websocket.TextMessage, data)
	
	// Keep connection alive
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (h *DashboardHandler) handleDashboardUI(c *fiber.Ctx) error {
	c.Type("html")
	return c.SendString(dashboardHTML)
}

const dashboardHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Axon Dashboard</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; border-radius: 10px; margin-bottom: 20px; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; }
        .card { background: white; padding: 20px; border-radius: 10px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .card h3 { margin: 0 0 10px 0; color: #333; font-size: 14px; text-transform: uppercase; }
        .card .value { font-size: 32px; font-weight: bold; color: #667eea; }
        .card .sub { font-size: 12px; color: #666; margin-top: 5px; }
        .status { display: inline-block; width: 10px; height: 10px; border-radius: 50%; margin-right: 5px; }
        .status.online { background: #4caf50; }
        .status.offline { background: #f44336; }
        #connection { position: fixed; top: 20px; right: 20px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🧠 Axon Dashboard</h1>
        <p>Real-time browser automation metrics</p>
        <span id="connection"><span class="status online"></span> Connected</span>
    </div>
    
    <div class="grid">
        <div class="card">
            <h3>Active Sessions</h3>
            <div class="value" id="active-sessions">0</div>
            <div class="sub">Total: <span id="total-sessions">0</span></div>
        </div>
        
        <div class="card">
            <h3>Total Requests</h3>
            <div class="value" id="total-requests">0</div>
            <div class="sub">Success Rate: <span id="success-rate">0</span>%</div>
        </div>
        
        <div class="card">
            <h3>Avg Latency</h3>
            <div class="value" id="avg-latency">0ms</div>
            <div class="sub">P95: <span id="p95-latency">0</span> | P99: <span id="p99-latency">0</span></div>
        </div>
        
        <div class="card">
            <h3>Tokens Saved</h3>
            <div class="value" id="tokens-saved">0</div>
            <div class="sub">Through semantic compression</div>
        </div>
        
        <div class="card">
            <h3>Browser Pool</h3>
            <div class="value" id="browser-pool">0</div>
            <div class="sub">Active Contexts: <span id="active-contexts">0</span></div>
        </div>
        
        <div class="card">
            <h3>Memory Usage</h3>
            <div class="value" id="memory-usage">0</div>
            <div class="sub">Go runtime heap</div>
        </div>
        
        <div class="card">
            <h3>Snapshots Taken</h3>
            <div class="value" id="snapshots-taken">0</div>
            <div class="sub">Avg Time: <span id="avg-snapshot-time">0</span></div>
        </div>
        
        <div class="card">
            <h3>Retries</h3>
            <div class="value" id="retry-count">0</div>
            <div class="sub">Errors: <span id="total-errors">0</span></div>
        </div>
    </div>
    
    <script>
        const ws = new WebSocket('ws://' + window.location.host + '/ws/stats');
        
        ws.onmessage = (event) => {
            const stats = JSON.parse(event.data);
            document.getElementById('active-sessions').textContent = stats.active_sessions;
            document.getElementById('total-sessions').textContent = stats.total_sessions;
            document.getElementById('total-requests').textContent = stats.total_requests;
            document.getElementById('success-rate').textContent = stats.success_rate.toFixed(1);
            document.getElementById('avg-latency').textContent = stats.avg_latency;
            document.getElementById('p95-latency').textContent = stats.p95_latency;
            document.getElementById('p99-latency').textContent = stats.p99_latency;
            document.getElementById('tokens-saved').textContent = stats.total_tokens_saved.toLocaleString();
            document.getElementById('browser-pool').textContent = stats.browser_pool_size;
            document.getElementById('active-contexts').textContent = stats.active_contexts;
            document.getElementById('memory-usage').textContent = stats.memory_usage;
            document.getElementById('snapshots-taken').textContent = stats.snapshots_taken;
            document.getElementById('avg-snapshot-time').textContent = stats.avg_snapshot_time;
            document.getElementById('retry-count').textContent = stats.retry_count;
            document.getElementById('total-errors').textContent = stats.total_errors;
        };
        
        ws.onclose = () => {
            document.querySelector('#connection .status').className = 'status offline';
            document.querySelector('#connection').innerHTML = '<span class="status offline"></span> Disconnected';
        };
        
        ws.onerror = () => {
            document.querySelector('#connection .status').className = 'status offline';
        };
    </script>
</body>
</html>`
