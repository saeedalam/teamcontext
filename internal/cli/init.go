package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/saeedalam/teamcontext/internal/git"
	"github.com/saeedalam/teamcontext/internal/storage"
	"github.com/saeedalam/teamcontext/internal/worker"
	"github.com/saeedalam/teamcontext/pkg/types"
	"github.com/spf13/cobra"
)

var projectName string
var skipIDESetup bool
var skipIndexing bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize TeamContext in current directory",
	Long: `Initialize TeamContext in the current directory.

This creates a .teamcontext/ directory with the necessary structure
for storing project knowledge, decisions, warnings, patterns, and feature contexts.

After initialization, it:
1. Automatically indexes all code files (skeletons, imports, graph)
2. Configures detected IDEs (Cursor, VS Code, etc.) with project-level MCP settings

Example:
  teamcontext init
  teamcontext init --name "my-project"
  teamcontext init --skip-ide       # Skip automatic IDE configuration
  teamcontext init --skip-indexing  # Skip initial code indexing`,
	Run: runInit,
}

func init() {
	println("[DEBUG] cli/init.go: init")
	initCmd.Flags().StringVarP(&projectName, "name", "n", "", "Project name")
	initCmd.Flags().BoolVar(&skipIDESetup, "skip-ide", false, "Skip automatic IDE configuration")
	initCmd.Flags().BoolVar(&skipIndexing, "skip-indexing", false, "Skip initial code indexing")
}

