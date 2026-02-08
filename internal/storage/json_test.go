package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/saeedalam/teamcontext/pkg/types"
)

func setupTestStore(t *testing.T) (*JSONStore, func()) {
	t.Helper()
	
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "teamcontext-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Create required subdirectories
	dirs := []string{"knowledge", "index", "features"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create %s dir: %v", dir, err)
		}
	}
	
	store := NewJSONStore(tmpDir)
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}
	
	return store, cleanup
}

// =============================================================================
// CONFIG TESTS
// =============================================================================

func TestConfigSaveAndLoad(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	
	config := &types.Config{
		Name:    "test-project",
		Version: "1.0.0",
		Server: types.ServerConfig{
			MCPEnabled: true,
		},
	}
	
	// Save
	if err := store.SaveConfig(config); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}
	
	// Load
	loaded, err := store.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	
	// Verify
	if loaded.Name != "test-project" {
		t.Errorf("Expected project name 'test-project', got '%s'", loaded.Name)
	}
	if loaded.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", loaded.Version)
	}
	if !loaded.Server.MCPEnabled {
		t.Error("Expected MCPEnabled to be true")
	}
}

// =============================================================================
// DECISION TESTS
// =============================================================================

func TestDecisionCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	
	// Create decision
	decision := &types.Decision{
		Content:      "Use Zod for validation",
		Reason:       "Better TypeScript inference",
		Context:      "Choosing validation library",
		Alternatives: []string{"class-validator", "yup"},
		Author:       "test-user",
		Status:       "active",
		Tags:         []string{"validation", "typescript"},
	}
	
	// Add
	if err := store.AddDecision(decision); err != nil {
		t.Fatalf("AddDecision failed: %v", err)
	}
	
	// Verify ID was generated
	if decision.ID == "" {
		t.Error("Decision ID should be auto-generated")
	}
	
	// List all
	decisions, err := store.GetDecisions()
	if err != nil {
		t.Fatalf("GetDecisions failed: %v", err)
	}
	
	if len(decisions) != 1 {
		t.Fatalf("Expected 1 decision, got %d", len(decisions))
	}
	
	// Verify content
	if decisions[0].Content != "Use Zod for validation" {
		t.Errorf("Expected content 'Use Zod for validation', got '%s'", decisions[0].Content)
	}
	if decisions[0].Reason != "Better TypeScript inference" {
		t.Errorf("Expected reason 'Better TypeScript inference', got '%s'", decisions[0].Reason)
	}
}

func TestDecisionMultiple(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	
	// Add multiple decisions
	for i := 0; i < 5; i++ {
		decision := &types.Decision{
			Content: "Decision " + string(rune('A'+i)),
			Reason:  "Reason " + string(rune('A'+i)),
			Status:  "active",
		}
		if err := store.AddDecision(decision); err != nil {
			t.Fatalf("AddDecision %d failed: %v", i, err)
		}
	}
	
	// List all
	decisions, err := store.GetDecisions()
	if err != nil {
		t.Fatalf("GetDecisions failed: %v", err)
	}
	
	if len(decisions) != 5 {
		t.Errorf("Expected 5 decisions, got %d", len(decisions))
	}
}

// =============================================================================
// WARNING TESTS
// =============================================================================

func TestWarningCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	
	warning := &types.Warning{
		Content:  "Don't use localStorage for tokens",
		Reason:   "Security vulnerability",
		Evidence: "XSS attacks can steal tokens",
		Severity: "critical",
		Author:   "test-user",
		Tags:     []string{"security", "auth"},
	}
	
	// Add
	if err := store.AddWarning(warning); err != nil {
		t.Fatalf("AddWarning failed: %v", err)
	}
	
	// Verify ID was generated
	if warning.ID == "" {
		t.Error("Warning ID should be auto-generated")
	}
	
	// List all
	warnings, err := store.GetWarnings()
	if err != nil {
		t.Fatalf("GetWarnings failed: %v", err)
	}
	
	if len(warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(warnings))
	}
	
	if warnings[0].Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", warnings[0].Severity)
	}
}

// =============================================================================
// FEATURE TESTS
// =============================================================================

func TestFeatureLifecycle(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	
	// Create feature
	feature := &types.Feature{
		ID:          "auth-refactor",
		Status:      "active",
		Description: "Refactor authentication system",
		Owner:       "test-user",
	}
	
	// Create
	if err := store.CreateFeature(feature); err != nil {
		t.Fatalf("CreateFeature failed: %v", err)
	}
	
	// Get
	loaded, err := store.GetFeature("auth-refactor")
	if err != nil {
		t.Fatalf("GetFeature failed: %v", err)
	}
	
	if loaded.Description != "Refactor authentication system" {
		t.Errorf("Expected description 'Refactor authentication system', got '%s'", loaded.Description)
	}
	
	// Update
	loaded.CurrentState = "Implementing JWT tokens"
	if err := store.UpdateFeature(loaded); err != nil {
		t.Fatalf("UpdateFeature failed: %v", err)
	}
	
	// Verify update
	updated, err := store.GetFeature("auth-refactor")
	if err != nil {
		t.Fatalf("GetFeature after update failed: %v", err)
	}
	
	if updated.CurrentState != "Implementing JWT tokens" {
		t.Errorf("Expected current_state 'Implementing JWT tokens', got '%s'", updated.CurrentState)
	}
}

