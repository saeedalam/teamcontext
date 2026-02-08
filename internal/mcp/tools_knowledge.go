package mcp

import (
"encoding/json"
"fmt"
"os"
"path/filepath"
"regexp"
"strings"
"time"

"github.com/saeedalam/teamcontext/internal/skeleton"
"github.com/saeedalam/teamcontext/internal/storage"
"github.com/saeedalam/teamcontext/pkg/types"
)

// =============================================================================
// KNOWLEDGE MANAGEMENT TOOLS
// Index files, add decisions/warnings/insights, manage features
// =============================================================================

func (s *Server) handleIndexFile(params json.RawMessage) (interface{}, error) {
	var file types.FileIndex
	if err := json.Unmarshal(params, &file); err != nil {
		return nil, err
	}

	if file.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if file.Summary == "" {
		return nil, fmt.Errorf("summary is required")
	}

	// Save to JSON store
	if err := s.jsonStore.SaveFileIndex(&file); err != nil {
		return nil, err
	}

	// Index in SQLite
	s.sqliteIndex.IndexFile(&file)

	// Add to knowledge graph
	s.addFileEdges(&file)

	// Auto-create import edges from the imports array
	importEdgesCreated := 0
	for _, imp := range file.Imports {
		// Resolve relative imports
		importPath := imp
		if strings.HasPrefix(imp, ".") {
			importPath = filepath.Clean(filepath.Join(filepath.Dir(file.Path), imp))
		}

		// Create "imports" edge
		s.jsonStore.AddEdge(&types.Edge{
			FromType: "file",
			FromID:   file.Path,
			ToType:   "file",
			ToID:     importPath,
			Relation: "imports",
		})

		// Create reverse "imported_by" edge
		s.jsonStore.AddEdge(&types.Edge{
			FromType: "file",
			FromID:   importPath,
			ToType:   "file",
			ToID:     file.Path,
			Relation: "imported_by",
		})

		importEdgesCreated += 2
	}

	// Auto-index file content for code search (background, don't fail if error)
	chunksCreated := 0
	if content, err := os.ReadFile(file.Path); err == nil {
		lines := strings.Split(string(content), "\n")
		language := detectLanguageFromPath(file.Path)
		chunkSize := 50

		var chunks []storage.CodeChunk

		// Try semantic chunks first
		if sk, err := skeleton.ParseFile(file.Path); err == nil && sk != nil {
			for _, fn := range sk.Functions {
				startLine := fn.Line
				endLine := findBlockEnd(lines, startLine-1, chunkSize)
				chunks = append(chunks, storage.CodeChunk{
					FilePath:  file.Path,
					ChunkType: "function",
					ChunkName: fn.Name,
					StartLine: startLine,
					EndLine:   endLine,
					Content:   getLines(lines, startLine, endLine),
					Language:  language,
				})
			}
			for _, class := range sk.Classes {
				startLine := class.Line
				endLine := findBlockEnd(lines, startLine-1, chunkSize*2)
				chunks = append(chunks, storage.CodeChunk{
					FilePath:  file.Path,
					ChunkType: "class",
					ChunkName: class.Name,
					StartLine: startLine,
					EndLine:   endLine,
					Content:   getLines(lines, startLine, endLine),
					Language:  language,
				})
			}
		}

		// Add line-based chunks for uncovered code
		if len(chunks) == 0 || len(lines) > len(chunks)*chunkSize*2 {
			for i := 0; i < len(lines); i += chunkSize {
				endLine := i + chunkSize
				if endLine > len(lines) {
					endLine = len(lines)
				}
				chunkContent := strings.Join(lines[i:endLine], "\n")
				if strings.TrimSpace(chunkContent) != "" {
					chunks = append(chunks, storage.CodeChunk{
						FilePath:  file.Path,
						ChunkType: "lines",
						ChunkName: fmt.Sprintf("lines:%d-%d", i+1, endLine),
						StartLine: i + 1,
						EndLine:   endLine,
						Content:   chunkContent,
						Language:  language,
					})
				}
			}
		}

		if err := s.sqliteIndex.IndexCodeChunks(file.Path, chunks); err == nil {
			chunksCreated = len(chunks)
		}
	}

	return map[string]interface{}{
		"success":             true,
		"indexed_at":          file.IndexedAt,
		"path":                file.Path,
		"exports":             len(file.Exports),
		"patterns":            len(file.Patterns),
		"graph_edges_created": len(file.Patterns) + len(file.RelatedFiles) + importEdgesCreated,
		"import_edges":        importEdgesCreated,
		"content_chunks":      chunksCreated,
	}, nil
}

