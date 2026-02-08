// Package blueprint provides task blueprint generation for AI agents.
// Blueprints combine patterns, decisions, warnings, and examples into
// actionable plans that reduce AI token usage and improve accuracy.
//
// v2: Includes code snippets, import maps, convention detection, and
// app-specific checklists — eliminating follow-up Read calls.
package blueprint

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/saeedalam/teamcontext/internal/imports"
	"github.com/saeedalam/teamcontext/internal/search"
	"github.com/saeedalam/teamcontext/internal/skeleton"
	"github.com/saeedalam/teamcontext/internal/storage"
	"github.com/saeedalam/teamcontext/pkg/types"
)

// TaskType represents a common development task
type TaskType string

const (
	TaskAddEndpoint TaskType = "add-endpoint"
	TaskAddFeature  TaskType = "add-feature"
	TaskAddService  TaskType = "add-service"
	TaskFixBug      TaskType = "fix-bug"
	TaskRefactor    TaskType = "refactor"
	TaskAddTest     TaskType = "add-test"
)

// maxSnippetLines caps each snippet to keep response compact.
const maxSnippetLines = 20

// maxExamples caps the number of examples returned.
const maxExamples = 3

// maxImportsPerType caps imports per file type.
const maxImportsPerType = 5



// tokenBudget is the approximate max token count for the whole response.
const tokenBudget = 4000

// charsPerToken is a rough estimate for token counting.
const charsPerToken = 4

// Blueprint is the structured output for AI agents
type Blueprint struct {
	TaskType    TaskType `json:"task_type"`
	App         string   `json:"app,omitempty"`
	Path        string   `json:"path,omitempty"`
	Description string   `json:"description"`

	// File structure to create
	FilePattern *FilePattern `json:"file_pattern,omitempty"`

	// Examples to follow (real files from the codebase, with skeleton)
	Examples []Example `json:"examples,omitempty"`

	// Templatized code snippets per file type
	Snippets map[string]*SnippetEntry `json:"snippets,omitempty"`

	// Required imports per file type
	Imports map[string][]string `json:"imports,omitempty"`

	// Auto-detected app conventions
	Conventions *Conventions `json:"conventions,omitempty"`

	// Relevant team decisions
	Decisions []Decision `json:"decisions,omitempty"`

	// Warnings and pitfalls to avoid
	Warnings []Warning `json:"warnings,omitempty"`

	// Files that typically change together
	Correlations []Correlation `json:"correlations,omitempty"`

	// Step-by-step checklist (app-specific)
	Checklist []string `json:"checklist,omitempty"`

	// Metadata
	Confidence float64 `json:"confidence"` // 0-1, how confident we are in this blueprint
	Source     string  `json:"source"`     // Where this blueprint came from
}

// FilePattern describes the file structure for a task
type FilePattern struct {
	BasePath    string   `json:"base_path"`
	Files       []string `json:"files"`
	Directories []string `json:"directories,omitempty"`
	RegisterIn  []string `json:"register_in,omitempty"`
}

// Example is a real file to use as a pattern
type Example struct {
	Path        string `json:"path"`
	Description string `json:"description"`
	Skeleton    string `json:"skeleton,omitempty"`
}

// SnippetEntry is a templatized code snippet for a file type.
type SnippetEntry struct {
	Description string `json:"description"`
	Code        string `json:"code"`
	SourceFile  string `json:"source_file"`
}

// Conventions holds auto-detected app conventions.
type Conventions struct {
	AuthGuard       string           `json:"auth_guard,omitempty"`
	Validation      string           `json:"validation,omitempty"`
	ResponseEnvelope string          `json:"response_envelope,omitempty"`
	Logging         string           `json:"logging,omitempty"`
	DI              string           `json:"di,omitempty"`
	ErrorHandling   string           `json:"error_handling,omitempty"`
	Naming          *NamingConvention `json:"naming,omitempty"`
}

// NamingConvention holds detected naming patterns.
type NamingConvention struct {
	Files   string `json:"files,omitempty"`
	Classes string `json:"classes,omitempty"`
	Methods string `json:"methods,omitempty"`
	Types   string `json:"types,omitempty"`
}

