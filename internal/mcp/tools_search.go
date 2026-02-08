package mcp

import (
"encoding/json"
"fmt"
"sort"
"strings"

"github.com/saeedalam/teamcontext/internal/git"
"github.com/saeedalam/teamcontext/pkg/types"
)

// =============================================================================
// SEARCH & QUERY TOOLS
// Query the codebase, get context, and search knowledge
// =============================================================================

// --- New TeamContext Tool Handlers ---

func (s *Server) handleQuery(params json.RawMessage) (interface{}, error) {
	var p struct {
		Question    string `json:"question"`
		Scope       string `json:"scope"`
		IncludeCode bool   `json:"include_code"`
		MaxTokens   int    `json:"max_tokens"`
	}
	json.Unmarshal(params, &p)

	if p.Question == "" {
		return nil, fmt.Errorf("question is required")
	}

	// Get all relevant knowledge
	decisions, _ := s.jsonStore.GetDecisions()
	warnings, _ := s.jsonStore.GetWarnings()
	patterns, _ := s.jsonStore.GetPatterns()

	// Filter by scope if specified
	var featureID string
	if len(p.Scope) > 8 && p.Scope[:8] == "feature:" {
		featureID = p.Scope[8:]
	}

	// Simple keyword-based relevance scoring
	query := p.Question
	var relevantDecisions []types.Decision
	var relevantWarnings []types.Warning

	for _, d := range decisions {
		if featureID != "" && d.Feature != featureID {
			continue
		}
		// Simple relevance check - contains query terms
		if containsAny(d.Content+d.Reason+d.Context, query) {
			relevantDecisions = append(relevantDecisions, d)
		}
	}

	for _, w := range warnings {
		if featureID != "" && w.Feature != featureID {
			continue
		}
		if containsAny(w.Content+w.Reason+w.Evidence, query) {
			relevantWarnings = append(relevantWarnings, w)
		}
	}

	// Limit results
	if len(relevantDecisions) > 10 {
		relevantDecisions = relevantDecisions[:10]
	}
	if len(relevantWarnings) > 5 {
		relevantWarnings = relevantWarnings[:5]
	}

	// Search indexed files
	var relevantFiles []types.FileIndex
	files, err := s.sqliteIndex.SearchFiles(query, "", 10)
	if err == nil && len(files) > 0 {
		relevantFiles = files
	}

	// Search git experts for matching contributors or areas
	var gitExperts []types.GitExpertHit
	var cachedExperts []git.DirectoryExpert
	if err := s.loadGitKnowledge("git-experts.json", &cachedExperts); err == nil {
		queryLower := strings.ToLower(query)
		queryWords := strings.Fields(queryLower)
		for _, de := range cachedExperts {
			dirLower := strings.ToLower(de.Directory)
			dirMatch := false
			for _, w := range queryWords {
				if len(w) > 2 && strings.Contains(dirLower, w) {
					dirMatch = true
					break
				}
			}
			if !dirMatch {
				continue
			}
			for _, expert := range de.TopExperts {
				gitExperts = append(gitExperts, types.GitExpertHit{
					Name:      expert.Name,
					Email:     expert.Email,
					Area:      de.Directory,
					Ownership: expert.Ownership,
					Active:    expert.Active,
				})
			}
		}
		// Also match by contributor name
		for _, de := range cachedExperts {
			for _, expert := range de.TopExperts {
				nameLower := strings.ToLower(expert.Name)
				for _, w := range queryWords {
					if len(w) > 2 && strings.Contains(nameLower, w) {
						// Avoid duplicates
						dup := false
						for _, existing := range gitExperts {
							if existing.Email == expert.Email && existing.Area == de.Directory {
								dup = true
								break
							}
						}
						if !dup {
							gitExperts = append(gitExperts, types.GitExpertHit{
								Name:      expert.Name,
								Email:     expert.Email,
								Area:      de.Directory,
								Ownership: expert.Ownership,
								Active:    expert.Active,
							})
						}
						break
					}
				}
			}
		}
		if len(gitExperts) > 20 {
			gitExperts = gitExperts[:20]
		}
	}

	// Semantic search: also find results via TF-IDF similarity
	var relevantConversations []types.Conversation
	semanticSource := ""
	engine := s.getTFIDFEngine()
	if engine != nil && len(engine.Vocabulary) > 0 {
		queryVec := engine.Vectorize(query)
		if queryVec != nil {
			semanticResults, err := s.sqliteIndex.SearchSemantic(queryVec, "", 10)
			if err == nil && len(semanticResults) > 0 {
				semanticSource = "tfidf"
				// Merge semantic decisions/warnings that weren't already found by keyword
				existingDecIDs := make(map[string]bool)
				for _, d := range relevantDecisions {
					existingDecIDs[d.ID] = true
				}
				existingWarnIDs := make(map[string]bool)
				for _, w := range relevantWarnings {
					existingWarnIDs[w.ID] = true
				}

				var semanticConvIDs []string
				for _, sr := range semanticResults {
					switch sr.DocType {
					case "decision":
						if !existingDecIDs[sr.ID] {
							for _, d := range decisions {
								if d.ID == sr.ID {
									relevantDecisions = append(relevantDecisions, d)
									break
								}
							}
						}
					case "warning":
						if !existingWarnIDs[sr.ID] {
							for _, w := range warnings {
								if w.ID == sr.ID {
									relevantWarnings = append(relevantWarnings, w)
									break
								}
							}
						}
					case "conversation":
						semanticConvIDs = append(semanticConvIDs, sr.ID)
					}
				}

				// Load matching conversations from semantic results
				if len(semanticConvIDs) > 0 {
					allConvs, convErr := s.jsonStore.GetAllConversations()
					if convErr == nil {
						convIDSet := make(map[string]bool)
						for _, cid := range semanticConvIDs {
							convIDSet[cid] = true
						}
						for _, c := range allConvs {
							if convIDSet[c.ID] {
								relevantConversations = append(relevantConversations, c)
							}
						}
					}
				}
			}
		}
	}

	resp := &types.QueryResponse{
		Decisions:     relevantDecisions,
		Warnings:      relevantWarnings,
		Patterns:      patterns,
		Files:         relevantFiles,
		GitExperts:    gitExperts,
		Conversations: relevantConversations,
	}
	_ = semanticSource // available for future use in response metadata
	return resp, nil
}

