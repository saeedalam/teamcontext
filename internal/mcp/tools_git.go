package mcp

import (
"encoding/json"
"fmt"
"path/filepath"
"strings"

"github.com/saeedalam/teamcontext/internal/git"
)

// =============================================================================
// GIT INTELLIGENCE HANDLERS
// Mining the team's institutional memory from Git history
// =============================================================================


// =============================================================================
// GIT INTELLIGENCE HANDLERS
// Mining the team's institutional memory from Git history
// =============================================================================

func (s *Server) handleFindExperts(params json.RawMessage) (interface{}, error) {
	var p struct {
		Files []string `json:"files"`
		Area  string   `json:"area"`
		Limit int      `json:"limit"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Limit <= 0 {
		p.Limit = 20 // Default limit to prevent massive outputs
	}
	if p.Limit > 50 {
		p.Limit = 50 // Max limit
	}

	// Build search context
	searchContext := "files"
	if p.Area != "" {
		searchContext = "area: " + p.Area
	} else if len(p.Files) > 0 {
		searchContext = strings.Join(p.Files, ", ")
	}

	// Try pre-computed cache first
	var cachedExperts []git.DirectoryExpert
	if err := s.loadGitKnowledge("git-experts.json", &cachedExperts); err == nil && len(cachedExperts) > 0 {
		var filtered []git.DirectoryExpert
		areaLower := strings.ToLower(p.Area)
		for _, de := range cachedExperts {
			dirLower := strings.ToLower(de.Directory)
			matched := false
			if p.Area != "" && strings.Contains(dirLower, areaLower) {
				matched = true
			}
			if len(p.Files) > 0 {
				for _, f := range p.Files {
					if strings.HasPrefix(strings.ToLower(f), dirLower) {
						matched = true
						break
					}
				}
			}
			if p.Area == "" && len(p.Files) == 0 {
				matched = true
			}
			if matched {
				filtered = append(filtered, de)
			}
		}
		if len(filtered) > 0 {
			totalCount := len(filtered)
			truncated := false
			if len(filtered) > p.Limit {
				filtered = filtered[:p.Limit]
				truncated = true
			}
			return map[string]interface{}{
				"search":       searchContext,
				"experts":      filtered,
				"expert_count": len(filtered),
				"total_count":  totalCount,
				"truncated":    truncated,
				"source":       "cached",
				"note":         "Experts ranked by ownership % and activity. Check 'is_active' to see if they're still contributing.",
			}, nil
		}
	}

	// Fall back to live git analysis
	projectRoot := filepath.Dir(s.basePath)
	analyzer := git.NewHistoryAnalyzer(projectRoot)

	experts, err := analyzer.FindExperts(p.Files, p.Area)
	if err != nil {
		return nil, err
	}

	totalCount := len(experts)
	truncated := false
	if len(experts) > p.Limit {
		experts = experts[:p.Limit]
		truncated = true
	}
	return map[string]interface{}{
		"search":       searchContext,
		"experts":      experts,
		"expert_count": len(experts),
		"total_count":  totalCount,
		"truncated":    truncated,
		"source":       "live",
		"note":         "Experts ranked by ownership % and activity. Check 'is_active' to see if they're still contributing.",
	}, nil
}

func (s *Server) handleGetFileHistory(params json.RawMessage) (interface{}, error) {
	var p struct {
		File string `json:"file"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.File == "" {
		return nil, fmt.Errorf("file is required")
	}

	projectRoot := filepath.Dir(s.basePath)
	analyzer := git.NewHistoryAnalyzer(projectRoot)

	expertise, err := analyzer.GetFileExpertise(p.File)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"file":             expertise.File,
		"total_commits":    expertise.TotalCommits,
		"total_lines":      expertise.TotalLines,
		"churn_rate":       expertise.ChurnRate,
		"last_modified":    expertise.LastModified,
		"last_modified_by": expertise.LastModifiedBy,
		"contributors":     expertise.Contributors,
		"contributor_count": len(expertise.Contributors),
	}, nil
}