// Decision is a team decision relevant to the task
type Decision struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Rationale string   `json:"rationale,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// Warning is a pitfall to avoid
type Warning struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Severity    string `json:"severity,omitempty"`
}

// Correlation shows files that change together
type Correlation struct {
	Files      []string `json:"files"`
	Confidence float64  `json:"confidence"`
	Reason     string   `json:"reason,omitempty"`
}

// Generator creates blueprints from project data
type Generator struct {
	projectRoot string
	tcDir       string
	jsonStore   *storage.JSONStore
}

// NewGenerator creates a blueprint generator
func NewGenerator(projectRoot, tcDir string, jsonStore *storage.JSONStore) *Generator {
	return &Generator{
		projectRoot: projectRoot,
		tcDir:       tcDir,
		jsonStore:   jsonStore,
	}
}

// Generate creates a blueprint for the given task
func (g *Generator) Generate(taskType TaskType, app, path string) (*Blueprint, error) {
	bp := &Blueprint{
		TaskType:   taskType,
		App:        app,
		Path:       path,
		Confidence: 0.5,
		Source:     "pattern-analysis-v2",
	}

	bp.Description = g.getTaskDescription(taskType)

	switch taskType {
	case TaskAddEndpoint:
		g.generateEndpointBlueprint(bp)
	case TaskAddFeature, TaskAddService:
		g.generateFeatureBlueprint(bp)
	case TaskFixBug:
		g.generateBugFixBlueprint(bp)
	case TaskRefactor:
		g.generateRefactorBlueprint(bp)
	case TaskAddTest:
		g.generateTestBlueprint(bp)
	default:
		g.generateGenericBlueprint(bp)
	}

	g.addRelevantDecisions(bp)
	g.addRelevantWarnings(bp)
	g.addCorrelations(bp)

	// Enforce token budget
	g.enforceTokenBudget(bp)

	return bp, nil
}

func (g *Generator) getTaskDescription(taskType TaskType) string {
	descriptions := map[TaskType]string{
		TaskAddEndpoint: "Add a new REST API endpoint with controller, service, and module",
		TaskAddFeature:  "Add a new feature module with full structure",
		TaskAddService:  "Add a new service with dependency injection",
		TaskFixBug:      "Fix a bug in existing code",
		TaskRefactor:    "Refactor existing code while preserving behavior",
		TaskAddTest:     "Add tests for existing functionality",
	}
	if desc, ok := descriptions[taskType]; ok {
		return desc
	}
	return "Complete the requested task following project conventions"
}

// ---------------------------------------------------------------------------
// Task-type generators
// ---------------------------------------------------------------------------

func (g *Generator) generateEndpointBlueprint(bp *Blueprint) {
	framework := g.detectFramework()

	// Find examples (prefer newest/best)
	examples := g.findEndpointExamples(bp.App)
	bp.Examples = examples
	if len(examples) > 0 {
		bp.Confidence += 0.3
	}

	// File pattern based on detected framework
	switch framework {
	case "nestjs":
		bp.FilePattern = g.nestJSEndpointPattern(bp.App)
	case "express":
		bp.FilePattern = g.expressEndpointPattern(bp.App)
	// Go frameworks
	case "go-gin":
		bp.FilePattern = g.goGinEndpointPattern(bp.App)
	case "go-echo":
		bp.FilePattern = g.goEchoEndpointPattern(bp.App)
	case "go":
		bp.FilePattern = g.goGenericEndpointPattern(bp.App)
	// Python frameworks
	case "python-fastapi":
		bp.FilePattern = g.pythonFastAPIEndpointPattern(bp.App)
	case "python-flask":
		bp.FilePattern = g.pythonFlaskEndpointPattern(bp.App)
	case "python-django":
		bp.FilePattern = g.pythonDjangoEndpointPattern(bp.App)
	// Rust frameworks
	case "rust-actix":
		bp.FilePattern = g.rustActixEndpointPattern(bp.App)
	case "rust-axum":
		bp.FilePattern = g.rustAxumEndpointPattern(bp.App)
	case "rust":
		bp.FilePattern = g.rustGenericEndpointPattern(bp.App)
	default:
		bp.FilePattern = g.genericEndpointPattern(bp.App)
	}

	// Store detected framework for checklist generation
	bp.Source = "pattern-analysis:" + framework

	// --- v2: Extract snippets from best example ---
	if len(examples) > 0 {
		bestExample := examples[0]
		exDir := filepath.Join(g.projectRoot, bestExample.Path)
		featureName := g.extractFeatureName(bestExample.Path)

		bp.Snippets = g.extractSnippets(exDir, featureName)
		bp.Imports = g.extractImports(exDir, featureName)

		// Add skeleton to the best example
		if len(bp.Snippets) > 0 {
			bp.Confidence += 0.1
		}
	}

	// --- v2: Detect conventions ---
	bp.Conventions = g.detectConventions(bp.App)
	if bp.Conventions != nil {
		bp.Confidence += 0.1
	}

	// --- v2: App-specific checklist based on framework ---
	bp.Checklist = g.buildEndpointChecklist(bp.Conventions, framework)
}

func (g *Generator) generateFeatureBlueprint(bp *Blueprint) {
	examples := g.findFeatureExamples(bp.App)
	bp.Examples = examples
	if len(examples) > 0 {
		bp.Confidence += 0.3
	}

	bp.FilePattern = &FilePattern{
		BasePath: g.inferBasePath(bp.App, "feature"),
		Files: []string{
			"{name}.module.ts",
			"{name}.service.ts",
			"{name}.controller.ts",
			"types/{name}.types.ts",
		},
		RegisterIn: []string{"app.module.ts"},
	}

	// v2: snippets from best example (skip controller snippet for pure services)
	if len(examples) > 0 {
		bestExample := examples[0]
		exDir := filepath.Join(g.projectRoot, bestExample.Path)
		featureName := g.extractFeatureName(bestExample.Path)
		bp.Snippets = g.extractSnippets(exDir, featureName)
		bp.Imports = g.extractImports(exDir, featureName)
	}

	bp.Conventions = g.detectConventions(bp.App)
	bp.Checklist = g.buildFeatureChecklist(bp.Conventions)
}

func (g *Generator) generateBugFixBlueprint(bp *Blueprint) {
	if bp.Path == "" {
		bp.Confidence = 0.1
		bp.Checklist = []string{
			"Provide a file path for targeted bug-fix guidance",
			"Without a path, only generic advice is available",
		}
		return
	}

	bp.Confidence = 0.6

	// Extract skeleton of the target file
	absPath := bp.Path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(g.projectRoot, absPath)
	}
	if sk, err := skeleton.ParseFile(absPath); err == nil {
		bp.Examples = []Example{{
			Path:        bp.Path,
			Description: "Target file structure",
			Skeleton:    skeleton.FormatSkeleton(sk),
		}}
	}

	// File correlations for the target
	bp.Correlations = g.getFileCorrelations(bp.Path)

	// Find experts
	experts := g.findExperts(bp.Path)

	bp.Checklist = []string{
		"Reproduce the bug and understand root cause",
		"Check git blame for context on why code exists",
		"Write a failing test that captures the bug",
		"Fix the code",
		"Verify test passes",
		"Check for similar patterns elsewhere",
	}
	if len(experts) > 0 {
		bp.Checklist = append(bp.Checklist,
			"Consider consulting: "+strings.Join(experts, ", "))
	}
}

func (g *Generator) generateRefactorBlueprint(bp *Blueprint) {
	if bp.Path == "" {
		bp.Confidence = 0.1
		bp.Checklist = []string{
			"Provide a file path for targeted refactor guidance",
			"Without a path, only generic advice is available",
		}
		return
	}

	bp.Confidence = 0.6

	// Extract skeleton of the target file
	absPath := bp.Path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(g.projectRoot, absPath)
	}
	if sk, err := skeleton.ParseFile(absPath); err == nil {
		bp.Examples = []Example{{
			Path:        bp.Path,
			Description: "Target file structure",
			Skeleton:    skeleton.FormatSkeleton(sk),
		}}
	}

	// Correlations — crucial for refactoring
	bp.Correlations = g.getFileCorrelations(bp.Path)

	bp.Checklist = []string{
		"Ensure tests exist for current behavior",
		"Identify all usages of code being refactored",
		"Make incremental changes",
		"Run tests after each change",
		"Update imports and references",
		"Check for breaking changes in public APIs",
	}
}

func (g *Generator) generateTestBlueprint(bp *Blueprint) {
	testFramework := g.detectTestFramework()

	bp.FilePattern = &FilePattern{
		BasePath: "{same_dir_as_source}",
		Files:    []string{"{name}.spec.ts"},
	}

	// v2: Scope test examples to the correct app
	bp.Examples = g.findTestExamplesForApp(bp.App, testFramework)
	if len(bp.Examples) > 0 {
		bp.Confidence += 0.2
		// Extract test snippet from best example
		bestTest := bp.Examples[0]
		testPath := filepath.Join(g.projectRoot, bestTest.Path)
		featureName := g.extractFeatureNameFromFile(bestTest.Path)
		snippet := g.extractSingleSnippet(testPath, featureName, "Unit test pattern")
		if snippet != nil {
			bp.Snippets = map[string]*SnippetEntry{"test": snippet}
		}
		bp.Imports = g.extractImportsFromFile(testPath, featureName)
	}

	bp.Conventions = g.detectConventions(bp.App)
	bp.Checklist = g.buildTestChecklist(bp.Conventions, testFramework)
}

func (g *Generator) generateGenericBlueprint(bp *Blueprint) {
	bp.Checklist = []string{
		"Understand the requirements",
		"Check existing patterns in the codebase",
		"Follow established conventions",
		"Write tests for new functionality",
		"Update documentation if needed",
	}
}

// ---------------------------------------------------------------------------
// Snippet extraction (v2 core)
// ---------------------------------------------------------------------------

// extractSnippets reads files from the example directory, parses their
// skeleton, and templatizes the feature name to {Name}/{name}.
func (g *Generator) extractSnippets(exDir, featureName string) map[string]*SnippetEntry {
	snippets := make(map[string]*SnippetEntry)

	// Map of suffix -> snippet key
	fileTypes := map[string]string{
		".controller.ts": "controller",
		".service.ts":    "service",
		".module.ts":     "module",
	}

	entries, err := os.ReadDir(exDir)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		for suffix, key := range fileTypes {
			if strings.HasSuffix(e.Name(), suffix) {
				absPath := filepath.Join(exDir, e.Name())
				relPath, _ := filepath.Rel(g.projectRoot, absPath)
				desc := g.snippetDescription(key)
				snippet := g.extractSingleSnippet(absPath, featureName, desc)
				if snippet != nil {
					snippet.SourceFile = relPath
					snippets[key] = snippet
				}
			}
		}
	}

	// Check for schemas/ and types/ subdirectories
	schemasDir := filepath.Join(exDir, "schemas")
	if entries, err := os.ReadDir(schemasDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".ts") {
				absPath := filepath.Join(schemasDir, e.Name())
				relPath, _ := filepath.Rel(g.projectRoot, absPath)
				snippet := g.extractSingleSnippet(absPath, featureName, "Zod validation schema pattern")
				if snippet != nil {
					snippet.SourceFile = relPath
					snippets["schema"] = snippet
					break // only first schema file
				}
			}
		}
	}

	typesDir := filepath.Join(exDir, "types")
	if entries, err := os.ReadDir(typesDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".ts") {
				absPath := filepath.Join(typesDir, e.Name())
				relPath, _ := filepath.Rel(g.projectRoot, absPath)
				snippet := g.extractSingleSnippet(absPath, featureName, "Type definitions pattern")
				if snippet != nil {
					snippet.SourceFile = relPath
					snippets["types"] = snippet
					break
				}
			}
		}
	}

	// Check for test file
	testPatterns := []string{".service.spec.ts", ".controller.spec.ts", ".spec.ts"}
	for _, tp := range testPatterns {
		testFile := filepath.Join(exDir, featureName+tp)
		if _, err := os.Stat(testFile); err == nil {
			relPath, _ := filepath.Rel(g.projectRoot, testFile)
			snippet := g.extractSingleSnippet(testFile, featureName, "Unit test pattern")
			if snippet != nil {
				snippet.SourceFile = relPath
				snippets["test"] = snippet
				break
			}
		}
	}

	if len(snippets) == 0 {
		return nil
	}
	return snippets
}

// extractSingleSnippet reads a file, gets its skeleton, templatizes it,
// and caps at maxSnippetLines.
func (g *Generator) extractSingleSnippet(absPath, featureName, description string) *SnippetEntry {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	// Build a condensed representation: imports + skeleton
	lines := strings.Split(string(content), "\n")
	condensed := g.condenseFile(lines, featureName)
	if condensed == "" {
		return nil
	}

	return &SnippetEntry{
		Description: description,
		Code:        condensed,
	}
}

// condenseFile builds a condensed, templatized view of a source file.
// It keeps decorators, class/function signatures, constructor, and strips
// method bodies, JSDoc comments, and blank lines.
func (g *Generator) condenseFile(lines []string, featureName string) string {
	if len(lines) == 0 {
		return ""
	}

	// Pre-compiled patterns
	importRe := regexp.MustCompile(`^\s*import\s+`)
	jsdocStartRe := regexp.MustCompile(`^\s*/\*\*`)
	jsdocEndRe := regexp.MustCompile(`\*/\s*$`)
	blockCommentRe := regexp.MustCompile(`^\s*\*`)
	lineCommentRe := regexp.MustCompile(`^\s*//`)
	decoratorRe := regexp.MustCompile(`^\s*@(\w+)`)
	classRe := regexp.MustCompile(`^\s*(export\s+)?(abstract\s+)?class\s+\w+`)
	constructorRe := regexp.MustCompile(`^\s*constructor\s*\(`)
	methodRe := regexp.MustCompile(`^\s*(private\s+|public\s+|protected\s+)?(static\s+)?(async\s+)?(\w+)\s*(<[^>]+>)?\s*\(`)
	exportConstRe := regexp.MustCompile(`^\s*export\s+const\s+\w+`)
	exportTypeRe := regexp.MustCompile(`^\s*export\s+(?:type|interface)\s+`)

	// Phase 1: skip leading imports
	startIdx := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || importRe.MatchString(line) {
			startIdx = i + 1
			continue
		}
		break
	}

	var result []string
	braceDepth := 0      // global brace tracking
	classDepth := -1      // brace depth where class body starts (-1 = not in class)
	skipUntilBrace := -1  // skip body lines until braceDepth drops to this
	inJSDoc := false
	seenLines := make(map[string]bool) // deduplication

	addLine := func(s string) {
		key := strings.TrimSpace(s)
		if key == "" {
			// Allow one blank line between sections
			if len(result) > 0 && strings.TrimSpace(result[len(result)-1]) != "" {
				result = append(result, "")
			}
			return
		}
		if seenLines[key] {
			return
		}
		seenLines[key] = true
		result = append(result, s)
	}

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Track JSDoc / block comments — skip entirely
		if jsdocStartRe.MatchString(trimmed) {
			inJSDoc = true
			if jsdocEndRe.MatchString(trimmed) {
				inJSDoc = false
			}
			continue
		}
		if inJSDoc {
			if jsdocEndRe.MatchString(trimmed) {
				inJSDoc = false
			}
			continue
		}
		// Skip standalone comment lines
		if blockCommentRe.MatchString(trimmed) || lineCommentRe.MatchString(trimmed) {
			continue
		}

		// If we are skipping a method/function body, count braces
		if skipUntilBrace >= 0 {
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth < 0 {
				braceDepth = 0
			}
			if braceDepth <= skipUntilBrace {
				skipUntilBrace = -1
				// Add closing brace at class level
				if classDepth >= 0 && braceDepth == classDepth {
					addLine("}")
					classDepth = -1
				}
			}
			continue
		}

		// Count opening/closing braces on this line
		lineOpen := strings.Count(line, "{")
		lineClose := strings.Count(line, "}")

		// --- @Module / @Injectable decorator blocks (check BEFORE generic decorator) ---
		if strings.HasPrefix(trimmed, "@Module") || strings.HasPrefix(trimmed, "@Injectable") {
			block := g.collectDecoratorBlock(lines, i)
			addLine(block)
			blockLines := strings.Count(block, "\n")
			i += blockLines
			braceDepth += strings.Count(block, "{") - strings.Count(block, "}")
			continue
		}

		// --- Decorators (@UseGuards, @Controller, @Post, etc.) ---
		if decoratorRe.MatchString(trimmed) {
			// Skip parameter decorators used inside method signatures (e.g. @Body, @Query, @Param)
			paramDecorators := map[string]bool{
				"Body": true, "Query": true, "Param": true, "Headers": true,
				"Req": true, "Res": true, "Session": true, "UploadedFile": true,
				"UploadedFiles": true, "Ip": true, "HostParam": true,
			}
			if sub := decoratorRe.FindStringSubmatch(trimmed); len(sub) > 1 && paramDecorators[sub[1]] {
				continue
			}
			addLine(line)
			continue
		}

		// --- Class declaration ---
		if classRe.MatchString(trimmed) {
			addLine(line)
			braceDepth += lineOpen - lineClose
			classDepth = braceDepth - 1 // class body is one level above closing
			continue
		}

		// --- Constructor ---
		if classDepth >= 0 && constructorRe.MatchString(trimmed) {
			sig := g.collectSignature(lines, i)
			addLine(sig)
			sigLines := strings.Count(sig, "\n")
			i += sigLines
			sigOpen := strings.Count(sig, "{")
			sigClose := strings.Count(sig, "}")
			braceDepth += sigOpen - sigClose
			// Only skip body if the signature opened a brace that isn't closed
			if sigOpen > sigClose {
				skipUntilBrace = braceDepth - (sigOpen - sigClose)
			}
			continue
		}

		// --- Methods (inside class body) ---
		if classDepth >= 0 && methodRe.MatchString(trimmed) {
			// Ignore control flow keywords that look like methods
			sub := methodRe.FindStringSubmatch(trimmed)
			if len(sub) > 4 {
				name := sub[4]
				if name == "if" || name == "for" || name == "while" || name == "switch" || name == "catch" {
					braceDepth += lineOpen - lineClose
					continue
				}
			}
			sig := g.collectSignature(lines, i)
			addLine(sig)
			sigLines := strings.Count(sig, "\n")
			i += sigLines
			sigOpen := strings.Count(sig, "{")
			sigClose := strings.Count(sig, "}")
			braceDepth += sigOpen - sigClose
			if sigOpen > sigClose {
				skipUntilBrace = braceDepth - (sigOpen - sigClose)
			}
			continue
		}

		// --- export const (schemas, const-enums) ---
		if exportConstRe.MatchString(trimmed) {
			addLine(line)
			if lineOpen > lineClose {
				addLine("  // ...")
				addLine("});")
				// Skip until matching close
				braceDepth += lineOpen - lineClose
				skipUntilBrace = braceDepth - lineOpen + lineClose
			} else {
				braceDepth += lineOpen - lineClose
			}
			continue
		}

		// --- export type/interface ---
		if exportTypeRe.MatchString(trimmed) {
			addLine(line)
			if lineOpen > lineClose {
				addLine("  // fields...")
				addLine("}")
				braceDepth += lineOpen - lineClose
				skipUntilBrace = braceDepth - lineOpen + lineClose
			} else {
				braceDepth += lineOpen - lineClose
			}
			continue
		}

		// --- Track brace depth for everything else ---
		braceDepth += lineOpen - lineClose
		if braceDepth < 0 {
			braceDepth = 0
		}

		// Closing brace of the class
		if classDepth >= 0 && braceDepth <= classDepth {
			addLine("}")
			classDepth = -1
		}
	}

	// Cap at maxSnippetLines
	if len(result) > maxSnippetLines {
		result = result[:maxSnippetLines]
	}

	// Templatize
	text := strings.Join(result, "\n")
	text = g.templatize(text, featureName)
	return strings.TrimSpace(text)
}