func TestListFeatures(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	
	// Create multiple features
	features := []types.Feature{
		{ID: "feature-1", Status: "active", Description: "Feature 1"},
		{ID: "feature-2", Status: "active", Description: "Feature 2"},
		{ID: "feature-3", Status: "archived", Description: "Feature 3"},
	}
	
	for _, f := range features {
		feature := f // copy
		if err := store.CreateFeature(&feature); err != nil {
			t.Fatalf("CreateFeature failed: %v", err)
		}
	}
	
	// List all
	all, err := store.GetFeatures()
	if err != nil {
		t.Fatalf("GetFeatures failed: %v", err)
	}
	
	if len(all) != 3 {
		t.Errorf("Expected 3 features, got %d", len(all))
	}
}

// =============================================================================
// FILE INDEX TESTS
// =============================================================================

func TestFileIndexCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	
	fileIndex := &types.FileIndex{
		Path:      "src/auth/login.ts",
		Summary:   "Handles user login",
		Language:  "typescript",
		LineCount: 150,
		SizeBytes: 4500,
		IndexedAt: time.Now(),
	}
	
	// Save
	if err := store.SaveFileIndex(fileIndex); err != nil {
		t.Fatalf("SaveFileIndex failed: %v", err)
	}
	
	// Get all
	files, err := store.GetFilesIndex()
	if err != nil {
		t.Fatalf("GetFilesIndex failed: %v", err)
	}
	
	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}
	
	if file, ok := files["src/auth/login.ts"]; ok {
		if file.Summary != "Handles user login" {
			t.Errorf("Expected summary 'Handles user login', got '%s'", file.Summary)
		}
	} else {
		t.Error("File 'src/auth/login.ts' not found in index")
	}
}

// =============================================================================
// EVOLUTION EVENT TESTS
// =============================================================================

func TestEvolutionEventCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	
	event := &types.EvolutionEvent{
		EventType:   "decision",
		Title:       "Migrated to Zod",
		Description: "Switched from class-validator to Zod for validation",
		Author:      "test-user",
		Impact:      "medium",
	}
	
	// Add
	if err := store.AddEvolutionEvent(event); err != nil {
		t.Fatalf("AddEvolutionEvent failed: %v", err)
	}
	
	// Get all
	timeline, err := store.GetEvolutionTimeline()
	if err != nil {
		t.Fatalf("GetEvolutionTimeline failed: %v", err)
	}
	
	if len(timeline.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(timeline.Events))
	}
	
	if timeline.Events[0].Title != "Migrated to Zod" {
		t.Errorf("Expected title 'Migrated to Zod', got '%s'", timeline.Events[0].Title)
	}
}

// =============================================================================
// KNOWLEDGE GRAPH TESTS
// =============================================================================

func TestKnowledgeGraphEdges(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	
	// Add edge
	edge := &types.Edge{
		FromType: "decision",
		FromID:   "dec-001",
		ToType:   "file",
		ToID:     "src/auth/login.ts",
		Relation: "affects",
	}
	
	if err := store.AddEdge(edge); err != nil {
		t.Fatalf("AddEdge failed: %v", err)
	}
	
	// Get graph
	graph, err := store.GetKnowledgeGraph()
	if err != nil {
		t.Fatalf("GetKnowledgeGraph failed: %v", err)
	}
	
	if len(graph.Edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(graph.Edges))
	}
	
	if graph.Edges[0].Relation != "affects" {
		t.Errorf("Expected relation 'affects', got '%s'", graph.Edges[0].Relation)
	}
}

// =============================================================================
// CONVERSATION TESTS
// =============================================================================

func TestConversationCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	
	// First create a feature
	feature := &types.Feature{
		ID:     "test-feature",
		Status: "active",
	}
	if err := store.CreateFeature(feature); err != nil {
		t.Fatalf("CreateFeature failed: %v", err)
	}
	
	conv := &types.Conversation{
		Feature:        "test-feature",
		Summary:        "Discussed authentication flow",
		KeyPoints:      []string{"Use JWT", "Add refresh tokens"},
		FilesDiscussed: []string{"src/auth/jwt.ts"},
	}
	
	// Save
	if err := store.SaveConversation(conv); err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}
	
	// Verify ID was generated
	if conv.ID == "" {
		t.Error("Conversation ID should be auto-generated")
	}
	
	// List for feature
	convs, err := store.GetConversations("test-feature")
	if err != nil {
		t.Fatalf("GetConversations failed: %v", err)
	}
	
	if len(convs) != 1 {
		t.Fatalf("Expected 1 conversation, got %d", len(convs))
	}
	
	if convs[0].Summary != "Discussed authentication flow" {
		t.Errorf("Expected summary 'Discussed authentication flow', got '%s'", convs[0].Summary)
	}
}

// =============================================================================
// CONCURRENT ACCESS TESTS
// =============================================================================

func TestConcurrentDecisionWrites(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	
	done := make(chan bool, 10)
	
	// Write 10 decisions concurrently
	for i := 0; i < 10; i++ {
		go func(idx int) {
			decision := &types.Decision{
				Content: "Decision " + string(rune('A'+idx)),
				Reason:  "Reason " + string(rune('A'+idx)),
				Status:  "active",
			}
			if err := store.AddDecision(decision); err != nil {
				t.Errorf("Concurrent AddDecision %d failed: %v", idx, err)
			}
			done <- true
		}(i)
	}
	
	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify all were written
	decisions, err := store.GetDecisions()
	if err != nil {
		t.Fatalf("GetDecisions failed: %v", err)
	}
	
	if len(decisions) != 10 {
		t.Errorf("Expected 10 decisions after concurrent writes, got %d", len(decisions))
	}
}
