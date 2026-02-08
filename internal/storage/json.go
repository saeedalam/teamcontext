package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/saeedalam/teamcontext/pkg/types"
)

// JSONStore handles JSON file storage operations
type JSONStore struct {
	basePath string
	mu       sync.RWMutex
}

// NewJSONStore creates a new JSON store
func NewJSONStore(basePath string) *JSONStore {
	return &JSONStore{
		basePath: basePath,
	}
}

// BasePath returns the base path of the store
func (s *JSONStore) BasePath() string {
	return s.basePath
}

// --- Config ---

func (s *JSONStore) GetConfig() (*types.Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "config.json")
	return readJSON[types.Config](path)
}

func (s *JSONStore) SaveConfig(config *types.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "config.json")
	return writeJSON(path, config)
}

// --- Project ---

func (s *JSONStore) GetProject() (*types.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "knowledge", "project.json")
	return readJSON[types.Project](path)
}

func (s *JSONStore) SaveProject(project *types.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	project.UpdatedAt = time.Now()
	path := filepath.Join(s.basePath, "knowledge", "project.json")
	return writeJSON(path, project)
}

// --- Files Index ---

func (s *JSONStore) GetFilesIndex() (map[string]types.FileIndex, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "index", "files.json")
	result, err := readJSON[map[string]types.FileIndex](path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]types.FileIndex), nil
		}
		return nil, err
	}
	if result == nil {
		return make(map[string]types.FileIndex), nil
	}
	return *result, nil
}

func (s *JSONStore) SaveFileIndex(file *types.FileIndex) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "index", "files.json")

	files, err := readJSON[map[string]types.FileIndex](path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if files == nil {
		m := make(map[string]types.FileIndex)
		files = &m
	}

	file.IndexedAt = time.Now()
	(*files)[file.Path] = *file

	return writeJSON(path, files)
}

func (s *JSONStore) SaveFilesIndexBulk(files map[string]types.FileIndex) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "index", "files.json")
	return writeJSON(path, files)
}

func (s *JSONStore) GetFileIndex(filePath string) (*types.FileIndex, error) {
	files, err := s.GetFilesIndex()
	if err != nil {
		return nil, err
	}

	if file, ok := files[filePath]; ok {
		return &file, nil
	}
	return nil, fmt.Errorf("file not indexed: %s", filePath)
}

// --- Decisions ---

func (s *JSONStore) GetDecisions() ([]types.Decision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "knowledge", "decisions.json")
	result, err := readJSON[[]types.Decision](path)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Decision{}, nil
		}
		return nil, err
	}
	if result == nil {
		return []types.Decision{}, nil
	}
	return *result, nil
}

func (s *JSONStore) AddDecision(decision *types.Decision) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "knowledge", "decisions.json")

	decisions, err := readJSON[[]types.Decision](path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if decisions == nil {
		empty := []types.Decision{}
		decisions = &empty
	}

	decision.ID = generateID("dec")
	decision.CreatedAt = time.Now()
	if decision.Status == "" {
		decision.Status = "active"
	}

	*decisions = append(*decisions, *decision)

	return writeJSON(path, decisions)
}

func (s *JSONStore) GetDecisionsByFeature(feature string) ([]types.Decision, error) {
	decisions, err := s.GetDecisions()
	if err != nil {
		return nil, err
	}

	var filtered []types.Decision
	for _, d := range decisions {
		if d.Feature == feature {
			filtered = append(filtered, d)
		}
	}
	return filtered, nil
}

// --- Warnings ---

func (s *JSONStore) GetWarnings() ([]types.Warning, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "knowledge", "warnings.json")
	result, err := readJSON[[]types.Warning](path)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Warning{}, nil
		}
		return nil, err
	}
	if result == nil {
		return []types.Warning{}, nil
	}
	return *result, nil
}

func (s *JSONStore) AddWarning(warning *types.Warning) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "knowledge", "warnings.json")

	warnings, err := readJSON[[]types.Warning](path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if warnings == nil {
		empty := []types.Warning{}
		warnings = &empty
	}

	warning.ID = generateID("warn")
	warning.CreatedAt = time.Now()
	if warning.Severity == "" {
		warning.Severity = "warning"
	}

	*warnings = append(*warnings, *warning)

	return writeJSON(path, warnings)
}

// --- Insights ---

func (s *JSONStore) GetInsights() ([]types.Insight, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "knowledge", "insights.json")
	result, err := readJSON[[]types.Insight](path)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Insight{}, nil
		}
		return nil, err
	}
	if result == nil {
		return []types.Insight{}, nil
	}
	return *result, nil
}