// collectSignature collects a function/constructor signature, possibly multi-line.
func (g *Generator) collectSignature(lines []string, start int) string {
	sig := lines[start]
	// If the line doesn't contain '{', collect continuation lines
	if !strings.Contains(sig, "{") {
		for j := start + 1; j < len(lines) && j < start+5; j++ {
			sig += "\n" + lines[j]
			if strings.Contains(lines[j], "{") || strings.Contains(lines[j], ")") {
				break
			}
		}
	}
	// Strip the body opening brace content
	if idx := strings.LastIndex(sig, "{"); idx > 0 {
		sig = strings.TrimRight(sig[:idx+1], " ") + " }"
	}
	return sig
}

// collectDecoratorBlock collects a decorator and its argument block (e.g. @Module({...})).
func (g *Generator) collectDecoratorBlock(lines []string, start int) string {
	block := lines[start]
	depth := strings.Count(block, "(") - strings.Count(block, ")")
	depth += strings.Count(block, "{") - strings.Count(block, "}")
	for j := start + 1; j < len(lines) && depth > 0; j++ {
		block += "\n" + lines[j]
		depth += strings.Count(lines[j], "(") - strings.Count(lines[j], ")")
		depth += strings.Count(lines[j], "{") - strings.Count(lines[j], "}")
	}
	return block
}

