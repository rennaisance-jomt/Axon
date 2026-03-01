package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/rennaisance-jomt/axon/internal/browser"
	"github.com/rennaisance-jomt/axon/internal/config"
	"github.com/rennaisance-jomt/axon/internal/storage"
	"github.com/rennaisance-jomt/axon/pkg/logger"
	"github.com/rennaisance-jomt/axon/pkg/types"
)

// MCPMessage represents an MCP protocol message
type MCPMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCPTool represents a tool definition
type MCPTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// MCPServer represents the MCP server
type MCPServer struct {
	sessions    *browser.SessionManager
	pool        *browser.Pool
	db          *storage.DB
	cfg         *config.Config
	reader      *bufio.Reader
	writer      io.Writer
	currentSession string
}

// NewMCPServer creates a new MCP server
func NewMCPServer(pool *browser.Pool, db *storage.DB, cfg *config.Config) *MCPServer {
	return &MCPServer{
		sessions: browser.NewSessionManager(pool, cfg.Browser.MaxSessionLife),
		pool:     pool,
		db:       db,
		cfg:      cfg,
		reader:   bufio.NewReader(os.Stdin),
		writer:   os.Stdout,
	}
}

// SetIO allows setting custom reader/writer for testing
func (s *MCPServer) SetIO(reader io.Reader, writer io.Writer) {
	s.reader = bufio.NewReader(reader)
	s.writer = writer
}

// Run starts the MCP server loop
func (s *MCPServer) Run() error {
	logger.System("MCP Server started, waiting for messages...")
	
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read message: %w", err)
		}
		
		var msg MCPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}
		
		if err := s.handleMessage(&msg); err != nil {
			logger.Error("MCP handling error: %v", err)
		}
	}
}

func (s *MCPServer) handleMessage(msg *MCPMessage) error {
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "tools/list":
		return s.handleToolsList(msg)
	case "tools/call":
		return s.handleToolsCall(msg)
	default:
		return s.sendError(msg.ID, -32601, "Method not found", nil)
	}
}

func (s *MCPServer) handleInitialize(msg *MCPMessage) error {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "axon-mcp-server",
			"version": "1.0.0",
		},
	}
	return s.sendResult(msg.ID, result)
}

func (s *MCPServer) handleToolsList(msg *MCPMessage) error {
	tools := []MCPTool{
		{
			Name:        "axon_navigate",
			Description: "Navigate to a URL in the browser session",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"url": {
						Type:        "string",
						Description: "The URL to navigate to",
					},
					"session_id": {
						Type:        "string",
						Description: "Optional session ID (creates new if not provided)",
					},
				},
				Required: []string{"url"},
			},
		},
		{
			Name:        "axon_snapshot",
			Description: "Get a semantic snapshot of the current page with interactive elements",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"session_id": {
						Type:        "string",
						Description: "Session ID (uses default if not provided)",
					},
					"depth": {
						Type:        "string",
						Description: "Snapshot depth: compact, standard, or full",
						Enum:        []string{"compact", "standard", "full"},
					},
				},
				Required: []string{},
			},
		},
		{
			Name:        "axon_act",
			Description: "Perform an action on an element (click, fill, press, etc.)",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"ref": {
						Type:        "string",
						Description: "Element reference from snapshot (e.g., 'b4', 't12')",
					},
					"action": {
						Type:        "string",
						Description: "Action to perform",
						Enum:        []string{"click", "fill", "press", "select", "hover", "scroll"},
					},
					"value": {
						Type:        "string",
						Description: "Value for fill/select actions",
					},
					"session_id": {
						Type:        "string",
						Description: "Session ID",
					},
					"confirm": {
						Type:        "boolean",
						Description: "Confirm irreversible actions",
					},
				},
				Required: []string{"ref", "action"},
			},
		},
		{
			Name:        "axon_find_and_act",
			Description: "Find an element by intent description and perform an action",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"intent": {
						Type:        "string",
						Description: "Description of the element (e.g., 'search box', 'login button')",
					},
					"action": {
						Type:        "string",
						Description: "Action to perform",
						Enum:        []string{"click", "fill", "press", "select", "hover", "scroll"},
					},
					"value": {
						Type:        "string",
						Description: "Value for fill/select actions",
					},
					"session_id": {
						Type:        "string",
						Description: "Session ID",
					},
				},
				Required: []string{"intent", "action"},
			},
		},
		{
			Name:        "axon_get_status",
			Description: "Get current page status and state information",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"session_id": {
						Type:        "string",
						Description: "Session ID",
					},
				},
				Required: []string{},
			},
		},
	}
	
	return s.sendResult(msg.ID, map[string]interface{}{
		"tools": tools,
	})
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

