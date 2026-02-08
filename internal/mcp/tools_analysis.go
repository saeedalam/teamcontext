package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/saeedalam/teamcontext/internal/git"
	"github.com/saeedalam/teamcontext/internal/imports"
	"github.com/saeedalam/teamcontext/internal/search"
	"github.com/saeedalam/teamcontext/internal/skeleton"
	"github.com/saeedalam/teamcontext/internal/typeregistry"
	"github.com/saeedalam/teamcontext/pkg/types"
)

// =============================================================================
// CODE ANALYSIS & TOKEN-SAVING TOOLS
// Understand code structure and save tokens
// =============================================================================

// --- Analysis Tool Handlers ---

func (s *Server) handleScanImports(params json.RawMessage) (interface{}, error) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	info, err := os.Stat(p.Path)
	if err != nil {
		return nil, fmt.Errorf("path not found: %w", err)
	}

	var allResults []types.ImportResult
	var filesProcessed int

	if info.IsDir() {
		// Walk directory and scan all source files
		err := filepath.Walk(p.Path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				if info != nil && info.IsDir() {
					name := info.Name()
					if name == "node_modules" || name == "dist" || name == ".git" || name == "vendor" {
						return filepath.SkipDir
					}
				}
				return nil
			}

			ext := strings.ToLower(filepath.Ext(filePath))
			if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".go" && ext != ".py" {
				return nil
			}

			results, err := imports.ScanFile(filePath)
			if err == nil {
				allResults = append(allResults, results...)
				filesProcessed++
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Single file
		results, err := imports.ScanFile(p.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to scan imports: %w", err)
		}
		allResults = results
		filesProcessed = 1
	}

	return map[string]interface{}{
		"path":            p.Path,
		"files_processed": filesProcessed,
		"imports":         allResults,
		"total":           len(allResults),
	}, nil
}

func (s *Server) handleGetCodeMap(params json.RawMessage) (interface{}, error) {
	// Read pre-computed codetree.txt (generated during init/index)
	treePath := filepath.Join(s.basePath, "codetree.txt")
	data, err := os.ReadFile(treePath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{
				"error": "Code tree not generated yet. Run 'teamcontext index' first.",
				"hint":  "The code tree is pre-computed during indexing for ultra-fast access.",
			}, nil
		}
		return nil, err
	}

	return map[string]interface{}{
		"tree": string(data),
		"note": "Pre-computed during index. Run 'teamcontext index' to refresh.",
	}, nil
}