func (s *Server) handleAddDecision(params json.RawMessage) (interface{}, error) {
	var decision types.Decision
	if err := json.Unmarshal(params, &decision); err != nil {
		return nil, err
	}

	if decision.Content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if decision.Reason == "" {
		return nil, fmt.Errorf("reason is required")
	}

	// Save to JSON store (generates ID)
	if err := s.jsonStore.AddDecision(&decision); err != nil {
		return nil, err
	}

	// Index in SQLite
	s.sqliteIndex.IndexDecision(&decision)

	// Add to knowledge graph
	s.addDecisionEdges(&decision)

	// Add evolution event for significant decisions
	s.jsonStore.AddEvolutionEvent(&types.EvolutionEvent{
		EventType:   "decision",
		Title:       decision.Content,
		Description: decision.Reason,
		Author:      decision.Author,
		Impact:      "medium",
		RelatedIDs:  []string{decision.ID},
	})

	// Store semantic vector for TF-IDF search
	s.storeSemanticVector(decision.ID, "decision", decision.Content+" "+decision.Reason+" "+decision.Context)

	return map[string]interface{}{
		"id":         decision.ID,
		"created_at": decision.CreatedAt,
		"graph_edges_created": len(decision.RelatedFiles) + len(decision.RelatedDecisions) + 1,
	}, nil
}

func (s *Server) handleAddWarning(params json.RawMessage) (interface{}, error) {
	var warning types.Warning
	if err := json.Unmarshal(params, &warning); err != nil {
		return nil, err
	}

	if warning.Content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if warning.Reason == "" {
		return nil, fmt.Errorf("reason is required")
	}

	// Save to JSON store (generates ID)
	if err := s.jsonStore.AddWarning(&warning); err != nil {
		return nil, err
	}

	// Index in SQLite
	s.sqliteIndex.IndexWarning(&warning)

	// Add to knowledge graph
	s.addWarningEdges(&warning)

	// Add evolution event for warnings
	if warning.Severity == "critical" || warning.Severity == "warning" {
		s.jsonStore.AddEvolutionEvent(&types.EvolutionEvent{
			EventType:   "warning",
			Title:       warning.Content,
			Description: warning.Reason,
			Author:      warning.Author,
			Impact:      warning.Severity,
			RelatedIDs:  []string{warning.ID},
		})
	}

	// Store semantic vector for TF-IDF search
	s.storeSemanticVector(warning.ID, "warning", warning.Content+" "+warning.Reason+" "+warning.Evidence)

	return map[string]interface{}{
		"id":         warning.ID,
		"created_at": warning.CreatedAt,
		"graph_edges_created": len(warning.RelatedFiles) + len(warning.RelatedDecisions) + 1,
	}, nil
}

func (s *Server) handleAddInsight(params json.RawMessage) (interface{}, error) {
	var insight types.Insight
	if err := json.Unmarshal(params, &insight); err != nil {
		return nil, err
	}

	if insight.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	if err := s.jsonStore.AddInsight(&insight); err != nil {
		return nil, err
	}

	// Add evolution event for insight capture
	s.jsonStore.AddEvolutionEvent(&types.EvolutionEvent{
		EventType:   "insight",
		Title:       insight.Content,
		Description: insight.Context,
		Author:      insight.Author,
		Impact:      "low",
		RelatedIDs:  []string{insight.ID},
	})

	// Store semantic vector for TF-IDF search
	s.storeSemanticVector(insight.ID, "insight", insight.Content+" "+insight.Context)

	return map[string]interface{}{
		"id":         insight.ID,
		"created_at": insight.CreatedAt,
	}, nil
}


