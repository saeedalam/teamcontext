package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/teamcontext/teamcontext/internal/search"
	"github.com/teamcontext/teamcontext/internal/storage"
	"github.com/teamcontext/teamcontext/internal/worker"
)

// SessionTracker tracks tool calls and accumulated context for auto-saving conversations
type SessionTracker struct {
	StartedAt           time.Time
	ToolCalls           int
	ToolNames           []string // last N tool names called
	FilesTouched        map[string]bool
	DecisionsMade       []string
	WarningsAdded       []string
	InsightsAdded       []string
	PatternsAdded       []string
	ActiveFeature       string // auto-detected from tool calls
	LastSaveAt          time.Time
	SaveCount           int
	ToolCallsAtLastSave int    // tool calls count at last checkpoint
	LastCheckpointID    string // ID of last auto-saved conversation
	FeaturesStarted     []string
	FeaturesArchived    []string
}

func newSessionTracker() *SessionTracker {
	return &SessionTracker{
		StartedAt:    time.Now(),
		FilesTouched: make(map[string]bool),
	}
}

// Server is the MCP server
type Server struct {
	jsonStore     *storage.JSONStore
	sqliteIndex   *storage.SQLiteIndex
	workerManager *worker.Manager
	basePath      string
	tools         map[string]ToolHandler
	tfidfEngine   *search.TFIDFEngine // lazy-loaded TF-IDF engine for semantic search
	session       *SessionTracker
}

// ToolHandler handles a tool call
type ToolHandler func(params json.RawMessage) (interface{}, error)

// Request is a JSON-RPC request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error is a JSON-RPC error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// InitializeResult is the result of initialize
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
}

// ServerInfo contains server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Capabilities contains server capabilities
type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// ToolsCapability contains tools capability
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolInfo describes a tool
type ToolInfo struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema describes tool input
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// Property describes a property
type Property struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// NewServer creates a new MCP server
func NewServer(basePath string) (*Server, error) {
	jsonStore := storage.NewJSONStore(basePath)

	sqliteIndex, err := storage.NewSQLiteIndex(basePath)
	if err != nil {
		return nil, err
	}

	// Create worker manager for background tasks
	workerManager := worker.NewManager(basePath, jsonStore, sqliteIndex)

	s := &Server{
		jsonStore:     jsonStore,
		sqliteIndex:   sqliteIndex,
		workerManager: workerManager,
		basePath:      basePath,
		tools:         make(map[string]ToolHandler),
		session:       newSessionTracker(),
	}

	s.registerTools()

	return s, nil
}

func (s *Server) registerTools() {
	// Query tools
	s.tools["query"] = s.handleQuery
	s.tools["get_context"] = s.handleGetContext
	s.tools["search"] = s.handleSearch
	s.tools["search_files"] = s.handleSearchFiles
	s.tools["search_code"] = s.handleSearchCode

	// Read tools
	s.tools["get_project"] = s.handleGetProject
	s.tools["get_feature"] = s.handleGetFeature
	s.tools["list_features"] = s.handleListFeatures
	s.tools["list_decisions"] = s.handleListDecisions
	s.tools["list_warnings"] = s.handleListWarnings
	s.tools["list_patterns"] = s.handleListPatterns
	s.tools["get_stats"] = s.handleGetStats
	s.tools["get_architecture"] = s.handleGetArchitecture
	s.tools["get_evolution_timeline"] = s.handleGetEvolutionTimeline

	// Write tools
	s.tools["index_file"] = s.handleIndexFile
	s.tools["add_decision"] = s.handleAddDecision
	s.tools["add_warning"] = s.handleAddWarning
	s.tools["add_insight"] = s.handleAddInsight
	s.tools["add_pattern"] = s.handleAddPattern
	s.tools["add_evolution_event"] = s.handleAddEvolutionEvent
	s.tools["save_conversation"] = s.handleSaveConversation
	s.tools["compact_conversation"] = s.handleCompactConversation
	s.tools["update_feature_state"] = s.handleUpdateFeatureState
	s.tools["update_architecture"] = s.handleUpdateArchitecture
	s.tools["update_project"] = s.handleUpdateProject

	// Feature lifecycle tools
	s.tools["start_feature"] = s.handleStartFeature
	s.tools["archive_feature"] = s.handleArchiveFeature
	s.tools["recall_feature"] = s.handleRecallFeature

	// Knowledge graph traversal
	s.tools["get_related"] = s.handleGetRelated

	// Indexing tools
	s.tools["index"] = s.handleIndex
	s.tools["index_status"] = s.handleIndexStatus
	s.tools["get_graph"] = s.handleGetGraph

	// Analysis tools
	s.tools["scan_imports"] = s.handleScanImports
	s.tools["get_code_map"] = s.handleGetCodeMap
	s.tools["get_tree"] = s.handleGetTree
	s.tools["get_dependencies"] = s.handleGetDependencies
	s.tools["trace_flow"] = s.handleTraceFlow

	// Token-saving tools
	s.tools["get_signature"] = s.handleGetSignature
	s.tools["get_skeleton"] = s.handleGetSkeleton
	s.tools["get_types"] = s.handleGetTypes
	s.tools["search_snippets"] = s.handleSearchSnippets
	s.tools["get_recent_changes"] = s.handleGetRecentChanges
	s.tools["resume_context"] = s.handleResumeContext
	s.tools["list_conversations"] = s.handleListConversations
	s.tools["get_task_context"] = s.handleGetTaskContext

	// High-impact extraction tools
	s.tools["get_api_surface"] = s.handleGetAPISurface
	s.tools["get_schema_models"] = s.handleGetSchemaModels
	s.tools["get_config_map"] = s.handleGetConfigMap
	s.tools["get_blueprint"] = s.handleGetBlueprint

	// Compliance & onboarding tools
	s.tools["check_compliance"] = s.handleCheckCompliance
	s.tools["onboard"] = s.handleOnboard
	s.tools["get_feed"] = s.handleGetFeed

	// Git Intelligence tools
	s.tools["find_experts"] = s.handleFindExperts
	s.tools["get_file_history"] = s.handleGetFileHistory
	s.tools["get_knowledge_risks"] = s.handleGetKnowledgeRisks
	s.tools["get_file_correlations"] = s.handleGetFileCorrelations
	s.tools["get_commit_context"] = s.handleGetCommitContext
}