func (s *JSONStore) AddInsight(insight *types.Insight) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "knowledge", "insights.json")

	insights, err := readJSON[[]types.Insight](path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if insights == nil {
		empty := []types.Insight{}
		insights = &empty
	}

	insight.ID = generateID("ins")
	insight.CreatedAt = time.Now()

	*insights = append(*insights, *insight)

	return writeJSON(path, insights)
}

// --- Features ---

func (s *JSONStore) GetFeatures() ([]types.Feature, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	featuresDir := filepath.Join(s.basePath, "features")
	entries, err := os.ReadDir(featuresDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Feature{}, nil
		}
		return nil, err
	}

	var features []types.Feature
	for _, entry := range entries {
		if entry.IsDir() {
			metaPath := filepath.Join(featuresDir, entry.Name(), "meta.json")
			feature, err := readJSON[types.Feature](metaPath)
			if err == nil && feature != nil {
				features = append(features, *feature)
			}
		}
	}

	return features, nil
}

func (s *JSONStore) GetFeature(id string) (*types.Feature, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metaPath := filepath.Join(s.basePath, "features", id, "meta.json")
	return readJSON[types.Feature](metaPath)
}

func (s *JSONStore) CreateFeature(feature *types.Feature) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	featureDir := filepath.Join(s.basePath, "features", feature.ID)

	// Create feature directory
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		return err
	}

	// Create conversations subdirectory
	if err := os.MkdirAll(filepath.Join(featureDir, "conversations"), 0755); err != nil {
		return err
	}

	feature.Status = "active"
	feature.CreatedAt = time.Now()
	feature.LastAccessed = time.Now()

	metaPath := filepath.Join(featureDir, "meta.json")
	return writeJSON(metaPath, feature)
}

func (s *JSONStore) UpdateFeature(feature *types.Feature) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	feature.LastAccessed = time.Now()
	metaPath := filepath.Join(s.basePath, "features", feature.ID, "meta.json")
	return writeJSON(metaPath, feature)
}

func (s *JSONStore) ArchiveFeature(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read current feature
	metaPath := filepath.Join(s.basePath, "features", id, "meta.json")
	feature, err := readJSON[types.Feature](metaPath)
	if err != nil {
		return err
	}

	// Update status
	feature.Status = "archived"
	feature.ArchivedAt = time.Now()

	// Save updated meta
	if err := writeJSON(metaPath, feature); err != nil {
		return err
	}

	// Move to archive directory
	srcDir := filepath.Join(s.basePath, "features", id)
	dstDir := filepath.Join(s.basePath, "archive", id)

	return os.Rename(srcDir, dstDir)
}

func (s *JSONStore) RecallFeature(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Move from archive to features
	srcDir := filepath.Join(s.basePath, "archive", id)
	dstDir := filepath.Join(s.basePath, "features", id)

	if err := os.Rename(srcDir, dstDir); err != nil {
		return err
	}

	// Read and update feature
	metaPath := filepath.Join(dstDir, "meta.json")
	feature, err := readJSON[types.Feature](metaPath)
	if err != nil {
		return err
	}

	feature.Status = "active"
	feature.LastAccessed = time.Now()

	return writeJSON(metaPath, feature)
}

// --- Feature Ancestry ---

// GetFeatureAncestors walks the Extends chain up to maxDepth levels, returning ancestor features (parent first).
func (s *JSONStore) GetFeatureAncestors(featureID string, maxDepth int) ([]types.Feature, error) {
	if maxDepth <= 0 {
		maxDepth = 5
	}

	var ancestors []types.Feature
	visited := make(map[string]bool)
	currentID := featureID

	for i := 0; i < maxDepth; i++ {
		feature, err := s.GetFeature(currentID)
		if err != nil {
			break
		}
		if feature.Extends == "" {
			break
		}
		if visited[feature.Extends] {
			break // cycle detection
		}
		visited[feature.Extends] = true

		parent, err := s.GetFeature(feature.Extends)
		if err != nil {
			// Try archived features
			archivedPath := filepath.Join(s.basePath, "archive", feature.Extends, "meta.json")
			parent, err = readJSON[types.Feature](archivedPath)
			if err != nil {
				break
			}
		}
		ancestors = append(ancestors, *parent)
		currentID = parent.ID
	}

	return ancestors, nil
}

// --- Conversations ---

func (s *JSONStore) GetConversations(featureID string) ([]types.Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	convDir := filepath.Join(s.basePath, "features", featureID, "conversations")
	entries, err := os.ReadDir(convDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Conversation{}, nil
		}
		return nil, err
	}

	var conversations []types.Conversation
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			convPath := filepath.Join(convDir, entry.Name())
			conv, err := readJSON[types.Conversation](convPath)
			if err == nil && conv != nil {
				conversations = append(conversations, *conv)
			}
		}
	}

	return conversations, nil
}