func (g *Generator) snippetDescription(key string) string {
	descs := map[string]string{
		"controller": "REST controller pattern for this app",
		"service":    "Injectable service pattern",
		"module":     "Module registration pattern",
		"schema":     "Zod validation schema pattern",
		"types":      "Type definitions pattern",
		"test":       "Unit test pattern",
	}
	if d, ok := descs[key]; ok {
		return d
	}
	return key + " pattern"
}

// ---------------------------------------------------------------------------
// Import extraction (v2)
// ---------------------------------------------------------------------------

// extractImports reads imports from example files and templatizes them.
func (g *Generator) extractImports(exDir, featureName string) map[string][]string {
	result := make(map[string][]string)

	fileTypes := map[string]string{
		".controller.ts": "controller",
		".service.ts":    "service",
		".module.ts":     "module",
	}

	entries, err := os.ReadDir(exDir)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		for suffix, key := range fileTypes {
			if strings.HasSuffix(e.Name(), suffix) {
				absPath := filepath.Join(exDir, e.Name())
				imps := g.extractImportsFromFile(absPath, featureName)
				if lines, ok := imps[key]; ok && len(lines) > 0 {
					result[key] = lines
				} else {
					// Fall back to using the filename-based key
					for k, v := range imps {
						result[k] = v
					}
				}
			}
		}
	}

	// Test file imports
	testPatterns := []string{".service.spec.ts", ".controller.spec.ts", ".spec.ts"}
	for _, tp := range testPatterns {
		testFile := filepath.Join(exDir, featureName+tp)
		if _, err := os.Stat(testFile); err == nil {
			imps := g.extractImportsFromFile(testFile, featureName)
			for _, v := range imps {
				result["test"] = v
				break
			}
			break
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// extractImportsFromFile reads the import block from a single file,
// templatizes it, and returns categorized imports.
func (g *Generator) extractImportsFromFile(absPath, featureName string) map[string][]string {
	f, err := os.Open(absPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var rawImports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") {
			rawImports = append(rawImports, trimmed)
		} else if trimmed == "" {
			continue
		} else if len(rawImports) > 0 {
			// Past the import block
			break
		}
	}

	// Templatize
	var templatized []string
	for _, imp := range rawImports {
		imp = g.templatize(imp, featureName)
		templatized = append(templatized, imp)
	}

	// Cap
	if len(templatized) > maxImportsPerType {
		templatized = templatized[:maxImportsPerType]
	}

	// Determine the key from file suffix
	base := filepath.Base(absPath)
	key := "other"
	if strings.Contains(base, ".controller.") {
		key = "controller"
	} else if strings.Contains(base, ".service.spec.") || strings.Contains(base, ".spec.") {
		key = "test"
	} else if strings.Contains(base, ".service.") {
		key = "service"
	} else if strings.Contains(base, ".module.") {
		key = "module"
	}

	return map[string][]string{key: templatized}
}

// ---------------------------------------------------------------------------
// Convention auto-detection (v2)
// ---------------------------------------------------------------------------

// detectConventions analyzes existing code to detect app conventions.
func (g *Generator) detectConventions(app string) *Conventions {
	searchPath := g.appSourcePath(app)
	if searchPath == "" {
		return nil
	}

	conv := &Conventions{}
	detected := false

	// Auth guard detection: grep controllers for @UseGuards(...)
	if guards := g.detectAuthGuard(searchPath); guards != "" {
		conv.AuthGuard = guards
		detected = true
	}

	// Validation pipe detection: ZodPipe vs ValidationPipe
	if validation := g.detectValidation(searchPath); validation != "" {
		conv.Validation = validation
		detected = true
	}

	// Response envelope detection
	if envelope := g.detectResponseEnvelope(searchPath); envelope != "" {
		conv.ResponseEnvelope = envelope
		detected = true
	}

	// Logging pattern detection
	if logging := g.detectLogging(searchPath); logging != "" {
		conv.Logging = logging
		detected = true
	}

	// DI pattern
	conv.DI = "constructor(private readonly dep: Dep) — always private readonly"

	// Error handling
	conv.ErrorHandling = g.detectErrorHandling(searchPath)

	// Naming conventions
	conv.Naming = g.detectNamingConventions(searchPath)
	if conv.Naming != nil {
		detected = true
	}

	if !detected {
		return nil
	}
	return conv
}

func (g *Generator) detectAuthGuard(searchPath string) string {
	matches, err := search.SearchCode(`@UseGuards\(\w+`, searchPath, "*.controller.ts", 20)
	if err != nil || len(matches) == 0 {
		return ""
	}

	// Count guard types
	guardCounts := make(map[string]int)
	guardRe := regexp.MustCompile(`@UseGuards\((\w+)\)`)
	for _, m := range matches {
		if sub := guardRe.FindStringSubmatch(m.Content); len(sub) > 1 {
			guardCounts[sub[1]]++
		}
	}

	// Most common guard
	var bestGuard string
	bestCount := 0
	for guard, count := range guardCounts {
		if count > bestCount {
			bestGuard = guard
			bestCount = count
		}
	}

	if bestGuard == "" {
		return ""
	}
	return bestGuard + " (class-level on all REST controllers)"
}

func (g *Generator) detectValidation(searchPath string) string {
	zodCount := 0
	cvCount := 0

	if matches, err := search.SearchCode(`ZodPipe`, searchPath, "*.ts", 20); err == nil {
		zodCount = len(matches)
	}
	if matches, err := search.SearchCode(`ValidationPipe`, searchPath, "*.ts", 20); err == nil {
		cvCount = len(matches)
	}

	if zodCount > cvCount && zodCount > 0 {
		return "ZodPipe — never class-validator"
	} else if cvCount > zodCount && cvCount > 0 {
		return "ValidationPipe (class-validator)"
	} else if zodCount > 0 {
		return "ZodPipe"
	}
	return ""
}

func (g *Generator) detectResponseEnvelope(searchPath string) string {
	// Check for { statusCode, data } pattern in controllers
	matches, err := search.SearchCode(`statusCode.*data|data.*statusCode`, searchPath, "*.controller.ts", 10)
	if err == nil && len(matches) > 0 {
		return "{ statusCode: number, data: T } — controller wraps, service returns plain"
	}
	return ""
}

func (g *Generator) detectLogging(searchPath string) string {
	matches, err := search.SearchCode(`new Logger\(`, searchPath, "*.service.ts", 10)
	if err == nil && len(matches) > 0 {
		return "private readonly logger = new Logger(ClassName.name)"
	}
	return ""
}

func (g *Generator) detectErrorHandling(searchPath string) string {
	matches, err := search.SearchCode(`NotFoundException|ConflictException|BadRequestException`, searchPath, "*.service.ts", 10)
	if err == nil && len(matches) > 0 {
		return "Throw NestJS exceptions (NotFoundException, ConflictException) — services transform external errors"
	}
	return "Throw framework-specific exceptions from service layer"
}

func (g *Generator) detectNamingConventions(searchPath string) *NamingConvention {
	nc := &NamingConvention{}

	// Check if file names are singular or plural
	entries, err := os.ReadDir(searchPath)
	if err != nil {
		// Try v1 subdirectory
		v1Path := filepath.Join(searchPath, "v1")
		entries, err = os.ReadDir(v1Path)
		if err != nil {
			return nil
		}
		_ = v1Path // Use v1Path as new search base
	}

	singularCount := 0
	pluralCount := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "schemas" || name == "types" || name == "node_modules" {
			continue
		}
		if strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss") {
			pluralCount++
		} else {
			singularCount++
		}
	}

	if singularCount >= pluralCount {
		nc.Files = "singular: alarm.controller.ts (not alarms.controller.ts)"
	} else {
		nc.Files = "plural: alarms.controller.ts"
	}

	nc.Classes = "PascalCase: AlarmController, AlarmService"
	nc.Methods = "camelCase: createIssue, findDevices"
	nc.Types = "Create{Name}Request, Create{Name}Response, Create{Name}ResponseData"

	return nc
}

// ---------------------------------------------------------------------------
// Correlation fix (v2): always try to load from git-correlations.json
// ---------------------------------------------------------------------------

func (g *Generator) addCorrelations(bp *Blueprint) {
	// Already populated by task-specific generators (e.g., refactor, fix-bug)
	if len(bp.Correlations) > 0 {
		return
	}

	// Try path first
	if bp.Path != "" {
		bp.Correlations = g.getFileCorrelations(bp.Path)
		if len(bp.Correlations) > 0 {
			return
		}
	}

	// Try example paths
	if len(bp.Examples) > 0 {
		bp.Correlations = g.getFileCorrelations(bp.Examples[0].Path)
		if len(bp.Correlations) > 0 {
			return
		}
	}

	// For add-endpoint tasks, try to find correlations for controller+module patterns
	if bp.TaskType == TaskAddEndpoint && bp.App != "" {
		bp.Correlations = g.getEndpointCorrelations(bp.App)
	}
}