func runInit(cmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	tcDir := filepath.Join(cwd, ".teamcontext")

	// Check if already initialized
	if _, err := os.Stat(tcDir); err == nil {
		// Check for existing knowledge that would be lost
		decisionsFile := filepath.Join(tcDir, "knowledge", "decisions.json")
		warningsFile := filepath.Join(tcDir, "knowledge", "warnings.json")
		insightsFile := filepath.Join(tcDir, "knowledge", "insights.json")

		hasKnowledge := false
		for _, f := range []string{decisionsFile, warningsFile, insightsFile} {
			if data, err := os.ReadFile(f); err == nil && len(data) > 3 { // More than "[]"
				hasKnowledge = true
				break
			}
		}

		if hasKnowledge {
			fmt.Println("TeamContext already initialized with existing knowledge.")
			fmt.Println("")
			fmt.Println("To preserve knowledge while re-indexing:")
			fmt.Println("  teamcontext reindex    # Re-index files, preserve decisions/warnings")
			fmt.Println("")
			fmt.Println("To completely reset (DATA LOSS WARNING):")
			fmt.Println("  rm -rf .teamcontext && teamcontext init")
			fmt.Println("")
			fmt.Println("Use 'teamcontext status' to see current state.")
		} else {
			fmt.Println("TeamContext already initialized in this directory.")
			fmt.Println("Use 'teamcontext status' to see current state.")
			fmt.Println("Use 'teamcontext reindex' to re-scan the codebase.")
		}
		return
	}

	// Get project name from directory if not provided
	if projectName == "" {
		projectName = filepath.Base(cwd)
	}

	fmt.Printf("Initializing TeamContext for '%s'...\n", projectName)

	// Create directory structure
	dirs := []string{
		tcDir,
		filepath.Join(tcDir, "index"),
		filepath.Join(tcDir, "knowledge"),
		filepath.Join(tcDir, "features"),
		filepath.Join(tcDir, "archive"),
		filepath.Join(tcDir, "cache"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			return
		}
	}

	// Create config file with MCP enabled by default
	config := types.Config{
		Name:      projectName,
		Version:   "0.1.0",
		CreatedAt: time.Now(),
		Server: types.ServerConfig{
			MCPEnabled:  true, // Enable MCP by default - this is the primary use case
			RESTEnabled: false,
		},
	}
	writeJSONFile(filepath.Join(tcDir, "config.json"), config)

	// Create empty project knowledge
	project := types.Project{
		Name:      projectName,
		UpdatedAt: time.Now(),
	}
	writeJSONFile(filepath.Join(tcDir, "knowledge", "project.json"), project)

	// Create empty index files
	writeJSONFile(filepath.Join(tcDir, "index", "files.json"), map[string]interface{}{})
	writeJSONFile(filepath.Join(tcDir, "index", "architecture.json"), map[string]interface{}{})
	writeJSONFile(filepath.Join(tcDir, "index", "api-surface.json"), []interface{}{})

	// Create empty knowledge files
	writeJSONFile(filepath.Join(tcDir, "knowledge", "decisions.json"), []interface{}{})
	writeJSONFile(filepath.Join(tcDir, "knowledge", "warnings.json"), []interface{}{})
	writeJSONFile(filepath.Join(tcDir, "knowledge", "insights.json"), []interface{}{})
	writeJSONFile(filepath.Join(tcDir, "knowledge", "patterns.json"), []interface{}{})
	writeJSONFile(filepath.Join(tcDir, "knowledge", "graph.json"), map[string]interface{}{"edges": []interface{}{}})
	writeJSONFile(filepath.Join(tcDir, "knowledge", "evolution.json"), map[string]interface{}{"events": []interface{}{}})

	// Create .gitignore
	gitignore := `# TeamContext cache (regenerated from JSON)
cache/
`
	os.WriteFile(filepath.Join(tcDir, ".gitignore"), []byte(gitignore), 0644)

	// Run full project indexing and git history processing in parallel
	if !skipIndexing {
		fmt.Println("")
		fmt.Println("Indexing project files and processing git history...")

		var wg sync.WaitGroup

		// Goroutine 1: Index project files
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("  [DEBUG] Goroutine 1: Starting file indexing...")
			jsonStore := storage.NewJSONStore(tcDir)
			sqliteIndex, err := storage.NewSQLiteIndex(tcDir)
			if err != nil {
				fmt.Printf("  [ERROR] Could not initialize SQLite index: %v\n", err)
				return
			}
			workerMgr := worker.NewManager(tcDir, jsonStore, sqliteIndex)
			indexed, err := workerMgr.InitProject()
			if err != nil {
				fmt.Printf("  [WARNING] Indexing completed with errors: %v\n", err)
			}
			fmt.Printf("  ✓ [DEBUG] Goroutine 1 complete: Indexed %d files\n", indexed)
		}()

		// Goroutine 2: Process git history
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("  [DEBUG] Goroutine 2: Starting git history processing...")
			report, err := git.ProcessGitHistory(cwd)
			if err != nil {
				fmt.Printf("  [WARNING] Git history processing failed: %v\n", err)
				return
			}

			// Cross-reference linked repos if configured
			configPath := filepath.Join(tcDir, "config.json")
			if configData, err := os.ReadFile(configPath); err == nil {
				var cfg types.Config
				if json.Unmarshal(configData, &cfg) == nil && len(cfg.LinkedRepos) > 0 {
					git.CrossReferenceLinkedRepos(report, cfg.LinkedRepos)
				}
			}

			knowledgeDir := filepath.Join(tcDir, "knowledge")
			if err := git.WriteReportFiles(report, knowledgeDir); err != nil {
				fmt.Printf("  [WARNING] Could not write git reports: %v\n", err)
			} else {
				fmt.Printf("  ✓ [DEBUG] Goroutine 2 complete: Processed %d commits\n", report.CommitCount)
			}
		}()

		wg.Wait()

		// Auto-populate project metadata from indexed data
		autoPopulateProjectMetadata(cwd, tcDir)
	}

	// Create AGENT.md with balanced instructions
	agentMD := `# TeamContext Agent Instructions

## When to Use TeamContext vs Native Tools

TeamContext excels at **structured intelligence** that native tools can't provide:

| Use TeamContext for...                     | Use native Read/Grep for...             |
|------------------------------------------|-----------------------------------------|
| Who owns this code? (find_experts)       | Reading a specific file                 |
| What decisions were made? (get_context)  | Searching for exact strings             |
| API surface extraction (get_api_surface) | Quick file lookups                      |
| Code structure overview (get_skeleton)   | Reading implementation details          |
| Risk analysis (get_knowledge_risks)      | Line-by-line debugging                  |
| Tracing data flow (trace_flow)           | Small file reads                        |

**Fallback strategy:** If a TeamContext tool returns too much data or errors, fall back to native tools.

## Quick Reference

### Unique Value Tools (no native equivalent)
- **find_experts** - Git ownership percentages
- **get_knowledge_risks** - Bus factor analysis
- **get_file_correlations** - Files that change together
- **get_api_surface** - REST/Kafka endpoints structured
- **add_decision/add_warning** - Persistent knowledge capture

### Structure Tools (prefer over reading full files)
- **get_skeleton** - Code signatures without bodies (use limit param for directories)
- **get_types** - TypeScript interfaces/types only
- **get_schema_models** - Database models from Prisma/GORM/etc
- **get_code_map** - Directory overview (use path param to zoom in)

### Search Tools
- **search_code** - Full-text with TF-IDF ranking
- **query** - Natural language questions about codebase

### Knowledge Capture (proactive)
When you discover something important:
- **add_decision** - Architectural choices and reasoning
- **add_warning** - Gotchas, footguns, pitfalls
- **add_insight** - Non-obvious behavior
- **add_pattern** - Recurring structures

## Session Workflow

1. **Start:** resume_context or get_context (loads decisions + warnings)
2. **Explore:** Use TeamContext for structure, native tools for content
3. **Record:** Capture decisions/warnings as you discover them
4. **End:** save_conversation for substantial sessions

## Important Notes

- Output limits: Most tools have limit parameters - use them to avoid overflow
- Errors: If a tool fails or returns empty, fall back to native tools
- MCP availability: If TeamContext MCP is offline, proceed with native tools
`
	os.WriteFile(filepath.Join(tcDir, "AGENT.md"), []byte(agentMD), 0644)

	fmt.Println("")
	fmt.Println("TeamContext initialized successfully!")
	fmt.Println("")
	fmt.Println("Directory structure created:")
	fmt.Println("  .teamcontext/")
	fmt.Println("  ├── config.json")
	fmt.Println("  ├── AGENT.md         # Instructions for AI agents")
	fmt.Println("  ├── index/           # Codebase index, architecture")
	fmt.Println("  ├── knowledge/       # Decisions, warnings, patterns, graph")
	fmt.Println("  ├── features/        # Feature contexts")
	fmt.Println("  ├── archive/         # Archived features")
	fmt.Println("  └── cache/           # SQLite search index (not git-tracked)")

	// Auto-configure IDEs
	if !skipIDESetup {
		fmt.Println("")
		configuredIDEs := autoConfigureIDEs(cwd)
		if len(configuredIDEs) > 0 {
			fmt.Println("IDE configurations created:")
			for _, ide := range configuredIDEs {
				fmt.Printf("  ✓ %s\n", ide)
			}
		}
	}

	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("  1. Restart your IDE to load the MCP server")
	fmt.Println("  2. Start a feature context: teamcontext start <feature-name>")
	fmt.Println("  3. Ask the AI: \"What TeamContext tools are available?\"")
	fmt.Println("")
}

