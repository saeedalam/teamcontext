package worker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/saeedalam/teamcontext/internal/git"
	"github.com/saeedalam/teamcontext/internal/imports"
	"github.com/saeedalam/teamcontext/internal/search"
	"github.com/saeedalam/teamcontext/internal/skeleton"
	"github.com/saeedalam/teamcontext/internal/storage"
	"github.com/saeedalam/teamcontext/pkg/types"
)

// WorkerConfig configures the background workers
type WorkerConfig struct {
	GitWatchInterval    time.Duration `json:"git_watch_interval"`    // How often to check git for changes
	ReindexInterval     time.Duration `json:"reindex_interval"`      // How often to do full reindex
	AutoDiscoverInterval time.Duration `json:"auto_discover_interval"` // How often to scan for new files
	SkeletonCacheEnable bool          `json:"skeleton_cache_enable"` // Cache skeletons on index
	AutoDiscoverEnable  bool          `json:"auto_discover_enable"`  // Auto-discover and index new files
	Enabled             bool          `json:"enabled"`
}

// DefaultConfig returns sensible defaults
func DefaultConfig() WorkerConfig {
	return WorkerConfig{
		GitWatchInterval:     30 * time.Second,
		ReindexInterval:      5 * time.Minute,
		AutoDiscoverInterval: 10 * time.Minute,
		SkeletonCacheEnable:  true,
		AutoDiscoverEnable:   true,
		Enabled:              true,
	}
}

// Manager manages background workers
type Manager struct {
	config      WorkerConfig
	jsonStore   *storage.JSONStore
	sqliteIndex *storage.SQLiteIndex
	projectRoot string
	basePath    string

	stopChan    chan struct{}
	wg          sync.WaitGroup
	mu          sync.RWMutex
	running     bool
	lastGitHash string

	// Cached data
	skeletonCache map[string]*types.CodeSkeleton
	cacheMu       sync.RWMutex

	// Statistics
	stats WorkerStats
}

// WorkerStats tracks worker activity
type WorkerStats struct {
	GitChecks        int       `json:"git_checks"`
	FilesReindexed   int       `json:"files_reindexed"`
	SkeletonsCached  int       `json:"skeletons_cached"`
	LastGitCheck     time.Time `json:"last_git_check"`
	LastReindex      time.Time `json:"last_reindex"`
	ChangesDetected  int       `json:"changes_detected"`
	ErrorCount       int       `json:"error_count"`
	LastError        string    `json:"last_error,omitempty"`
}

// NewManager creates a new worker manager
func NewManager(basePath string, jsonStore *storage.JSONStore, sqliteIndex *storage.SQLiteIndex) *Manager {
	// Project root is parent of .teamcontext
	projectRoot := filepath.Dir(basePath)

	return &Manager{
		config:        DefaultConfig(),
		jsonStore:     jsonStore,
		sqliteIndex:   sqliteIndex,
		projectRoot:   projectRoot,
		basePath:      basePath,
		stopChan:      make(chan struct{}),
		skeletonCache: make(map[string]*types.CodeSkeleton),
	}
}

// SetConfig updates the worker configuration
func (m *Manager) SetConfig(config WorkerConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// GetConfig returns current configuration
func (m *Manager) GetConfig() WorkerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Start begins background workers
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("workers already running")
	}
	m.running = true
	m.stopChan = make(chan struct{})
	m.mu.Unlock()

	// Start git watcher
	m.wg.Add(1)
	go m.gitWatcher()

	// Start periodic reindexer
	m.wg.Add(1)
	go m.periodicReindexer()

	// Start auto-discovery (if enabled)
	if m.config.AutoDiscoverEnable {
		m.wg.Add(1)
		go m.autoDiscoverer()
	}

	// Log startup
	m.logEvent("Workers started", nil)

	return nil
}

// Stop halts all background workers
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	close(m.stopChan)
	m.mu.Unlock()

	m.wg.Wait()
	m.logEvent("Workers stopped", nil)
}

// IsRunning returns whether workers are active
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetStats returns worker statistics
func (m *Manager) GetStats() WorkerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// GetCachedSkeleton returns a cached skeleton if available
func (m *Manager) GetCachedSkeleton(path string) (*types.CodeSkeleton, bool) {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()
	sk, ok := m.skeletonCache[path]
	return sk, ok
}

// toRelativePath converts an absolute path to a relative path from projectRoot
func (m *Manager) toRelativePath(absPath string) string {
	rel, err := filepath.Rel(m.projectRoot, absPath)
	if err != nil {
		return absPath
	}
	return rel
}



// gitWatcher monitors git for changes and triggers reindexing
func (m *Manager) gitWatcher() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.GitWatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkGitChanges()
		}
	}
}