func (s *Server) handleGetContext(params json.RawMessage) (interface{}, error) {
	var p struct {
		Intent           string   `json:"intent"`
		TargetFiles      []string `json:"target_files"`
		ProposedApproach string   `json:"proposed_approach"`
		MaxTokens        int      `json:"max_tokens"`
	}
	json.Unmarshal(params, &p)

	if p.Intent == "" {
		return nil, fmt.Errorf("intent is required")
	}
	if p.MaxTokens <= 0 {
		p.MaxTokens = 8000
	}

	// Get all knowledge
	decisions, _ := s.jsonStore.GetDecisions()
	warnings, _ := s.jsonStore.GetWarnings()
	patterns, _ := s.jsonStore.GetPatterns()

	// Scored items for relevance ranking
	type scoredDecision struct {
		decision types.Decision
		score    float64
	}
	type scoredWarning struct {
		warning types.Warning
		score   float64
	}
	type scoredPattern struct {
		pattern types.Pattern
		score   float64
	}

	var scoredDecs []scoredDecision
	var scoredWarns []scoredWarning
	var scoredPats []scoredPattern

	existingDecIDs := make(map[string]bool)
	existingWarnIDs := make(map[string]bool)

	// Score decisions
	for _, d := range decisions {
		score := 0.0
		if hasOverlap(d.RelatedFiles, p.TargetFiles) {
			score = 1.0 // file path match
		} else if containsAny(d.Content+d.Reason, p.Intent) {
			score = 0.5 // keyword match
		}
		if score > 0 {
			scoredDecs = append(scoredDecs, scoredDecision{d, score})
			existingDecIDs[d.ID] = true
		}
	}

	// Score warnings (critical warnings get boosted)
	for _, w := range warnings {
		score := 0.0
		if hasOverlap(w.RelatedFiles, p.TargetFiles) {
			score = 1.0
		} else if containsAny(w.Content+w.Reason, p.Intent) {
			score = 0.5
		}
		if w.Severity == "critical" && score > 0 {
			score += 0.5 // boost critical warnings
		}
		if score > 0 {
			scoredWarns = append(scoredWarns, scoredWarning{w, score})
			existingWarnIDs[w.ID] = true
		}
	}

	// Score patterns
	for _, pat := range patterns {
		if containsAny(pat.Name+pat.Description, p.Intent) {
			scoredPats = append(scoredPats, scoredPattern{pat, 0.5})
		}
	}

	// Semantic search: boost items found via TF-IDF similarity
	engine := s.getTFIDFEngine()
	if engine != nil && len(engine.Vocabulary) > 0 {
		queryVec := engine.Vectorize(p.Intent)
		if queryVec != nil {
			semanticResults, err := s.sqliteIndex.SearchSemantic(queryVec, "", 20)
			if err == nil {
				for _, sr := range semanticResults {
					switch sr.DocType {
					case "decision":
						if !existingDecIDs[sr.ID] {
							for _, d := range decisions {
								if d.ID == sr.ID {
									scoredDecs = append(scoredDecs, scoredDecision{d, sr.Similarity})
									existingDecIDs[d.ID] = true
									break
								}
							}
						}
					case "warning":
						if !existingWarnIDs[sr.ID] {
							for _, w := range warnings {
								if w.ID == sr.ID {
									scoredWarns = append(scoredWarns, scoredWarning{w, sr.Similarity})
									existingWarnIDs[w.ID] = true
									break
								}
							}
						}
					}
				}
			}
		}
	}

	// Graph traversal: find connected knowledge for target files
	for _, tf := range p.TargetFiles {
		edges, err := s.jsonStore.TraverseGraph("file", tf, 2)
		if err != nil {
			continue
		}
		for _, e := range edges {
			if e.ToType == "decision" && !existingDecIDs[e.ToID] {
				for _, d := range decisions {
					if d.ID == e.ToID {
						scoredDecs = append(scoredDecs, scoredDecision{d, 0.7})
						existingDecIDs[d.ID] = true
						break
					}
				}
			}
			if e.ToType == "warning" && !existingWarnIDs[e.ToID] {
				for _, w := range warnings {
					if w.ID == e.ToID {
						scoredWarns = append(scoredWarns, scoredWarning{w, 0.7})
						existingWarnIDs[w.ID] = true
						break
					}
				}
			}
			if e.FromType == "decision" && !existingDecIDs[e.FromID] {
				for _, d := range decisions {
					if d.ID == e.FromID {
						scoredDecs = append(scoredDecs, scoredDecision{d, 0.7})
						existingDecIDs[d.ID] = true
						break
					}
				}
			}
			if e.FromType == "warning" && !existingWarnIDs[e.FromID] {
				for _, w := range warnings {
					if w.ID == e.FromID {
						scoredWarns = append(scoredWarns, scoredWarning{w, 0.7})
						existingWarnIDs[w.ID] = true
						break
					}
				}
			}
		}
	}

	// Merge ancestor context
	features, _ := s.jsonStore.GetFeatures()
	for _, feat := range features {
		if feat.Status == "active" && feat.Extends != "" {
			ancestors, err := s.jsonStore.GetFeatureAncestors(feat.ID, 5)
			if err == nil {
				for _, ancestor := range ancestors {
					for _, decID := range ancestor.Decisions {
						if !existingDecIDs[decID] {
							for _, d := range decisions {
								if d.ID == decID {
									scoredDecs = append(scoredDecs, scoredDecision{d, 0.6})
									existingDecIDs[d.ID] = true
									break
								}
							}
						}
					}
					for _, warnID := range ancestor.Warnings {
						if !existingWarnIDs[warnID] {
							for _, w := range warnings {
								if w.ID == warnID {
									scoredWarns = append(scoredWarns, scoredWarning{w, 0.6})
									existingWarnIDs[w.ID] = true
									break
								}
							}
						}
					}
				}
			}
		}
	}

	// Sort by score descending
	sort.Slice(scoredDecs, func(i, j int) bool { return scoredDecs[i].score > scoredDecs[j].score })
	sort.Slice(scoredWarns, func(i, j int) bool { return scoredWarns[i].score > scoredWarns[j].score })
	sort.Slice(scoredPats, func(i, j int) bool { return scoredPats[i].score > scoredPats[j].score })

	// Token budget filling
	estimateTokens := func(text string) int { return len(text) / 4 }
	tokensUsed := 0

	// 1. Critical warnings first
	var relevantWarnings []types.Warning
	for _, sw := range scoredWarns {
		cost := estimateTokens(sw.warning.Content + sw.warning.Reason)
		if tokensUsed+cost > p.MaxTokens {
			break
		}
		relevantWarnings = append(relevantWarnings, sw.warning)
		tokensUsed += cost
	}

	// 2. Decisions matching target files
	var relevantDecisions []types.Decision
	for _, sd := range scoredDecs {
		cost := estimateTokens(sd.decision.Content + sd.decision.Reason)
		if tokensUsed+cost > p.MaxTokens {
			break
		}
		relevantDecisions = append(relevantDecisions, sd.decision)
		tokensUsed += cost
	}

	// 3. Patterns
	var relevantPatterns []types.Pattern
	for _, sp := range scoredPats {
		cost := estimateTokens(sp.pattern.Name + sp.pattern.Description)
		if tokensUsed+cost > p.MaxTokens {
			break
		}
		relevantPatterns = append(relevantPatterns, sp.pattern)
		tokensUsed += cost
	}

	// 4. Files
	var fileList []string
	fileList = append(fileList, p.TargetFiles...)
	searchedFiles, err := s.sqliteIndex.SearchFiles(p.Intent, "", 10)
	if err == nil {
		for _, f := range searchedFiles {
			fileList = append(fileList, f.Path)
		}
	}
	for _, tf := range p.TargetFiles {
		edgesFrom, _ := s.jsonStore.GetEdgesFrom("file", tf)
		for _, e := range edgesFrom {
			if e.ToType == "file" {
				fileList = append(fileList, e.ToID)
			}
		}
		edgesTo, _ := s.jsonStore.GetEdgesTo("file", tf)
		for _, e := range edgesTo {
			if e.FromType == "file" {
				fileList = append(fileList, e.FromID)
			}
		}
	}
	fileList = uniqueStrings(fileList)
	// Estimate file list tokens
	for _, f := range fileList {
		tokensUsed += estimateTokens(f)
	}

	// 5. Git experts
	var gitExperts []types.GitExpertHit
	var cachedExperts []git.DirectoryExpert
	if err := s.loadGitKnowledge("git-experts.json", &cachedExperts); err == nil {
		for _, tf := range p.TargetFiles {
			tfLower := strings.ToLower(tf)
			for _, de := range cachedExperts {
				dirLower := strings.ToLower(de.Directory)
				if strings.HasPrefix(tfLower, dirLower) && len(de.TopExperts) > 0 {
					top := de.TopExperts[0]
					gitExperts = append(gitExperts, types.GitExpertHit{
						Name:      top.Name,
						Email:     top.Email,
						Area:      de.Directory,
						Ownership: top.Ownership,
						Active:    top.Active,
					})
				}
			}
		}
		if len(gitExperts) > 5 {
			gitExperts = gitExperts[:5]
		}
	}

	// Generate suggestions
	var suggestions []string
	if len(relevantWarnings) > 0 {
		suggestions = append(suggestions, "Review warnings before proceeding")
	}
	if len(relevantDecisions) > 0 {
		suggestions = append(suggestions, "Consider existing decisions that may affect your approach")
	}
	arch, _ := s.jsonStore.GetArchitecture()
	if arch != nil && arch.Description != "" {
		if containsAny(arch.Description, p.Intent) {
			suggestions = append(suggestions, "This touches the documented architecture - review get_architecture output")
		}
	}

	return &types.ContextResponse{
		Intent:      p.Intent,
		Decisions:   relevantDecisions,
		Warnings:    relevantWarnings,
		Patterns:    relevantPatterns,
		Files:       fileList,
		Suggestions: suggestions,
		GitExperts:  gitExperts,
		TokenBudget: &types.TokenBudget{
			Requested: p.MaxTokens,
			Used:      tokensUsed,
			Remaining: p.MaxTokens - tokensUsed,
		},
	}, nil
}