func (s *Server) handleGetKnowledgeRisks(params json.RawMessage) (interface{}, error) {
	var p struct {
		Area  string `json:"area"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Limit <= 0 {
		p.Limit = 30 // Default limit to prevent massive outputs
	}
	if p.Limit > 100 {
		p.Limit = 100 // Max limit
	}

	var risks []git.KnowledgeRisk
	source := "live"

	// Try pre-computed cache first
	var cachedRisks []git.KnowledgeRisk
	if err := s.loadGitKnowledge("git-risks.json", &cachedRisks); err == nil {
		risks = cachedRisks
		source = "cached"
	} else {
		// Fall back to live git analysis
		projectRoot := filepath.Dir(s.basePath)
		analyzer := git.NewHistoryAnalyzer(projectRoot)

		var err error
		risks, err = analyzer.AnalyzeKnowledgeRisk()
		if err != nil {
			return nil, err
		}
	}

	// Filter by area if specified
	if p.Area != "" {
		var filtered []git.KnowledgeRisk
		for _, r := range risks {
			if strings.Contains(strings.ToLower(r.Area), strings.ToLower(p.Area)) {
				filtered = append(filtered, r)
			}
		}
		risks = filtered
	}

	// Count by risk level (before truncation)
	riskCounts := map[string]int{}
	for _, r := range risks {
		riskCounts[r.RiskLevel]++
	}

	// Apply limit
	totalCount := len(risks)
	truncated := false
	if len(risks) > p.Limit {
		risks = risks[:p.Limit]
		truncated = true
	}

	return map[string]interface{}{
		"risks":        risks,
		"risk_counts":  riskCounts,
		"returned":     len(risks),
		"total_risks":  totalCount,
		"truncated":    truncated,
		"source":       source,
		"note":         "CRITICAL risks need immediate attention. HIGH risks should be addressed soon.",
	}, nil
}

func (s *Server) handleGetFileCorrelations(params json.RawMessage) (interface{}, error) {
	var p struct {
		File           string  `json:"file"`
		MinCorrelation float64 `json:"min_correlation"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.File == "" {
		return nil, fmt.Errorf("file is required")
	}
	if p.MinCorrelation == 0 {
		p.MinCorrelation = 0.3 // Default to 30% correlation
	}

	// Try pre-computed cache first
	var cachedCorrelations []git.FileCorrelation
	if err := s.loadGitKnowledge("git-correlations.json", &cachedCorrelations); err == nil {
		fileLower := strings.ToLower(p.File)
		var filtered []git.FileCorrelation
		for _, c := range cachedCorrelations {
			if c.Correlation < p.MinCorrelation {
				continue
			}
			if strings.ToLower(c.File1) == fileLower || strings.ToLower(c.File2) == fileLower {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) > 0 {
			return map[string]interface{}{
				"file":              p.File,
				"correlations":      filtered,
				"correlation_count": len(filtered),
				"source":            "cached",
				"note":              "Files that usually change together. Consider updating these when modifying the source file.",
			}, nil
		}
	}

	// Fall back to live git analysis
	projectRoot := filepath.Dir(s.basePath)
	analyzer := git.NewHistoryAnalyzer(projectRoot)

	correlations, err := analyzer.GetFileCorrelations(p.File, p.MinCorrelation)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"file":              p.File,
		"correlations":      correlations,
		"correlation_count": len(correlations),
		"source":            "live",
		"note":              "Files that usually change together. Consider updating these when modifying the source file.",
	}, nil
}

func (s *Server) handleGetCommitContext(params json.RawMessage) (interface{}, error) {
	var p struct {
		File  string `json:"file"`
		Lines []int  `json:"lines"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.File == "" {
		return nil, fmt.Errorf("file is required")
	}

	projectRoot := filepath.Dir(s.basePath)
	analyzer := git.NewHistoryAnalyzer(projectRoot)

	context, err := analyzer.GetCommitContext(p.File, p.Lines)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"file":         context.File,
		"lines":        context.Lines,
		"commits":      context.Commits,
		"commit_count": len(context.Commits),
		"contributors": context.Contributors,
		"time_span":    context.TimeSpan,
		"summary":      context.Summary,
	}, nil
}
