package types

import "time"

// =============================================================================
// CORE KNOWLEDGE TYPES (TeamContext)
// =============================================================================

// FileIndex represents indexed file metadata
type FileIndex struct {
	Path           string    `json:"path"`
	Summary        string    `json:"summary"`
	Exports        []Export  `json:"exports,omitempty"`
	Imports        []string  `json:"imports,omitempty"`
	Dependencies   []string  `json:"dependencies,omitempty"` // Internal dependencies
	Language       string    `json:"language,omitempty"`
	Patterns       []string  `json:"patterns,omitempty"` // Pattern IDs this file follows
	RelatedFiles   []string  `json:"related_files,omitempty"`
	ContentHash    string    `json:"content_hash,omitempty"`
	SizeBytes      int64     `json:"size_bytes,omitempty"`
	LineCount      int       `json:"line_count,omitempty"`
	IndexedAt      time.Time `json:"indexed_at"`
}

// Export represents an exported symbol from a file
type Export struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // function, class, struct, type, const, variable
	Line int    `json:"line,omitempty"`
}

// Decision represents a recorded decision with full reasoning
type Decision struct {
	ID               string    `json:"id"`
	Content          string    `json:"content"`
	Reason           string    `json:"reason"`
	Context          string    `json:"context,omitempty"`
	Alternatives     []string  `json:"alternatives,omitempty"` // What else was considered
	Feature          string    `json:"feature,omitempty"`
	Author           string    `json:"author,omitempty"`
	Status           string    `json:"status"` // active, superseded, archived
	Supersedes       string    `json:"supersedes,omitempty"`
	RelatedFiles     []string  `json:"related_files,omitempty"`
	RelatedDecisions []string  `json:"related_decisions,omitempty"`
	Tags             []string  `json:"tags,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// Warning represents a documented pitfall to avoid
type Warning struct {
	ID               string    `json:"id"`
	Content          string    `json:"content"`
	Reason           string    `json:"reason"`
	Evidence         string    `json:"evidence,omitempty"`
	Severity         string    `json:"severity"` // info, warning, critical
	Feature          string    `json:"feature,omitempty"`
	Author           string    `json:"author,omitempty"`
	RelatedFiles     []string  `json:"related_files,omitempty"`
	RelatedDecisions []string  `json:"related_decisions,omitempty"`
	Tags             []string  `json:"tags,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// Insight represents a captured insight
type Insight struct {
	ID           string    `json:"id"`
	Content      string    `json:"content"`
	Context      string    `json:"context,omitempty"`
	Feature      string    `json:"feature,omitempty"`
	Author       string    `json:"author,omitempty"`
	RelatedFiles []string  `json:"related_files,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Pattern represents a recognized way of doing things
type Pattern struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Examples     []string `json:"examples,omitempty"`      // File paths showing pattern
	Rules        []string `json:"rules,omitempty"`         // What defines this pattern
	AntiPatterns []string `json:"anti_patterns,omitempty"` // What violates it
	Confidence   float64  `json:"confidence,omitempty"`    // For auto-detected patterns
	Source       string   `json:"source"`                  // manual, detected
	CreatedAt    time.Time `json:"created_at"`
}

// Feature represents a feature context (working memory)
type Feature struct {
	ID             string    `json:"id"`
	Status         string    `json:"status"` // active, paused, archived
	Description    string    `json:"description,omitempty"`
	Branch         string    `json:"branch,omitempty"`
	Extends        string    `json:"extends,omitempty"` // Parent feature for inheritance
	Owner          string    `json:"owner,omitempty"`
	CurrentState   string    `json:"current_state,omitempty"`
	RelevantFiles  []string  `json:"relevant_files,omitempty"`
	Decisions      []string  `json:"decisions,omitempty"`      // Decision IDs
	Warnings       []string  `json:"warnings,omitempty"`       // Warning IDs
	Contributors   []string  `json:"contributors,omitempty"`
	ArchiveSummary string    `json:"archive_summary,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	LastAccessed   time.Time `json:"last_accessed"`
	ArchivedAt     time.Time `json:"archived_at,omitempty"`
}

// Conversation represents a compressed conversation
type Conversation struct {
	ID              string    `json:"id"`
	Feature         string    `json:"feature"`
	Summary         string    `json:"summary"`
	KeyPoints       []string  `json:"key_points,omitempty"`
	FilesDiscussed  []string  `json:"files_discussed,omitempty"`
	DecisionsMade   []string  `json:"decisions_made,omitempty"`
	OriginalTokens  int       `json:"original_tokens,omitempty"`
	CompressedTokens int      `json:"compressed_tokens,omitempty"`
	StartTime       time.Time `json:"start_time,omitempty"`
	EndTime         time.Time `json:"end_time,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// =============================================================================
// KNOWLEDGE GRAPH TYPES
// =============================================================================

// KnowledgeGraph represents the connected knowledge structure
type KnowledgeGraph struct {
	Edges []Edge `json:"edges"`
}

// Edge represents a relationship between two nodes
type Edge struct {
	FromType string `json:"from_type"` // decision, warning, pattern, file, feature
	FromID   string `json:"from_id"`
	ToType   string `json:"to_type"`
	ToID     string `json:"to_id"`
	Relation string `json:"relation"` // affects, warns, follows, supersedes, related_to, belongs_to
}

// =============================================================================
// EVOLUTION TIMELINE TYPES
// =============================================================================

// EvolutionEvent represents a point in the evolution timeline
type EvolutionEvent struct {
	ID          string    `json:"id"`
	EventType   string    `json:"event_type"` // decision, architecture_change, pattern_adopted, milestone, warning
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Author      string    `json:"author,omitempty"`
	Impact      string    `json:"impact"` // low, medium, high
	RelatedIDs  []string  `json:"related_ids,omitempty"` // Decision/warning IDs
	Timestamp   time.Time `json:"timestamp"`
}

// EvolutionTimeline holds all evolution events
type EvolutionTimeline struct {
	Events []EvolutionEvent `json:"events"`
}

// =============================================================================
// ARCHITECTURE TYPES
// =============================================================================

// Architecture represents high-level system structure
type Architecture struct {
	Description string        `json:"description"`
	Diagram     string        `json:"diagram,omitempty"` // ASCII or mermaid
	Services    []ServiceNode `json:"services,omitempty"`
	DataFlows   []DataFlow    `json:"data_flows,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// ServiceNode represents a service in the architecture
type ServiceNode struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Files        []string `json:"files,omitempty"`        // Files that belong to this service
	Dependencies []string `json:"dependencies,omitempty"` // Other services it depends on
}

// DataFlow represents data movement between services
type DataFlow struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Description string `json:"description"`
}