// periodicReindexer does periodic full reindex checks
func (m *Manager) periodicReindexer() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.ReindexInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.periodicReindex()
		}
	}
}

// autoDiscoverer scans for new files and auto-indexes them
func (m *Manager) autoDiscoverer() {
	defer m.wg.Done()

	// Run immediately on start
	m.discoverAndIndexFiles()

	ticker := time.NewTicker(m.config.AutoDiscoverInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.discoverAndIndexFiles()
		}
	}
}

// discoverAndIndexFiles walks the project and indexes new source files
func (m *Manager) discoverAndIndexFiles() {
	// Get already indexed files
	existingFiles, _ := m.jsonStore.GetFilesIndex()
	existingPaths := make(map[string]bool)
	for path := range existingFiles {
		existingPaths[path] = true
	}

	newFilesIndexed := 0

	// Walk the project looking for source files
	err := filepath.Walk(m.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories we don't care about
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == ".git" || name == "dist" ||
				name == "vendor" || name == ".teamcontext" || name == "coverage" ||
				name == "__pycache__" || name == ".next" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip if already indexed
		if existingPaths[path] {
			return nil
		}

		// Check if it's a source file we care about
		ext := strings.ToLower(filepath.Ext(path))
		if !isSourceFile(ext) {
			return nil
		}

		// Auto-index the file
		if err := m.autoIndexFile(path); err == nil {
			newFilesIndexed++
		}

		return nil
	})

	if err != nil {
		m.recordError("auto-discover walk", err)
	}

	if newFilesIndexed > 0 {
		m.logEvent(fmt.Sprintf("Auto-discovered and indexed %d new files", newFilesIndexed), nil)
	}
}

// autoIndexFile creates a basic index entry for a file without LLM summary
func (m *Manager) autoIndexFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Detect language
	ext := strings.ToLower(filepath.Ext(path))
	language := extToLanguage(ext)

	// Create basic file index without summary (agent can add summary later)
	fileIndex := &types.FileIndex{
		Path:        m.toRelativePath(path),
		Summary:     "[auto-indexed - needs summary]",
		Language:    language,
		IndexedAt:   time.Now(),
		ContentHash: fmt.Sprintf("%d", info.ModTime().UnixNano()),
	}

	// Try to extract exports from skeleton
	if sk, err := skeleton.ParseFile(path); err == nil && sk != nil {
		for _, fn := range sk.Functions {
			fileIndex.Exports = append(fileIndex.Exports, types.Export{
				Name: fn.Name,
				Kind: "function",
				Line: fn.Line,
			})
		}
		for _, class := range sk.Classes {
			fileIndex.Exports = append(fileIndex.Exports, types.Export{
				Name: class.Name,
				Kind: "class",
				Line: class.Line,
			})
		}
	}

	// Try to extract imports
	if results, err := imports.ScanFile(path); err == nil {
		for _, r := range results {
			fileIndex.Imports = append(fileIndex.Imports, r.Imported)
		}
	}

	// Save to JSON store
	if err := m.jsonStore.SaveFileIndex(fileIndex); err != nil {
		return err
	}

	// Index in SQLite
	m.sqliteIndex.IndexFile(fileIndex)

	// Index content for search
	m.indexFileContent(path, language)

	// Cache skeleton if enabled
	if m.config.SkeletonCacheEnable {
		if sk, err := skeleton.ParseFile(path); err == nil {
			m.cacheMu.Lock()
			m.skeletonCache[path] = sk
			m.stats.SkeletonsCached = len(m.skeletonCache)
			m.cacheMu.Unlock()
		}
	}

	return nil
}

