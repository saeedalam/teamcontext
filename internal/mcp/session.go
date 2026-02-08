package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/saeedalam/teamcontext/pkg/types"
)

// =============================================================================
// SESSION TRACKING & AUTO-SAVE
// Track tool calls and automatically save session metadata
// =============================================================================

// =============================================================================
// SESSION TRACKING & AUTO-SAVE
// =============================================================================

// trackToolCall records a tool call in the session tracker
func (s *Server) trackToolCall(toolName string, args json.RawMessage) {
	s.session.ToolCalls++

	// Keep last 50 tool names for summary
	if len(s.session.ToolNames) < 50 {
		s.session.ToolNames = append(s.session.ToolNames, toolName)
	}

	// Extract file paths and feature from arguments
	var argMap map[string]interface{}
	if err := json.Unmarshal(args, &argMap); err == nil {
		// Track files touched
		for _, key := range []string{"path", "file", "file_path", "directory"} {
			if v, ok := argMap[key].(string); ok && v != "" {
				s.session.FilesTouched[v] = true
			}
		}
		if files, ok := argMap["files"].([]interface{}); ok {
			for _, f := range files {
				if fs, ok := f.(string); ok {
					s.session.FilesTouched[fs] = true
				}
			}
		}

		// Auto-detect active feature
		if f, ok := argMap["feature"].(string); ok && f != "" {
			s.session.ActiveFeature = f
		}
	}

	// Note: knowledge item IDs (decisions, warnings, etc.) are tracked
	// in trackResultIDs() which runs after the handler completes
}

// trackResultIDs extracts IDs from tool results to track knowledge items created
func (s *Server) trackResultIDs(toolName string, result interface{}) {
	if result == nil {
		return
	}
	// Try to extract "id" from result map
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return
	}
	id, ok := resultMap["id"].(string)
	if !ok || id == "" {
		return
	}

	switch toolName {
	case "add_decision":
		s.session.DecisionsMade = append(s.session.DecisionsMade, id)
	case "add_warning":
		s.session.WarningsAdded = append(s.session.WarningsAdded, id)
	case "add_insight":
		s.session.InsightsAdded = append(s.session.InsightsAdded, id)
	case "add_pattern":
		s.session.PatternsAdded = append(s.session.PatternsAdded, id)
	case "start_feature":
		s.session.FeaturesStarted = append(s.session.FeaturesStarted, id)
	case "archive_feature":
		s.session.FeaturesArchived = append(s.session.FeaturesArchived, id)
	}
}

const autoSaveToolCallInterval = 25

// checkAutoCaptureTriggers checks if an auto-save should fire based on tool-call
// thresholds and knowledge-creation events.
func (s *Server) checkAutoCaptureTriggers(toolName string) {
	callsSinceLastSave := s.session.ToolCalls - s.session.ToolCallsAtLastSave
	trigger := ""

	// Checkpoint: every 25 tool calls
	if callsSinceLastSave >= autoSaveToolCallInterval {
		trigger = "checkpoint"
	}
	// Knowledge: after decisions/warnings (if some activity accumulated)
	if (toolName == "add_decision" || toolName == "add_warning") && callsSinceLastSave >= 5 {
		trigger = "knowledge_created"
	}
	// Lifecycle: after feature start/archive
	if toolName == "start_feature" || toolName == "archive_feature" {
		trigger = "feature_lifecycle"
	}

	if trigger != "" {
		s.autoSaveSession(trigger)
	}
}