// autoConfigureIDEs detects and configures available IDEs
func autoConfigureIDEs(projectRoot string) []string {
	var configured []string

	// Find the teamcontext binary
	binaryPath, err := findBinaryPath()
	if err != nil {
		// Try to find it in PATH
		binaryPath, err = exec.LookPath("teamcontext")
		if err != nil {
			fmt.Println("Warning: Could not find teamcontext binary for IDE config")
			return configured
		}
	}

	// Configure each IDE that supports project-level configs
	for ideName, ide := range supportedIDEs {
		if ide.ProjectConfigDir == "" {
			continue // Skip IDEs that don't support project-level configs (like Claude Desktop)
		}

		configFileName := "mcp.json"
		if ideName == "claudecode" && ide.ProjectConfigDir == "." {
			configFileName = ".mcp.json"
		}
		configPath := filepath.Join(projectRoot, ide.ProjectConfigDir, configFileName)

		// Create directory if needed
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			continue
		}

		// Read or create config
		config, err := readOrCreateConfig(configPath)
		if err != nil {
			continue
		}

		// Add TeamContext with project path
		if err := addTeamContextToConfigWithPath(config, binaryPath, projectRoot); err != nil {
			continue
		}

		// Write config back
		if err := writeConfig(configPath, config); err != nil {
			continue
		}

		displayPath := filepath.Join(ide.ProjectConfigDir, configFileName)
		configured = append(configured, fmt.Sprintf("%s (%s)", ide.Name, displayPath))

		// For Cursor, also check if there's a global config that might conflict
		if ideName == "cursor" || ideName == "windsurf" || ideName == "vscode" {
			globalPath, err := findConfigPath(ide.ConfigPaths)
			if err == nil {
				globalConfig, err := readOrCreateConfig(globalPath)
				if err == nil {
					if mcpServers, ok := globalConfig["mcpServers"].(map[string]interface{}); ok {
						if _, exists := mcpServers["teamcontext"]; exists {
							fmt.Printf("  Note: TeamContext exists in global %s config - project config will take precedence\n", ide.Name)
						}
					}
				}
			}
		}
	}

	return configured
}