// indexFileContent indexes file content chunks for search
func (m *Manager) indexFileContent(path string, language string) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	relPath := m.toRelativePath(path)
	lines := strings.Split(string(content), "\n")
	chunkSize := 50

	var chunks []storage.CodeChunk

	// Try semantic chunks first
	if sk, err := skeleton.ParseFile(path); err == nil && sk != nil {
		for _, fn := range sk.Functions {
			startLine := fn.Line
			endLine := findBlockEndLines(lines, startLine-1, chunkSize)
			chunks = append(chunks, storage.CodeChunk{
				FilePath:  relPath,
				ChunkType: "function",
				ChunkName: fn.Name,
				StartLine: startLine,
				EndLine:   endLine,
				Content:   getLinesContent(lines, startLine, endLine),
				Language:  language,
			})
		}
		for _, class := range sk.Classes {
			startLine := class.Line
			endLine := findBlockEndLines(lines, startLine-1, chunkSize*2)
			chunks = append(chunks, storage.CodeChunk{
				FilePath:  relPath,
				ChunkType: "class",
				ChunkName: class.Name,
				StartLine: startLine,
				EndLine:   endLine,
				Content:   getLinesContent(lines, startLine, endLine),
				Language:  language,
			})
		}
	}

	// Add line-based chunks for coverage
	if len(chunks) == 0 || len(lines) > len(chunks)*chunkSize*2 {
		for i := 0; i < len(lines); i += chunkSize {
			endLine := i + chunkSize
			if endLine > len(lines) {
				endLine = len(lines)
			}
			chunkContent := strings.Join(lines[i:endLine], "\n")
			if strings.TrimSpace(chunkContent) != "" {
				chunks = append(chunks, storage.CodeChunk{
					FilePath:  relPath,
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

	m.sqliteIndex.IndexCodeChunks(relPath, chunks)
}

// Helper functions

func isSourceFile(ext string) bool {
	sourceExts := map[string]bool{
		".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".mjs": true,
		".go": true, ".py": true, ".java": true, ".cs": true, ".rs": true,
		".c": true, ".cpp": true, ".h": true, ".hpp": true,
		".rb": true, ".php": true, ".swift": true, ".kt": true, ".scala": true,
	}
	return sourceExts[ext]
}

func extToLanguage(ext string) string {
	langMap := map[string]string{
		".ts": "typescript", ".tsx": "typescript",
		".js": "javascript", ".jsx": "javascript", ".mjs": "javascript",
		".go": "go", ".py": "python", ".java": "java", ".cs": "csharp",
		".rs": "rust", ".c": "c", ".cpp": "cpp", ".h": "c", ".hpp": "cpp",
		".rb": "ruby", ".php": "php", ".swift": "swift", ".kt": "kotlin", ".scala": "scala",
		".json": "json", ".yaml": "yaml", ".yml": "yaml", ".toml": "toml",
		".sql": "sql", ".prisma": "prisma", ".graphql": "graphql", ".gql": "graphql",
		".md": "markdown", ".mdx": "markdown",
		".sh": "shell", ".bash": "shell", ".zsh": "shell",
		".dockerfile": "dockerfile",
		".xml": "xml", ".html": "html", ".css": "css", ".scss": "scss", ".less": "less",
	}
	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return "unknown"
}

func findBlockEndLines(lines []string, startIdx int, maxLines int) int {
	if startIdx >= len(lines) {
		return len(lines)
	}
	braceCount := 0
	started := false
	endIdx := startIdx
	for i := startIdx; i < len(lines) && i < startIdx+maxLines; i++ {
		line := lines[i]
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")
		if strings.Contains(line, "{") {
			started = true
		}
		endIdx = i + 1
		if started && braceCount <= 0 {
			break
		}
	}
	return endIdx
}

func getLinesContent(lines []string, startLine, endLine int) string {
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine {
		return ""
	}
	return strings.Join(lines[startLine-1:endLine], "\n")
}

// buildFileSummary creates a descriptive summary from parsed skeleton data
func buildFileSummary(language string, sk *types.CodeSkeleton) string {
	if sk == nil {
		return fmt.Sprintf("Auto-indexed %s file", language)
	}

	// Collect all export names by kind
	var classes, functions, interfaces, typeDefs []string
	for _, c := range sk.Classes {
		classes = append(classes, c.Name)
	}
	for _, f := range sk.Functions {
		functions = append(functions, f.Name)
	}
	for _, i := range sk.Interfaces {
		interfaces = append(interfaces, i.Name)
	}
	for _, t := range sk.Types {
		typeDefs = append(typeDefs, t.Name)
	}

	// Detect NestJS patterns from class names
	nestPatterns := map[string]string{
		"Controller":  "NestJS controller",
		"Module":      "NestJS module",
		"Service":     "NestJS service",
		"Guard":       "NestJS guard",
		"Pipe":        "NestJS pipe",
		"Interceptor": "NestJS interceptor",
		"Middleware":   "NestJS middleware",
		"Gateway":     "NestJS gateway",
		"Resolver":    "NestJS resolver",
		"Filter":      "NestJS exception filter",
	}

	for _, name := range classes {
		for suffix, label := range nestPatterns {
			if strings.HasSuffix(name, suffix) {
				return fmt.Sprintf("%s: %s", label, name)
			}
		}
	}

	// Also check function decorators for NestJS patterns
	for _, cls := range sk.Classes {
		for _, dec := range cls.Methods {
			for _, d := range dec.Decorators {
				if strings.Contains(d, "Controller") || strings.Contains(d, "Injectable") {
					return fmt.Sprintf("NestJS class: %s", cls.Name)
				}
			}
		}
	}

	// Build summary from what we found
	nameList := func(names []string, max int) string {
		if len(names) > max {
			return strings.Join(names[:max], ", ") + fmt.Sprintf(" (+%d more)", len(names)-max)
		}
		return strings.Join(names, ", ")
	}

	if len(classes) > 0 {
		return fmt.Sprintf("%d class(es): %s", len(classes), nameList(classes, 3))
	}
	if len(interfaces) > 0 {
		return fmt.Sprintf("%d interface(s): %s", len(interfaces), nameList(interfaces, 3))
	}
	if len(functions) > 0 {
		return fmt.Sprintf("%d function(s): %s", len(functions), nameList(functions, 3))
	}
	if len(typeDefs) > 0 {
		return fmt.Sprintf("%d type(s): %s", len(typeDefs), nameList(typeDefs, 3))
	}

	// Fallback with counts
	total := len(classes) + len(functions) + len(interfaces) + len(typeDefs) + len(sk.Enums) + len(sk.Constants)
	if total > 0 {
		return fmt.Sprintf("%s file with %d exports", language, total)
	}

	return fmt.Sprintf("Auto-indexed %s file", language)
}

// checkGitChanges looks for git changes and reindexes affected files
func (m *Manager) checkGitChanges() {
	m.mu.Lock()
	m.stats.GitChecks++
	m.stats.LastGitCheck = time.Now()
	m.mu.Unlock()

	// Get current HEAD
	currentHash, err := m.getCurrentGitHash()
	if err != nil {
		m.recordError("git hash", err)
		return
	}

	// First run - just record the hash
	if m.lastGitHash == "" {
		m.lastGitHash = currentHash
		return
	}

	// No changes
	if currentHash == m.lastGitHash {
		return
	}

	// Get changed files since last check
	changedFiles, err := m.getChangedFiles(m.lastGitHash, currentHash)
	if err != nil {
		m.recordError("git diff", err)
		return
	}

	m.lastGitHash = currentHash

	if len(changedFiles) == 0 {
		return
	}

	m.mu.Lock()
	m.stats.ChangesDetected += len(changedFiles)
	m.mu.Unlock()

	// Process ALL changed files (new or existing)
	indexed := 0
	graphEdgesCreated := 0

	for _, file := range changedFiles {
		fullPath := filepath.Join(m.projectRoot, file)

		// Check if file exists (might be deleted)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// File deleted - remove from index and graph
			m.handleDeletedFile(fullPath)
			continue
		}

		// Check if file was previously indexed
		existing, err := m.jsonStore.GetFileIndex(fullPath)
		if err != nil {
			// NEW file - auto-index it!
			if err := m.autoIndexNewFile(fullPath); err != nil {
				m.recordError("auto-index "+file, err)
			} else {
				indexed++
			}
		} else {
			// EXISTING file - reindex it
			if err := m.fullReindexFile(fullPath, existing); err != nil {
				m.recordError("reindex "+file, err)
			} else {
				indexed++
			}
		}

		// Always update graph edges for imports
		edges, err := m.updateGraphEdgesForFile(fullPath)
		if err == nil {
			graphEdgesCreated += edges
		}
	}

	m.mu.Lock()
	m.stats.FilesReindexed += indexed
	m.mu.Unlock()

	m.logEvent(fmt.Sprintf("Git change: indexed %d files, created %d graph edges", indexed, graphEdgesCreated), changedFiles)
}

// periodicReindex validates index integrity
func (m *Manager) periodicReindex() {
	m.mu.Lock()
	m.stats.LastReindex = time.Now()
	m.mu.Unlock()

	// Get all indexed files
	files, err := m.jsonStore.GetFilesIndex()
	if err != nil {
		m.recordError("get index", err)
		return
	}

	reindexed := 0
	for path, file := range files {
		// Check if file still exists
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			// File deleted - could remove from index
			continue
		}

		// Check if file was modified since last index
		if info.ModTime().After(file.IndexedAt) {
			if err := m.reindexFile(path); err != nil {
				m.recordError("periodic reindex "+path, err)
			} else {
				reindexed++
			}
		}
	}

	if reindexed > 0 {
		m.mu.Lock()
		m.stats.FilesReindexed += reindexed
		m.mu.Unlock()
		m.logEvent(fmt.Sprintf("Periodic reindex: %d files", reindexed), nil)
	}
}

// reindexFile updates the index for a single file
func (m *Manager) reindexFile(path string) error {
	// Get existing index entry
	existing, err := m.jsonStore.GetFileIndex(path)
	if err != nil {
		return err
	}

	// Update timestamp and re-save
	existing.IndexedAt = time.Now()

	// If skeleton caching is enabled, update skeleton cache
	if m.config.SkeletonCacheEnable {
		sk, err := skeleton.ParseFile(path)
		if err == nil {
			m.cacheMu.Lock()
			m.skeletonCache[path] = sk
			m.stats.SkeletonsCached = len(m.skeletonCache)
			m.cacheMu.Unlock()
		}
	}

	// Save back to JSON store
	if err := m.jsonStore.SaveFileIndex(existing); err != nil {
		return err
	}

	// Update SQLite index
	m.sqliteIndex.IndexFile(existing)

	return nil
}

// getCurrentGitHash returns current HEAD hash
func (m *Manager) getCurrentGitHash() (string, error) {
	changes, err := git.GetRecentChanges(m.projectRoot, "", 1)
	if err != nil {
		return "", err
	}
	if len(changes) == 0 {
		return "", fmt.Errorf("no git commits found")
	}
	return changes[0].Hash, nil
}

// getChangedFiles returns files changed between two commits
func (m *Manager) getChangedFiles(fromHash, toHash string) ([]string, error) {
	changes, err := git.GetRecentChanges(m.projectRoot, "", 50)
	if err != nil {
		return nil, err
	}

	var files []string
	collecting := false
	for _, c := range changes {
		if c.Hash == toHash {
			collecting = true
		}
		if collecting {
			files = append(files, c.FilesChanged...)
		}
		if c.Hash == fromHash {
			break
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, f := range files {
		if !seen[f] {
			seen[f] = true
			unique = append(unique, f)
		}
	}

	return unique, nil
}

// recordError records an error in stats
func (m *Manager) recordError(context string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.ErrorCount++
	m.stats.LastError = fmt.Sprintf("%s: %v", context, err)
}

// logEvent writes an event to the worker log
func (m *Manager) logEvent(message string, data interface{}) {
	logPath := filepath.Join(m.basePath, "cache", "worker.log")

	entry := struct {
		Time    time.Time   `json:"time"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	}{
		Time:    time.Now(),
		Message: message,
		Data:    data,
	}

	line, _ := json.Marshal(entry)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	f.WriteString(string(line) + "\n")
}

// TriggerReindex manually triggers a reindex of specific files
func (m *Manager) TriggerReindex(paths []string) (int, error) {
	reindexed := 0
	for _, path := range paths {
		if err := m.reindexFile(path); err != nil {
			m.recordError("manual reindex "+path, err)
		} else {
			reindexed++
		}
	}
	return reindexed, nil
}

// ClearCache clears the skeleton cache
func (m *Manager) ClearCache() {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	m.skeletonCache = make(map[string]*types.CodeSkeleton)
	m.stats.SkeletonsCached = 0
}

// =============================================================================
// ENHANCED AUTO-INDEXING (Git-Connected)
// =============================================================================

// InitProject does a full project scan and indexes all relevant files
// Call this on first setup or to rebuild the entire index
func (m *Manager) InitProject() (int, error) {
	m.logEvent("Starting project initialization", nil)

	indexed := 0
	graphEdges := 0

	supportedExts := map[string]bool{
		".ts": true, ".tsx": true, ".js": true, ".jsx": true,
		".go": true, ".py": true, ".java": true, ".cs": true,
		".rb": true, ".rs": true, ".kt": true, ".swift": true,
		".prisma": true, ".sql": true,
		".json": true, ".yaml": true, ".yml": true, ".toml": true,
	}

	skipDirs := map[string]bool{
		"node_modules": true, ".git": true, "vendor": true,
		"dist": true, "build": true, "target": true,
		"__pycache__": true, ".next": true, ".nuxt": true,
		"coverage": true, ".cache": true,
	}

	allFiles := make(map[string]types.FileIndex)
	var allEdges []types.Edge

	err := filepath.Walk(m.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories
		if info.IsDir() {
			name := info.Name()
			if skipDirs[name] || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check extension
		ext := strings.ToLower(filepath.Ext(path))
		if !supportedExts[ext] {
			return nil
		}

		// Skip very large files (> 1MB)
		if info.Size() > 1024*1024 {
			return nil
		}

		// Prepare file index entry
		fileIndex, err := m.prepareFileIndex(path)
		if err != nil {
			m.recordError("init-index-prep "+path, err)
			return nil
		}
		allFiles[fileIndex.Path] = *fileIndex
		indexed++

		if indexed%50 == 0 {
			fmt.Printf("  ... processed %d files\n", indexed)
		}

		// Save to SQLite
		m.sqliteIndex.IndexFile(fileIndex)

		// Index code chunks for content search in SQLite
		m.indexFileContent(path, fileIndex.Language)

		// Collect graph edges (imports)
		importResults, _ := imports.ScanFile(path)
		for _, imp := range importResults {
			if imp.ImportType == "package" || imp.ImportType == "builtin" {
				continue
			}

			targetPath := imp.Imported
			dir := filepath.Dir(path)
			if imp.ImportType == "relative" && !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(dir, targetPath)
			}

			// Basic resolution for typical extensions
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".go", ".py"} {
					if _, err := os.Stat(targetPath + ext); err == nil {
						targetPath = targetPath + ext
						break
					}
				}
			}

			edge := types.Edge{
				FromType: "file",
				FromID:   m.toRelativePath(path),
				ToType:   "file",
				ToID:     m.toRelativePath(targetPath),
				Relation: "imports",
			}
			allEdges = append(allEdges, edge)
		}

		return nil
	})

	if err != nil {
		fmt.Printf("  [DEBUG] Error during file walk: %v\n", err)
	}

	// Perform bulk saves to JSON store—this is the primary performance win
	if err == nil {
		fmt.Printf("  [DEBUG] Saving bulk index for %d files...\n", indexed)
		if saveErr := m.jsonStore.SaveFilesIndexBulk(allFiles); saveErr != nil {
			fmt.Printf("  [WARNING] Error saving bulk file index: %v\n", saveErr)
		}
		fmt.Printf("  [DEBUG] Saving bulk graph for %d edges...\n", len(allEdges))
		if saveErr := m.jsonStore.AddEdgesBulk(allEdges); saveErr != nil {
			fmt.Printf("  [WARNING] Error saving bulk graph edges: %v\n", saveErr)
		}
		graphEdges = len(allEdges)
	}

	m.logEvent(fmt.Sprintf("Project init complete: %d files indexed, %d graph edges", indexed, graphEdges), nil)

	// Build semantic index for TF-IDF search (only if knowledge items exist;
	// on first init there are typically none yet, so skip to avoid OOM on large codebases)
	decisions, _ := m.jsonStore.GetDecisions()
	warnings, _ := m.jsonStore.GetWarnings()
	patterns, _ := m.jsonStore.GetPatterns()
	insights, _ := m.jsonStore.GetInsights()
	knowledgeCount := len(decisions) + len(warnings) + len(patterns) + len(insights)
	if knowledgeCount > 0 {
		semanticCount, semanticErr := m.BuildSemanticIndex()
		if semanticErr != nil {
			m.logEvent("Semantic index build failed: "+semanticErr.Error(), nil)
		} else {
			m.logEvent(fmt.Sprintf("Semantic index built: %d documents", semanticCount), nil)
		}
	} else {
		m.logEvent("Skipping semantic index (no knowledge items yet)", nil)
	}

	return indexed, err
}



// handleDeletedFile removes a file from the index and graph
func (m *Manager) handleDeletedFile(path string) {
	// Remove from JSON store (if exists)
	// Note: We don't have a delete method, but could add one
	// For now, just log
	m.logEvent("File deleted", path)

	// Note: Edge cleanup would require a DeleteEdge method
	// For now, edges will become stale but won't cause issues
}

// autoIndexNewFile creates a full index entry for a new file and saves it
func (m *Manager) autoIndexNewFile(path string) error {
	fileIndex, err := m.prepareFileIndex(path)
	if err != nil {
		return err
	}

	// Save to JSON store
	if err := m.jsonStore.SaveFileIndex(fileIndex); err != nil {
		return err
	}

	// Save to SQLite
	m.sqliteIndex.IndexFile(fileIndex)

	// Index code chunks
	m.indexFileContent(path, fileIndex.Language)

	return nil
}

// prepareFileIndex builds a FileIndex struct for a file without saving it
func (m *Manager) prepareFileIndex(path string) (*types.FileIndex, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	language := extToLanguage(ext)

	var sk *types.CodeSkeleton
	if m.config.SkeletonCacheEnable {
		sk, _ = skeleton.ParseFile(path)
	}

	importResults, _ := imports.ScanFile(path)
	var relImportPaths []string
	for _, imp := range importResults {
		relImportPaths = append(relImportPaths, m.toRelativePath(imp.Imported))
	}

	var exports []types.Export
	if sk != nil {
		for _, fn := range sk.Functions {
			exports = append(exports, types.Export{Name: fn.Name, Kind: "function", Line: fn.Line})
		}
		for _, cls := range sk.Classes {
			exports = append(exports, types.Export{Name: cls.Name, Kind: "class", Line: cls.Line})
		}
		for _, iface := range sk.Interfaces {
			exports = append(exports, types.Export{Name: iface.Name, Kind: "interface", Line: iface.Line})
		}
		for _, t := range sk.Types {
			exports = append(exports, types.Export{Name: t.Name, Kind: "type", Line: t.Line})
		}
	}

	return &types.FileIndex{
		Path:      m.toRelativePath(path),
		Summary:   buildFileSummary(language, sk),
		Exports:   exports,
		Imports:   relImportPaths,
		Language:  language,
		SizeBytes: info.Size(),
		LineCount: strings.Count(string(content), "\n") + 1,
		IndexedAt: time.Now(),
	}, nil
}

// fullReindexFile does a complete reindex of an existing file
func (m *Manager) fullReindexFile(path string, existing *types.FileIndex) error {
	// Read current content
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Parse skeleton
	var sk *types.CodeSkeleton
	if m.config.SkeletonCacheEnable {
		sk, _ = skeleton.ParseFile(path)
	}

	// Extract imports
	importResults, _ := imports.ScanFile(path)
	var importPaths []string
	for _, imp := range importResults {
		importPaths = append(importPaths, imp.Imported)
	}

	// Extract exports from skeleton
	var exports []types.Export
	if sk != nil {
		for _, fn := range sk.Functions {
			exports = append(exports, types.Export{Name: fn.Name, Kind: "function", Line: fn.Line})
		}
		for _, cls := range sk.Classes {
			exports = append(exports, types.Export{Name: cls.Name, Kind: "class", Line: cls.Line})
		}
		for _, iface := range sk.Interfaces {
			exports = append(exports, types.Export{Name: iface.Name, Kind: "interface", Line: iface.Line})
		}
		for _, t := range sk.Types {
			exports = append(exports, types.Export{Name: t.Name, Kind: "type", Line: t.Line})
		}
	}

	// Update existing entry
	existing.Exports = exports
	existing.Imports = importPaths
	existing.SizeBytes = info.Size()
	existing.LineCount = strings.Count(string(content), "\n") + 1
	existing.IndexedAt = time.Now()

	// Save to JSON store
	if err := m.jsonStore.SaveFileIndex(existing); err != nil {
		return err
	}

	// Update SQLite
	m.sqliteIndex.IndexFile(existing)

	// Re-index code chunks
	m.indexFileContent(path, existing.Language)

	// Update skeleton cache
	if sk != nil {
		m.cacheMu.Lock()
		m.skeletonCache[path] = sk
		m.stats.SkeletonsCached = len(m.skeletonCache)
		m.cacheMu.Unlock()
	}

	return nil
}

// updateGraphEdgesForFile creates/updates graph edges based on imports
func (m *Manager) updateGraphEdgesForFile(path string) (int, error) {
	// Scan imports
	importResults, err := imports.ScanFile(path)
	if err != nil || len(importResults) == 0 {
		return 0, err
	}

	edgesCreated := 0
	dir := filepath.Dir(path)

	for _, imp := range importResults {
		targetPath := imp.Imported

		// Skip external packages — no graph value
		if imp.ImportType == "package" || imp.ImportType == "builtin" {
			continue
		}

		// Only resolve if not already absolute (resolveRelative in scanner already resolved it)
		if imp.ImportType == "relative" && !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(dir, targetPath)
		}

		// Try to resolve with extensions if file not found
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			resolved := false
			for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".go", ".py"} {
				if _, err := os.Stat(targetPath + ext); err == nil {
					targetPath = targetPath + ext
					resolved = true
					break
				}
			}
			// Also try index files (e.g. ./foo -> ./foo/index.ts)
			if !resolved {
				for _, ext := range []string{".ts", ".tsx", ".js", ".jsx"} {
					indexPath := filepath.Join(targetPath, "index"+ext)
					if _, err := os.Stat(indexPath); err == nil {
						targetPath = indexPath
						break
					}
				}
			}
		}

		// Convert to relative paths for storage
		fromRel := m.toRelativePath(path)
		toRel := m.toRelativePath(targetPath)

		// Create "imports" edge using types.Edge
		edge := &types.Edge{
			FromType: "file",
			FromID:   fromRel,
			ToType:   "file",
			ToID:     toRel,
			Relation: "imports",
		}

		if err := m.jsonStore.AddEdge(edge); err == nil {
			edgesCreated++
		}
	}

	return edgesCreated, nil
}

// BuildSemanticIndex collects all knowledge documents, builds a TF-IDF vocabulary,
// vectorizes each document, and stores vectors + vocab in SQLite.
func (m *Manager) BuildSemanticIndex() (int, error) {
	type docEntry struct {
		id      string
		docType string
		text    string
	}

	var docs []docEntry

	// Collect decisions
	decisions, _ := m.jsonStore.GetDecisions()
	for _, d := range decisions {
		docs = append(docs, docEntry{
			id:      d.ID,
			docType: "decision",
			text:    d.Content + " " + d.Reason + " " + d.Context,
		})
	}

	// Collect warnings
	warnings, _ := m.jsonStore.GetWarnings()
	for _, w := range warnings {
		docs = append(docs, docEntry{
			id:      w.ID,
			docType: "warning",
			text:    w.Content + " " + w.Reason + " " + w.Evidence,
		})
	}

	// Collect patterns
	patterns, _ := m.jsonStore.GetPatterns()
	for _, p := range patterns {
		docs = append(docs, docEntry{
			id:      p.ID,
			docType: "pattern",
			text:    p.Name + " " + p.Description,
		})
	}

	// Collect insights
	insights, _ := m.jsonStore.GetInsights()
	for _, i := range insights {
		docs = append(docs, docEntry{
			id:      i.ID,
			docType: "insight",
			text:    i.Content + " " + i.Context,
		})
	}

	// Collect file summaries (cap at 500 to avoid OOM on large codebases).
	// Sort deterministically by priority score then path to ensure consistent selection.
	files, _ := m.jsonStore.GetFilesIndex()
	type filePriority struct {
		path  string
		file  types.FileIndex
		score int
	}
	filePriorities := make([]filePriority, 0, len(files))
	keyDirs := map[string]bool{
		"src": true, "lib": true, "pkg": true, "internal": true,
		"cmd": true, "app": true, "core": true, "api": true,
	}
	for path, f := range files {
		score := len(f.Exports) // more exports = more important
		// Boost files in key directories
		parts := strings.Split(filepath.ToSlash(path), "/")
		for _, p := range parts {
			if keyDirs[p] {
				score += 5
				break
			}
		}
		filePriorities = append(filePriorities, filePriority{path: path, file: f, score: score})
	}
	sort.Slice(filePriorities, func(i, j int) bool {
		if filePriorities[i].score != filePriorities[j].score {
			return filePriorities[i].score > filePriorities[j].score
		}
		return filePriorities[i].path < filePriorities[j].path
	})
	if len(filePriorities) > 500 {
		filePriorities = filePriorities[:500]
	}
	for _, fp := range filePriorities {
		exportNames := ""
		for _, e := range fp.file.Exports {
			exportNames += " " + e.Name
		}
		docs = append(docs, docEntry{
			id:      fp.path,
			docType: "file",
			text:    fp.file.Summary + " " + fp.path + exportNames,
		})
	}

	// Collect git expert knowledge for vectorization
	knowledgeDir := filepath.Join(m.basePath, "knowledge")
	if expertsData, err := os.ReadFile(filepath.Join(knowledgeDir, "git-experts.json")); err == nil {
		var experts []git.DirectoryExpert
		if json.Unmarshal(expertsData, &experts) == nil {
			for _, de := range experts {
				for _, exp := range de.TopExperts {
					status := "inactive"
					if exp.Active {
						status = "active"
					}
					docs = append(docs, docEntry{
						id:      fmt.Sprintf("expert:%s:%s", de.Directory, exp.Email),
						docType: "git_expert",
						text: fmt.Sprintf("Expert: %s in %s with %.0f%% ownership, %d commits, %s",
							exp.Name, de.Directory, exp.Ownership*100, exp.Commits, status),
					})
				}
			}
		}
	}

	// Collect git risk knowledge for vectorization
	if risksData, err := os.ReadFile(filepath.Join(knowledgeDir, "git-risks.json")); err == nil {
		var risks []git.KnowledgeRisk
		if json.Unmarshal(risksData, &risks) == nil {
			for i, risk := range risks {
				docs = append(docs, docEntry{
					id:      fmt.Sprintf("risk:%s:%d", risk.Area, i),
					docType: "git_risk",
					text: fmt.Sprintf("Risk: %s in %s - %s. Primary expert: %s",
						risk.RiskLevel, risk.Area, risk.Reason, risk.PrimaryExpert),
				})
			}
		}
	}

	// Collect conversations for vectorization
	allConvs, _ := m.jsonStore.GetAllConversations()
	for _, conv := range allConvs {
		text := conv.Summary
		for _, kp := range conv.KeyPoints {
			text += " " + kp
		}
		for _, f := range conv.FilesDiscussed {
			text += " " + f
		}
		docs = append(docs, docEntry{id: conv.ID, docType: "conversation", text: text})
	}

	if len(docs) == 0 {
		return 0, nil
	}

	// Build corpus text for vocabulary
	corpus := make([]string, len(docs))
	for i, d := range docs {
		corpus[i] = d.text
	}

	// Build TF-IDF engine
	engine := search.NewTFIDFEngine()
	engine.BuildVocabulary(corpus)

	// Store vocabulary
	if err := m.sqliteIndex.StoreVocab(engine); err != nil {
		return 0, fmt.Errorf("storing vocab: %w", err)
	}

	// Vectorize and store each document
	stored := 0
	for _, d := range docs {
		vec := engine.Vectorize(d.text)
		if vec == nil {
			continue
		}
		packed := search.PackVector(vec)
		if err := m.sqliteIndex.StoreSemanticVector(d.id, d.docType, d.text, packed); err != nil {
			continue
		}
		stored++
	}

	return stored, nil
}

// Note: extToLanguage is defined earlier in this file (around line 431)
