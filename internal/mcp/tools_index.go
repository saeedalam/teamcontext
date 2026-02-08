package mcp

import (
"encoding/json"
"fmt"
"os"
"path/filepath"

"github.com/saeedalam/teamcontext/internal/blueprint"
"github.com/saeedalam/teamcontext/internal/extractor"
"github.com/saeedalam/teamcontext/internal/search"
"github.com/saeedalam/teamcontext/pkg/types"
)

// =============================================================================
// INDEXING & GRAPH TOOLS
// Index files and traverse the knowledge graph
// =============================================================================

// --- Indexing Handlers ---

func (s *Server) handleIndex(params json.RawMessage) (interface{}, error) {
	var p struct {
		Incremental bool     `json:"incremental"`
		Paths       []string `json:"paths"`
	}
	json.Unmarshal(params, &p)

	// Get current index stats before
	statsBefore, _ := s.jsonStore.GetStats()

	// Return instructions for agent-driven indexing
	return map[string]interface{}{
		"status": "ready",
		"message": "TeamContext is ready for indexing. Use index_file tool to index each file.",
		"instructions": []string{
			"1. Read each file you want to index",
			"2. Generate a one-sentence summary",
			"3. Extract exports (functions, classes, types)",
			"4. Call index_file with the data",
			"5. TeamContext stores and indexes the data",
		},
		"current_stats": map[string]interface{}{
			"files_indexed": statsBefore.FilesIndexed,
			"decisions":     statsBefore.Decisions,
			"warnings":      statsBefore.Warnings,
		},
		"incremental": p.Incremental,
		"paths":       p.Paths,
	}, nil
}

func (s *Server) handleIndexStatus(params json.RawMessage) (interface{}, error) {
	stats, err := s.jsonStore.GetStats()
	if err != nil {
		return nil, err
	}

	files, _ := s.jsonStore.GetFilesIndex()

	// Get list of indexed file paths
	var indexedPaths []string
	for path := range files {
		indexedPaths = append(indexedPaths, path)
	}

	return map[string]interface{}{
		"files_indexed":   stats.FilesIndexed,
		"indexed_paths":   indexedPaths,
		"decisions":       stats.Decisions,
		"warnings":        stats.Warnings,
		"patterns":        stats.Patterns,
		"features":        stats.Features,
		"active_features": stats.ActiveFeatures,
	}, nil
}

func (s *Server) handleGetGraph(params json.RawMessage) (interface{}, error) {
	var p struct {
		NodeType string `json:"node_type"`
		NodeID   string `json:"node_id"`
	}
	json.Unmarshal(params, &p)

	graph, err := s.jsonStore.GetKnowledgeGraph()
	if err != nil {
		return nil, err
	}

	var edges []types.Edge

	if p.NodeType != "" && p.NodeID != "" {
		// Get edges for specific node
		edgesFrom, _ := s.jsonStore.GetEdgesFrom(p.NodeType, p.NodeID)
		edgesTo, _ := s.jsonStore.GetEdgesTo(p.NodeType, p.NodeID)
		edges = append(edgesFrom, edgesTo...)
	} else if p.NodeType != "" {
		// Get all edges for node type
		for _, e := range graph.Edges {
			if e.FromType == p.NodeType || e.ToType == p.NodeType {
				edges = append(edges, e)
			}
		}
	} else {
		// Return all edges
		edges = graph.Edges
	}

	return map[string]interface{}{
		"edges": edges,
		"total": len(edges),
	}, nil
}