func (s *MCPServer) handleToolsCall(msg *MCPMessage) error {
	var params ToolCallParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, -32602, "Invalid params", err.Error())
	}
	
	var result interface{}
	var err error
	
	switch params.Name {
	case "axon_navigate":
		result, err = s.toolNavigate(params.Arguments)
	case "axon_snapshot":
		result, err = s.toolSnapshot(params.Arguments)
	case "axon_act":
		result, err = s.toolAct(params.Arguments)
	case "axon_find_and_act":
		result, err = s.toolFindAndAct(params.Arguments)
	case "axon_get_status":
		result, err = s.toolGetStatus(params.Arguments)
	default:
		return s.sendError(msg.ID, -32602, "Unknown tool", params.Name)
	}
	
	if err != nil {
		return s.sendError(msg.ID, -32000, "Tool execution failed", err.Error())
	}
	
	return s.sendResult(msg.ID, result)
}

func (s *MCPServer) toolNavigate(args map[string]interface{}) (interface{}, error) {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}
	
	sessionID := ""
	if sid, ok := args["session_id"].(string); ok {
		sessionID = sid
	}
	
	// Use default session if none provided
	if sessionID == "" {
		sessionID = "default"
	}
	
	// Create session if it doesn't exist
	session, err := s.sessions.Get(sessionID)
	if err != nil {
		session, err = s.sessions.Create(sessionID, "")
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}
	
	s.currentSession = sessionID
	
	if err := session.Navigate(url, "load"); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}
	
	return map[string]interface{}{
		"success": true,
		"url":     session.URL,
		"title":   session.Title,
		"state":   session.PageState,
	}, nil
}

func (s *MCPServer) toolSnapshot(args map[string]interface{}) (interface{}, error) {
	sessionID := "default"
	if sid, ok := args["session_id"].(string); ok && sid != "" {
		sessionID = sid
	}
	
	depth := "compact"
	if d, ok := args["depth"].(string); ok && d != "" {
		depth = d
	}
	
	session, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	
	extractor := browser.NewSnapshotExtractor()
	snapshot, err := extractor.Extract(session.Page, depth, "")
	if err != nil {
		return nil, fmt.Errorf("snapshot failed: %w", err)
	}
	
	session.SetLastElements(snapshot.Elements)
	
	return map[string]interface{}{
		"content":   snapshot.Content,
		"url":       snapshot.URL,
		"title":     snapshot.Title,
		"elements":  snapshot.Elements,
		"warnings":  snapshot.Warnings,
	}, nil
}