func (s *Server) handleGetDependencies(params json.RawMessage) (interface{}, error) {
	var p struct {
		Path      string `json:"path"`
		Direction string `json:"direction"`
		Depth     int    `json:"depth"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if p.Direction == "" {
		p.Direction = "both"
	}
	if p.Depth <= 0 {
		p.Depth = 1
	}
	if p.Depth > 5 {
		p.Depth = 5
	}

	files, _ := s.jsonStore.GetFilesIndex()

	var upstream, downstream []types.DependencyNode

	if p.Direction == "upstream" || p.Direction == "both" {
		visited := map[string]bool{}
		upstream = s.collectDependencies(p.Path, "imports", p.Depth, 0, visited, files)
	}

	if p.Direction == "downstream" || p.Direction == "both" {
		visited := map[string]bool{}
		downstream = s.collectDependencies(p.Path, "imported_by", p.Depth, 0, visited, files)
	}

	result := map[string]interface{}{
		"path":      p.Path,
		"direction": p.Direction,
		"depth":     p.Depth,
	}

	if p.Direction == "upstream" || p.Direction == "both" {
		result["upstream"] = upstream
		result["upstream_count"] = countDepNodes(upstream)
	}
	if p.Direction == "downstream" || p.Direction == "both" {
		result["downstream"] = downstream
		result["downstream_count"] = countDepNodes(downstream)
	}

	return result, nil
}

func (s *Server) handleTraceFlow(params json.RawMessage) (interface{}, error) {
	var p struct {
		Path      string `json:"path"`
		Direction string `json:"direction"`
		Depth     int    `json:"depth"`
		Query     string `json:"query"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if p.Direction == "" {
		p.Direction = "forward"
	}
	if p.Depth <= 0 {
		p.Depth = 3
	}
	if p.Depth > 10 {
		p.Depth = 10
	}

	files, _ := s.jsonStore.GetFilesIndex()

	relation := "imports"
	if p.Direction == "backward" {
		relation = "imported_by"
	}

	// DFS traversal
	var chain []types.FlowNode
	visited := map[string]bool{}

	var dfs func(filePath string, depth int)
	dfs = func(filePath string, depth int) {
		if depth > p.Depth || visited[filePath] {
			return
		}
		visited[filePath] = true

		node := types.FlowNode{
			Path:  filePath,
			Depth: depth,
		}

		// Enrich with file index data
		if fi, ok := files[filePath]; ok {
			node.Summary = fi.Summary
			node.Language = fi.Language
			node.Exports = fi.Exports
		}

		// Calculate relevance if query provided
		if p.Query != "" {
			searchText := node.Summary + " " + node.Path
			for _, exp := range node.Exports {
				searchText += " " + exp.Name
			}
			if containsAny(searchText, p.Query) {
				node.Relevance = 1.0
			}
		}

		chain = append(chain, node)

		// Follow edges
		edges, _ := s.jsonStore.GetEdgesFrom("file", filePath)
		for _, e := range edges {
			if e.Relation == relation && e.ToType == "file" {
				dfs(e.ToID, depth+1)
			}
		}
	}

	dfs(p.Path, 0)

	// If query provided, sort by relevance (relevant first), keeping depth order as secondary
	if p.Query != "" {
		sort.SliceStable(chain, func(i, j int) bool {
			if chain[i].Relevance != chain[j].Relevance {
				return chain[i].Relevance > chain[j].Relevance
			}
			return chain[i].Depth < chain[j].Depth
		})
	}

	return map[string]interface{}{
		"path":      p.Path,
		"direction": p.Direction,
		"depth":     p.Depth,
		"chain":     chain,
		"total":     len(chain),
	}, nil
}



// collectDependencies does BFS traversal to collect dependency nodes
func (s *Server) collectDependencies(filePath, relation string, maxDepth, currentDepth int, visited map[string]bool, files map[string]types.FileIndex) []types.DependencyNode {
	if currentDepth >= maxDepth || visited[filePath] {
		return nil
	}
	visited[filePath] = true

	edges, _ := s.jsonStore.GetEdgesFrom("file", filePath)

	var nodes []types.DependencyNode
	for _, e := range edges {
		if e.Relation != relation || e.ToType != "file" {
			continue
		}

		node := types.DependencyNode{
			Path:  e.ToID,
			Depth: currentDepth + 1,
		}
		if fi, ok := files[e.ToID]; ok {
			node.Summary = fi.Summary
			node.Language = fi.Language
		}
		node.Children = s.collectDependencies(e.ToID, relation, maxDepth, currentDepth+1, visited, files)
		nodes = append(nodes, node)
	}
	return nodes
}

// countDepNodes counts total nodes in a dependency tree
func countDepNodes(nodes []types.DependencyNode) int {
	count := len(nodes)
	for _, n := range nodes {
		count += countDepNodes(n.Children)
	}
	return count
}

// uniqueStrings deduplicates a string slice preserving order
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range input {
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// --- Token-Saving Tool Handlers ---

func (s *Server) handleGetSkeleton(params json.RawMessage) (interface{}, error) {
	var p struct {
		Path      string `json:"path"`
		Format    string `json:"format"`
		Limit     int    `json:"limit"`
		MaxChars  int    `json:"max_chars"`
		Recursive bool   `json:"recursive"` // Default to false
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if p.Format == "" {
		p.Format = "text"
	}
	// Default limits to prevent output explosion
	if p.Limit <= 0 {
		p.Limit = 20 // Max 20 files by default for directories
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.MaxChars <= 0 {
		p.MaxChars = 50000 // ~12K tokens max
	}
	if p.MaxChars > 100000 {
		p.MaxChars = 100000
	}

	info, err := os.Stat(p.Path)
	if err != nil {
		return nil, fmt.Errorf("path not found: %w", err)
	}

	if info.IsDir() {
		// Walk directory and extract skeletons from supported files (with limit)
		var allSkeletons []*types.CodeSkeleton
		var totalOriginalLines, totalSkeletonLines int
		filesProcessed := 0
		filesSkipped := 0

		supportedExts := map[string]bool{
			".ts": true, ".tsx": true, ".js": true, ".jsx": true,
			".go": true, ".py": true, ".java": true, ".cs": true,
			".rb": true, ".rs": true, ".kt": true, ".swift": true,
		}

		err := filepath.Walk(p.Path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				if info != nil && info.IsDir() {
					name := info.Name()
					if name == "node_modules" || name == ".git" || name == "vendor" ||
						name == "dist" || name == "target" || name == "__pycache__" {
						return filepath.SkipDir
					}
					// If not recursive, skip subdirectories
					if !p.Recursive && filePath != p.Path {
						return filepath.SkipDir
					}
				}
				return nil
			}

			ext := strings.ToLower(filepath.Ext(filePath))
			if !supportedExts[ext] {
				return nil
			}

			// Enforce file limit
			if filesProcessed >= p.Limit {
				filesSkipped++
				return nil
			}

			sk, err := skeleton.ParseFile(filePath)
			if err == nil {
				allSkeletons = append(allSkeletons, sk)
				totalOriginalLines += sk.LineCount
				totalSkeletonLines += sk.SkeletonLines
				filesProcessed++
			}
			return nil
		})

		if err != nil {
			return nil, err
		}

		savingsPercent := 0
		if totalOriginalLines > 0 {
			savingsPercent = 100 - (totalSkeletonLines * 100 / totalOriginalLines)
		}

		result := map[string]interface{}{
			"path":            p.Path,
			"files_processed": len(allSkeletons),
			"files_skipped":   filesSkipped,
			"truncated":       filesSkipped > 0,
			"original_lines":  totalOriginalLines,
			"skeleton_lines":  totalSkeletonLines,
			"tokens_saved":    fmt.Sprintf("%d%%", savingsPercent),
		}

		if p.Format == "json" {
			result["skeletons"] = allSkeletons
		} else {
			var combined strings.Builder
			for _, sk := range allSkeletons {
				combined.WriteString(fmt.Sprintf("\n// === %s ===\n", sk.Path))
				formatted := skeleton.FormatSkeleton(sk)
				// Check if we'd exceed max_chars
				if combined.Len()+len(formatted) > p.MaxChars {
					combined.WriteString("\n// ... output truncated (max_chars reached) ...\n")
					result["output_truncated"] = true
					break
				}
				combined.WriteString(formatted)
			}
			result["skeleton"] = combined.String()
		}

		if filesSkipped > 0 {
			result["note"] = fmt.Sprintf("Use 'limit' param to increase (current: %d). Use 'path' to target specific subdirectory.", p.Limit)
		}

		return result, nil
	}

	// Single file
	sk, err := skeleton.ParseFile(p.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse skeleton: %w", err)
	}

	// Calculate token savings estimate
	originalTokens := sk.LineCount * 4 // rough estimate: 4 tokens per line
	skeletonTokens := sk.SkeletonLines * 4
	savingsPercent := 0
	if originalTokens > 0 {
		savingsPercent = 100 - (skeletonTokens * 100 / originalTokens)
	}

	result := map[string]interface{}{
		"path":            p.Path,
		"language":        sk.Language,
		"original_lines":  sk.LineCount,
		"skeleton_lines":  sk.SkeletonLines,
		"tokens_saved":    fmt.Sprintf("%d%%", savingsPercent),
		"classes":         len(sk.Classes),
		"functions":       len(sk.Functions),
		"interfaces":      len(sk.Interfaces),
		"types":           len(sk.Types),
		"enums":           len(sk.Enums),
	}

	if p.Format == "json" {
		result["skeleton"] = sk
	} else {
		result["skeleton"] = skeleton.FormatSkeleton(sk)
	}

	return result, nil
}

func (s *Server) handleGetTypes(params json.RawMessage) (interface{}, error) {
	var p struct {
		Path   string `json:"path"`
		Format string `json:"format"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if p.Format == "" {
		p.Format = "text"
	}

	var allTypeDefs []types.TypeDef
	var allEnumDefs []types.EnumDef
	var filesProcessed int

	info, err := os.Stat(p.Path)
	if err != nil {
		return nil, fmt.Errorf("path not found: %w", err)
	}

	if info.IsDir() {
		// Walk directory and extract types from all TypeScript files
		err := filepath.Walk(p.Path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				if info != nil && info.IsDir() {
					name := info.Name()
					if name == "node_modules" || name == "dist" || name == ".git" {
						return filepath.SkipDir
					}
				}
				return nil
			}

			ext := strings.ToLower(filepath.Ext(filePath))
			if ext != ".ts" && ext != ".tsx" {
				return nil
			}

			typeDefs, enumDefs, err := typeregistry.ExtractTypes(filePath)
			if err == nil {
				allTypeDefs = append(allTypeDefs, typeDefs...)
				allEnumDefs = append(allEnumDefs, enumDefs...)
				filesProcessed++
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Single file
		typeDefs, enumDefs, err := typeregistry.ExtractTypes(p.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to extract types: %w", err)
		}
		allTypeDefs = typeDefs
		allEnumDefs = enumDefs
		filesProcessed = 1
	}

	result := map[string]interface{}{
		"path":            p.Path,
		"files_processed": filesProcessed,
		"interfaces":      len(allTypeDefs),
		"enums":           len(allEnumDefs),
	}

	if p.Format == "json" {
		result["types"] = allTypeDefs
		result["enum_defs"] = allEnumDefs
	} else {
		result["types"] = typeregistry.FormatTypeDefs(allTypeDefs, allEnumDefs)
	}

	return result, nil
}

func (s *Server) handleSearchSnippets(params json.RawMessage) (interface{}, error) {
	var p struct {
		Query    string `json:"query"`
		Language string `json:"language"`
		MaxLines int    `json:"max_lines"`
		Limit    int    `json:"limit"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if p.MaxLines <= 0 {
		p.MaxLines = 50
	}
	if p.Limit <= 0 {
		p.Limit = 5
	}

	var snippets []types.CodeSnippet
	searchSource := "indexed"

	// First, try searching indexed code chunks (if content has been indexed)
	chunks, err := s.sqliteIndex.SearchCodeContent(p.Query, p.Language, p.Limit)
	if err == nil && len(chunks) > 0 {
		// Convert chunks to snippets
		for _, chunk := range chunks {
			tokenCount := len(chunk.Content) / 4
			snippet := types.CodeSnippet{
				Path:       chunk.FilePath,
				StartLine:  chunk.StartLine,
				EndLine:    chunk.EndLine,
				Content:    chunk.Content,
				Context:    fmt.Sprintf("%s: %s", chunk.ChunkType, chunk.ChunkName),
				Relevance:  1.0,
				TokenCount: tokenCount,
			}
			snippets = append(snippets, snippet)
		}
	}

	// If no indexed results, fallback to grep-based search
	if len(snippets) == 0 {
		searchSource = "grep"
		projectRoot := s.basePath[:len(s.basePath)-len("/.teamcontext")]

		matches, err := search.SearchCode(p.Query, projectRoot, p.Language, p.Limit*3)
		if err != nil {
			return nil, err
		}

		// Group matches by file and extract snippets
		fileMatches := make(map[string][]types.CodeMatch)
		for _, m := range matches {
			fileMatches[m.Path] = append(fileMatches[m.Path], m)
		}

		for path, pathMatches := range fileMatches {
			if len(snippets) >= p.Limit {
				break
			}

			// Read file to extract context
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			lines := strings.Split(string(content), "\n")

			for _, m := range pathMatches {
				if len(snippets) >= p.Limit {
					break
				}

				// Calculate snippet bounds
				startLine := m.Line - 5
				if startLine < 1 {
					startLine = 1
				}
				endLine := m.Line + p.MaxLines - 5
				if endLine > len(lines) {
					endLine = len(lines)
				}

				// Extract snippet
				snippetLines := lines[startLine-1 : endLine]
				snippetContent := strings.Join(snippetLines, "\n")
				tokenCount := len(snippetContent) / 4

				snippet := types.CodeSnippet{
					Path:       path,
					StartLine:  startLine,
					EndLine:    endLine,
					Content:    snippetContent,
					Context:    fmt.Sprintf("Match for '%s' at line %d", p.Query, m.Line),
					Relevance:  1.0,
					TokenCount: tokenCount,
				}
				snippets = append(snippets, snippet)
			}
		}
	}

	totalTokens := 0
	for _, sn := range snippets {
		totalTokens += sn.TokenCount
	}

	return map[string]interface{}{
		"query":         p.Query,
		"snippets":      snippets,
		"total":         len(snippets),
		"total_tokens":  totalTokens,
		"search_source": searchSource,
		"hint":          "Use index_content to index file contents for faster, more accurate search",
	}, nil
}

func (s *Server) handleGetRecentChanges(params json.RawMessage) (interface{}, error) {
	var p struct {
		Since   string `json:"since"`
		Path    string `json:"path"`
		Limit   int    `json:"limit"`
		Feature string `json:"feature"`
	}
	json.Unmarshal(params, &p)

	if p.Limit <= 0 {
		p.Limit = 10
	}

	// Get project root
	projectRoot := s.basePath[:len(s.basePath)-len("/.teamcontext")]

	var changes []types.GitChange
	var err error

	if p.Path != "" {
		changes, err = git.GetRecentChangesForPath(projectRoot, p.Path, p.Limit)
	} else {
		changes, err = git.GetRecentChanges(projectRoot, p.Since, p.Limit)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get git changes: %w", err)
	}

	// Filter by feature if specified
	if p.Feature != "" {
		var filtered []types.GitChange
		for _, c := range changes {
			if strings.Contains(strings.ToLower(c.Message), strings.ToLower(p.Feature)) {
				c.RelatedFeature = p.Feature
				filtered = append(filtered, c)
			}
		}
		changes = filtered
	}

	// Get current branch
	branch, _ := git.GetBranch(projectRoot)

	// Calculate total impact
	totalInsertions := 0
	totalDeletions := 0
	allFiles := []string{}
	for _, c := range changes {
		totalInsertions += c.Insertions
		totalDeletions += c.Deletions
		allFiles = append(allFiles, c.FilesChanged...)
	}

	return map[string]interface{}{
		"branch":           branch,
		"changes":          changes,
		"total":            len(changes),
		"total_insertions": totalInsertions,
		"total_deletions":  totalDeletions,
		"files_affected":   len(uniqueStrings(allFiles)),
	}, nil
}

func (s *Server) handleListConversations(params json.RawMessage) (interface{}, error) {
	var p struct {
		Feature string `json:"feature"`
		Since   string `json:"since"`
		Limit   int    `json:"limit"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Limit <= 0 {
		p.Limit = 20
	}

	var sinceTime time.Time
	if p.Since != "" {
		parsed, err := time.Parse(time.RFC3339, p.Since)
		if err == nil {
			sinceTime = parsed
		}
	}

	var conversations []types.Conversation
	if p.Feature != "" {
		convs, err := s.jsonStore.GetConversations(p.Feature)
		if err != nil {
			return nil, err
		}
		conversations = convs
	} else {
		convs, err := s.jsonStore.GetAllConversations()
		if err != nil {
			return nil, err
		}
		conversations = convs
	}

	// Filter by since time
	if !sinceTime.IsZero() {
		var filtered []types.Conversation
		for _, c := range conversations {
			if c.CreatedAt.After(sinceTime) {
				filtered = append(filtered, c)
			}
		}
		conversations = filtered
	}

	// Sort by created_at descending (most recent first)
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].CreatedAt.After(conversations[j].CreatedAt)
	})

	// Apply limit
	if len(conversations) > p.Limit {
		conversations = conversations[:p.Limit]
	}

	return map[string]interface{}{
		"conversations": conversations,
		"total":         len(conversations),
	}, nil
}

func (s *Server) handleResumeContext(params json.RawMessage) (interface{}, error) {
	var p struct {
		Feature string `json:"feature"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Feature == "" {
		return nil, fmt.Errorf("feature is required")
	}

	// Get feature
	feature, err := s.jsonStore.GetFeature(p.Feature)
	if err != nil {
		return nil, fmt.Errorf("feature not found: %w", err)
	}

	// Get conversations for this feature
	conversations, _ := s.jsonStore.GetConversations(p.Feature)

	// Sort conversations chronologically (oldest first)
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].CreatedAt.Before(conversations[j].CreatedAt)
	})

	// Get decisions for this feature
	decisions, _ := s.jsonStore.GetDecisionsByFeature(p.Feature)

	// Build resume context
	context := map[string]interface{}{
		"feature":        feature.ID,
		"status":         feature.Status,
		"description":    feature.Description,
		"current_state":  feature.CurrentState,
		"relevant_files": feature.RelevantFiles,
		"branch":         feature.Branch,
		"last_accessed":  feature.LastAccessed,
	}

	// Include conversation summaries with start/end times and latest marker
	var convSummaries []map[string]interface{}
	for i, conv := range conversations {
		entry := map[string]interface{}{
			"id":              conv.ID,
			"summary":         conv.Summary,
			"key_points":      conv.KeyPoints,
			"files_discussed": conv.FilesDiscussed,
			"decisions_made":  conv.DecisionsMade,
			"start_time":      conv.StartTime,
			"end_time":        conv.EndTime,
			"created_at":      conv.CreatedAt,
		}
		if i == len(conversations)-1 {
			entry["is_latest"] = true
		}
		convSummaries = append(convSummaries, entry)
	}
	context["conversations"] = convSummaries
	context["decisions"] = decisions
	context["decision_count"] = len(decisions)
	context["conversation_count"] = len(conversations)

	// Calculate tokens saved
	estimatedFullTokens := 50000 // typical full context rebuild
	compressedTokens := 2000 + len(decisions)*200 + len(conversations)*500
	context["tokens_saved_estimate"] = fmt.Sprintf("%d%%", 100-(compressedTokens*100/estimatedFullTokens))

	// Add hooks to remind AI about conversation management
	context["_hooks"] = []types.ConversationHook{
		{
			Action:    types.HookActionSaveConversation,
			Tool:      "save_conversation",
			Priority:  types.HookPrioritySuggested,
			Reason:    "Remember to save this conversation when it gets long (>20 messages) or when important decisions are made.",
			Condition: "conversation_length > 20 OR important_decision_made",
			Arguments: map[string]interface{}{
				"feature": p.Feature,
			},
		},
	}

	return context, nil
}

func (s *Server) handleGetTaskContext(params json.RawMessage) (interface{}, error) {
	var p struct {
		Task   string `json:"task"`
		Domain string `json:"domain"`
		File   string `json:"file"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Task == "" {
		return nil, fmt.Errorf("task is required")
	}

	bundle := types.TaskContextBundle{
		Task:   p.Task,
		Domain: p.Domain,
	}

	// Get patterns for this domain
	patterns, _ := s.jsonStore.GetPatterns()
	for _, pat := range patterns {
		if p.Domain == "" || containsAny(pat.Description, p.Domain) {
			bundle.Patterns = append(bundle.Patterns, pat.Name+": "+pat.Description)
		}
	}

	// Get relevant warnings
	warnings, _ := s.jsonStore.GetWarnings()
	for _, w := range warnings {
		if containsAny(w.Content, p.Task) || (p.Domain != "" && containsAny(w.Content, p.Domain)) {
			bundle.Warnings = append(bundle.Warnings, w.Content)
		}
	}

	// Task-specific checklists
	switch p.Task {
	case "add-api-endpoint":
		bundle.Checklist = []string{
			"Create/update DTO with validation",
			"Add controller method with proper decorators",
			"Add to module providers if needed",
			"Update OpenAPI/Swagger spec",
			"Add auth guard if required",
			"Write unit tests",
			"Test with curl/Postman",
		}
	case "add-service":
		bundle.Checklist = []string{
			"Create service class with @Injectable()",
			"Define interface for the service",
			"Add to module providers",
			"Inject dependencies in constructor",
			"Add error handling",
			"Write unit tests with mocks",
		}
	case "add-test":
		bundle.Checklist = []string{
			"Create test file with .spec.ts suffix",
			"Set up test module with mocked dependencies",
			"Write describe/it blocks",
			"Test happy path",
			"Test error cases",
			"Test edge cases",
			"Run coverage check",
		}
	case "refactor":
		bundle.Checklist = []string{
			"Identify all usages of code to refactor",
			"Write tests for existing behavior if missing",
			"Make changes in small incremental steps",
			"Run tests after each change",
			"Update imports in dependent files",
			"Check for breaking changes",
		}
	case "debug":
		bundle.Checklist = []string{
			"Reproduce the issue consistently",
			"Check recent changes (get_recent_changes)",
			"Add logging/breakpoints",
			"Trace the data flow (trace_flow)",
			"Check error handling paths",
			"Verify inputs and outputs at each step",
		}
	default:
		bundle.Checklist = []string{
			"Understand existing patterns (list_patterns)",
			"Check for related decisions (list_decisions)",
			"Review relevant warnings (list_warnings)",
			"Identify files to modify",
			"Plan incremental changes",
			"Test thoroughly",
		}
	}

	// If a specific file is provided, get its context
	if p.File != "" {
		fileInfo, err := s.jsonStore.GetFileIndex(p.File)
		if err == nil {
			bundle.RelatedDocs = []string{
				fmt.Sprintf("File: %s", fileInfo.Path),
				fmt.Sprintf("Summary: %s", fileInfo.Summary),
				fmt.Sprintf("Language: %s", fileInfo.Language),
				fmt.Sprintf("Exports: %d", len(fileInfo.Exports)),
			}
		}
	}

	return map[string]interface{}{
		"task":        bundle.Task,
		"domain":      bundle.Domain,
		"patterns":    bundle.Patterns,
		"warnings":    bundle.Warnings,
		"checklist":   bundle.Checklist,
		"related_docs": bundle.RelatedDocs,
	}, nil
}


// Helper functions for content indexing

func detectLanguageFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs":
		return "javascript"
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".cs":
		return "csharp"
	case ".rs":
		return "rust"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".scala", ".sc":
		return "scala"
	default:
		return "unknown"
	}
}

func findBlockEnd(lines []string, startIdx int, maxLines int) int {
	if startIdx >= len(lines) {
		return len(lines)
	}

	// Simple brace counting for block detection
	braceCount := 0
	started := false

	endIdx := startIdx
	for i := startIdx; i < len(lines) && i < startIdx+maxLines; i++ {
		line := lines[i]
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")

		if strings.Contains(line, "{") {
			started = true
		}

		endIdx = i + 1

		if started && braceCount <= 0 {
			break
		}
	}

	return endIdx
}

func getLines(lines []string, startLine, endLine int) string {
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine {
		return ""
	}
	return strings.Join(lines[startLine-1:endLine], "\n")
}