// ApiEndpoint represents an API endpoint
type ApiEndpoint struct {
	Method  string `json:"method"` // GET, POST, etc.
	Path    string `json:"path"`   // /api/payments/:id
	Handler string `json:"handler,omitempty"`
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
}

// =============================================================================
// PROJECT & CONFIG TYPES
// =============================================================================

// Project represents project-level knowledge
type Project struct {
	Name         string       `json:"name"`
	Description  string       `json:"description,omitempty"`
	Languages    []string     `json:"languages,omitempty"`
	Architecture *Architecture `json:"architecture,omitempty"`
	Goals        []Goal       `json:"goals,omitempty"`
	TeamMindset  TeamMindset  `json:"team_mindset,omitempty"`
	Resources    []Resource   `json:"resources,omitempty"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Goal represents a project goal
type Goal struct {
	Type        string `json:"type"` // business, technical
	Description string `json:"description"`
	Status      string `json:"status,omitempty"` // active, completed, abandoned
}

// TeamMindset represents team conventions
type TeamMindset struct {
	CodingStandards string   `json:"coding_standards,omitempty"`
	ReviewProcess   string   `json:"review_process,omitempty"`
	Conventions     string   `json:"conventions,omitempty"`
	DontDo          []string `json:"dont_do,omitempty"` // Things to avoid
	AlwaysDo        []string `json:"always_do,omitempty"` // Things to always do
}

// Resource represents a project resource
type Resource struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Type        string `json:"type,omitempty"`        // docs, api, repo
	Description string `json:"description,omitempty"`
}

// Config represents TeamContext configuration
type Config struct {
	Name        string       `json:"name"`
	Version     string       `json:"version"`
	CreatedAt   time.Time    `json:"created_at"`
	Index       IndexConfig  `json:"index,omitempty"`
	Server      ServerConfig `json:"server,omitempty"`
	LinkedRepos []string     `json:"linked_repos,omitempty"` // sibling repo paths for cross-repo activity
}

// IndexConfig represents indexing configuration
type IndexConfig struct {
	Exclude    []string `json:"exclude,omitempty"`    // Patterns to exclude
	Include    []string `json:"include,omitempty"`    // Patterns to include
	MaxFileSize int64   `json:"max_file_size,omitempty"` // Max file size in bytes
}

// ServerConfig represents server configuration
type ServerConfig struct {
	MCPEnabled  bool   `json:"mcp_enabled"`
	RESTEnabled bool   `json:"rest_enabled"`
	RESTPort    int    `json:"rest_port,omitempty"`
}

// =============================================================================
// SEARCH & QUERY TYPES
// =============================================================================

// CodeMatch represents a code search result
type CodeMatch struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
	Context string `json:"context,omitempty"` // Surrounding lines
}

// FeatureSummary is a lightweight feature representation
type FeatureSummary struct {
	ID           string    `json:"id"`
	Status       string    `json:"status"`
	CurrentState string    `json:"current_state,omitempty"`
	Owner        string    `json:"owner,omitempty"`
	LastAccessed time.Time `json:"last_accessed"`
}

// QueryResponse represents a response to a knowledge query
type QueryResponse struct {
	Answer        string            `json:"answer,omitempty"`
	Sources       []Source          `json:"sources,omitempty"`
	Decisions     []Decision        `json:"decisions,omitempty"`
	Warnings      []Warning         `json:"warnings,omitempty"`
	Files         []FileIndex       `json:"files,omitempty"`
	Patterns      []Pattern         `json:"patterns,omitempty"`
	GitExperts    []GitExpertHit    `json:"git_experts,omitempty"`
	Conversations []Conversation    `json:"conversations,omitempty"`
	TokensUsed    int               `json:"tokens_used,omitempty"`
	TokensSaved   int               `json:"tokens_saved,omitempty"`
}

// GitExpertHit represents a git expert matched by a query
type GitExpertHit struct {
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Area      string  `json:"area"`
	Ownership float64 `json:"ownership"`
	Active    bool    `json:"active"`
}

// Source represents a source reference in a query response
type Source struct {
	Type      string  `json:"type"` // decision, warning, file, conversation
	ID        string  `json:"id"`
	Relevance float64 `json:"relevance,omitempty"` // 0-1
}

// ContextResponse represents relevant context for an intent
type ContextResponse struct {
	Intent       string        `json:"intent"`
	Decisions    []Decision    `json:"decisions,omitempty"`
	Warnings     []Warning     `json:"warnings,omitempty"`
	Patterns     []Pattern     `json:"patterns,omitempty"`
	Files        []string      `json:"files,omitempty"` // Recommended files to load
	Suggestions  []string      `json:"suggestions,omitempty"`
	TokenBudget  *TokenBudget  `json:"token_budget,omitempty"`
	GitExperts   []GitExpertHit `json:"git_experts,omitempty"`
}

// TokenBudget tracks context loading token usage
type TokenBudget struct {
	Requested int `json:"requested"`
	Used      int `json:"used"`
	Remaining int `json:"remaining"`
}

// ImportResult represents a parsed import from a source file
type ImportResult struct {
	Source     string `json:"source"`
	Imported   string `json:"imported"`
	ImportType string `json:"import_type"` // "relative", "package", "builtin"
	Raw        string `json:"raw"`
}

// CodeMapNode represents a node in the code map tree
type CodeMapNode struct {
	Name     string        `json:"name"`
	Path     string        `json:"path"`
	Type     string        `json:"type"` // "directory" or "file"
	Summary  string        `json:"summary,omitempty"`
	Language string        `json:"language,omitempty"`
	Exports  int           `json:"exports,omitempty"`
	Children []CodeMapNode `json:"children,omitempty"`
}

// DependencyNode represents a node in the dependency tree
type DependencyNode struct {
	Path     string           `json:"path"`
	Summary  string           `json:"summary,omitempty"`
	Language string           `json:"language,omitempty"`
	Depth    int              `json:"depth"`
	Children []DependencyNode `json:"children,omitempty"`
}

// FlowNode represents a node in a flow trace
type FlowNode struct {
	Path      string   `json:"path"`
	Summary   string   `json:"summary,omitempty"`
	Language  string   `json:"language,omitempty"`
	Exports   []Export `json:"exports,omitempty"`
	Depth     int      `json:"depth"`
	Relevance float64  `json:"relevance,omitempty"`
}

// =============================================================================
// TOKEN-SAVING TYPES (Skeleton, Snippets, Types)
// =============================================================================

// CodeSkeleton represents a file's structure without implementation details
// This saves 90%+ tokens when understanding code structure
type CodeSkeleton struct {
	Path       string          `json:"path"`
	Language   string          `json:"language"`
	Summary    string          `json:"summary,omitempty"`
	Classes    []ClassSkeleton `json:"classes,omitempty"`
	Functions  []FunctionSig   `json:"functions,omitempty"`
	Interfaces []TypeDef       `json:"interfaces,omitempty"`
	Types      []TypeDef       `json:"types,omitempty"`
	Enums      []EnumDef       `json:"enums,omitempty"`
	Constants  []ConstDef      `json:"constants,omitempty"`
	LineCount  int             `json:"line_count"`
	SkeletonLines int          `json:"skeleton_lines"` // Lines in skeleton vs original
}

// ClassSkeleton represents a class without method bodies
type ClassSkeleton struct {
	Name       string        `json:"name"`
	Line       int           `json:"line"`
	Extends    string        `json:"extends,omitempty"`
	Implements []string      `json:"implements,omitempty"`
	IsAbstract bool          `json:"is_abstract,omitempty"`
	IsExported bool          `json:"is_exported,omitempty"`
	Properties []PropertyDef `json:"properties,omitempty"`
	Methods    []FunctionSig `json:"methods,omitempty"`
	Constructor *FunctionSig `json:"constructor,omitempty"`
}

// FunctionSig represents a function/method signature
type FunctionSig struct {
	Name       string      `json:"name"`
	Line       int         `json:"line"`
	Params     []ParamDef  `json:"params,omitempty"`
	ReturnType string      `json:"return_type,omitempty"`
	IsAsync    bool        `json:"is_async,omitempty"`
	IsStatic   bool        `json:"is_static,omitempty"`
	IsPrivate  bool        `json:"is_private,omitempty"`
	IsExported bool        `json:"is_exported,omitempty"`
	Decorators []string    `json:"decorators,omitempty"`
	DocComment string      `json:"doc_comment,omitempty"`
}

// ParamDef represents a function parameter
type ParamDef struct {
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	Default  string `json:"default,omitempty"`
}

// PropertyDef represents a class property
type PropertyDef struct {
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	IsPrivate  bool   `json:"is_private,omitempty"`
	IsReadonly bool   `json:"is_readonly,omitempty"`
	IsStatic   bool   `json:"is_static,omitempty"`
}

// TypeDef represents a TypeScript interface or type alias
type TypeDef struct {
	Name       string        `json:"name"`
	Line       int           `json:"line"`
	Kind       string        `json:"kind"` // "interface", "type", "alias"
	IsExported bool          `json:"is_exported,omitempty"`
	Extends    []string      `json:"extends,omitempty"`
	Properties []PropertyDef `json:"properties,omitempty"`
	RawDef     string        `json:"raw_def,omitempty"` // For complex type aliases
}

// EnumDef represents an enum definition
type EnumDef struct {
	Name       string   `json:"name"`
	Line       int      `json:"line"`
	IsExported bool     `json:"is_exported,omitempty"`
	Members    []string `json:"members,omitempty"`
}

// ConstDef represents a constant definition
type ConstDef struct {
	Name       string `json:"name"`
	Line       int    `json:"line"`
	Type       string `json:"type,omitempty"`
	Value      string `json:"value,omitempty"`
	IsExported bool   `json:"is_exported,omitempty"`
}

// CodeSnippet represents a relevant code fragment from search
type CodeSnippet struct {
	Path       string  `json:"path"`
	StartLine  int     `json:"start_line"`
	EndLine    int     `json:"end_line"`
	Content    string  `json:"content"`
	Context    string  `json:"context,omitempty"` // Why this snippet is relevant
	Relevance  float64 `json:"relevance"`
	TokenCount int     `json:"token_count,omitempty"`
}

// =============================================================================
// GIT INTEGRATION TYPES
// =============================================================================

// GitChange represents a recent change in the repository
type GitChange struct {
	Hash          string    `json:"hash"`
	ShortHash     string    `json:"short_hash"`
	Message       string    `json:"message"`
	Author        string    `json:"author"`
	AuthorEmail   string    `json:"author_email,omitempty"`
	Date          time.Time `json:"date"`
	FilesChanged  []string  `json:"files_changed,omitempty"`
	Insertions    int       `json:"insertions"`
	Deletions     int       `json:"deletions"`
	AISummary     string    `json:"ai_summary,omitempty"`     // Brief AI-generated summary
	ImpactLevel   string    `json:"impact_level,omitempty"`   // low, medium, high
	RelatedFeature string   `json:"related_feature,omitempty"`
}

// GitDiff represents a diff for a specific file
type GitDiff struct {
	Path       string   `json:"path"`
	Status     string   `json:"status"` // added, modified, deleted, renamed
	OldPath    string   `json:"old_path,omitempty"` // For renames
	Insertions int      `json:"insertions"`
	Deletions  int      `json:"deletions"`
	Hunks      []string `json:"hunks,omitempty"` // Diff hunks
}

// =============================================================================
// CONTEXT BUNDLE TYPES
// =============================================================================

// TaskContextBundle represents precomputed context for common tasks
type TaskContextBundle struct {
	Task        string          `json:"task"`         // e.g., "add-api-endpoint"
	Domain      string          `json:"domain,omitempty"` // e.g., "notification"
	Patterns    []string        `json:"patterns,omitempty"`
	ExampleCode []CodeSnippet   `json:"example_code,omitempty"`
	TypesNeeded []TypeDef       `json:"types_needed,omitempty"`
	Warnings    []string        `json:"warnings,omitempty"`
	Checklist   []string        `json:"checklist,omitempty"`
	RelatedDocs []string        `json:"related_docs,omitempty"`
	TokensSaved int             `json:"tokens_saved,omitempty"`
}

// =============================================================================
// ENHANCED CONVERSATION MEMORY
// =============================================================================

// ConversationMemory extends Conversation for session resume
type ConversationMemory struct {
	Conversation
	NextSteps      []string      `json:"next_steps,omitempty"`
	OpenQuestions  []string      `json:"open_questions,omitempty"`
	CodeSnippets   []CodeSnippet `json:"code_snippets,omitempty"`
	Context        string        `json:"context,omitempty"` // What led to this session
	Blockers       []string      `json:"blockers,omitempty"`
	ProgressPct    int           `json:"progress_pct,omitempty"` // 0-100
}

// Stats represents system statistics
type Stats struct {
	FilesIndexed    int `json:"files_indexed"`
	Decisions       int `json:"decisions"`
	Warnings        int `json:"warnings"`
	Patterns        int `json:"patterns"`
	Insights        int `json:"insights"`
	Features        int `json:"features"`
	ActiveFeatures  int `json:"active_features"`
	ArchivedFeatures int `json:"archived_features"`
	Conversations   int `json:"conversations"`
}

// =============================================================================
// CONVERSATION HOOKS (Auto-delegation to AI)
// =============================================================================

// ConversationHook instructs the AI agent to automatically call a tool
// These hooks are returned in MCP tool responses and tell the AI what to do next
type ConversationHook struct {
	Action    string                 `json:"action"`     // Human-readable action name
	Tool      string                 `json:"tool"`       // MCP tool to call
	Arguments map[string]interface{} `json:"arguments"`  // Pre-filled or template arguments
	Priority  string                 `json:"priority"`   // "auto" (do immediately), "suggested" (ask first), "optional"
	Reason    string                 `json:"reason"`     // Why this hook was triggered
	Condition string                 `json:"condition,omitempty"` // Optional condition for the AI to evaluate
}

// Common hook actions
const (
	HookActionSaveConversation = "save_conversation"      // Save/compact conversation memory
	HookActionAddDecision      = "add_decision"           // Record an architectural decision
	HookActionAddInsight       = "add_insight"            // Record a learned insight
	HookActionAddWarning       = "add_warning"            // Record a discovered pitfall
	HookActionIndexFile        = "index_file"             // Index a modified/new file
	HookActionUpdateFeature    = "update_feature"         // Update feature progress
	HookActionNotifyUser       = "notify_user"            // Prompt user about something
)

// Hook priorities
const (
	HookPriorityAuto      = "auto"      // Execute immediately without asking
	HookPrioritySuggested = "suggested" // Suggest to user, execute if approved
	HookPriorityOptional  = "optional"  // Low priority, user can skip
)

// HookableResponse is a wrapper that tool handlers can use to return hooks
type HookableResponse struct {
	Data  interface{}        `json:"data"`
	Hooks []ConversationHook `json:"_hooks,omitempty"` // Underscore prefix = metadata
}

// ConversationCompaction represents a compacted conversation summary
type ConversationCompaction struct {
	OriginalID      string    `json:"original_id"`
	Summary         string    `json:"summary"`
	KeyDecisions    []string  `json:"key_decisions,omitempty"`
	KeyInsights     []string  `json:"key_insights,omitempty"`
	FilesDiscussed  []string  `json:"files_discussed,omitempty"`
	OpenItems       []string  `json:"open_items,omitempty"`
	MessageCount    int       `json:"message_count"`
	CompactedAt     time.Time `json:"compacted_at"`
	TokensSaved     int       `json:"tokens_saved,omitempty"`
}