// getEndpointCorrelations looks for common endpoint-related correlations.
func (g *Generator) getEndpointCorrelations(app string) []Correlation {
	correlationsFile := filepath.Join(g.tcDir, "knowledge", "git-correlations.json")
	data, err := os.ReadFile(correlationsFile)
	if err != nil {
		return nil
	}

	var rawCorrelations []map[string]interface{}
	if err := json.Unmarshal(data, &rawCorrelations); err != nil {
		return nil
	}

	// Look for correlations involving controller/service/module files in this app
	var result []Correlation
	appPrefix := "apps/" + app

	for _, rc := range rawCorrelations {
		files, ok := rc["files"].([]interface{})
		if !ok {
			continue
		}

		relevant := false
		var fileStrings []string
		for _, f := range files {
			if fs, ok := f.(string); ok {
				fileStrings = append(fileStrings, fs)
				if strings.Contains(fs, appPrefix) &&
					(strings.Contains(fs, ".controller.") ||
						strings.Contains(fs, ".service.") ||
						strings.Contains(fs, ".module.") ||
						strings.Contains(fs, "app.module")) {
					relevant = true
				}
			}
		}

		if relevant && len(fileStrings) > 1 {
			confidence := 0.5
			if c, ok := rc["confidence"].(float64); ok {
				confidence = c
			}
			reason := ""
			if r, ok := rc["reason"].(string); ok {
				reason = r
			}
			if reason == "" {
				reason = "These files typically change together when adding/modifying endpoints"
			}
			result = append(result, Correlation{
				Files:      fileStrings,
				Confidence: confidence,
				Reason:     reason,
			})
		}

		if len(result) >= 5 {
			break
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// App-specific checklists (v2)
// ---------------------------------------------------------------------------

func (g *Generator) buildEndpointChecklist(conv *Conventions, framework string) []string {
	// Framework-specific checklists
	switch framework {
	case "go-gin":
		return g.buildGoGinChecklist()
	case "go-echo":
		return g.buildGoEchoChecklist()
	case "go":
		return g.buildGoGenericChecklist()
	case "python-fastapi":
		return g.buildFastAPIChecklist()
	case "python-flask":
		return g.buildFlaskChecklist()
	case "python-django":
		return g.buildDjangoChecklist()
	case "rust-actix":
		return g.buildActixChecklist()
	case "rust-axum":
		return g.buildAxumChecklist()
	case "rust":
		return g.buildRustGenericChecklist()
	default:
		// NestJS/Express/TypeScript default
		return g.buildNestJSChecklist(conv)
	}
}

func (g *Generator) buildNestJSChecklist(conv *Conventions) []string {
	checklist := []string{}

	checklist = append(checklist, "Create types in types/{name}.types.ts — const enums + interfaces")
	checklist = append(checklist, "Create Zod schemas in schemas/{name}.schemas.ts — export schema + z.infer type")

	if conv != nil && conv.Logging != "" {
		checklist = append(checklist, "Create service with @Injectable(), Logger, constructor DI")
	} else {
		checklist = append(checklist, "Create service with @Injectable() and constructor DI")
	}

	if conv != nil && conv.AuthGuard != "" {
		guard := strings.Split(conv.AuthGuard, " ")[0]
		if conv.Validation != "" {
			valPipe := strings.Split(conv.Validation, " ")[0]
			checklist = append(checklist, "Create controller with @UseGuards("+guard+"), "+valPipe+" validation")
		} else {
			checklist = append(checklist, "Create controller with @UseGuards("+guard+")")
		}
	} else {
		checklist = append(checklist, "Create controller with route decorators")
	}

	checklist = append(checklist, "Create module — register controller + service + PrismaService in providers")
	checklist = append(checklist, "Register module in app.module.ts imports array")
	checklist = append(checklist, "Add unit test {name}.service.spec.ts — mock Prisma before imports")

	return checklist
}

// Go Framework Checklists

func (g *Generator) buildGoGinChecklist() []string {
	return []string{
		"Create DTO structs in {name}_dto.go — request/response with json tags",
		"Create handler in {name}_handler.go — gin.HandlerFunc with c *gin.Context",
		"Create service in {name}_service.go — business logic with interface",
		"Create repository in {name}_repository.go — data access layer",
		"Register routes in router.go — r.Group('/{name}').POST/GET/etc.",
		"Add middleware if auth needed — c.Get('user') from JWT middleware",
		"Add unit tests — mock repository with testify/mock",
		"Run go fmt and go vet",
	}
}

func (g *Generator) buildGoEchoChecklist() []string {
	return []string{
		"Create model in {name}_model.go — request/response structs",
		"Create handler in {name}_handler.go — echo.HandlerFunc with c echo.Context",
		"Create service in {name}_service.go — business logic interface",
		"Create repository in {name}_repository.go — database operations",
		"Register routes in routes.go — e.Group('/{name}').POST/GET/etc.",
		"Add middleware for auth — echo.JWT() or custom",
		"Add validation with echo.Bind() and validator",
		"Add unit tests with testify",
	}
}

func (g *Generator) buildGoGenericChecklist() []string {
	return []string{
		"Define request/response structs with json tags",
		"Create handler function — http.HandlerFunc signature",
		"Create service with interface for testability",
		"Register routes in main.go or router",
		"Add middleware for logging/auth if needed",
		"Add unit tests using testing package",
		"Run go fmt, go vet, and golint",
	}
}

// Python Framework Checklists

func (g *Generator) buildFastAPIChecklist() []string {
	return []string{
		"Create Pydantic models in schemas/{name}.py — BaseModel for request/response",
		"Create router in {name}.py — @router.post/get decorators",
		"Create service in services/{name}.py — business logic functions",
		"Create SQLAlchemy models in models/{name}.py if needed",
		"Register router in main.py — app.include_router()",
		"Add dependencies for auth — Depends(get_current_user)",
		"Add response_model for type hints and docs",
		"Add unit tests with pytest — use TestClient",
	}
}

func (g *Generator) buildFlaskChecklist() []string {
	return []string{
		"Create route in {name}.py — @bp.route decorators",
		"Create service in {name}_service.py — business logic",
		"Create model in {name}_model.py if using SQLAlchemy",
		"Register blueprint in __init__.py — app.register_blueprint()",
		"Add request validation with marshmallow or WTForms",
		"Add auth decorator if needed — @login_required",
		"Add unit tests with pytest",
	}
}

func (g *Generator) buildDjangoChecklist() []string {
	return []string{
		"Create model in models/{name}.py — Django ORM model",
		"Create serializer in serializers/{name}.py — DRF serializer",
		"Create view in views/{name}.py — APIView or ViewSet",
		"Add URL pattern in urls.py — path('/{name}/', ...)",
		"Register in settings.py INSTALLED_APPS if new app",
		"Add permissions — IsAuthenticated, etc.",
		"Run migrations — makemigrations + migrate",
		"Add unit tests with Django TestCase",
	}
}

// Rust Framework Checklists

func (g *Generator) buildActixChecklist() []string {
	return []string{
		"Create model structs in {name}_model.rs — #[derive(Serialize, Deserialize)]",
		"Create handler in {name}.rs — async fn with web::Json/Path extractors",
		"Create service in {name}_service.rs — business logic trait + impl",
		"Update mod.rs — pub mod {name};",
		"Register routes in main.rs — .service(web::resource('/{name}')...)",
		"Add middleware for auth if needed — HttpAuthentication::bearer()",
		"Add error handling with custom error types",
		"Add unit tests with #[actix_rt::test]",
	}
}

func (g *Generator) buildAxumChecklist() []string {
	return []string{
		"Create model structs in {name}_models.rs — #[derive(Serialize, Deserialize)]",
		"Create handler in {name}_handlers.rs — async fn with axum extractors",
		"Create service trait and impl for business logic",
		"Update mod.rs — pub mod {name};",
		"Register routes in router.rs — Router::new().route('/{name}', ...)",
		"Add middleware layers for auth/logging",
		"Add error handling with IntoResponse",
		"Add unit tests with tokio::test",
	}
}

func (g *Generator) buildRustGenericChecklist() []string {
	return []string{
		"Define structs for request/response with serde derives",
		"Create handler module with async functions",
		"Implement business logic in separate module",
		"Update mod.rs to export new module",
		"Register routes in main.rs",
		"Add error handling with Result types",
		"Add unit tests with #[test]",
	}
}

func (g *Generator) buildFeatureChecklist(conv *Conventions) []string {
	checklist := []string{
		"Create module structure with @Module decorator",
		"Implement core service logic with @Injectable()",
		"Define types and interfaces",
		"Add controller if API exposure needed",
		"Register in parent module",
		"Add unit tests",
	}

	if conv != nil && conv.Validation != "" {
		valPipe := strings.Split(conv.Validation, " ")[0]
		checklist = append(checklist, "Use "+valPipe+" for input validation (not class-validator)")
	}

	return checklist
}

func (g *Generator) buildTestChecklist(conv *Conventions, framework string) []string {
	checklist := []string{
		"Create test file next to source file",
		"Import the module/function to test",
	}

	if framework == "jest" {
		checklist = append(checklist, "Mock external dependencies (jest.mock) before imports")
	}

	checklist = append(checklist,
		"Write describe block for the unit",
		"Write test cases covering happy path",
		"Write test cases covering edge cases",
		"Write test cases covering error cases",
		"Mock external dependencies",
	)

	return checklist
}

// ---------------------------------------------------------------------------
// Token budget enforcement
// ---------------------------------------------------------------------------

func (g *Generator) enforceTokenBudget(bp *Blueprint) {
	estimated := g.estimateTokens(bp)
	if estimated <= tokenBudget {
		return
	}

	// Drop snippets in order: test, types, schema, module, service, controller
	dropOrder := []string{"test", "types", "schema", "module", "service", "controller"}
	for _, key := range dropOrder {
		if bp.Snippets != nil {
			delete(bp.Snippets, key)
		}
		estimated = g.estimateTokens(bp)
		if estimated <= tokenBudget {
			return
		}
	}

	// Drop imports
	bp.Imports = nil
	estimated = g.estimateTokens(bp)
	if estimated <= tokenBudget {
		return
	}

	// Drop conventions
	bp.Conventions = nil
}

func (g *Generator) estimateTokens(bp *Blueprint) int {
	data, err := json.Marshal(bp)
	if err != nil {
		return 0
	}
	return len(data) / charsPerToken
}

// ---------------------------------------------------------------------------
// Templatize helpers
// ---------------------------------------------------------------------------

// templatize replaces a feature name (e.g. "rma") with {name} and its
// PascalCase variant (e.g. "Rma") with {Name}.
func (g *Generator) templatize(text, featureName string) string {
	if featureName == "" {
		return text
	}

	pascal := toPascalCase(featureName)

	// Replace PascalCase first (longer match) to avoid partial replacements
	text = strings.ReplaceAll(text, pascal, "{Name}")
	text = strings.ReplaceAll(text, featureName, "{name}")

	return text
}

// toPascalCase converts "rma" -> "Rma", "alarm-events" -> "AlarmEvents"
func toPascalCase(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			result.WriteString(string(runes))
		}
	}
	if result.Len() == 0 {
		// Single word, just capitalize first letter
		runes := []rune(s)
		runes[0] = unicode.ToUpper(runes[0])
		return string(runes)
	}
	return result.String()
}

// extractFeatureName extracts the feature name from an example directory path.
// e.g. "apps/smart-smoke/src/app/v1/rma" -> "rma"
func (g *Generator) extractFeatureName(dirPath string) string {
	return filepath.Base(dirPath)
}

// extractFeatureNameFromFile extracts the feature name from a file path.
// e.g. "apps/smart-smoke/src/app/v1/rma/rma.service.spec.ts" -> "rma"
func (g *Generator) extractFeatureNameFromFile(filePath string) string {
	dir := filepath.Dir(filePath)
	return filepath.Base(dir)
}

// appSourcePath returns the source directory for an app.
func (g *Generator) appSourcePath(app string) string {
	if app == "" {
		srcPath := filepath.Join(g.projectRoot, "src", "app")
		if _, err := os.Stat(srcPath); err == nil {
			return srcPath
		}
		return filepath.Join(g.projectRoot, "src")
	}

	paths := []string{
		filepath.Join(g.projectRoot, "apps", app, "src", "app"),
		filepath.Join(g.projectRoot, "apps", app, "src"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Existing helpers (kept from v1)
// ---------------------------------------------------------------------------

func (g *Generator) detectFramework() string {
	// Check for Node.js frameworks (package.json)
	packageJSON := filepath.Join(g.projectRoot, "package.json")
	if data, err := os.ReadFile(packageJSON); err == nil {
		content := string(data)
		// NestJS
		nestIndicators := []string{"@nestjs/core", "@Module", "@Controller"}
		for _, indicator := range nestIndicators {
			if strings.Contains(content, indicator) {
				return "nestjs"
			}
		}
		// Express
		if strings.Contains(content, "express") {
			return "express"
		}
	}

	// Check for Go frameworks (go.mod)
	goMod := filepath.Join(g.projectRoot, "go.mod")
	if data, err := os.ReadFile(goMod); err == nil {
		content := string(data)
		if strings.Contains(content, "github.com/gin-gonic/gin") {
			return "go-gin"
		}
		if strings.Contains(content, "github.com/labstack/echo") {
			return "go-echo"
		}
		// Generic Go if no specific framework
		return "go"
	}

	// Check for Python frameworks (requirements.txt, pyproject.toml, setup.py)
	pythonFiles := []string{"requirements.txt", "pyproject.toml", "setup.py", "Pipfile"}
	for _, pf := range pythonFiles {
		pyFile := filepath.Join(g.projectRoot, pf)
		if data, err := os.ReadFile(pyFile); err == nil {
			content := string(data)
			if strings.Contains(content, "fastapi") {
				return "python-fastapi"
			}
			if strings.Contains(content, "flask") {
				return "python-flask"
			}
			if strings.Contains(content, "django") {
				return "python-django"
			}
		}
	}

	// Check for Rust frameworks (Cargo.toml)
	cargoToml := filepath.Join(g.projectRoot, "Cargo.toml")
	if data, err := os.ReadFile(cargoToml); err == nil {
		content := string(data)
		if strings.Contains(content, "actix-web") {
			return "rust-actix"
		}
		if strings.Contains(content, "axum") {
			return "rust-axum"
		}
		if strings.Contains(content, "rocket") {
			return "rust-rocket"
		}
		// Generic Rust
		return "rust"
	}

	return "unknown"
}

func (g *Generator) detectTestFramework() string {
	packageJSON := filepath.Join(g.projectRoot, "package.json")
	if data, err := os.ReadFile(packageJSON); err == nil {
		content := string(data)
		if strings.Contains(content, "jest") {
			return "jest"
		}
		if strings.Contains(content, "mocha") {
			return "mocha"
		}
		if strings.Contains(content, "vitest") {
			return "vitest"
		}
	}
	return "unknown"
}

func (g *Generator) nestJSEndpointPattern(app string) *FilePattern {
	basePath := g.inferBasePath(app, "endpoint")
	return &FilePattern{
		BasePath: basePath,
		Files: []string{
			"{name}.controller.ts",
			"{name}.service.ts",
			"{name}.module.ts",
			"schemas/{name}.schemas.ts",
			"types/{name}.types.ts",
		},
		Directories: []string{
			"schemas",
			"types",
		},
		RegisterIn: []string{
			g.findRegisterInPath(app),
		},
	}
}

func (g *Generator) expressEndpointPattern(app string) *FilePattern {
	return &FilePattern{
		BasePath: g.inferBasePath(app, "route"),
		Files: []string{
			"{name}.route.ts",
			"{name}.controller.ts",
			"{name}.service.ts",
		},
		RegisterIn: []string{
			"routes/index.ts",
		},
	}
}

func (g *Generator) genericEndpointPattern(app string) *FilePattern {
	return &FilePattern{
		BasePath: g.inferBasePath(app, "endpoint"),
		Files: []string{
			"{name}.ts",
		},
	}
}

// ---------------------------------------------------------------------------
// Go Framework Patterns
// ---------------------------------------------------------------------------

func (g *Generator) goGinEndpointPattern(app string) *FilePattern {
	return &FilePattern{
		BasePath: g.inferGoBasePath(app, "handler"),
		Files: []string{
			"{name}_handler.go",
			"{name}_service.go",
			"{name}_repository.go",
			"{name}_dto.go",
		},
		Directories: []string{},
		RegisterIn: []string{
			"cmd/server/main.go",
			"internal/router/router.go",
		},
	}
}

func (g *Generator) goEchoEndpointPattern(app string) *FilePattern {
	return &FilePattern{
		BasePath: g.inferGoBasePath(app, "handler"),
		Files: []string{
			"{name}_handler.go",
			"{name}_service.go",
			"{name}_repository.go",
			"{name}_model.go",
		},
		Directories: []string{},
		RegisterIn: []string{
			"cmd/server/main.go",
			"internal/routes/routes.go",
		},
	}
}

func (g *Generator) goGenericEndpointPattern(app string) *FilePattern {
	return &FilePattern{
		BasePath: g.inferGoBasePath(app, "handler"),
		Files: []string{
			"{name}.go",
			"{name}_test.go",
		},
		RegisterIn: []string{
			"main.go",
		},
	}
}

func (g *Generator) inferGoBasePath(app, kind string) string {
	// Standard Go project layout patterns
	patterns := []string{
		"internal/handlers/{name}/",
		"internal/handler/{name}/",
		"internal/api/{name}/",
		"pkg/handlers/{name}/",
		"api/{name}/",
		"handlers/{name}/",
	}

	for _, pattern := range patterns {
		testPath := strings.ReplaceAll(pattern, "{name}", "")
		fullPath := filepath.Join(g.projectRoot, testPath)
		if _, err := os.Stat(filepath.Dir(fullPath)); err == nil {
			return pattern
		}
	}

	// Default Go layout
	return "internal/handlers/{name}/"
}

// ---------------------------------------------------------------------------
// Python Framework Patterns
// ---------------------------------------------------------------------------

func (g *Generator) pythonFastAPIEndpointPattern(app string) *FilePattern {
	return &FilePattern{
		BasePath: g.inferPythonBasePath(app, "router"),
		Files: []string{
			"{name}.py",
			"schemas/{name}.py",
			"services/{name}.py",
			"models/{name}.py",
		},
		Directories: []string{
			"schemas",
			"services",
			"models",
		},
		RegisterIn: []string{
			"app/main.py",
			"app/routers/__init__.py",
		},
	}
}

func (g *Generator) pythonFlaskEndpointPattern(app string) *FilePattern {
	return &FilePattern{
		BasePath: g.inferPythonBasePath(app, "route"),
		Files: []string{
			"{name}.py",
			"{name}_service.py",
			"{name}_model.py",
		},
		RegisterIn: []string{
			"app/__init__.py",
			"app/routes/__init__.py",
		},
	}
}

func (g *Generator) pythonDjangoEndpointPattern(app string) *FilePattern {
	return &FilePattern{
		BasePath: g.inferPythonBasePath(app, "view"),
		Files: []string{
			"views/{name}.py",
			"serializers/{name}.py",
			"models/{name}.py",
			"urls.py",
		},
		Directories: []string{
			"views",
			"serializers",
			"models",
		},
		RegisterIn: []string{
			"urls.py",
			"settings.py",
		},
	}
}

func (g *Generator) inferPythonBasePath(app, kind string) string {
	// Common Python project patterns
	patterns := []string{
		"app/routers/",
		"app/api/",
		"src/routers/",
		"api/",
		"routers/",
		"views/",
	}

	for _, pattern := range patterns {
		fullPath := filepath.Join(g.projectRoot, pattern)
		if _, err := os.Stat(fullPath); err == nil {
			return pattern + "{name}/"
		}
	}

	return "app/routers/{name}/"
}

// ---------------------------------------------------------------------------
// Rust Framework Patterns
// ---------------------------------------------------------------------------

func (g *Generator) rustActixEndpointPattern(app string) *FilePattern {
	return &FilePattern{
		BasePath: g.inferRustBasePath(app, "handler"),
		Files: []string{
			"{name}.rs",
			"{name}_service.rs",
			"{name}_model.rs",
			"mod.rs",
		},
		RegisterIn: []string{
			"src/main.rs",
			"src/routes/mod.rs",
		},
	}
}

func (g *Generator) rustAxumEndpointPattern(app string) *FilePattern {
	return &FilePattern{
		BasePath: g.inferRustBasePath(app, "handler"),
		Files: []string{
			"{name}.rs",
			"{name}_handlers.rs",
			"{name}_models.rs",
			"mod.rs",
		},
		RegisterIn: []string{
			"src/main.rs",
			"src/router.rs",
		},
	}
}

func (g *Generator) rustGenericEndpointPattern(app string) *FilePattern {
	return &FilePattern{
		BasePath: g.inferRustBasePath(app, "module"),
		Files: []string{
			"{name}.rs",
			"mod.rs",
		},
		RegisterIn: []string{
			"src/main.rs",
			"src/lib.rs",
		},
	}
}

func (g *Generator) inferRustBasePath(app, kind string) string {
	// Common Rust project patterns
	patterns := []string{
		"src/handlers/",
		"src/routes/",
		"src/api/",
		"src/controllers/",
	}

	for _, pattern := range patterns {
		fullPath := filepath.Join(g.projectRoot, pattern)
		if _, err := os.Stat(fullPath); err == nil {
			return pattern + "{name}/"
		}
	}

	return "src/handlers/{name}/"
}

func (g *Generator) findRegisterInPath(app string) string {
	// Try to find the actual app.module.ts
	candidates := []string{
		filepath.Join("apps", app, "src", "app", "app.module.ts"),
		filepath.Join("apps", app, "src", "app.module.ts"),
		filepath.Join("src", "app", "app.module.ts"),
	}
	for _, c := range candidates {
		fullPath := filepath.Join(g.projectRoot, c)
		if _, err := os.Stat(fullPath); err == nil {
			return c
		}
	}
	return "app.module.ts"
}

func (g *Generator) inferBasePath(app, kind string) string {
	patterns := []string{
		"apps/%s/src/app/v1/{name}/",
		"apps/%s/src/%s/",
		"src/%s/",
		"lib/%s/",
	}

	for _, pattern := range patterns {
		testPath := strings.ReplaceAll(pattern, "%s", app)
		testPath = strings.ReplaceAll(testPath, "{name}", "")
		fullPath := filepath.Join(g.projectRoot, testPath)
		if _, err := os.Stat(filepath.Dir(fullPath)); err == nil {
			return strings.ReplaceAll(pattern, "%s", app)
		}
	}

	return "apps/" + app + "/src/app/v1/{name}/"
}

func (g *Generator) findEndpointExamples(app string) []Example {
	var examples []Example
	framework := g.detectFramework()

	// Framework-specific search paths and patterns
	var searchPaths []string
	var patterns []*regexp.Regexp
	var descSuffix string

	switch {
	case strings.HasPrefix(framework, "go"):
		searchPaths = []string{
			filepath.Join(g.projectRoot, "internal", "handlers"),
			filepath.Join(g.projectRoot, "internal", "handler"),
			filepath.Join(g.projectRoot, "internal", "api"),
			filepath.Join(g.projectRoot, "pkg", "handlers"),
			filepath.Join(g.projectRoot, "handlers"),
		}
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`_handler\.go$`),
			regexp.MustCompile(`handler\.go$`),
		}
		descSuffix = " endpoint (handler + service)"

	case strings.HasPrefix(framework, "python"):
		searchPaths = []string{
			filepath.Join(g.projectRoot, "app", "routers"),
			filepath.Join(g.projectRoot, "app", "api"),
			filepath.Join(g.projectRoot, "routers"),
			filepath.Join(g.projectRoot, "api"),
			filepath.Join(g.projectRoot, "views"),
		}
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`^[a-z_]+\.py$`), // e.g., users.py, orders.py
		}
		descSuffix = " endpoint (router + service)"

	case strings.HasPrefix(framework, "rust"):
		searchPaths = []string{
			filepath.Join(g.projectRoot, "src", "handlers"),
			filepath.Join(g.projectRoot, "src", "routes"),
			filepath.Join(g.projectRoot, "src", "api"),
		}
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`\.rs$`),
		}
		descSuffix = " endpoint (handler + service)"

	default: // NestJS/Express/TypeScript
		searchPaths = []string{
			filepath.Join(g.projectRoot, "apps", app, "src", "app", "v1"),
			filepath.Join(g.projectRoot, "apps", app, "src", "app"),
			filepath.Join(g.projectRoot, "src", "app"),
		}
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`\.controller\.ts$`),
		}
		descSuffix = " endpoint (controller + service + module)"
	}

	type exampleWithMtime struct {
		example Example
		mtime   int64
	}
	var candidates []exampleWithMtime

	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); err != nil {
			continue
		}

		filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			// Skip common non-source files
			basename := filepath.Base(path)
			if basename == "mod.rs" || basename == "__init__.py" || basename == "index.ts" {
				return nil
			}

			// Check against patterns
			for _, pattern := range patterns {
				if pattern.MatchString(basename) {
					relPath, _ := filepath.Rel(g.projectRoot, path)
					dirPath := filepath.Dir(relPath)

					// Extract name from filename
					name := basename
					for _, suffix := range []string{".controller.ts", "_handler.go", ".go", ".py", ".rs"} {
						name = strings.TrimSuffix(name, suffix)
					}

					candidates = append(candidates, exampleWithMtime{
						example: Example{
							Path:        dirPath,
							Description: name + descSuffix,
						},
						mtime: info.ModTime().Unix(),
					})
					break
				}
			}
			return nil
		})

		if len(candidates) > 0 {
			break
		}
	}

	// Sort by modification time (newest first — best example)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].mtime > candidates[j].mtime
	})

	for i, c := range candidates {
		if i >= maxExamples {
			break
		}
		ex := c.example
		if i == 0 {
			ex.Description += " — newest, best example to follow"
		}
		examples = append(examples, ex)
	}

	return examples
}