func (s *Server) handleSearch(params json.RawMessage) (interface{}, error) {
	var p struct {
		Query string   `json:"query"`
		Types []string `json:"types"`
		Limit int      `json:"limit"`
	}
	json.Unmarshal(params, &p)

	if p.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	if p.Limit == 0 {
		p.Limit = 20
	}

	results := make(map[string]interface{})

	// Search decisions
	if len(p.Types) == 0 || containsString(p.Types, "decision") {
		decisions, _ := s.sqliteIndex.SearchDecisions(p.Query, "", "", p.Limit)
		if len(decisions) > 0 {
			results["decisions"] = decisions
		}
	}

	// Search warnings
	if len(p.Types) == 0 || containsString(p.Types, "warning") {
		warnings, _ := s.sqliteIndex.SearchWarnings(p.Query, "", "", p.Limit)
		if len(warnings) > 0 {
			results["warnings"] = warnings
		}
	}

	// Search files
	if len(p.Types) == 0 || containsString(p.Types, "file") {
		files, _ := s.sqliteIndex.SearchFiles(p.Query, "", p.Limit)
		if len(files) > 0 {
			results["files"] = files
		}
	}

	// Search patterns
	if len(p.Types) == 0 || containsString(p.Types, "pattern") {
		patterns, _ := s.jsonStore.GetPatterns()
		var matchedPatterns []types.Pattern
		for _, pat := range patterns {
			if containsAny(pat.Name+pat.Description, p.Query) {
				matchedPatterns = append(matchedPatterns, pat)
			}
		}
		if len(matchedPatterns) > 0 {
			results["patterns"] = matchedPatterns
		}
	}

	return results, nil
}