func (s *JSONStore) SaveConversation(conv *types.Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv.ID = generateID("conv")
	conv.CreatedAt = time.Now()

	convDir := filepath.Join(s.basePath, "features", conv.Feature, "conversations")
	if err := os.MkdirAll(convDir, 0755); err != nil {
		return err
	}

	convPath := filepath.Join(convDir, conv.ID+".json")
	return writeJSON(convPath, conv)
}

// GetAllConversations returns conversations from all features.
func (s *JSONStore) GetAllConversations() ([]types.Conversation, error) {
	features, err := s.GetFeatures()
	if err != nil {
		return nil, err
	}
	var all []types.Conversation
	for _, f := range features {
		convs, _ := s.GetConversations(f.ID)
		all = append(all, convs...)
	}
	return all, nil
}

// --- Patterns ---

func (s *JSONStore) GetPatterns() ([]types.Pattern, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "knowledge", "patterns.json")
	result, err := readJSON[[]types.Pattern](path)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Pattern{}, nil
		}
		return nil, err
	}
	if result == nil {
		return []types.Pattern{}, nil
	}
	return *result, nil
}

func (s *JSONStore) AddPattern(pattern *types.Pattern) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "knowledge", "patterns.json")

	patterns, err := readJSON[[]types.Pattern](path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if patterns == nil {
		empty := []types.Pattern{}
		patterns = &empty
	}

	pattern.ID = generateID("pat")
	pattern.CreatedAt = time.Now()
	if pattern.Source == "" {
		pattern.Source = "manual"
	}

	*patterns = append(*patterns, *pattern)

	return writeJSON(path, patterns)
}

func (s *JSONStore) GetPattern(id string) (*types.Pattern, error) {
	patterns, err := s.GetPatterns()
	if err != nil {
		return nil, err
	}

	for _, p := range patterns {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("pattern not found: %s", id)
}

// --- Knowledge Graph ---

func (s *JSONStore) GetKnowledgeGraph() (*types.KnowledgeGraph, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "knowledge", "graph.json")
	result, err := readJSON[types.KnowledgeGraph](path)
	if err != nil {
		if os.IsNotExist(err) {
			return &types.KnowledgeGraph{Edges: []types.Edge{}}, nil
		}
		return nil, err
	}
	if result == nil {
		return &types.KnowledgeGraph{Edges: []types.Edge{}}, nil
	}
	return result, nil
}

func (s *JSONStore) AddEdge(edge *types.Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "knowledge", "graph.json")

	graph, err := readJSON[types.KnowledgeGraph](path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if graph == nil {
		graph = &types.KnowledgeGraph{Edges: []types.Edge{}}
	}

	// Check for duplicate edge
	for _, e := range graph.Edges {
		if e.FromType == edge.FromType && e.FromID == edge.FromID &&
			e.ToType == edge.ToType && e.ToID == edge.ToID && e.Relation == edge.Relation {
			return nil // Edge already exists
		}
	}

	graph.Edges = append(graph.Edges, *edge)

	return writeJSON(path, graph)
}

func (s *JSONStore) AddEdgesBulk(edges []types.Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "knowledge", "graph.json")

	graph, err := readJSON[types.KnowledgeGraph](path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if graph == nil {
		graph = &types.KnowledgeGraph{Edges: []types.Edge{}}
	}

	graph.Edges = append(graph.Edges, edges...)

	return writeJSON(path, graph)
}

func (s *JSONStore) GetEdgesFrom(nodeType, nodeID string) ([]types.Edge, error) {
	graph, err := s.GetKnowledgeGraph()
	if err != nil {
		return nil, err
	}

	var edges []types.Edge
	for _, e := range graph.Edges {
		if e.FromType == nodeType && e.FromID == nodeID {
			edges = append(edges, e)
		}
	}
	return edges, nil
}

func (s *JSONStore) GetEdgesTo(nodeType, nodeID string) ([]types.Edge, error) {
	graph, err := s.GetKnowledgeGraph()
	if err != nil {
		return nil, err
	}

	var edges []types.Edge
	for _, e := range graph.Edges {
		if e.ToType == nodeType && e.ToID == nodeID {
			edges = append(edges, e)
		}
	}
	return edges, nil
}

// --- Graph Traversal ---