func (g *Generator) findFeatureExamples(app string) []Example {
	var examples []Example

	searchPath := filepath.Join(g.projectRoot, "apps", app, "src")
	if app == "" {
		searchPath = filepath.Join(g.projectRoot, "src")
	}

	modulePattern := regexp.MustCompile(`\.module\.ts$`)

	filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if modulePattern.MatchString(path) && !strings.Contains(path, "app.module") && len(examples) < maxExamples {
			relPath, _ := filepath.Rel(g.projectRoot, path)
			examples = append(examples, Example{
				Path:        filepath.Dir(relPath),
				Description: strings.TrimSuffix(filepath.Base(path), ".module.ts") + " feature module",
			})
		}
		return nil
	})

	return examples
}

// findTestExamplesForApp scopes test examples to the correct app.
func (g *Generator) findTestExamplesForApp(app, framework string) []Example {
	var examples []Example

	searchPath := g.appSourcePath(app)
	if searchPath == "" {
		// Fallback to project root
		return g.findTestExamples(framework)
	}

	testPattern := regexp.MustCompile(`\.spec\.ts$|\.test\.ts$`)

	filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if strings.Contains(path, "node_modules") {
			return filepath.SkipDir
		}

		if testPattern.MatchString(path) && len(examples) < 2 {
			relPath, _ := filepath.Rel(g.projectRoot, path)
			examples = append(examples, Example{
				Path:        relPath,
				Description: "Test file example (" + filepath.Base(path) + ")",
			})
		}
		return nil
	})

	return examples
}

