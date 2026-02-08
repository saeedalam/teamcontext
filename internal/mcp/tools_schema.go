package mcp

// handleToolsList returns the schema definitions for all 54 MCP tools.
// This is called when the client requests "tools/list".

func (s *Server) handleToolsList(req *Request) {
	tools := []ToolInfo{
		// === SEARCH & QUERY TOOLS ===
		// Use these to find information before making changes
		{
			Name:        "query",
			Description: "ASK A QUESTION about the codebase. Use for ANY question: 'who developed X?', 'how does X work?', 'what are the risks in X?'. Returns relevant files, decisions, warnings, AND git experts (who owns/developed each area with ownership %). Always try this first.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"question":     {Type: "string", Description: "Your question in plain English"},
					"scope":        {Type: "string", Description: "Optional: 'feature:ID' to limit to a feature, default searches everything"},
					"include_code": {Type: "boolean", Description: "Set true to include matching code snippets"},
				},
				Required: []string{"question"},
			},
		},
		{
			Name:        "get_context",
			Description: "GET CONTEXT BEFORE MODIFYING CODE. Use before making changes. Returns decisions, warnings, patterns, related files, AND suggests experts to consult (with ownership %). Supports token budgeting and relevance ranking.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"intent":            {Type: "string", Description: "What you plan to do, e.g., 'add new API endpoint for users'"},
					"target_files":      {Type: "array", Description: "List of file paths you plan to modify"},
					"proposed_approach": {Type: "string", Description: "Optional: your planned approach, system will validate against existing decisions"},
					"max_tokens":        {Type: "integer", Description: "Maximum token budget for context (default 8000). Results ranked by relevance and trimmed to fit."},
				},
				Required: []string{"intent"},
			},
		},
		{
			Name:        "search",
			Description: "SEARCH ALL KNOWLEDGE. Use when you need to find specific information across files, decisions, warnings, and patterns.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {Type: "string", Description: "What to search for"},
					"types": {Type: "array", Description: "Optional filter: ['file', 'decision', 'warning', 'pattern']"},
					"limit": {Type: "integer", Description: "Max results, default 20"},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "search_files",
			Description: "FIND FILES by name, content, or description. Returns file paths with summaries. Use when you need to locate files.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query":    {Type: "string", Description: "Search term (matches path, summary, exports)"},
					"language": {Type: "string", Description: "Optional: 'typescript', 'go', 'python', etc."},
					"limit":    {Type: "integer", Description: "Max results, default 20"},
				},
			},
		},
		{
			Name:        "search_code",
			Description: "SEARCH CODE CONTENT with regex. Use when you need to find specific code patterns, function calls, or variable usage.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"pattern":       {Type: "string", Description: "Regex pattern to match"},
					"glob":          {Type: "string", Description: "Optional file filter: '*.ts', 'src/**/*.go'"},
					"context_lines": {Type: "integer", Description: "Lines of context around match, default 2"},
					"limit":         {Type: "integer", Description: "Max results, default 50"},
				},
				Required: []string{"pattern"},
			},
		},
		// === READ TOOLS ===
		// Use these to read existing knowledge
		{
			Name:        "get_project",
			Description: "GET PROJECT OVERVIEW. Use at the start of a session to understand the project's architecture, patterns, goals, and team conventions.",
			InputSchema: InputSchema{
				Type: "object",
			},
		},
		{
			Name:        "get_feature",
			Description: "GET FEATURE DETAILS. Use when working on a specific feature to see its current state, decisions, warnings, and conversation history.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {Type: "string", Description: "Feature ID (e.g., 'auth-refactor', 'payment-v2')"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "list_features",
			Description: "LIST ALL FEATURES. Use to see what work is in progress, paused, or completed.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"status": {Type: "string", Description: "Filter: 'active', 'paused', or 'archived'"},
				},
			},
		},
		{
			Name:        "list_decisions",
			Description: "LIST DECISIONS made in this project. Use to understand why things are done a certain way.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"feature": {Type: "string", Description: "Filter by feature ID"},
					"status":  {Type: "string", Description: "Filter: 'active' or 'superseded'"},
					"tags":    {Type: "array", Description: "Filter by tags"},
					"limit":   {Type: "integer", Description: "Max results, default 50"},
				},
			},
		},
		{
			Name:        "list_warnings",
			Description: "LIST WARNINGS about things to avoid. Use before making changes to check for known pitfalls.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"feature":  {Type: "string", Description: "Filter by feature ID"},
					"severity": {Type: "string", Description: "Filter: 'info', 'warning', 'critical'"},
				},
			},
		},
		{
			Name:        "list_patterns",
			Description: "LIST CODE PATTERNS used in this project. Use to understand conventions to follow.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"source": {Type: "string", Description: "Filter: 'manual' or 'detected'"},
				},
			},
		},
		{
			Name:        "get_stats",
			Description: "GET INDEX STATISTICS. Shows how much knowledge is indexed (files, decisions, warnings, etc.).",
			InputSchema: InputSchema{
				Type: "object",
			},
		},
		{
			Name:        "get_architecture",
			Description: "GET SYSTEM ARCHITECTURE. Shows the high-level structure: components, layers, data flow.",
			InputSchema: InputSchema{
				Type: "object",
			},
		},
		{
			Name:        "get_evolution_timeline",
			Description: "GET PROJECT HISTORY. Shows how the project has evolved over time.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"event_type": {Type: "string", Description: "Filter: 'decision', 'architecture_change', 'milestone'"},
					"limit":      {Type: "integer", Description: "Max results, default 50"},
				},
			},
		},
		// === WRITE TOOLS ===
		// Use these to record knowledge as you work
		{
			Name:        "index_file",
			Description: "INDEX A FILE after reading it. Records the file's summary, exports, imports, and language. Content is automatically indexed for search.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":       {Type: "string", Description: "Absolute file path"},
					"summary":    {Type: "string", Description: "One-sentence description of what this file does"},
					"exports":    {Type: "array", Description: "Exported symbols: [{name: 'funcName', kind: 'function', line: 10}]"},
					"imports":    {Type: "array", Description: "Imported modules: ['./utils', 'express']"},
					"language":   {Type: "string", Description: "Language: 'typescript', 'go', 'python', etc."},
					"patterns":   {Type: "array", Description: "Pattern IDs this file uses"},
					"line_count": {Type: "integer", Description: "Total lines in file"},
				},
				Required: []string{"path", "summary"},
			},
		},
		{
			Name:        "add_decision",
			Description: "RECORD A DECISION you made. Use when you choose between alternatives or establish a pattern. Future agents will see this.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"content":           {Type: "string", Description: "What was decided (e.g., 'Use JWT for API authentication')"},
					"reason":            {Type: "string", Description: "Why this choice was made"},
					"context":           {Type: "string", Description: "What problem/situation led to this"},
					"alternatives":      {Type: "array", Description: "Other options considered"},
					"feature":           {Type: "string", Description: "Feature ID if related to specific feature"},
					"related_files":     {Type: "array", Description: "File paths affected by this decision"},
					"related_decisions": {Type: "array", Description: "IDs of related decisions"},
					"tags":              {Type: "array", Description: "Tags: ['security', 'performance', 'api']"},
				},
				Required: []string{"content", "reason"},
			},
		},
		{
			Name:        "add_warning",
			Description: "RECORD A WARNING about something to avoid. Use when you discover a pitfall, bug, or anti-pattern. Future agents will be warned.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"content":       {Type: "string", Description: "What to avoid (e.g., 'Don't use sync API in async context')"},
					"reason":        {Type: "string", Description: "Why it's dangerous"},
					"evidence":      {Type: "string", Description: "What went wrong when this was tried"},
					"severity":      {Type: "string", Description: "'info', 'warning', or 'critical'"},
					"feature":       {Type: "string", Description: "Feature ID if specific to a feature"},
					"related_files": {Type: "array", Description: "File paths this warning applies to"},
					"tags":          {Type: "array", Description: "Tags for categorization"},
				},
				Required: []string{"content", "reason"},
			},
		},
		{
			Name:        "add_insight",
			Description: "RECORD AN INSIGHT or learning. Use when you discover something useful about the codebase.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"content":       {Type: "string", Description: "The insight"},
					"context":       {Type: "string", Description: "Context for the insight"},
					"feature":       {Type: "string", Description: "Related feature ID"},
					"related_files": {Type: "array", Description: "Related files"},
					"tags":          {Type: "array", Description: "Tags for categorization"},
				},
				Required: []string{"content"},
			},
		},
		{
			Name:        "add_pattern",
			Description: "Add a recognized pattern.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"name":          {Type: "string", Description: "Pattern name (e.g., Repository Pattern)"},
					"description":   {Type: "string", Description: "Pattern description"},
					"examples":      {Type: "array", Description: "File paths showing this pattern"},
					"rules":         {Type: "array", Description: "Rules that define this pattern"},
					"anti_patterns": {Type: "array", Description: "What violates this pattern"},
				},
				Required: []string{"name", "description"},
			},
		},
		{
			Name:        "add_evolution_event",
			Description: "Add an event to the evolution timeline.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"event_type":  {Type: "string", Description: "Type: decision, architecture_change, pattern_adopted, milestone, warning"},
					"title":       {Type: "string", Description: "Event title"},
					"description": {Type: "string", Description: "Event description"},
					"impact":      {Type: "string", Description: "Impact: low, medium, high"},
					"related_ids": {Type: "array", Description: "Related decision/warning IDs"},
				},
				Required: []string{"event_type", "title", "description"},
			},
		},
		{
			Name:        "save_conversation",
			Description: "SAVE CONVERSATION MEMORY. Call this when: (1) a conversation exceeds ~20 messages, (2) important decisions were made, (3) the user is done for now. This compresses the conversation so future sessions can resume efficiently.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"feature":           {Type: "string", Description: "Feature ID this conversation relates to"},
					"summary":           {Type: "string", Description: "What was accomplished in this conversation"},
					"key_points":        {Type: "array", Description: "Key takeaways and decisions"},
					"files_discussed":   {Type: "array", Description: "Files that were read or modified"},
					"decisions_made":    {Type: "array", Description: "Decision IDs if any were recorded"},
					"open_items":        {Type: "array", Description: "Items left to do or discuss"},
					"original_tokens":   {Type: "integer", Description: "Approximate token count before compression"},
					"compressed_tokens": {Type: "integer", Description: "Token count after compression"},
				},
				Required: []string{"feature", "summary"},
			},
		},
		{
			Name:        "compact_conversation",
			Description: "COMPACT A LONG CONVERSATION into a dense summary. Use this when the conversation history is getting long (>50 messages) to reduce tokens while preserving key context. Returns a structured summary.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"conversation_id": {Type: "string", Description: "ID of conversation to compact (or 'current')"},
					"feature":         {Type: "string", Description: "Feature this conversation relates to"},
					"full_text":       {Type: "string", Description: "The full conversation text to compact"},
					"preserve_code":   {Type: "boolean", Description: "Keep code snippets intact (default: true)"},
				},
				Required: []string{"feature", "full_text"},
			},
		},
		{
			Name:        "update_feature_state",
			Description: "Update the current state of a feature.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"feature":        {Type: "string", Description: "Feature ID"},
					"state":          {Type: "string", Description: "Current state description"},
					"relevant_files": {Type: "array", Description: "Files this feature touches"},
				},
				Required: []string{"feature", "state"},
			},
		},
		{
			Name:        "update_architecture",
			Description: "Update the high-level system architecture.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"description": {Type: "string", Description: "Architecture description"},
					"diagram":     {Type: "string", Description: "ASCII or mermaid diagram"},
					"services":    {Type: "array", Description: "Service nodes [{name, description, files, dependencies}]"},
					"data_flows":  {Type: "array", Description: "Data flows [{from, to, description}]"},
				},
			},
		},
		{
			Name:        "update_project",
			Description: "Update project-level knowledge.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"name":        {Type: "string", Description: "Project name"},
					"description": {Type: "string", Description: "Project description"},
					"languages":   {Type: "array", Description: "Programming languages used"},
				},
			},
		},
		// Feature lifecycle tools
		{
			Name:        "start_feature",
			Description: "Create a new feature context.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id":          {Type: "string", Description: "Feature ID (e.g., payment-retry)"},
					"branch":      {Type: "string", Description: "Git branch name"},
					"extends":     {Type: "string", Description: "Parent feature to inherit from"},
					"description": {Type: "string", Description: "Initial description"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "archive_feature",
			Description: "Archive a completed feature.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id":      {Type: "string", Description: "Feature ID to archive"},
					"summary": {Type: "string", Description: "Archive summary"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "recall_feature",
			Description: "Recall an archived feature back to active.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {Type: "string", Description: "Feature ID to recall"},
				},
				Required: []string{"id"},
			},
		},
		// === INDEX & GRAPH TOOLS ===
		{
			Name:        "index",
			Description: "TRIGGER INDEXING. Usually not needed - use index_file for individual files.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"incremental": {Type: "boolean", Description: "Only index changed files"},
					"paths":       {Type: "array", Description: "Specific paths to index"},
				},
			},
		},
		{
			Name:        "index_status",
			Description: "CHECK INDEX STATUS. Shows what has been indexed and any pending work.",
			InputSchema: InputSchema{
				Type: "object",
			},
		},
		{
			Name:        "get_graph",
			Description: "GET KNOWLEDGE GRAPH. Shows connections between files, decisions, warnings, and features. Use to understand relationships.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"node_type": {Type: "string", Description: "Filter: 'decision', 'warning', 'file', 'pattern', 'feature'"},
					"node_id":   {Type: "string", Description: "Get edges for a specific node"},
				},
			},
		},
		{
			Name:        "get_related",
			Description: "Traverse the knowledge graph from a starting node. Find all connected decisions, warnings, patterns, and files.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"node_type": {Type: "string", Description: "Starting node type: 'decision', 'warning', 'file', 'pattern', 'feature'"},
					"node_id":   {Type: "string", Description: "Starting node ID"},
					"max_depth": {Type: "integer", Description: "Max traversal depth (default 2, max 5)"},
				},
				Required: []string{"node_type", "node_id"},
			},
		},
		// === CODE ANALYSIS TOOLS ===
		// Use these to understand code structure and dependencies
		{
			Name:        "scan_imports",
			Description: "EXTRACT IMPORTS from a file. Use to understand what a file depends on. Supports TS/JS, Go, Python.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {Type: "string", Description: "File path to scan"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "get_code_map",
			Description: "GET PROJECT STRUCTURE/DIR MAP. Use to understand how the codebase is organized. Returns limited depth/files by default to keep output manageable. For high-level overview, use dirs_only=true.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":       {Type: "string", Description: "Filter to a subdirectory (recommended for large projects)"},
					"language":   {Type: "string", Description: "Filter by language"},
					"max_depth":  {Type: "integer", Description: "Max directory depth (default: 3, max: 10)"},
					"limit":      {Type: "integer", Description: "Max items to return (default: 100, max: 1000)"},
					"recursive":  {Type: "boolean", Description: "Whether to walk subdirectories (default: true)"},
					"dirs_only":  {Type: "boolean", Description: "Set true to ONLY show directory names (no files) — perfect for 'DIR map'"},
				},
			},
		},
		{
			Name:        "get_dependencies",
			Description: "GET FILE DEPENDENCIES. Use to understand what a file imports (upstream) or what imports it (downstream).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":      {Type: "string", Description: "File path to analyze"},
					"direction": {Type: "string", Description: "'upstream' (what I import), 'downstream' (what imports me), or 'both'"},
					"depth":     {Type: "integer", Description: "How many levels deep (default 1, max 5)"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "trace_flow",
			Description: "TRACE CODE FLOW. Use to understand how data/requests flow through the codebase. Example: 'How does a login request flow?'",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":      {Type: "string", Description: "Starting file"},
					"direction": {Type: "string", Description: "'forward' (follow imports) or 'backward' (follow importers)"},
					"depth":     {Type: "integer", Description: "How far to trace (default 3, max 10)"},
					"query":     {Type: "string", Description: "Optional: filter nodes by relevance to this topic"},
				},
				Required: []string{"path"},
			},
		},
		// === TOKEN-EFFICIENT TOOLS ===
		// Use these instead of reading full files to save tokens
		{
			Name:        "get_skeleton",
			Description: "GET CODE STRUCTURE without implementation. Returns function/class signatures - NOT bodies. Use this FIRST when exploring code. For directories, it is NON-RECURSIVE by default (only files in the target dir). Set recursive=true for deep exploration.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":      {Type: "string", Description: "File or directory path"},
					"format":    {Type: "string", Description: "'json' or 'text' (default: text)"},
					"limit":     {Type: "integer", Description: "Max files to process for directories (default: 20, max: 100)"},
					"max_chars": {Type: "integer", Description: "Max output characters (default: 50000, max: 100000)"},
					"recursive": {Type: "boolean", Description: "Set true to walk subdirectories (default: false)"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "get_types",
			Description: "GET TYPE DEFINITIONS from TypeScript files. Returns interfaces, types, enums - perfect for understanding data models. Saves 70% tokens.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":   {Type: "string", Description: "File or directory path"},
					"format": {Type: "string", Description: "'json' or 'text' (default: text)"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "search_snippets",
			Description: "SEARCH CODE and get only relevant snippets, not full files. Use when you need to find specific code. Saves 80% tokens.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query":    {Type: "string", Description: "What to search for"},
					"language": {Type: "string", Description: "Optional language filter"},
					"limit":    {Type: "integer", Description: "Max snippets (default 5)"},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "get_recent_changes",
			Description: "GET GIT HISTORY with impact analysis. Use to see what changed recently and who changed it. Saves 60-80% tokens vs git log.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"since":   {Type: "string", Description: "Time filter: '3 days ago', '1 week ago'"},
					"path":    {Type: "string", Description: "Filter to changes in this path"},
					"limit":   {Type: "integer", Description: "Max commits (default 10)"},
					"feature": {Type: "string", Description: "Filter to commits mentioning this feature"},
				},
			},
		},
		{
			Name:        "resume_context",
			Description: "RESUME PREVIOUS WORK on a feature. Loads saved state from last session. Use at start of session to continue where you left off. Saves 95% tokens.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"feature": {Type: "string", Description: "Feature ID to resume"},
				},
				Required: []string{"feature"},
			},
		},
		{
			Name:        "list_conversations",
			Description: "LIST SAVED CONVERSATIONS. Shows conversation history across features, sorted by most recent first. Use to review past sessions and decisions.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"feature": {Type: "string", Description: "Optional: filter to a specific feature ID"},
					"since":   {Type: "string", Description: "Optional: ISO8601 date to filter conversations after (e.g. '2025-01-01T00:00:00Z')"},
					"limit":   {Type: "integer", Description: "Max conversations to return (default 20)"},
				},
			},
		},
		{
			Name:        "get_task_context",
			Description: "GET STARTER CONTEXT for a task type. Use when starting common tasks to get relevant patterns, warnings, and checklist. One call instead of 5-10 searches.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"task":   {Type: "string", Description: "'add-api-endpoint', 'add-service', 'add-test', 'refactor', 'debug'"},
					"domain": {Type: "string", Description: "Module name: 'notification', 'payment', 'auth'"},
					"file":   {Type: "string", Description: "Specific file to get context for"},
				},
				Required: []string{"task"},
			},
		},
		// === COMPLIANCE & ONBOARDING TOOLS ===
		{
			Name:        "check_compliance",
			Description: "CHECK CODE COMPLIANCE against recorded decisions and patterns. Use before committing or after generating code to verify it follows team conventions. Returns violations with references to the decision/pattern being violated.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_path": {Type: "string", Description: "File path to check (reads current content)"},
					"diff":      {Type: "string", Description: "Code diff to check (alternative to file_path)"},
					"code":      {Type: "string", Description: "Code snippet to check (alternative to file_path)"},
				},
			},
		},
		{
			Name:        "onboard",
			Description: "GET STRUCTURED ONBOARDING for a new team member. Returns the project's architecture, top decisions, active warnings, key patterns, code map, and expert contacts. One call to understand the entire project.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"focus": {Type: "string", Description: "Optional area to focus on: 'architecture', 'patterns', 'warnings', 'all' (default: 'all')"},
					"role":  {Type: "string", Description: "Optional: 'frontend', 'backend', 'fullstack', 'devops' — tailors the onboarding"},
				},
			},
		},
		// === TEAM ACTIVITY TOOLS ===
		{
			Name:        "get_feed",
			Description: "GET RECENT TEAM ACTIVITY. Shows a timeline of recent decisions, warnings, patterns, insights, conversations, and events. Use to see what the team has been working on or what changed recently.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit": {Type: "integer", Description: "Max entries to return (default 20)"},
					"since": {Type: "string", Description: "Optional: ISO8601 date or duration (e.g., '7d', '24h', '2025-01-01')"},
					"type":  {Type: "string", Description: "Optional: filter by type — 'decision', 'warning', 'pattern', 'insight', 'conversation', 'event'"},
				},
			},
		},
		// === HIGH-IMPACT EXTRACTION TOOLS ===
		{
			Name:        "get_api_surface",
			Description: "GET ALL API ENDPOINTS from multiple languages: TypeScript (NestJS/Express), Go (gin/echo), Python (Flask/FastAPI/Django), Java (Spring), C# (ASP.NET). Also extracts Kafka consumers/producers. Saves 80% tokens vs reading controller files.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {Type: "string", Description: "Directory or file path to scan"},
					"app":  {Type: "string", Description: "App name for labeling (e.g., 'notification', 'gateway')"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "get_schema_models",
			Description: "GET DATABASE MODELS from multiple languages: Prisma, Go (GORM/sqlx), Python (SQLAlchemy/Django), Java (JPA/Hibernate), TypeScript (TypeORM). Extracts models, fields, relations, enums. Saves 70% tokens vs reading full files.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {Type: "string", Description: "File or directory to scan for database models"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "get_config_map",
			Description: "GET ALL CONFIG/ENV VARS used in a project. Scans for process.env, os.Getenv, config files. Saves 60% tokens vs manually searching.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {Type: "string", Description: "Directory to scan for config usage"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "get_blueprint",
			Description: "GET TASK BLUEPRINT - The most powerful tool. Returns a complete action plan with file patterns, examples to follow, relevant decisions, warnings, and a checklist. Use this FIRST for any development task. Saves 50-70% tokens by eliminating exploration. Task types: 'add-endpoint', 'add-feature', 'add-service', 'fix-bug', 'refactor', 'add-test'.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"task": {Type: "string", Description: "Task type: 'add-endpoint', 'add-feature', 'add-service', 'fix-bug', 'refactor', 'add-test'"},
					"app":  {Type: "string", Description: "App/module name (e.g., 'smart-smoke', 'notification')"},
					"path": {Type: "string", Description: "Optional: specific path context for the task"},
				},
				Required: []string{"task"},
			},
		},
		// === GIT INTELLIGENCE TOOLS ===
		// Mine the team's institutional memory from Git history
		{
			Name:        "find_experts",
			Description: "WHO WROTE/OWNS THIS CODE? Use when asked 'who developed X?', 'who is the expert on X?', 'who owns X?'. Pass an area keyword (e.g. 'notification', 'sms', 'fraud') or specific file paths. Returns top developers ranked by ownership % with activity status. Instant from cache.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"files": {Type: "array", Description: "List of file paths to find experts for"},
					"area":  {Type: "string", Description: "Or specify an area/keyword to search (e.g., 'sms', 'payment')"},
					"limit": {Type: "integer", Description: "Max results to return (default: 20, max: 50)"},
				},
			},
		},
		{
			Name:        "get_file_history",
			Description: "GET EXPERTISE ANALYSIS for a file. Shows all contributors, their ownership %, lines changed, activity status, and churn rate.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"file": {Type: "string", Description: "File path to analyze"},
				},
				Required: []string{"file"},
			},
		},
		{
			Name:        "get_knowledge_risks",
			Description: "IDENTIFY BUS FACTOR / KNOWLEDGE RISKS. Shows areas where the main developer left (CRITICAL), single-person ownership (MEDIUM), or few contributors (HIGH). Use when asked about risks, bus factor, knowledge gaps, or team coverage. Returns top risks by severity.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"area":  {Type: "string", Description: "Optional: limit to specific area/directory"},
					"limit": {Type: "integer", Description: "Max risks to return (default: 30, max: 100)"},
				},
			},
		},
		{
			Name:        "get_file_correlations",
			Description: "FIND FILES THAT CHANGE TOGETHER. Before modifying a file, check what other files usually change with it. Prevents incomplete changes (e.g. forgot to update the test, the schema, the DTO). Instant from cache.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"file":            {Type: "string", Description: "File to find correlations for"},
					"min_correlation": {Type: "number", Description: "Minimum correlation (0-1, default 0.3)"},
				},
				Required: []string{"file"},
			},
		},
		{
			Name:        "get_commit_context",
			Description: "GET WHY CODE EXISTS. For a file (and optionally specific lines), shows the commits that created/modified it, who did it, and why (commit messages).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"file":  {Type: "string", Description: "File path"},
					"lines": {Type: "array", Description: "Optional: specific line numbers to get context for"},
				},
				Required: []string{"file"},
			},
		},
	}

	s.sendResult(req.ID, map[string]interface{}{"tools": tools})
}
