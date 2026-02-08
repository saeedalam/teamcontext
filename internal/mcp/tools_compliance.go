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
)

// =============================================================================
// COMPLIANCE & ONBOARDING TOOLS
// Check code compliance and onboard new team members
// =============================================================================

// --- Compliance & Onboarding Tools ---

func (s *Server) handleCheckCompliance(params json.RawMessage) (interface{}, error) {
	var p struct {
		FilePath string `json:"file_path"`
		Diff     string `json:"diff"`
		Code     string `json:"code"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// Get the code to check
	var codeContent string
	var sourceName string
	if p.FilePath != "" {
		absPath := p.FilePath
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(filepath.Dir(s.basePath), absPath)
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("cannot read file: %w", err)
		}
		codeContent = string(data)
		sourceName = p.FilePath
	} else if p.Diff != "" {
		codeContent = p.Diff
		sourceName = "diff"
	} else if p.Code != "" {
		codeContent = p.Code
		sourceName = "code snippet"
	} else {
		return nil, fmt.Errorf("one of file_path, diff, or code is required")
	}

	codeLower := strings.ToLower(codeContent)

	// Load all decisions and patterns to check against
	decisions, _ := s.jsonStore.GetDecisions()
	warnings, _ := s.jsonStore.GetWarnings()
	patterns, _ := s.jsonStore.GetPatterns()

	type violation struct {
		Type      string `json:"type"`      // "decision", "warning", "pattern"
		ID        string `json:"id"`        // ID of the violated item
		Severity  string `json:"severity"`  // "blocker", "warning", "info"
		Message   string `json:"message"`   // What was violated
		Reference string `json:"reference"` // The original decision/pattern text
	}

	var violations []violation

	// Check decisions — look for keywords that suggest violation
	for _, d := range decisions {
		if d.Status != "" && d.Status != "active" {
			continue
		}
		// Extract negative keywords from decisions (e.g., "avoid X", "don't use Y", "ban Z")
		contentLower := strings.ToLower(d.Content)
		reasonLower := strings.ToLower(d.Reason)

		// Check for "avoid/ban/don't use X" patterns
		avoidTerms := extractProhibitedTerms(contentLower + " " + reasonLower)
		for _, term := range avoidTerms {
			if term != "" && len(term) > 2 && strings.Contains(codeLower, term) {
				violations = append(violations, violation{
					Type:      "decision",
					ID:        d.ID,
					Severity:  "blocker",
					Message:   fmt.Sprintf("Code contains '%s' which conflicts with decision: %s", term, d.Content),
					Reference: d.Content + " — " + d.Reason,
				})
			}
		}

		// Check if decision mentions specific files and this is that file
		if sourceName != "diff" && sourceName != "code snippet" {
			for _, rf := range d.RelatedFiles {
				if strings.Contains(sourceName, rf) || strings.Contains(rf, sourceName) {
					// File matches a decision's related file — flag for awareness
					violations = append(violations, violation{
						Type:      "decision",
						ID:        d.ID,
						Severity:  "info",
						Message:   fmt.Sprintf("This file is governed by decision: %s", d.Content),
						Reference: d.Content + " — " + d.Reason,
					})
				}
			}
		}
	}

	// Check warnings — look for warning patterns in code
	for _, w := range warnings {
		if w.Severity == "critical" || w.Severity == "warning" {
			warnLower := strings.ToLower(w.Content)
			// Extract key terms from warning content
			terms := extractKeyTerms(warnLower)
			matchCount := 0
			for _, term := range terms {
				if len(term) > 3 && strings.Contains(codeLower, term) {
					matchCount++
				}
			}
			// If multiple warning terms appear in the code, flag it
			if matchCount >= 2 {
				sev := "warning"
				if w.Severity == "critical" {
					sev = "blocker"
				}
				violations = append(violations, violation{
					Type:      "warning",
					ID:        w.ID,
					Severity:  sev,
					Message:   fmt.Sprintf("Code may trigger known pitfall: %s", w.Content),
					Reference: w.Content + " — " + w.Reason,
				})
			}
		}
	}

	// Check pattern anti-patterns
	for _, pat := range patterns {
		for _, anti := range pat.AntiPatterns {
			antiLower := strings.ToLower(anti)
			terms := extractKeyTerms(antiLower)
			matchCount := 0
			for _, term := range terms {
				if len(term) > 3 && strings.Contains(codeLower, term) {
					matchCount++
				}
			}
			if matchCount >= 2 {
				violations = append(violations, violation{
					Type:      "pattern",
					ID:        pat.ID,
					Severity:  "warning",
					Message:   fmt.Sprintf("Code may violate pattern '%s': anti-pattern '%s' detected", pat.Name, anti),
					Reference: pat.Name + ": " + pat.Description,
				})
			}
		}
	}

	// Deduplicate by ID
	seen := make(map[string]bool)
	var unique []violation
	for _, v := range violations {
		key := v.Type + ":" + v.ID + ":" + v.Severity
		if !seen[key] {
			seen[key] = true
			unique = append(unique, v)
		}
	}

	compliant := len(unique) == 0
	blockers := 0
	for _, v := range unique {
		if v.Severity == "blocker" {
			blockers++
		}
	}

	return map[string]interface{}{
		"source":     sourceName,
		"compliant":  compliant,
		"violations": unique,
		"blockers":   blockers,
		"checked":    fmt.Sprintf("%d decisions, %d warnings, %d patterns", len(decisions), len(warnings), len(patterns)),
	}, nil
}

// extractProhibitedTerms extracts terms that follow "avoid", "ban", "don't use", "never use", etc.
func extractProhibitedTerms(text string) []string {
	var terms []string
	prefixes := []string{"avoid ", "ban ", "don't use ", "do not use ", "never use ", "prohibit ", "no "}
	for _, prefix := range prefixes {
		idx := strings.Index(text, prefix)
		for idx != -1 {
			rest := text[idx+len(prefix):]
			// Take the next word or quoted phrase
			term := extractNextTerm(rest)
			if term != "" {
				terms = append(terms, term)
			}
			nextIdx := strings.Index(text[idx+1:], prefix)
			if nextIdx == -1 {
				break
			}
			idx = idx + 1 + nextIdx
		}
	}
	return terms
}

func extractNextTerm(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	// Take until punctuation or space
	end := strings.IndexAny(text, " ,;.\n()")
	if end == -1 {
		end = len(text)
	}
	if end > 30 {
		end = 30
	}
	return strings.TrimSpace(text[:end])
}

func extractKeyTerms(text string) []string {
	// Split on whitespace and filter short/common words
	words := strings.Fields(text)
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "is": true, "are": true, "was": true,
		"this": true, "that": true, "these": true, "those": true, "it": true,
		"not": true, "no": true, "can": true, "will": true, "should": true,
		"use": true, "used": true, "using": true, "because": true, "when": true,
	}
	var terms []string
	for _, w := range words {
		w = strings.Trim(w, ",.;:!?\"'()[]{}")
		if len(w) > 2 && !stopWords[w] {
			terms = append(terms, w)
		}
	}
	return terms
}

func (s *Server) handleOnboard(params json.RawMessage) (interface{}, error) {
	var p struct {
		Focus string `json:"focus"`
		Role  string `json:"role"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Focus == "" {
		p.Focus = "all"
	}

	onboarding := map[string]interface{}{
		"welcome": "Welcome to the team! Here's everything you need to know about this project.",
	}

	// 1. Project overview
	if p.Focus == "all" || p.Focus == "architecture" {
		project, err := s.jsonStore.GetProject()
		if err == nil && project != nil {
			projectInfo := map[string]interface{}{
				"name":        project.Name,
				"description": project.Description,
				"languages":   project.Languages,
			}
			if project.Architecture != nil {
				projectInfo["architecture"] = project.Architecture
			}
			if len(project.Goals) > 0 {
				projectInfo["goals"] = project.Goals
			}
			projectInfo["team_mindset"] = project.TeamMindset
			onboarding["project"] = projectInfo
		}
	}

	// 2. Top decisions (architectural, binding)
	if p.Focus == "all" || p.Focus == "architecture" {
		decisions, _ := s.jsonStore.GetDecisions()
		var topDecisions []map[string]interface{}
		for _, d := range decisions {
			if d.Status != "" && d.Status != "active" {
				continue
			}
			entry := map[string]interface{}{
				"id":      d.ID,
				"what":    d.Content,
				"why":     d.Reason,
				"feature": d.Feature,
			}
			if len(d.Alternatives) > 0 {
				entry["alternatives_considered"] = d.Alternatives
			}
			topDecisions = append(topDecisions, entry)
			if len(topDecisions) >= 15 {
				break
			}
		}
		onboarding["key_decisions"] = topDecisions
		onboarding["decision_count"] = len(decisions)
	}

	// 3. Active warnings
	if p.Focus == "all" || p.Focus == "warnings" {
		warnings, _ := s.jsonStore.GetWarnings()
		var activeWarnings []map[string]interface{}
		for _, w := range warnings {
			entry := map[string]interface{}{
				"id":       w.ID,
				"severity": w.Severity,
				"what":     w.Content,
				"why":      w.Reason,
			}
			if w.Evidence != "" {
				entry["evidence"] = w.Evidence
			}
			activeWarnings = append(activeWarnings, entry)
			if len(activeWarnings) >= 10 {
				break
			}
		}
		onboarding["warnings"] = activeWarnings
	}

	// 4. Patterns
	if p.Focus == "all" || p.Focus == "patterns" {
		patterns, _ := s.jsonStore.GetPatterns()
		var patternList []map[string]interface{}
		for _, pat := range patterns {
			entry := map[string]interface{}{
				"name":        pat.Name,
				"description": pat.Description,
			}
			if len(pat.Rules) > 0 {
				entry["rules"] = pat.Rules
			}
			if len(pat.Examples) > 0 {
				entry["example_files"] = pat.Examples
			}
			if len(pat.AntiPatterns) > 0 {
				entry["anti_patterns"] = pat.AntiPatterns
			}
			patternList = append(patternList, entry)
		}
		onboarding["patterns"] = patternList
	}

	// 5. Code map — key files and their roles
	if p.Focus == "all" || p.Focus == "architecture" {
		files, err := s.sqliteIndex.SearchFiles("", "", 30)
		if err == nil && len(files) > 0 {
			var codeMap []map[string]string
			for _, f := range files {
				if f.Summary != "" {
					entry := map[string]string{
						"path":    f.Path,
						"summary": f.Summary,
					}
					if f.Language != "" {
						entry["language"] = f.Language
					}
					codeMap = append(codeMap, entry)
				}
			}
			onboarding["code_map"] = codeMap
		}
	}

	// 6. Active features
	features, _ := s.jsonStore.GetFeatures()
	var activeFeatures []map[string]interface{}
	for _, f := range features {
		if f.Status == "active" || f.Status == "paused" {
			activeFeatures = append(activeFeatures, map[string]interface{}{
				"id":          f.ID,
				"status":      f.Status,
				"description": f.Description,
				"branch":      f.Branch,
			})
		}
	}
	if len(activeFeatures) > 0 {
		onboarding["active_features"] = activeFeatures
	}

	// 7. Expert contacts
	var cachedExperts []git.DirectoryExpert
	if err := s.loadGitKnowledge("git-experts.json", &cachedExperts); err == nil && len(cachedExperts) > 0 {
		expertMap := make(map[string][]string)
		for _, de := range cachedExperts {
			for _, exp := range de.TopExperts {
				if exp.Active && exp.Ownership > 0.1 {
					key := fmt.Sprintf("%s (%s)", exp.Name, exp.Email)
					expertMap[key] = append(expertMap[key], de.Directory)
				}
			}
		}
		var expertList []map[string]interface{}
		for person, areas := range expertMap {
			if len(areas) > 5 {
				areas = areas[:5]
			}
			expertList = append(expertList, map[string]interface{}{
				"expert": person,
				"areas":  areas,
			})
		}
		// Cap expert list
		if len(expertList) > 15 {
			expertList = expertList[:15]
		}
		onboarding["experts"] = expertList
	}

	// 8. Role-specific tips
	if p.Role != "" {
		tips := getRoleTips(p.Role)
		if len(tips) > 0 {
			onboarding["role_tips"] = tips
		}
	}

	// 9. Knowledge risks (bus factor)
	var risks []git.KnowledgeRisk
	if err := s.loadGitKnowledge("git-risks.json", &risks); err == nil && len(risks) > 0 {
		var highRisks []map[string]string
		for _, r := range risks {
			if r.RiskLevel == "high" || r.RiskLevel == "critical" {
				highRisks = append(highRisks, map[string]string{
					"area":           r.Area,
					"risk":           r.RiskLevel,
					"reason":         r.Reason,
					"primary_expert": r.PrimaryExpert,
				})
			}
			if len(highRisks) >= 5 {
				break
			}
		}
		if len(highRisks) > 0 {
			onboarding["knowledge_risks"] = highRisks
		}
	}

	onboarding["next_steps"] = []string{
		"Use 'get_context' before modifying any file to check for related decisions/warnings.",
		"Use 'find_experts' to identify who to ask about specific areas.",
		"Use 'resume_context' with a feature ID to continue previous work.",
		"Use 'check_compliance' after writing code to validate against team conventions.",
	}

	return onboarding, nil
}

func getRoleTips(role string) []string {
	switch strings.ToLower(role) {
	case "frontend":
		return []string{
			"Check 'patterns' for UI component conventions.",
			"Look at warnings tagged with 'frontend' or 'ui'.",
			"Use 'get_api_surface' to understand backend endpoints you'll consume.",
		}
	case "backend":
		return []string{
			"Check 'patterns' for API and service conventions.",
			"Use 'get_schema_models' to understand data models.",
			"Look at warnings tagged with 'backend', 'api', or 'database'.",
		}
	case "fullstack":
		return []string{
			"Use 'get_api_surface' for endpoint overview.",
			"Use 'get_schema_models' for data model overview.",
			"Check patterns for both frontend and backend conventions.",
		}
	case "devops":
		return []string{
			"Use 'get_config_map' to understand all configuration files.",
			"Look at warnings tagged with 'infra', 'deploy', or 'config'.",
			"Check decisions related to deployment and infrastructure.",
		}
	default:
		return nil
	}
}

// handleGetFeed returns a timeline of recent team activity
func (s *Server) handleGetFeed(params json.RawMessage) (interface{}, error) {
	var p struct {
		Limit int    `json:"limit"`
		Since string `json:"since"`
		Type  string `json:"type"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if p.Limit <= 0 {
		p.Limit = 20
	}

	// Parse since
	var sinceTime time.Time
	if p.Since != "" {
		sinceTime = parseFeedSince(p.Since)
	}

	type feedItem struct {
		Type      string    `json:"type"`
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Detail    string    `json:"detail,omitempty"`
		Author    string    `json:"author,omitempty"`
		Feature   string    `json:"feature,omitempty"`
		Severity  string    `json:"severity,omitempty"`
		CreatedAt time.Time `json:"created_at"`
	}

	var items []feedItem

	// Decisions
	if p.Type == "" || p.Type == "decision" {
		decisions, _ := s.jsonStore.GetDecisions()
		for _, d := range decisions {
			if !sinceTime.IsZero() && d.CreatedAt.Before(sinceTime) {
				continue
			}
			items = append(items, feedItem{
				Type: "decision", ID: d.ID, Title: d.Content,
				Detail: d.Reason, Author: d.Author, Feature: d.Feature,
				CreatedAt: d.CreatedAt,
			})
		}
	}

	// Warnings
	if p.Type == "" || p.Type == "warning" {
		warnings, _ := s.jsonStore.GetWarnings()
		for _, w := range warnings {
			if !sinceTime.IsZero() && w.CreatedAt.Before(sinceTime) {
				continue
			}
			items = append(items, feedItem{
				Type: "warning", ID: w.ID, Title: w.Content,
				Detail: w.Reason, Author: w.Author, Feature: w.Feature,
				Severity: w.Severity, CreatedAt: w.CreatedAt,
			})
		}
	}

	// Patterns
	if p.Type == "" || p.Type == "pattern" {
		patterns, _ := s.jsonStore.GetPatterns()
		for _, pat := range patterns {
			if !sinceTime.IsZero() && pat.CreatedAt.Before(sinceTime) {
				continue
			}
			items = append(items, feedItem{
				Type: "pattern", ID: pat.ID, Title: pat.Name,
				Detail: pat.Description, CreatedAt: pat.CreatedAt,
			})
		}
	}

	// Insights
	if p.Type == "" || p.Type == "insight" {
		insights, _ := s.jsonStore.GetInsights()
		for _, ins := range insights {
			if !sinceTime.IsZero() && ins.CreatedAt.Before(sinceTime) {
				continue
			}
			items = append(items, feedItem{
				Type: "insight", ID: ins.ID, Title: ins.Content,
				Author: ins.Author, Feature: ins.Feature, CreatedAt: ins.CreatedAt,
			})
		}
	}

	// Conversations
	if p.Type == "" || p.Type == "conversation" {
		convs, _ := s.jsonStore.GetAllConversations()
		for _, c := range convs {
			if !sinceTime.IsZero() && c.CreatedAt.Before(sinceTime) {
				continue
			}
			items = append(items, feedItem{
				Type: "conversation", ID: c.ID, Title: c.Summary,
				Feature: c.Feature, CreatedAt: c.CreatedAt,
			})
		}
	}

	// Evolution events
	if p.Type == "" || p.Type == "event" {
		timeline, _ := s.jsonStore.GetEvolutionTimeline()
		if timeline != nil {
			for _, ev := range timeline.Events {
				if !sinceTime.IsZero() && ev.Timestamp.Before(sinceTime) {
					continue
				}
				items = append(items, feedItem{
					Type: "event", ID: ev.ID, Title: ev.Title,
					Detail: ev.Description, Author: ev.Author, CreatedAt: ev.Timestamp,
				})
			}
		}
	}

	// Sort by most recent
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	// Apply limit
	if len(items) > p.Limit {
		items = items[:p.Limit]
	}

	return map[string]interface{}{
		"entries": items,
		"total":   len(items),
	}, nil
}

func parseFeedSince(s string) time.Time {
	if len(s) > 1 {
		unit := s[len(s)-1]
		numStr := s[:len(s)-1]
		var num int
		if _, err := fmt.Sscanf(numStr, "%d", &num); err == nil {
			switch unit {
			case 'd':
				return time.Now().Add(-time.Duration(num) * 24 * time.Hour)
			case 'h':
				return time.Now().Add(-time.Duration(num) * time.Hour)
			case 'm':
				return time.Now().Add(-time.Duration(num) * time.Minute)
			}
		}
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}