func (g *Generator) findTestExamples(framework string) []Example {
	var examples []Example

	testPattern := regexp.MustCompile(`\.spec\.ts$|\.test\.ts$`)

	filepath.Walk(g.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if strings.Contains(path, "node_modules") {
			return filepath.SkipDir
		}

		if testPattern.MatchString(path) && len(examples) < 2 {
			relPath, _ := filepath.Rel(g.projectRoot, path)
			examples = append(examples, Example{
				Path:        relPath,
				Description: "Test file example",
			})
		}
		return nil
	})

	return examples
}

func (g *Generator) findExperts(path string) []string {
	expertsFile := filepath.Join(g.tcDir, "knowledge", "git-experts.json")
	data, err := os.ReadFile(expertsFile)
	if err != nil {
		return nil
	}

	var experts map[string]interface{}
	if err := json.Unmarshal(data, &experts); err != nil {
		return nil
	}

	var result []string
	for filePath, expertData := range experts {
		if strings.Contains(filePath, path) || strings.Contains(path, filePath) {
			if expMap, ok := expertData.(map[string]interface{}); ok {
				if contributors, ok := expMap["contributors"].([]interface{}); ok {
					for _, c := range contributors {
						if cMap, ok := c.(map[string]interface{}); ok {
							if name, ok := cMap["name"].(string); ok {
								result = append(result, name)
								if len(result) >= 2 {
									return result
								}
							}
						}
					}
				}
			}
		}
	}

	return result
}