func (s *Server) handleListPatterns(params json.RawMessage) (interface{}, error) {
	var p struct {
		Source string `json:"source"`
	}
	json.Unmarshal(params, &p)

	patterns, err := s.jsonStore.GetPatterns()
	if err != nil {
		return nil, err
	}

	// Filter by source if provided
	if p.Source != "" {
		var filtered []types.Pattern
		for _, pat := range patterns {
			if pat.Source == p.Source {
				filtered = append(filtered, pat)
			}
		}
		patterns = filtered
	}

	return map[string]interface{}{
		"patterns": patterns,
		"total":    len(patterns),
	}, nil
}

func (s *Server) handleGetStats(params json.RawMessage) (interface{}, error) {
	stats, err := s.jsonStore.GetStats()
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (s *Server) handleGetArchitecture(params json.RawMessage) (interface{}, error) {
	arch, err := s.jsonStore.GetArchitecture()
	if err != nil {
		return &types.Architecture{}, nil
	}
	return arch, nil
}

func (s *Server) handleGetEvolutionTimeline(params json.RawMessage) (interface{}, error) {
	var p struct {
		EventType string `json:"event_type"`
		Limit     int    `json:"limit"`
	}
	json.Unmarshal(params, &p)

	if p.Limit == 0 {
		p.Limit = 50
	}

	timeline, err := s.jsonStore.GetEvolutionTimeline()
	if err != nil {
		return nil, err
	}

	events := timeline.Events

	// Filter by event type if provided
	if p.EventType != "" {
		var filtered []types.EvolutionEvent
		for _, e := range events {
			if e.EventType == p.EventType {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	// Limit results
	if len(events) > p.Limit {
		events = events[:p.Limit]
	}

	return map[string]interface{}{
		"events": events,
		"total":  len(events),
	}, nil
}

func (s *Server) handleAddPattern(params json.RawMessage) (interface{}, error) {
	var pattern types.Pattern
	if err := json.Unmarshal(params, &pattern); err != nil {
		return nil, err
	}

	if pattern.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if pattern.Description == "" {
		return nil, fmt.Errorf("description is required")
	}

	if err := s.jsonStore.AddPattern(&pattern); err != nil {
		return nil, err
	}

	// Add evolution event for pattern adoption
	s.jsonStore.AddEvolutionEvent(&types.EvolutionEvent{
		EventType:   "pattern_adopted",
		Title:       pattern.Name,
		Description: pattern.Description,
		Impact:      "medium",
		RelatedIDs:  []string{pattern.ID},
	})

	// Store semantic vector for TF-IDF search
	s.storeSemanticVector(pattern.ID, "pattern", pattern.Name+" "+pattern.Description)

	return map[string]interface{}{
		"id":         pattern.ID,
		"created_at": pattern.CreatedAt,
	}, nil
}

func (s *Server) handleAddEvolutionEvent(params json.RawMessage) (interface{}, error) {
	var event types.EvolutionEvent
	if err := json.Unmarshal(params, &event); err != nil {
		return nil, err
	}

	if event.EventType == "" {
		return nil, fmt.Errorf("event_type is required")
	}
	if event.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if event.Description == "" {
		return nil, fmt.Errorf("description is required")
	}

	if err := s.jsonStore.AddEvolutionEvent(&event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":        event.ID,
		"timestamp": event.Timestamp,
	}, nil
}

func (s *Server) handleUpdateArchitecture(params json.RawMessage) (interface{}, error) {
	var arch types.Architecture
	if err := json.Unmarshal(params, &arch); err != nil {
		return nil, err
	}

	// Merge with existing
	existing, _ := s.jsonStore.GetArchitecture()
	if existing != nil {
		if arch.Description == "" {
			arch.Description = existing.Description
		}
		if arch.Diagram == "" {
			arch.Diagram = existing.Diagram
		}
		if len(arch.Services) == 0 {
			arch.Services = existing.Services
		}
		if len(arch.DataFlows) == 0 {
			arch.DataFlows = existing.DataFlows
		}
	}

	if err := s.jsonStore.SaveArchitecture(&arch); err != nil {
		return nil, err
	}

	// Add evolution event for architecture change
	s.jsonStore.AddEvolutionEvent(&types.EvolutionEvent{
		EventType:   "architecture_change",
		Title:       "Architecture updated",
		Description: arch.Description,
		Impact:      "high",
	})

	return map[string]interface{}{
		"success":    true,
		"updated_at": arch.UpdatedAt,
	}, nil
}

func (s *Server) handleRecallFeature(params json.RawMessage) (interface{}, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if p.ID == "" {
		return nil, fmt.Errorf("id is required")
	}

	if err := s.jsonStore.RecallFeature(p.ID); err != nil {
		return nil, err
	}

	// Get the recalled feature
	feature, _ := s.jsonStore.GetFeature(p.ID)

	return map[string]interface{}{
		"success": true,
		"feature": feature,
	}, nil
}