// Run starts the MCP server
func (s *Server) Run() {
	// Start background workers if enabled
	workerConfig := s.workerManager.GetConfig()
	if workerConfig.Enabled {
		if err := s.workerManager.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not start workers: %v\n", err)
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer size for large messages
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			// Don't send error with null ID - Cursor rejects it
			fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
			continue
		}

		s.handleRequest(&req)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Scanner error: %v\n", err)
	}

	// Graceful shutdown: auto-save session, stop workers, close storage
	s.autoSaveSession("session_end")

	if s.workerManager != nil && s.workerManager.IsRunning() {
		s.workerManager.Stop()
	}
	if s.sqliteIndex != nil {
		s.sqliteIndex.Close()
	}
}

func (s *Server) handleRequest(req *Request) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "initialized":
		// No response needed
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	default:
		s.sendError(req.ID, -32601, "Method not found", req.Method)
	}
}

func (s *Server) handleInitialize(req *Request) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    "teamcontext",
			Version: "0.2.0",
		},
		Capabilities: Capabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
		},
	}
	s.sendResult(req.ID, result)
}

func (s *Server) HandleToolCall(name string, params json.RawMessage) (interface{}, error) {
	handler, ok := s.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return handler(params)
}

func (s *Server) handleToolsCall(req *Request) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	handler, ok := s.tools[params.Name]
	if !ok {
		s.sendError(req.ID, -32601, "Tool not found", params.Name)
		return
	}

	// Track session activity
	s.trackToolCall(params.Name, params.Arguments)

	result, err := handler(params.Arguments)
	if err != nil {
		s.sendResult(req.ID, map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Error: %v", err),
				},
			},
			"isError": true,
		})
		return
	}

	// Post-call: track IDs of knowledge items created
	s.trackResultIDs(params.Name, result)

	// Format result as text content
	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	s.sendResult(req.ID, map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": string(resultJSON),
			},
		},
	})

	// Auto-capture replaces the AI-driven save instruction
	s.checkAutoCaptureTriggers(params.Name)
}

func (s *Server) sendResult(id interface{}, result interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.send(resp)
}

func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	// Don't send error responses for notifications (null/nil ID)
	if id == nil {
		fmt.Fprintf(os.Stderr, "Error (no id): %s: %v\n", message, data)
		return
	}
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.send(resp)
}

func (s *Server) send(resp Response) {
	output, _ := json.Marshal(resp)
	fmt.Println(string(output))
}

// --- Helper Functions ---

func (s *Server) loadGitKnowledge(filename string, target interface{}) error {
	knowledgePath := filepath.Join(s.basePath, "knowledge", filename)
	data, err := os.ReadFile(knowledgePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func containsAny(s string, query string) bool {
	s = strings.ToLower(s)
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return false
	}
	if strings.Contains(s, query) {
		return true
	}
	for _, word := range strings.Fields(query) {
		if strings.Contains(s, word) {
			return true
		}
	}
	return false
}

func hasOverlap(a, b []string) bool {
	m := make(map[string]bool)
	for _, item := range a {
		m[item] = true
	}
	for _, item := range b {
		if m[item] {
			return true
		}
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