func (g *Generator) getFileCorrelations(path string) []Correlation {
	correlationsFile := filepath.Join(g.tcDir, "knowledge", "git-correlations.json")
	data, err := os.ReadFile(correlationsFile)
	if err != nil {
		return nil
	}

	var rawCorrelations []map[string]interface{}
	if err := json.Unmarshal(data, &rawCorrelations); err != nil {
		return nil
	}

	var result []Correlation
	for _, rc := range rawCorrelations {
		files, ok := rc["files"].([]interface{})
		if !ok {
			continue
		}

		relevant := false
		var fileStrings []string
		for _, f := range files {
			if fs, ok := f.(string); ok {
				fileStrings = append(fileStrings, fs)
				if strings.Contains(fs, path) || strings.Contains(path, fs) {
					relevant = true
				}
			}
		}

		if relevant && len(fileStrings) > 0 {
			confidence := 0.5
			if c, ok := rc["confidence"].(float64); ok {
				confidence = c
			}
			reason := ""
			if r, ok := rc["reason"].(string); ok {
				reason = r
			}
			result = append(result, Correlation{
				Files:      fileStrings,
				Confidence: confidence,
				Reason:     reason,
			})
		}

		if len(result) >= 5 {
			break
		}
	}

	return result
}

func (g *Generator) addRelevantDecisions(bp *Blueprint) {
	decisionsFile := filepath.Join(g.tcDir, "knowledge", "decisions.json")
	data, err := os.ReadFile(decisionsFile)
	if err != nil {
		return
	}

	var decisions []types.Decision
	if err := json.Unmarshal(data, &decisions); err != nil {
		return
	}

	keywords := g.getTaskKeywords(bp.TaskType, bp.App)

	var relevant []Decision
	for _, d := range decisions {
		if g.matchesKeywords(d, keywords) {
			relevant = append(relevant, Decision{
				ID:        d.ID,
				Title:     d.Content,
				Rationale: d.Reason,
				Tags:      d.Tags,
			})
		}
	}

	sort.Slice(relevant, func(i, j int) bool {
		return len(relevant[i].Tags) > len(relevant[j].Tags)
	})

	if len(relevant) > 5 {
		relevant = relevant[:5]
	}

	bp.Decisions = relevant

	if len(relevant) > 0 {
		bp.Confidence += 0.1
	}
}

func (g *Generator) addRelevantWarnings(bp *Blueprint) {
	warningsFile := filepath.Join(g.tcDir, "knowledge", "warnings.json")
	data, err := os.ReadFile(warningsFile)
	if err != nil {
		return
	}

	var warnings []types.Warning
	if err := json.Unmarshal(data, &warnings); err != nil {
		return
	}

	keywords := g.getTaskKeywords(bp.TaskType, bp.App)

	var relevant []Warning
	for _, w := range warnings {
		if g.matchesWarningKeywords(w, keywords) {
			relevant = append(relevant, Warning{
				Title:       w.Content,
				Description: w.Reason,
				Severity:    w.Severity,
			})
		}
	}

	if len(relevant) > 5 {
		relevant = relevant[:5]
	}

	bp.Warnings = relevant

	if len(relevant) > 0 {
		bp.Confidence += 0.1
	}
}

func (g *Generator) getTaskKeywords(taskType TaskType, app string) []string {
	keywords := []string{}

	if app != "" {
		keywords = append(keywords, app)
	}

	switch taskType {
	case TaskAddEndpoint:
		keywords = append(keywords, "api", "endpoint", "controller", "route", "rest", "http", "validation", "guard", "auth")
	case TaskAddFeature:
		keywords = append(keywords, "feature", "module", "service")
	case TaskAddService:
		keywords = append(keywords, "service", "injection", "dependency")
	case TaskFixBug:
		keywords = append(keywords, "bug", "fix", "error", "issue")
	case TaskRefactor:
		keywords = append(keywords, "refactor", "clean", "restructure", "migration")
	case TaskAddTest:
		keywords = append(keywords, "test", "spec", "mock", "jest", "coverage")
	}

	return keywords
}

func (g *Generator) matchesKeywords(d types.Decision, keywords []string) bool {
	searchText := strings.ToLower(d.Content + " " + d.Reason + " " + d.Context + " " + strings.Join(d.Tags, " "))
	for _, kw := range keywords {
		if strings.Contains(searchText, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

func (g *Generator) matchesWarningKeywords(w types.Warning, keywords []string) bool {
	searchText := strings.ToLower(w.Content + " " + w.Reason + " " + strings.Join(w.Tags, " "))
	for _, kw := range keywords {
		if strings.Contains(searchText, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// Ensure imports package is used (scan_imports functionality)
var _ = imports.ScanFile
