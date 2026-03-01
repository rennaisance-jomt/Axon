package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// OverlayFrame represents a single frame of overlay data
type OverlayFrame struct {
	Type      string          `json:"type"` // "screenshot", "dom", "spatial", "console"
	Timestamp time.Time       `json:"timestamp"`
	SessionID string          `json:"session_id"`
	Data      json.RawMessage `json:"data"`
}

// OverlayEvent represents an event in the overlay
type OverlayEvent struct {
	Type        string          `json:"type"` // "click", "input", "navigate", "error"
	X           float64         `json:"x,omitempty"`
	Y           float64         `json:"y,omitempty"`
	Key         string          `json:"key,omitempty"`
	Value       string          `json:"value,omitempty"`
	URL         string          `json:"url,omitempty"`
	Message     string          `json:"message,omitempty"`
	Level       string          `json:"level,omitempty"` // "info", "warn", "error"
	Timestamp   time.Time       `json:"timestamp"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// OverlayClient represents a connected overlay viewer
type OverlayClient struct {
	conn      *websocket.Conn
	send      chan []byte
	sessionID string
	mu        sync.Mutex
	closed    bool
}

// OverlayServer manages WebSocket connections for vision overlay
type OverlayServer struct {
	mu       sync.RWMutex
	clients  map[string]*OverlayClient
	upgrader websocket.Upgrader
	config   *OverlayConfig
}

// OverlayConfig holds overlay configuration
type OverlayConfig struct {
	Enabled        bool
	MaxClients     int
	BufferSize     int
	CaptureInterval time.Duration
}

// NewOverlayServer creates a new overlay server
func NewOverlayServer(cfg *OverlayConfig) *OverlayServer {
	if cfg == nil {
		cfg = &OverlayConfig{
			Enabled:     true,
			MaxClients:  10,
			BufferSize:  100,
			CaptureInterval: 100 * time.Millisecond,
		}
	}

	return &OverlayServer{
		clients: make(map[string]*OverlayClient),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		config: cfg,
	}
}

// HandleWebSocket upgrades an HTTP connection to WebSocket
func (os *OverlayServer) HandleWebSocket(w http.ResponseWriter, r *http.Request, sessionID string) error {
	if !os.config.Enabled {
		return errors.New("overlay not enabled")
	}

	os.mu.Lock()
	if len(os.clients) >= os.config.MaxClients {
		os.mu.Unlock()
		return errors.New("max clients reached")
	}
	os.mu.Unlock()

	conn, err := os.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("failed to upgrade: %w", err)
	}

	client := &OverlayClient{
		conn:      conn,
		send:      make(chan []byte, os.config.BufferSize),
		sessionID: sessionID,
	}

	os.mu.Lock()
	os.clients[sessionID] = client
	os.mu.Unlock()

	go os.readPump(client)
	go os.writePump(client)

	return nil
}

// BroadcastFrame broadcasts a frame to all connected clients for a session
func (os *OverlayServer) BroadcastFrame(sessionID string, frame *OverlayFrame) {
	os.mu.RLock()
	client, ok := os.clients[sessionID]
	os.mu.RUnlock()

	if !ok || client == nil {
		return
	}

	data, err := json.Marshal(frame)
	if err != nil {
		return
	}

	select {
	case client.send <- data:
	default:
		// Buffer full, drop frame
	}
}

// BroadcastEvent broadcasts an event to all connected clients for a session
func (os *OverlayServer) BroadcastEvent(sessionID string, event *OverlayEvent) {
	os.mu.RLock()
	client, ok := os.clients[sessionID]
	os.mu.RUnlock()

	if !ok || client == nil {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	select {
	case client.send <- data:
	default:
	}
}

// readPump reads messages from the WebSocket
func (os *OverlayServer) readPump(client *OverlayClient) {
	defer func() {
		os.removeClient(client.sessionID)
		client.conn.Close()
	}()

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error
			}
			break
		}

		// Handle incoming messages (e.g., pause/resume, filters)
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			os.handleClientMessage(client.sessionID, msg)
		}
	}
}

// writePump writes messages to the WebSocket
func (os *OverlayServer) writePump(client *OverlayClient) {
	ticker := time.NewTicker(os.config.CaptureInterval)
	defer func() {
		ticker.Stop()
		os.removeClient(client.sessionID)
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			// Send keepalive
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (os *OverlayServer) handleClientMessage(sessionID string, msg map[string]interface{}) {
	// Handle client commands like "pause", "resume", "setFilter"
}

func (os *OverlayServer) removeClient(sessionID string) {
	os.mu.Lock()
	defer os.mu.Unlock()

	if client, ok := os.clients[sessionID]; ok {
		close(client.send)
		delete(os.clients, sessionID)
	}
}

// GetClientCount returns the number of connected clients
func (os *OverlayServer) GetClientCount() int {
	os.mu.RLock()
	defer os.mu.RUnlock()
	return len(os.clients)
}

// Close closes all client connections
func (os *OverlayServer) Close() {
	os.mu.Lock()
	defer os.mu.Unlock()

	for _, client := range os.clients {
		close(client.send)
		client.conn.Close()
	}
	os.clients = make(map[string]*OverlayClient)
}