func (s *Server) handleGetRelated(params json.RawMessage) (interface{}, error) {
	var p struct {
		NodeType string `json:"node_type"`
		NodeID   string `json:"node_id"`
		MaxDepth int    `json:"max_depth"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.NodeType == "" || p.NodeID == "" {
		return nil, fmt.Errorf("node_type and node_id are required")
	}
	if p.MaxDepth <= 0 {
		p.MaxDepth = 2
	}

	edges, err := s.jsonStore.TraverseGraph(p.NodeType, p.NodeID, p.MaxDepth)
	if err != nil {
		return nil, err
	}

	// Group results by type
	grouped := make(map[string][]string)
	for _, e := range edges {
		if e.FromType != p.NodeType || e.FromID != p.NodeID {
			key := e.FromType
			grouped[key] = appendUnique(grouped[key], e.FromID)
		}
		if e.ToType != p.NodeType || e.ToID != p.NodeID {
			key := e.ToType
			grouped[key] = appendUnique(grouped[key], e.ToID)
		}
	}

	return map[string]interface{}{
		"start_type":    p.NodeType,
		"start_id":      p.NodeID,
		"max_depth":     p.MaxDepth,
		"edges":         edges,
		"edge_count":    len(edges),
		"connected":     grouped,
	}, nil
}

// storeSemanticVector stores a semantic vector for a newly added document.
func (s *Server) storeSemanticVector(id, docType, text string) {
	engine := s.getTFIDFEngine()
	if engine == nil || len(engine.Vocabulary) == 0 {
		return
	}
	vec := engine.Vectorize(text)
	if vec == nil {
		return
	}
	s.sqliteIndex.StoreSemanticVector(id, docType, text, search.PackVector(vec))
}

// getTFIDFEngine lazy-loads the TF-IDF engine from SQLite vocab.
func (s *Server) getTFIDFEngine() *search.TFIDFEngine {
	if s.tfidfEngine != nil {
		return s.tfidfEngine
	}
	engine, err := s.sqliteIndex.LoadVocab()
	if err != nil {
		return nil
	}
	s.tfidfEngine = engine
	return engine
}

// appendUnique appends s to slice if not already present
func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

// --- Graph Integration Helpers ---

// addDecisionEdges creates graph edges for a decision
func (s *Server) addDecisionEdges(decision *types.Decision) {
	// Connect decision to related files
	for _, filePath := range decision.RelatedFiles {
		s.jsonStore.AddEdge(&types.Edge{
			FromType: "decision",
			FromID:   decision.ID,
			ToType:   "file",
			ToID:     filePath,
			Relation: "affects",
		})
	}

	// Connect decision to related decisions
	for _, decID := range decision.RelatedDecisions {
		s.jsonStore.AddEdge(&types.Edge{
			FromType: "decision",
			FromID:   decision.ID,
			ToType:   "decision",
			ToID:     decID,
			Relation: "related_to",
		})
	}

	// Connect decision to feature
	if decision.Feature != "" {
		s.jsonStore.AddEdge(&types.Edge{
			FromType: "decision",
			FromID:   decision.ID,
			ToType:   "feature",
			ToID:     decision.Feature,
			Relation: "belongs_to",
		})
	}

	// Connect superseded decision
	if decision.Supersedes != "" {
		s.jsonStore.AddEdge(&types.Edge{
			FromType: "decision",
			FromID:   decision.ID,
			ToType:   "decision",
			ToID:     decision.Supersedes,
			Relation: "supersedes",
		})
	}
}

// addWarningEdges creates graph edges for a warning
func (s *Server) addWarningEdges(warning *types.Warning) {
	// Connect warning to related files
	for _, filePath := range warning.RelatedFiles {
		s.jsonStore.AddEdge(&types.Edge{
			FromType: "warning",
			FromID:   warning.ID,
			ToType:   "file",
			ToID:     filePath,
			Relation: "warns",
		})
	}

	// Connect warning to related decisions
	for _, decID := range warning.RelatedDecisions {
		s.jsonStore.AddEdge(&types.Edge{
			FromType: "warning",
			FromID:   warning.ID,
			ToType:   "decision",
			ToID:     decID,
			Relation: "related_to",
		})
	}

	// Connect warning to feature
	if warning.Feature != "" {
		s.jsonStore.AddEdge(&types.Edge{
			FromType: "warning",
			FromID:   warning.ID,
			ToType:   "feature",
			ToID:     warning.Feature,
			Relation: "belongs_to",
		})
	}
}

// addFileEdges creates graph edges for a file
func (s *Server) addFileEdges(file *types.FileIndex) {
	// Connect file to patterns
	for _, patternID := range file.Patterns {
		s.jsonStore.AddEdge(&types.Edge{
			FromType: "file",
			FromID:   file.Path,
			ToType:   "pattern",
			ToID:     patternID,
			Relation: "follows",
		})
	}

	// Connect file to related files
	for _, relatedPath := range file.RelatedFiles {
		s.jsonStore.AddEdge(&types.Edge{
			FromType: "file",
			FromID:   file.Path,
			ToType:   "file",
			ToID:     relatedPath,
			Relation: "related_to",
		})
	}
}

// High-impact extraction handlers

func (s *Server) handleGetAPISurface(params json.RawMessage) (interface{}, error) {
	var p struct {
		Path string `json:"path"`
		App  string `json:"app"`
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

	var surface *extractor.APISurface

	if info.IsDir() {
		appName := p.App
		if appName == "" {
			appName = filepath.Base(p.Path)
		}
		surface, err = extractor.ExtractAPISurface(p.Path, appName)
	} else {
		surface, err = extractor.ExtractAPISurfaceFromFile(p.Path)
	}

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"app":             surface.App,
		"endpoints":       surface.Endpoints,
		"endpoint_count":  len(surface.Endpoints),
		"kafka_consumers": surface.KafkaConsumers,
		"kafka_producers": surface.KafkaProducers,
	}, nil
}

func (s *Server) handleGetSchemaModels(params json.RawMessage) (interface{}, error) {
	var p struct {
		Path string `json:"path"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if p.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Try Prisma-specific extraction first
	prismaSchema, prismaErr := extractor.ExtractSchemaModels(p.Path)

	// Also try multi-language extraction (Go GORM, Python SQLAlchemy/Django, Java JPA, TypeORM)
	multiSchema, multiErr := extractor.ExtractMultiLangSchema(p.Path)

	// Merge results with deduplication
	modelMap := make(map[string]extractor.SchemaModel) // key: "file:name"
	enumMap := make(map[string]extractor.SchemaEnum)   // key: "file:name"

	if prismaErr == nil && prismaSchema != nil {
		for _, m := range prismaSchema.Models {
			key := fmt.Sprintf("%s:%s", m.File, m.Name)
			modelMap[key] = m
		}
		for _, e := range prismaSchema.Enums {
			key := fmt.Sprintf("%s:%s", e.File, e.Name)
			enumMap[key] = e
		}
	}

	if multiErr == nil && multiSchema != nil {
		for _, m := range multiSchema.Models {
			key := fmt.Sprintf("%s:%s", m.File, m.Name)
			if _, exists := modelMap[key]; !exists {
				modelMap[key] = m
			}
		}
		for _, e := range multiSchema.Enums {
			key := fmt.Sprintf("%s:%s", e.File, e.Name)
			if _, exists := enumMap[key]; !exists {
				enumMap[key] = e
			}
		}
	}

	if len(modelMap) == 0 && prismaErr != nil && multiErr != nil {
		return nil, fmt.Errorf("no schema models found: prisma: %v, multi: %v", prismaErr, multiErr)
	}

	// Convert maps to slices
	allModels := make([]extractor.SchemaModel, 0, len(modelMap))
	for _, m := range modelMap {
		allModels = append(allModels, m)
	}
	allEnums := make([]extractor.SchemaEnum, 0, len(enumMap))
	for _, e := range enumMap {
		allEnums = append(allEnums, e)
	}

	// Group models by language/source
	modelsByLang := map[string]int{}
	for _, m := range allModels {
		ext := filepath.Ext(m.File)
		modelsByLang[ext]++
	}

	return map[string]interface{}{
		"models":          allModels,
		"model_count":     len(allModels),
		"enums":           allEnums,
		"enum_count":      len(allEnums),
		"models_by_type":  modelsByLang,
		"supported_langs": []string{"Prisma", "Go (GORM/sqlx)", "Python (SQLAlchemy/Django)", "Java (JPA/Hibernate)", "TypeScript (TypeORM)"},
	}, nil
}

func (s *Server) handleGetConfigMap(params json.RawMessage) (interface{}, error) {
	var p struct {
		Path string `json:"path"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if p.Path == "" {
		// Default to project root
		p.Path = filepath.Dir(s.basePath)
	}

	configMap, err := extractor.ExtractConfigMap(p.Path)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"env_vars":      configMap.EnvVars,
		"env_var_count": len(configMap.EnvVars),
		"config_files":  configMap.ConfigFiles,
	}, nil
}

func (s *Server) handleGetBlueprint(params json.RawMessage) (interface{}, error) {
	var p struct {
		Task string `json:"task"`
		App  string `json:"app"`
		Path string `json:"path"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if p.Task == "" {
		return nil, fmt.Errorf("task is required. Valid types: add-endpoint, add-feature, add-service, fix-bug, refactor, add-test")
	}

	// Convert string to TaskType
	taskType := blueprint.TaskType(p.Task)

	// Validate task type
	validTasks := map[blueprint.TaskType]bool{
		blueprint.TaskAddEndpoint: true,
		blueprint.TaskAddFeature:  true,
		blueprint.TaskAddService:  true,
		blueprint.TaskFixBug:      true,
		blueprint.TaskRefactor:    true,
		blueprint.TaskAddTest:     true,
	}

	if !validTasks[taskType] {
		return nil, fmt.Errorf("invalid task type '%s'. Valid types: add-endpoint, add-feature, add-service, fix-bug, refactor, add-test", p.Task)
	}

	// Create blueprint generator
	projectRoot := filepath.Dir(s.basePath)
	gen := blueprint.NewGenerator(projectRoot, s.basePath, s.jsonStore)

	// Generate blueprint
	bp, err := gen.Generate(taskType, p.App, p.Path)
	if err != nil {
		return nil, err
	}

	// Return the blueprint as a structured response (v2: includes snippets, imports, conventions)
	response := map[string]interface{}{
		"task_type":    bp.TaskType,
		"app":          bp.App,
		"path":         bp.Path,
		"description":  bp.Description,
		"file_pattern": bp.FilePattern,
		"examples":     bp.Examples,
		"decisions":    bp.Decisions,
		"warnings":     bp.Warnings,
		"correlations": bp.Correlations,
		"checklist":    bp.Checklist,
		"confidence":   bp.Confidence,
		"source":       bp.Source,
		"usage_hint":   "Follow the checklist. Use snippets as code templates ({Name}/{name} = your feature). Use imports as-is. Respect conventions, decisions and warnings.",
	}

	// v2 fields — only include when populated to keep response compact
	if bp.Snippets != nil {
		response["snippets"] = bp.Snippets
	}
	if bp.Imports != nil {
		response["imports"] = bp.Imports
	}
	if bp.Conventions != nil {
		response["conventions"] = bp.Conventions
	}

	return response, nil
}

