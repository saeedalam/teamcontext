package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/saeedalam/teamcontext/internal/search"
	"github.com/saeedalam/teamcontext/internal/storage"
	"github.com/saeedalam/teamcontext/internal/worker"
	"github.com/saeedalam/teamcontext/pkg/types"
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
	Type       string                 `json:"type"`
	Properties map[string]Property    `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
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

	// Start background workers if enabled
	workerConfig := workerManager.GetConfig()
	if workerConfig.Enabled {
		if err := workerManager.Start(); err != nil {
			// Log but don't fail - workers are optional
			fmt.Fprintf(os.Stderr, "Warning: could not start workers: %v\n", err)
		}
	}

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
	s.tools["get_dependencies"] = s.handleGetDependencies
	s.tools["trace_flow"] = s.handleTraceFlow

	// Token-saving tools
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

// --- Tool Handlers ---

func (s *Server) handleSearchFiles(params json.RawMessage) (interface{}, error) {
	var p struct {
		Query    string `json:"query"`
		Language string `json:"language"`
		Limit    int    `json:"limit"`
	}
	json.Unmarshal(params, &p)

	files, err := s.sqliteIndex.SearchFiles(p.Query, p.Language, p.Limit)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"files": files,
		"total": len(files),
	}, nil
}

func (s *Server) handleSearchCode(params json.RawMessage) (interface{}, error) {
	var p struct {
		Pattern string `json:"pattern"`
		Glob    string `json:"glob"`
		Limit   int    `json:"limit"`
	}
	json.Unmarshal(params, &p)

	if p.Pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	// Get the project root (parent of .teamcontext)
	projectRoot := s.basePath[:len(s.basePath)-len("/.teamcontext")]

	matches, err := search.SearchCode(p.Pattern, projectRoot, p.Glob, p.Limit)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"matches": matches,
		"total":   len(matches),
	}, nil
}

func (s *Server) handleGetProject(params json.RawMessage) (interface{}, error) {
	project, err := s.jsonStore.GetProject()
	if err != nil {
		// Return empty project if not found
		return &types.Project{}, nil
	}
	return project, nil
}

func (s *Server) handleGetFeature(params json.RawMessage) (interface{}, error) {
	var p struct {
		ID string `json:"id"`
	}
	json.Unmarshal(params, &p)

	if p.ID == "" {
		return nil, fmt.Errorf("id is required")
	}

	feature, err := s.jsonStore.GetFeature(p.ID)
	if err != nil {
		return nil, err
	}

	// Get feature-specific decisions
	decisions, _ := s.jsonStore.GetDecisionsByFeature(p.ID)

	// Get conversations
	conversations, _ := s.jsonStore.GetConversations(p.ID)

	return map[string]interface{}{
		"feature":       feature,
		"decisions":     decisions,
		"conversations": conversations,
	}, nil
}

func (s *Server) handleListFeatures(params json.RawMessage) (interface{}, error) {
	var p struct {
		Status string `json:"status"`
	}
	json.Unmarshal(params, &p)

	features, err := s.jsonStore.GetFeatures()
	if err != nil {
		return nil, err
	}

	// Filter by status if provided
	if p.Status != "" {
		var filtered []types.Feature
		for _, f := range features {
			if f.Status == p.Status {
				filtered = append(filtered, f)
			}
		}
		features = filtered
	}

	// Convert to summaries
	var summaries []types.FeatureSummary
	for _, f := range features {
		summaries = append(summaries, types.FeatureSummary{
			ID:           f.ID,
			Status:       f.Status,
			CurrentState: f.CurrentState,
			LastAccessed: f.LastAccessed,
		})
	}

	return map[string]interface{}{
		"features": summaries,
	}, nil
}

func (s *Server) handleListDecisions(params json.RawMessage) (interface{}, error) {
	var p struct {
		Feature string `json:"feature"`
		Status  string `json:"status"`
		Limit   int    `json:"limit"`
	}
	json.Unmarshal(params, &p)

	decisions, err := s.sqliteIndex.SearchDecisions("", p.Feature, p.Status, p.Limit)
	if err != nil {
		// Fallback to JSON store
		decisions, err = s.jsonStore.GetDecisions()
		if err != nil {
			return nil, err
		}
	}

	return map[string]interface{}{
		"decisions": decisions,
		"total":     len(decisions),
	}, nil
}

func (s *Server) handleListWarnings(params json.RawMessage) (interface{}, error) {
	var p struct {
		Feature  string `json:"feature"`
		Severity string `json:"severity"`
	}
	json.Unmarshal(params, &p)

	warnings, err := s.sqliteIndex.SearchWarnings("", p.Feature, p.Severity, 50)
	if err != nil {
		// Fallback to JSON store
		warnings, err = s.jsonStore.GetWarnings()
		if err != nil {
			return nil, err
		}
	}

	return map[string]interface{}{
		"warnings": warnings,
		"total":    len(warnings),
	}, nil
}

// and unmarshals it into the target. Returns an error if the file doesn't exist
// or can't be parsed.
func (s *Server) loadGitKnowledge(filename string, target interface{}) error {
	knowledgeDir := filepath.Join(s.basePath, "knowledge")
	data, err := os.ReadFile(filepath.Join(knowledgeDir, filename))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func containsAny(text, query string) bool {
	// Simple case-insensitive substring match
	text = strings.ToLower(text)
	query = strings.ToLower(query)
	words := strings.Fields(query)
	for _, word := range words {
		if strings.Contains(text, word) {
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

func hasOverlap(a, b []string) bool {
	for _, aItem := range a {
		for _, bItem := range b {
			if aItem == bItem {
				return true
			}
		}
	}
	return false
}