// autoPopulateProjectMetadata detects tech stack, architecture, and updates project.json
func autoPopulateProjectMetadata(projectRoot, tcDir string) {
	jsonStore := storage.NewJSONStore(tcDir)

	// --- E1: Auto-detect tech stack for project.json ---
	project, err := jsonStore.GetProject()
	if err != nil {
		project = &types.Project{Name: projectName, UpdatedAt: time.Now()}
	}

	// Count languages from indexed files
	files, _ := jsonStore.GetFilesIndex()
	langCount := make(map[string]int)
	for _, f := range files {
		if f.Language != "" && f.Language != "unknown" {
			langCount[f.Language]++
		}
	}

	// Build sorted language list
	type langStat struct {
		lang  string
		count int
	}
	var langStats []langStat
	for lang, count := range langCount {
		langStats = append(langStats, langStat{lang, count})
	}
	// Sort by count descending
	for i := 0; i < len(langStats); i++ {
		for j := i + 1; j < len(langStats); j++ {
			if langStats[j].count > langStats[i].count {
				langStats[i], langStats[j] = langStats[j], langStats[i]
			}
		}
	}
	var languages []string
	for _, ls := range langStats {
		languages = append(languages, ls.lang)
	}
	project.Languages = languages

	// Detect frameworks
	var frameworks []string

	// Check package.json for Node frameworks
	if pkgData, err := os.ReadFile(filepath.Join(projectRoot, "package.json")); err == nil {
		pkgStr := string(pkgData)
		if strings.Contains(pkgStr, "@nestjs/core") {
			frameworks = append(frameworks, "NestJS")
		}
		if strings.Contains(pkgStr, "express") && !strings.Contains(pkgStr, "@nestjs") {
			frameworks = append(frameworks, "Express")
		}
		if strings.Contains(pkgStr, "next") {
			frameworks = append(frameworks, "Next.js")
		}
	}

	// Check go.mod
	if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
		frameworks = append(frameworks, "Go")
	}

	// Detect monorepo tools
	var monorepoTool string
	if _, err := os.Stat(filepath.Join(projectRoot, "nx.json")); err == nil {
		monorepoTool = "Nx"
	} else if _, err := os.Stat(filepath.Join(projectRoot, "lerna.json")); err == nil {
		monorepoTool = "Lerna"
	} else if _, err := os.Stat(filepath.Join(projectRoot, "pnpm-workspace.yaml")); err == nil {
		monorepoTool = "pnpm workspace"
	}

	// Count apps and libs
	var appNames, libNames []string
	if entries, err := os.ReadDir(filepath.Join(projectRoot, "apps")); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				appNames = append(appNames, e.Name())
			}
		}
	}
	if entries, err := os.ReadDir(filepath.Join(projectRoot, "libs")); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				libNames = append(libNames, e.Name())
			}
		}
	}
	if entries, err := os.ReadDir(filepath.Join(projectRoot, "packages")); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				libNames = append(libNames, e.Name())
			}
		}
	}

	// Build description
	var descParts []string
	if len(frameworks) > 0 {
		descParts = append(descParts, strings.Join(frameworks, "/"))
	}
	if monorepoTool != "" {
		descParts = append(descParts, fmt.Sprintf("monorepo (%s)", monorepoTool))
	}
	if len(appNames) > 0 {
		descParts = append(descParts, fmt.Sprintf("%d apps", len(appNames)))
	}
	if len(libNames) > 0 {
		descParts = append(descParts, fmt.Sprintf("%d libs", len(libNames)))
	}
	if len(languages) > 0 {
		topLang := languages[0]
		if len(languages) > 1 {
			topLang = strings.Join(languages[:min(3, len(languages))], ", ")
		}
		descParts = append(descParts, topLang)
	}

	if len(descParts) > 0 {
		project.Description = strings.Join(descParts, " with ")
	}

	if err := jsonStore.SaveProject(project); err != nil {
		fmt.Printf("  Warning: Could not update project.json: %v\n", err)
	} else {
		fmt.Printf("  ✓ Auto-detected project metadata: %s\n", project.Description)
	}

	// --- E3: Auto-detect architecture from structure ---
	arch := &types.Architecture{
		UpdatedAt: time.Now(),
	}

	var services []types.ServiceNode
	for _, appName := range appNames {
		services = append(services, types.ServiceNode{
			Name:        appName,
			Description: fmt.Sprintf("Application: %s", appName),
		})
	}
	arch.Services = services

	archDesc := fmt.Sprintf("Project with %d apps and %d libs", len(appNames), len(libNames))
	if monorepoTool != "" {
		archDesc = fmt.Sprintf("%s monorepo with %d apps and %d libs", monorepoTool, len(appNames), len(libNames))
	}
	arch.Description = archDesc

	if err := jsonStore.SaveArchitecture(arch); err != nil {
		fmt.Printf("  Warning: Could not update architecture.json: %v\n", err)
	}
}

func writeJSONFile(path string, data interface{}) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}