// TraverseGraph does BFS traversal from a starting node, following edges in both directions up to maxDepth.
// Returns all edges found during traversal.
func (s *JSONStore) TraverseGraph(startType, startID string, maxDepth int) ([]types.Edge, error) {
	if maxDepth <= 0 {
		maxDepth = 2
	}
	if maxDepth > 5 {
		maxDepth = 5
	}

	graph, err := s.GetKnowledgeGraph()
	if err != nil {
		return nil, err
	}

	type node struct {
		nodeType string
		nodeID   string
	}

	visited := make(map[node]bool)
	var result []types.Edge
	seenEdges := make(map[string]bool)

	// BFS queue: each entry is (nodeType, nodeID, depth)
	type queueItem struct {
		n     node
		depth int
	}
	queue := []queueItem{{n: node{startType, startID}, depth: 0}}
	visited[node{startType, startID}] = true

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if item.depth >= maxDepth {
			continue
		}

		// Find all edges connected to this node (both directions)
		for _, e := range graph.Edges {
			edgeKey := fmt.Sprintf("%s:%s->%s:%s:%s", e.FromType, e.FromID, e.ToType, e.ToID, e.Relation)

			var neighbor node
			if e.FromType == item.n.nodeType && e.FromID == item.n.nodeID {
				neighbor = node{e.ToType, e.ToID}
			} else if e.ToType == item.n.nodeType && e.ToID == item.n.nodeID {
				neighbor = node{e.FromType, e.FromID}
			} else {
				continue
			}

			if !seenEdges[edgeKey] {
				seenEdges[edgeKey] = true
				result = append(result, e)
			}

			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, queueItem{n: neighbor, depth: item.depth + 1})
			}
		}
	}

	return result, nil
}

// --- Evolution Timeline ---

func (s *JSONStore) GetEvolutionTimeline() (*types.EvolutionTimeline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "knowledge", "evolution.json")
	result, err := readJSON[types.EvolutionTimeline](path)
	if err != nil {
		if os.IsNotExist(err) {
			return &types.EvolutionTimeline{Events: []types.EvolutionEvent{}}, nil
		}
		return nil, err
	}
	if result == nil {
		return &types.EvolutionTimeline{Events: []types.EvolutionEvent{}}, nil
	}
	return result, nil
}

func (s *JSONStore) AddEvolutionEvent(event *types.EvolutionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "knowledge", "evolution.json")

	timeline, err := readJSON[types.EvolutionTimeline](path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if timeline == nil {
		timeline = &types.EvolutionTimeline{Events: []types.EvolutionEvent{}}
	}

	event.ID = generateID("evt")
	event.Timestamp = time.Now()
	if event.Impact == "" {
		event.Impact = "medium"
	}

	timeline.Events = append(timeline.Events, *event)

	return writeJSON(path, timeline)
}

// --- Architecture ---

func (s *JSONStore) GetArchitecture() (*types.Architecture, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "index", "architecture.json")
	result, err := readJSON[types.Architecture](path)
	if err != nil {
		if os.IsNotExist(err) {
			return &types.Architecture{}, nil
		}
		return nil, err
	}
	return result, nil
}

func (s *JSONStore) SaveArchitecture(arch *types.Architecture) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	arch.UpdatedAt = time.Now()
	path := filepath.Join(s.basePath, "index", "architecture.json")
	return writeJSON(path, arch)
}

// --- API Endpoints ---

func (s *JSONStore) GetApiEndpoints() ([]types.ApiEndpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "index", "api-surface.json")
	result, err := readJSON[[]types.ApiEndpoint](path)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.ApiEndpoint{}, nil
		}
		return nil, err
	}
	if result == nil {
		return []types.ApiEndpoint{}, nil
	}
	return *result, nil
}

func (s *JSONStore) SaveApiEndpoints(endpoints []types.ApiEndpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "index", "api-surface.json")
	return writeJSON(path, endpoints)
}

// --- Stats ---

func (s *JSONStore) GetStats() (*types.Stats, error) {
	files, _ := s.GetFilesIndex()
	decisions, _ := s.GetDecisions()
	warnings, _ := s.GetWarnings()
	patterns, _ := s.GetPatterns()
	insights, _ := s.GetInsights()
	features, _ := s.GetFeatures()

	var activeFeatures, archivedFeatures int
	for _, f := range features {
		if f.Status == "active" {
			activeFeatures++
		} else if f.Status == "archived" {
			archivedFeatures++
		}
	}

	// Count conversations across all features
	var totalConversations int
	for _, f := range features {
		convs, _ := s.GetConversations(f.ID)
		totalConversations += len(convs)
	}

	return &types.Stats{
		FilesIndexed:     len(files),
		Decisions:        len(decisions),
		Warnings:         len(warnings),
		Patterns:         len(patterns),
		Insights:         len(insights),
		Features:         len(features),
		ActiveFeatures:   activeFeatures,
		ArchivedFeatures: archivedFeatures,
		Conversations:    totalConversations,
	}, nil
}

// --- Helpers ---

func readJSON[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func writeJSON(path string, v any) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	// Trailing newline for clean git diffs
	data = append(data, '\n')

	// Atomic write: write to temp file then rename to prevent corruption
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func generateID(prefix string) string {
	now := time.Now()
	short := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s-%s", prefix, now.Format("20060102"), short)
}