func (s *Server) handleSaveConversation(params json.RawMessage) (interface{}, error) {
	var conv types.Conversation
	if err := json.Unmarshal(params, &conv); err != nil {
		return nil, err
	}

	if conv.Feature == "" {
		return nil, fmt.Errorf("feature is required")
	}
	if conv.Summary == "" {
		return nil, fmt.Errorf("summary is required")
	}

	if err := s.jsonStore.SaveConversation(&conv); err != nil {
		return nil, err
	}

	// Index in semantic search
	vectorText := conv.Summary + " " + strings.Join(conv.KeyPoints, " ") + " " + strings.Join(conv.FilesDiscussed, " ")
	s.storeSemanticVector(conv.ID, "conversation", vectorText)

	return map[string]interface{}{
		"id":         conv.ID,
		"created_at": conv.CreatedAt,
		// Include hook to suggest indexing any files that were modified during the conversation
		"_hooks": []types.ConversationHook{
			{
				Action:   types.HookActionNotifyUser,
				Tool:     "",
				Priority: types.HookPriorityOptional,
				Reason:   "Conversation saved. Consider recording any important decisions with add_decision tool.",
			},
		},
	}, nil
}

func (s *Server) handleCompactConversation(params json.RawMessage) (interface{}, error) {
	var p struct {
		ConversationID string `json:"conversation_id"`
		Feature        string `json:"feature"`
		FullText       string `json:"full_text"`
		PreserveCode   bool   `json:"preserve_code"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if p.Feature == "" {
		return nil, fmt.Errorf("feature is required")
	}
	if p.FullText == "" {
		return nil, fmt.Errorf("full_text is required")
	}

	// Count approximate tokens (rough estimate: 4 chars per token)
	originalTokens := len(p.FullText) / 4

	// Create a compaction record
	compaction := types.ConversationCompaction{
		OriginalID:     p.ConversationID,
		Summary:        "", // AI should fill this based on full_text
		MessageCount:   strings.Count(p.FullText, "\n") / 2, // Rough estimate
		CompactedAt:    time.Now(),
		TokensSaved:    0,
	}

	// Extract code blocks if preserve_code is true
	var codeBlocks []string
	if p.PreserveCode {
		codePattern := regexp.MustCompile("```[\\s\\S]*?```")
		matches := codePattern.FindAllString(p.FullText, -1)
		codeBlocks = matches
	}

	// Return structure for AI to fill in
	return map[string]interface{}{
		"original_tokens":   originalTokens,
		"code_blocks":       codeBlocks,
		"code_blocks_count": len(codeBlocks),
		"compaction":        compaction,
		"instructions":      "Please create a dense summary of this conversation. Include: (1) What was the goal, (2) What was accomplished, (3) Key decisions made, (4) Open items remaining. Keep the summary under 500 words.",
		"_hooks": []types.ConversationHook{
			{
				Action:    types.HookActionSaveConversation,
				Tool:      "save_conversation",
				Priority:  types.HookPrioritySuggested,
				Reason:    "After compacting, save the summary to preserve conversation history",
				Arguments: map[string]interface{}{
					"feature": p.Feature,
				},
			},
		},
	}, nil
}

func (s *Server) handleUpdateFeatureState(params json.RawMessage) (interface{}, error) {
	var p struct {
		Feature       string   `json:"feature"`
		State         string   `json:"state"`
		RelevantFiles []string `json:"relevant_files"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if p.Feature == "" {
		return nil, fmt.Errorf("feature is required")
	}
	if p.State == "" {
		return nil, fmt.Errorf("state is required")
	}

	feature, err := s.jsonStore.GetFeature(p.Feature)
	if err != nil {
		return nil, err
	}

	feature.CurrentState = p.State
	if len(p.RelevantFiles) > 0 {
		feature.RelevantFiles = p.RelevantFiles
	}

	if err := s.jsonStore.UpdateFeature(feature); err != nil {
		return nil, err
	}

	// Update SQLite index
	s.sqliteIndex.IndexFeature(feature)

	return map[string]interface{}{
		"success":    true,
		"updated_at": feature.LastAccessed,
	}, nil
}

func (s *Server) handleStartFeature(params json.RawMessage) (interface{}, error) {
	var p struct {
		ID          string `json:"id"`
		Branch      string `json:"branch"`
		Extends     string `json:"extends"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if p.ID == "" {
		return nil, fmt.Errorf("id is required")
	}

	feature := &types.Feature{
		ID:          p.ID,
		Branch:      p.Branch,
		Extends:     p.Extends,
		Description: p.Description,
	}

	if err := s.jsonStore.CreateFeature(feature); err != nil {
		return nil, err
	}

	// Inherit parent context if extends is set
	inherited := 0
	if p.Extends != "" {
		parentFeature, err := s.jsonStore.GetFeature(p.Extends)
		if err == nil && parentFeature != nil {
			// Copy parent's relevant files
			if len(parentFeature.RelevantFiles) > 0 {
				feature.RelevantFiles = append(feature.RelevantFiles, parentFeature.RelevantFiles...)
			}
			// Copy parent's decisions
			if len(parentFeature.Decisions) > 0 {
				feature.Decisions = append(feature.Decisions, parentFeature.Decisions...)
			}
			// Copy parent's warnings
			if len(parentFeature.Warnings) > 0 {
				feature.Warnings = append(feature.Warnings, parentFeature.Warnings...)
			}
			inherited = len(parentFeature.RelevantFiles) + len(parentFeature.Decisions) + len(parentFeature.Warnings)
			// Save the updated feature with inherited data
			if inherited > 0 {
				s.jsonStore.UpdateFeature(feature)
			}
		}
	}

	// Index in SQLite
	s.sqliteIndex.IndexFeature(feature)

	// Add evolution event for feature start
	s.jsonStore.AddEvolutionEvent(&types.EvolutionEvent{
		EventType:   "milestone",
		Title:       "Feature started: " + p.ID,
		Description: feature.Description,
		Impact:      "medium",
	})

	result := map[string]interface{}{
		"success":    true,
		"created_at": feature.CreatedAt,
	}
	if inherited > 0 {
		result["inherited_from"] = p.Extends
		result["inherited_items"] = inherited
	}
	return result, nil
}

func (s *Server) handleArchiveFeature(params json.RawMessage) (interface{}, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if p.ID == "" {
		return nil, fmt.Errorf("id is required")
	}

	// Get feature to count decisions/warnings
	decisions, _ := s.jsonStore.GetDecisionsByFeature(p.ID)

	if err := s.jsonStore.ArchiveFeature(p.ID); err != nil {
		return nil, err
	}

	// Add evolution event for feature archive
	s.jsonStore.AddEvolutionEvent(&types.EvolutionEvent{
		EventType:  "milestone",
		Title:      "Feature archived: " + p.ID,
		Impact:     "medium",
		RelatedIDs: []string{p.ID},
	})

	return map[string]interface{}{
		"success":             true,
		"decisions_preserved": len(decisions),
	}, nil
}

func (s *Server) handleUpdateProject(params json.RawMessage) (interface{}, error) {
	var project types.Project
	if err := json.Unmarshal(params, &project); err != nil {
		return nil, err
	}

	// Get existing project to merge
	existing, _ := s.jsonStore.GetProject()
	if existing != nil {
		if project.Name == "" {
			project.Name = existing.Name
		}
		if project.Description == "" {
			project.Description = existing.Description
		}
		if len(project.Languages) == 0 {
			project.Languages = existing.Languages
		}
	}

	if err := s.jsonStore.SaveProject(&project); err != nil {
		return nil, err
	}

	// Add evolution event for project update
	s.jsonStore.AddEvolutionEvent(&types.EvolutionEvent{
		EventType:   "architecture_change",
		Title:       "Project updated",
		Description: project.Description,
		Impact:      "medium",
	})

	return map[string]interface{}{
		"success":    true,
		"updated_at": project.UpdatedAt,
	}, nil
}

// --- Helper Functions ---

// loadGitKnowledge reads a pre-computed JSON file from the knowledge/ directory