// autoSaveSession saves a metadata-only session record.
// Fires on shutdown, periodic checkpoints, and knowledge-creation events.
func (s *Server) autoSaveSession(trigger string) {
	// Use lower threshold for lifecycle triggers
	callsSinceLastSave := s.session.ToolCalls - s.session.ToolCallsAtLastSave
	minCalls := 5
	if trigger == "feature_lifecycle" {
		minCalls = 3
	}
	if callsSinceLastSave < minCalls {
		return
	}

	// Determine feature
	feature := s.session.ActiveFeature
	if feature == "" {
		features, err := s.jsonStore.GetFeatures()
		if err == nil {
			for _, f := range features {
				if f.Status == "active" {
					feature = f.ID
					break
				}
			}
		}
	}
	if feature == "" {
		feature = "_global"
	}

	// Ensure the feature directory exists
	featureDir := filepath.Join(s.basePath, "features", feature, "conversations")
	os.MkdirAll(featureDir, 0755)
	if feature == "_global" {
		metaPath := filepath.Join(s.basePath, "features", feature, "meta.json")
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			globalFeature := &types.Feature{
				ID:           "_global",
				Status:       "active",
				Description:  "Auto-saved sessions without a specific feature context",
				CreatedAt:    time.Now(),
				LastAccessed: time.Now(),
			}
			metaJSON, _ := json.MarshalIndent(globalFeature, "", "  ")
			os.WriteFile(metaPath, metaJSON, 0644)
		}
	}

	// Build summary from session metadata
	summary := s.buildSessionSummary(trigger)

	var filesTouched []string
	for f := range s.session.FilesTouched {
		filesTouched = append(filesTouched, f)
	}

	// Build key points
	var keyPoints []string
	toolUsage := make(map[string]int)
	for _, t := range s.session.ToolNames {
		toolUsage[t]++
	}
	for tool, count := range toolUsage {
		if count > 1 {
			keyPoints = append(keyPoints, fmt.Sprintf("Used %s %d times", tool, count))
		}
	}
	if len(s.session.DecisionsMade) > 0 {
		keyPoints = append(keyPoints, fmt.Sprintf("Made %d decision(s)", len(s.session.DecisionsMade)))
	}
	if len(s.session.WarningsAdded) > 0 {
		keyPoints = append(keyPoints, fmt.Sprintf("Added %d warning(s)", len(s.session.WarningsAdded)))
	}
	if len(s.session.InsightsAdded) > 0 {
		keyPoints = append(keyPoints, fmt.Sprintf("Added %d insight(s)", len(s.session.InsightsAdded)))
	}
	if len(s.session.FeaturesStarted) > 0 {
		keyPoints = append(keyPoints, fmt.Sprintf("Started %d feature(s)", len(s.session.FeaturesStarted)))
	}
	if len(s.session.FeaturesArchived) > 0 {
		keyPoints = append(keyPoints, fmt.Sprintf("Archived %d feature(s)", len(s.session.FeaturesArchived)))
	}

	// Use LastSaveAt as start time for incremental windows
	startTime := s.session.StartedAt
	if !s.session.LastSaveAt.IsZero() {
		startTime = s.session.LastSaveAt
	}

	conv := &types.Conversation{
		Feature:        feature,
		Summary:        summary,
		KeyPoints:      keyPoints,
		FilesDiscussed: filesTouched,
		DecisionsMade:  s.session.DecisionsMade,
		StartTime:      startTime,
		EndTime:        time.Now(),
	}

	if err := s.jsonStore.SaveConversation(conv); err != nil {
		fmt.Fprintf(os.Stderr, "Auto-save session failed: %v\n", err)
		return
	}

	// Update checkpoint tracking
	s.session.ToolCallsAtLastSave = s.session.ToolCalls
	s.session.LastSaveAt = time.Now()
	s.session.SaveCount++
	s.session.LastCheckpointID = conv.ID

	// Index in semantic search
	vectorText := conv.Summary + " " + strings.Join(conv.KeyPoints, " ") + " " + strings.Join(conv.FilesDiscussed, " ")
	s.storeSemanticVector(conv.ID, "conversation", vectorText)

	fmt.Fprintf(os.Stderr, "Session auto-saved [%s]: %s (%d tool calls since last save, %d files)\n",
		trigger, conv.ID, callsSinceLastSave, len(filesTouched))
}

// buildSessionSummary creates a human-readable summary of the session
func (s *Server) buildSessionSummary(trigger string) string {
	duration := time.Since(s.session.StartedAt).Round(time.Minute)

	// Count unique tools used
	toolSet := make(map[string]bool)
	for _, t := range s.session.ToolNames {
		toolSet[t] = true
	}

	parts := []string{
		fmt.Sprintf("Session: %d tool calls over %s", s.session.ToolCalls, duration),
	}

	if len(s.session.FilesTouched) > 0 {
		parts = append(parts, fmt.Sprintf("%d files touched", len(s.session.FilesTouched)))
	}

	knowledgeItems := len(s.session.DecisionsMade) + len(s.session.WarningsAdded) +
		len(s.session.InsightsAdded) + len(s.session.PatternsAdded)
	if knowledgeItems > 0 {
		parts = append(parts, fmt.Sprintf("%d knowledge items created", knowledgeItems))
	}

	// Identify primary activities from tool names
	var activities []string
	if toolSet["get_skeleton"] || toolSet["get_types"] || toolSet["search_code"] {
		activities = append(activities, "code exploration")
	}
	if toolSet["add_decision"] || toolSet["add_warning"] || toolSet["add_pattern"] {
		activities = append(activities, "knowledge capture")
	}
	if toolSet["index_file"] || toolSet["scan_imports"] {
		activities = append(activities, "indexing")
	}
	if toolSet["find_experts"] || toolSet["get_file_history"] {
		activities = append(activities, "git analysis")
	}
	if toolSet["query"] || toolSet["get_context"] {
		activities = append(activities, "context queries")
	}
	if len(activities) > 0 {
		parts = append(parts, "Activities: "+strings.Join(activities, ", "))
	}

	if len(s.session.FeaturesStarted) > 0 {
		parts = append(parts, fmt.Sprintf("Features started: %s", strings.Join(s.session.FeaturesStarted, ", ")))
	}
	if len(s.session.FeaturesArchived) > 0 {
		parts = append(parts, fmt.Sprintf("Features archived: %s", strings.Join(s.session.FeaturesArchived, ", ")))
	}

	switch trigger {
	case "session_end":
		parts = append(parts, "(auto-saved on session end)")
	case "checkpoint":
		parts = append(parts, "(auto-saved at checkpoint)")
	case "knowledge_created":
		parts = append(parts, "(auto-saved after knowledge creation)")
	case "feature_lifecycle":
		parts = append(parts, "(auto-saved on feature lifecycle event)")
	default:
		parts = append(parts, fmt.Sprintf("(auto-saved: %s)", trigger))
	}

	return strings.Join(parts, ". ")
}