func (s *MCPServer) toolAct(args map[string]interface{}) (interface{}, error) {
	ref, ok := args["ref"].(string)
	if !ok || ref == "" {
		return nil, fmt.Errorf("ref is required")
	}
	
	action, ok := args["action"].(string)
	if !ok || action == "" {
		return nil, fmt.Errorf("action is required")
	}
	
	sessionID := "default"
	if sid, ok := args["session_id"].(string); ok && sid != "" {
		sessionID = sid
	}
	
	value := ""
	if v, ok := args["value"].(string); ok {
		value = v
	}
	
	_ = false // confirm flag not used in this version
	if c, ok := args["confirm"].(bool); ok && c {
		// Confirmation logic handled elsewhere
		_ = c
	}
	
	session, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	
	// Validate action through middleware
	selector := fmt.Sprintf("[data-ref='%s']", ref)
	
	// Check if element exists and is of correct type
	elementType, err := s.getElementType(session, ref)
	if err != nil {
		return nil, fmt.Errorf("element lookup failed: %w", err)
	}
	
	// Validate action-type compatibility
	if err := s.validateAction(action, elementType); err != nil {
		return map[string]interface{}{
			"success":     false,
			"error_type":  types.ErrInvalidAction,
			"message":     err.Error(),
			"recoverable": true,
		}, nil
	}
	
	// Execute action
	switch action {
	case types.ActionClick:
		err = session.Click(selector)
	case types.ActionFill:
		err = session.Fill(selector, value)
	case types.ActionPress:
		err = session.Press(selector, value)
	case types.ActionSelect:
		err = session.SelectOption(selector, value)
	case types.ActionHover:
		err = session.Hover(selector)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
	
	if err != nil {
		return map[string]interface{}{
			"success":     false,
			"message":     err.Error(),
			"recoverable": true,
		}, nil
	}
	
	return map[string]interface{}{
		"success": true,
		"action":  action,
		"ref":     ref,
	}, nil
}

func (s *MCPServer) toolFindAndAct(args map[string]interface{}) (interface{}, error) {
	intent, ok := args["intent"].(string)
	if !ok || intent == "" {
		return nil, fmt.Errorf("intent is required")
	}
	
	action, ok := args["action"].(string)
	if !ok || action == "" {
		return nil, fmt.Errorf("action is required")
	}
	
	sessionID := "default"
	if sid, ok := args["session_id"].(string); ok && sid != "" {
		sessionID = sid
	}
	
	value := ""
	if v, ok := args["value"].(string); ok {
		value = v
	}
	
	session, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	
	// Use intent resolver to find element
	resolver := NewIntentResolver(s.db)
	ref, err := resolver.Resolve(session, intent)
	if err != nil {
		return nil, fmt.Errorf("could not resolve intent '%s': %w", intent, err)
	}
	
	// Execute the action
	return s.toolAct(map[string]interface{}{
		"ref":        ref,
		"action":     action,
		"value":      value,
		"session_id": sessionID,
	})
}

func (s *MCPServer) toolGetStatus(args map[string]interface{}) (interface{}, error) {
	sessionID := "default"
	if sid, ok := args["session_id"].(string); ok && sid != "" {
		sessionID = sid
	}
	
	session, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	
	return map[string]interface{}{
		"url":        session.URL,
		"title":      session.Title,
		"auth_state": session.AuthState,
		"page_state": session.PageState,
	}, nil
}

func (s *MCPServer) getElementType(session *browser.Session, ref string) (string, error) {
	elements := session.GetLastElements()
	for i := range elements {
		if elements[i].Ref == ref {
			return elements[i].Type, nil
		}
	}
	return "", fmt.Errorf("element %s not found in snapshot", ref)
}

func (s *MCPServer) validateAction(action, elementType string) error {
	switch action {
	case types.ActionClick:
		if elementType == "input" || elementType == "textarea" {
			return fmt.Errorf("cannot click an input element (did you mean to fill it?)")
		}
	case types.ActionFill:
		if elementType != "input" && elementType != "textarea" && elementType != "select" {
			return fmt.Errorf("cannot fill a %s element (only inputs, textareas, and selects can be filled)", elementType)
		}
	}
	return nil
}

func (s *MCPServer) sendResult(id interface{}, result interface{}) error {
	msg := MCPMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	return s.writeMessage(&msg)
}

func (s *MCPServer) sendError(id interface{}, code int, message string, data interface{}) error {
	msg := MCPMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	return s.writeMessage(&msg)
}

func (s *MCPServer) writeMessage(msg *MCPMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(s.writer, "%s\n", data)
	return err
}
