package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/saeedalam/teamcontext/internal/search"
	"github.com/saeedalam/teamcontext/pkg/types"
)

// SQLiteIndex provides full-text search capabilities
type SQLiteIndex struct {
	db       *sql.DB
	basePath string
}

// NewSQLiteIndex creates a new SQLite index
func NewSQLiteIndex(basePath string) (*SQLiteIndex, error) {
	dbPath := filepath.Join(basePath, "cache", "index.db")

	// Ensure cache directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	// Add connection parameters for better concurrency
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=10000", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Set connection limits
	db.SetMaxOpenConns(1) // Keep it simple for now to avoid locks

	idx := &SQLiteIndex{
		db:       db,
		basePath: basePath,
	}

	if err := idx.createTables(); err != nil {
		return nil, err
	}

	return idx, nil
}

func (idx *SQLiteIndex) createTables() error {
	schema := `
	-- Files table
	CREATE TABLE IF NOT EXISTS files (
		path TEXT PRIMARY KEY,
		summary TEXT,
		exports TEXT,
		imports TEXT,
		language TEXT,
		patterns TEXT,
		content_hash TEXT,
		indexed_at INTEGER
	);

	-- Files FTS
	CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
		path,
		summary,
		exports,
		content='files',
		content_rowid='rowid'
	);

	-- Triggers for FTS sync
	CREATE TRIGGER IF NOT EXISTS files_ai AFTER INSERT ON files BEGIN
		INSERT INTO files_fts(rowid, path, summary, exports)
		VALUES (new.rowid, new.path, new.summary, new.exports);
	END;

	CREATE TRIGGER IF NOT EXISTS files_ad AFTER DELETE ON files BEGIN
		INSERT INTO files_fts(files_fts, rowid, path, summary, exports)
		VALUES('delete', old.rowid, old.path, old.summary, old.exports);
	END;

	CREATE TRIGGER IF NOT EXISTS files_au AFTER UPDATE ON files BEGIN
		INSERT INTO files_fts(files_fts, rowid, path, summary, exports)
		VALUES('delete', old.rowid, old.path, old.summary, old.exports);
		INSERT INTO files_fts(rowid, path, summary, exports)
		VALUES (new.rowid, new.path, new.summary, new.exports);
	END;

	-- Decisions table
	CREATE TABLE IF NOT EXISTS decisions (
		id TEXT PRIMARY KEY,
		content TEXT,
		reason TEXT,
		context TEXT,
		feature TEXT,
		status TEXT,
		created_at INTEGER
	);

	-- Decisions FTS
	CREATE VIRTUAL TABLE IF NOT EXISTS decisions_fts USING fts5(
		content,
		reason,
		context,
		content='decisions',
		content_rowid='rowid'
	);

	-- Triggers for decisions FTS sync
	CREATE TRIGGER IF NOT EXISTS decisions_ai AFTER INSERT ON decisions BEGIN
		INSERT INTO decisions_fts(rowid, content, reason, context)
		VALUES (new.rowid, new.content, new.reason, new.context);
	END;

	CREATE TRIGGER IF NOT EXISTS decisions_ad AFTER DELETE ON decisions BEGIN
		INSERT INTO decisions_fts(decisions_fts, rowid, content, reason, context)
		VALUES('delete', old.rowid, old.content, old.reason, old.context);
	END;

	-- Warnings table
	CREATE TABLE IF NOT EXISTS warnings (
		id TEXT PRIMARY KEY,
		content TEXT,
		reason TEXT,
		evidence TEXT,
		severity TEXT,
		feature TEXT,
		created_at INTEGER
	);

	-- Warnings FTS
	CREATE VIRTUAL TABLE IF NOT EXISTS warnings_fts USING fts5(
		content,
		reason,
		evidence,
		content='warnings',
		content_rowid='rowid'
	);

	-- Triggers for warnings FTS sync
	CREATE TRIGGER IF NOT EXISTS warnings_ai AFTER INSERT ON warnings BEGIN
		INSERT INTO warnings_fts(rowid, content, reason, evidence)
		VALUES (new.rowid, new.content, new.reason, new.evidence);
	END;

	CREATE TRIGGER IF NOT EXISTS warnings_ad AFTER DELETE ON warnings BEGIN
		INSERT INTO warnings_fts(warnings_fts, rowid, content, reason, evidence)
		VALUES('delete', old.rowid, old.content, old.reason, old.evidence);
	END;

	-- Features table
	CREATE TABLE IF NOT EXISTS features (
		id TEXT PRIMARY KEY,
		status TEXT,
		current_state TEXT,
		relevant_files TEXT,
		created_at INTEGER,
		last_accessed INTEGER
	);

	-- Code chunks table for actual code content search
	CREATE TABLE IF NOT EXISTS code_chunks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT NOT NULL,
		chunk_type TEXT,
		chunk_name TEXT,
		start_line INTEGER,
		end_line INTEGER,
		content TEXT,
		language TEXT,
		indexed_at INTEGER
	);

	-- Code chunks FTS for full-text code search
	CREATE VIRTUAL TABLE IF NOT EXISTS code_chunks_fts USING fts5(
		file_path,
		chunk_name,
		content,
		content='code_chunks',
		content_rowid='id'
	);

	-- Triggers for code chunks FTS sync
	CREATE TRIGGER IF NOT EXISTS code_chunks_ai AFTER INSERT ON code_chunks BEGIN
		INSERT INTO code_chunks_fts(rowid, file_path, chunk_name, content)
		VALUES (new.id, new.file_path, new.chunk_name, new.content);
	END;

	CREATE TRIGGER IF NOT EXISTS code_chunks_ad AFTER DELETE ON code_chunks BEGIN
		INSERT INTO code_chunks_fts(code_chunks_fts, rowid, file_path, chunk_name, content)
		VALUES('delete', old.id, old.file_path, old.chunk_name, old.content);
	END;

	CREATE TRIGGER IF NOT EXISTS code_chunks_au AFTER UPDATE ON code_chunks BEGIN
		INSERT INTO code_chunks_fts(code_chunks_fts, rowid, file_path, chunk_name, content)
		VALUES('delete', old.id, old.file_path, old.chunk_name, old.content);
		INSERT INTO code_chunks_fts(rowid, file_path, chunk_name, content)
		VALUES (new.id, new.file_path, new.chunk_name, new.content);
	END;

	-- Index for faster file_path lookups
	CREATE INDEX IF NOT EXISTS idx_code_chunks_file ON code_chunks(file_path);

	-- Semantic vectors for TF-IDF search
	CREATE TABLE IF NOT EXISTS semantic_vectors (
		id TEXT PRIMARY KEY,
		doc_type TEXT,
		doc_text TEXT,
		vector BLOB,
		updated_at INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_semantic_doc_type ON semantic_vectors(doc_type);

	-- Semantic vocabulary (single row)
	CREATE TABLE IF NOT EXISTS semantic_vocab (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		vocabulary TEXT,
		idf TEXT,
		doc_count INTEGER
	);
	`

	_, err := idx.db.Exec(schema)
	return err
}

// Close closes the database connection
func (idx *SQLiteIndex) Close() error {
	return idx.db.Close()
}

type queryer interface {
	Exec(query string, args ...any) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// WithTransaction runs a function within a SQLite transaction
func (idx *SQLiteIndex) WithTransaction(fn func(tx *sql.Tx) error) error {
	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// --- Indexing ---

// IndexFile indexes a file for search
func (idx *SQLiteIndex) IndexFile(file *types.FileIndex) error {
	return idx.indexFile(idx.db, file)
}

// IndexFileTx indexes a file within a transaction
func (idx *SQLiteIndex) IndexFileTx(tx *sql.Tx, file *types.FileIndex) error {
	return idx.indexFile(tx, file)
}

func (idx *SQLiteIndex) indexFile(q queryer, file *types.FileIndex) error {
	exportsJSON, _ := json.Marshal(file.Exports)
	importsJSON, _ := json.Marshal(file.Imports)
	patternsJSON, _ := json.Marshal(file.Patterns)

	_, err := q.Exec(`
		INSERT OR REPLACE INTO files (path, summary, exports, imports, language, patterns, content_hash, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, file.Path, file.Summary, string(exportsJSON), string(importsJSON),
		file.Language, string(patternsJSON), file.ContentHash, file.IndexedAt.Unix())

	return err
}

// IndexDecision indexes a decision for search
func (idx *SQLiteIndex) IndexDecision(dec *types.Decision) error {
	_, err := idx.db.Exec(`
		INSERT OR REPLACE INTO decisions (id, content, reason, context, feature, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, dec.ID, dec.Content, dec.Reason, dec.Context, dec.Feature, dec.Status, dec.CreatedAt.Unix())

	return err
}

// IndexWarning indexes a warning for search
func (idx *SQLiteIndex) IndexWarning(warn *types.Warning) error {
	_, err := idx.db.Exec(`
		INSERT OR REPLACE INTO warnings (id, content, reason, evidence, severity, feature, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, warn.ID, warn.Content, warn.Reason, warn.Evidence, warn.Severity, warn.Feature, warn.CreatedAt.Unix())

	return err
}

// IndexFeature indexes a feature for search
func (idx *SQLiteIndex) IndexFeature(feat *types.Feature) error {
	filesJSON, _ := json.Marshal(feat.RelevantFiles)

	_, err := idx.db.Exec(`
		INSERT OR REPLACE INTO features (id, status, current_state, relevant_files, created_at, last_accessed)
		VALUES (?, ?, ?, ?, ?, ?)
	`, feat.ID, feat.Status, feat.CurrentState, string(filesJSON), feat.CreatedAt.Unix(), feat.LastAccessed.Unix())

	return err
}

// --- Search ---

// SearchFiles searches files by query
func (idx *SQLiteIndex) SearchFiles(query string, language string, limit int) ([]types.FileIndex, error) {
	if limit <= 0 {
		limit = 20
	}

	var args []interface{}
	var whereClause string

	if query != "" {
		// Use FTS5 for text search
		whereClause = `WHERE files.rowid IN (
			SELECT rowid FROM files_fts WHERE files_fts MATCH ?
		)`
		args = append(args, query)
	}

	if language != "" {
		if whereClause == "" {
			whereClause = "WHERE language = ?"
		} else {
			whereClause += " AND language = ?"
		}
		args = append(args, language)
	}

	sql := fmt.Sprintf(`
		SELECT path, summary, exports, imports, language, patterns, content_hash, indexed_at
		FROM files
		%s
		ORDER BY indexed_at DESC
		LIMIT ?
	`, whereClause)
	args = append(args, limit)

	rows, err := idx.db.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []types.FileIndex
	for rows.Next() {
		var f types.FileIndex
		var exportsJSON, importsJSON, patternsJSON string
		var indexedAt int64

		err := rows.Scan(&f.Path, &f.Summary, &exportsJSON, &importsJSON,
			&f.Language, &patternsJSON, &f.ContentHash, &indexedAt)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(exportsJSON), &f.Exports)
		json.Unmarshal([]byte(importsJSON), &f.Imports)
		json.Unmarshal([]byte(patternsJSON), &f.Patterns)

		files = append(files, f)
	}

	return files, nil
}

// SearchDecisions searches decisions by query
func (idx *SQLiteIndex) SearchDecisions(query string, feature string, status string, limit int) ([]types.Decision, error) {
	if limit <= 0 {
		limit = 50
	}

	var conditions []string
	var args []interface{}

	if query != "" {
		conditions = append(conditions, `decisions.rowid IN (
			SELECT rowid FROM decisions_fts WHERE decisions_fts MATCH ?
		)`)
		args = append(args, query)
	}

	if feature != "" {
		conditions = append(conditions, "feature = ?")
		args = append(args, feature)
	}

	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	sql := fmt.Sprintf(`
		SELECT id, content, reason, context, feature, status, created_at
		FROM decisions
		%s
		ORDER BY created_at DESC
		LIMIT ?
	`, whereClause)
	args = append(args, limit)

	rows, err := idx.db.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []types.Decision
	for rows.Next() {
		var d types.Decision
		var createdAt int64

		err := rows.Scan(&d.ID, &d.Content, &d.Reason, &d.Context, &d.Feature, &d.Status, &createdAt)
		if err != nil {
			continue
		}

		decisions = append(decisions, d)
	}

	return decisions, nil
}

// SearchWarnings searches warnings by query
func (idx *SQLiteIndex) SearchWarnings(query string, feature string, severity string, limit int) ([]types.Warning, error) {
	if limit <= 0 {
		limit = 50
	}

	var conditions []string
	var args []interface{}

	if query != "" {
		conditions = append(conditions, `warnings.rowid IN (
			SELECT rowid FROM warnings_fts WHERE warnings_fts MATCH ?
		)`)
		args = append(args, query)
	}

	if feature != "" {
		conditions = append(conditions, "feature = ?")
		args = append(args, feature)
	}

	if severity != "" {
		conditions = append(conditions, "severity = ?")
		args = append(args, severity)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	sql := fmt.Sprintf(`
		SELECT id, content, reason, evidence, severity, feature, created_at
		FROM warnings
		%s
		ORDER BY created_at DESC
		LIMIT ?
	`, whereClause)
	args = append(args, limit)

	rows, err := idx.db.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var warnings []types.Warning
	for rows.Next() {
		var w types.Warning
		var createdAt int64

		err := rows.Scan(&w.ID, &w.Content, &w.Reason, &w.Evidence, &w.Severity, &w.Feature, &createdAt)
		if err != nil {
			continue
		}

		warnings = append(warnings, w)
	}

	return warnings, nil
}

// --- Rebuild ---

// RebuildFromJSON rebuilds the SQLite index from JSON files
func (idx *SQLiteIndex) RebuildFromJSON(jsonStore *JSONStore) error {
	// Clear existing data
	tables := []string{"files", "decisions", "warnings", "features"}
	for _, table := range tables {
		idx.db.Exec("DELETE FROM " + table)
	}

	// Rebuild files
	files, err := jsonStore.GetFilesIndex()
	if err == nil {
		for _, f := range files {
			idx.IndexFile(&f)
		}
	}

	// Rebuild decisions
	decisions, err := jsonStore.GetDecisions()
	if err == nil {
		for _, d := range decisions {
			idx.IndexDecision(&d)
		}
	}

	// Rebuild warnings
	warnings, err := jsonStore.GetWarnings()
	if err == nil {
		for _, w := range warnings {
			idx.IndexWarning(&w)
		}
	}

	// Rebuild features
	features, err := jsonStore.GetFeatures()
	if err == nil {
		for _, f := range features {
			idx.IndexFeature(&f)
		}
	}

	return nil
}

// --- Stats ---

// GetStats returns index statistics
func (idx *SQLiteIndex) GetStats() (map[string]int, error) {
	stats := make(map[string]int)

	tables := []string{"files", "decisions", "warnings", "features", "code_chunks"}
	for _, table := range tables {
		var count int
		err := idx.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err == nil {
			stats[table] = count
		}
	}

	return stats, nil
}

// --- Code Content Indexing ---

// CodeChunk represents a searchable piece of code
type CodeChunk struct {
	ID        int64  `json:"id"`
	FilePath  string `json:"file_path"`
	ChunkType string `json:"chunk_type"` // "function", "class", "block", "lines"
	ChunkName string `json:"chunk_name"` // function/class name or "lines:10-50"
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Content   string `json:"content"`
	Language  string `json:"language"`
}

// IndexCodeChunks indexes code chunks for a file (replaces existing chunks for that file)
func (idx *SQLiteIndex) IndexCodeChunks(filePath string, chunks []CodeChunk) error {
	return idx.indexCodeChunks(idx.db, filePath, chunks)
}

// IndexCodeChunksTx indexes code chunks within a transaction
func (idx *SQLiteIndex) IndexCodeChunksTx(tx *sql.Tx, filePath string, chunks []CodeChunk) error {
	return idx.indexCodeChunks(tx, filePath, chunks)
}

func (idx *SQLiteIndex) indexCodeChunks(q queryer, filePath string, chunks []CodeChunk) error {
	// Delete existing chunks for this file
	_, err := q.Exec("DELETE FROM code_chunks WHERE file_path = ?", filePath)
	if err != nil {
		return err
	}

	// Insert new chunks
	stmt, err := q.Prepare(`
		INSERT INTO code_chunks (file_path, chunk_type, chunk_name, start_line, end_line, content, language, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := nowUnix()
	for _, chunk := range chunks {
		_, err := stmt.Exec(chunk.FilePath, chunk.ChunkType, chunk.ChunkName,
			chunk.StartLine, chunk.EndLine, chunk.Content, chunk.Language, now)
		if err != nil {
			return err
		}
	}

	return nil
}

// SearchCodeContent searches actual code content using FTS5
func (idx *SQLiteIndex) SearchCodeContent(query string, language string, limit int) ([]CodeChunk, error) {
	if limit <= 0 {
		limit = 20
	}

	var args []interface{}
	var conditions []string

	// FTS5 search
	if query != "" {
		conditions = append(conditions, `code_chunks.id IN (
			SELECT rowid FROM code_chunks_fts WHERE code_chunks_fts MATCH ?
		)`)
		args = append(args, query)
	}

	if language != "" {
		conditions = append(conditions, "language = ?")
		args = append(args, language)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	sql := fmt.Sprintf(`
		SELECT id, file_path, chunk_type, chunk_name, start_line, end_line, content, language
		FROM code_chunks
		%s
		ORDER BY indexed_at DESC
		LIMIT ?
	`, whereClause)
	args = append(args, limit)

	rows, err := idx.db.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []CodeChunk
	for rows.Next() {
		var c CodeChunk
		err := rows.Scan(&c.ID, &c.FilePath, &c.ChunkType, &c.ChunkName,
			&c.StartLine, &c.EndLine, &c.Content, &c.Language)
		if err != nil {
			continue
		}
		chunks = append(chunks, c)
	}

	return chunks, nil
}

// GetCodeChunksForFile returns all code chunks for a specific file
func (idx *SQLiteIndex) GetCodeChunksForFile(filePath string) ([]CodeChunk, error) {
	rows, err := idx.db.Query(`
		SELECT id, file_path, chunk_type, chunk_name, start_line, end_line, content, language
		FROM code_chunks
		WHERE file_path = ?
		ORDER BY start_line
	`, filePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []CodeChunk
	for rows.Next() {
		var c CodeChunk
		err := rows.Scan(&c.ID, &c.FilePath, &c.ChunkType, &c.ChunkName,
			&c.StartLine, &c.EndLine, &c.Content, &c.Language)
		if err != nil {
			continue
		}
		chunks = append(chunks, c)
	}

	return chunks, nil
}

// DeleteCodeChunksForFile removes all code chunks for a file
func (idx *SQLiteIndex) DeleteCodeChunksForFile(filePath string) error {
	_, err := idx.db.Exec("DELETE FROM code_chunks WHERE file_path = ?", filePath)
	return err
}

func nowUnix() int64 {
	return time.Now().Unix()
}

// --- Semantic Search ---

// StoreSemanticVector stores a document's TF-IDF vector for semantic search.
func (idx *SQLiteIndex) StoreSemanticVector(id, docType, text string, vector []byte) error {
	_, err := idx.db.Exec(`
		INSERT OR REPLACE INTO semantic_vectors (id, doc_type, doc_text, vector, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, docType, text, vector, nowUnix())
	return err
}

// SearchSemantic loads all vectors of the given type, computes cosine similarity, and returns top-N results.
// Uses brute-force scan with early termination. For corpora exceeding ~10K documents,
// consider replacing with an approximate nearest-neighbor index (e.g. HNSW or IVF).
func (idx *SQLiteIndex) SearchSemantic(queryVector []float64, docType string, limit int) ([]search.SemanticResult, error) {
	if limit <= 0 {
		limit = 10
	}

	var args []interface{}
	query := "SELECT id, doc_type, doc_text, vector FROM semantic_vectors"
	if docType != "" {
		query += " WHERE doc_type = ?"
		args = append(args, docType)
	}

	rows, err := idx.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []search.SemanticResult
	highQualityCount := 0 // tracks results with similarity > 0.6
	for rows.Next() {
		var id, dt, text string
		var vectorBlob []byte
		if err := rows.Scan(&id, &dt, &text, &vectorBlob); err != nil {
			continue
		}

		docVector := search.UnpackVector(vectorBlob)
		if docVector == nil {
			continue
		}

		similarity := search.CosineSimilarity(queryVector, docVector)
		if similarity > 0.15 { // minimum threshold
			results = append(results, search.SemanticResult{
				ID:         id,
				DocType:    dt,
				Text:       text,
				Similarity: similarity,
			})
			if similarity > 0.6 {
				highQualityCount++
			}
		}

		// Early termination: stop scanning when we have enough high-quality results
		if highQualityCount >= limit && len(results) >= limit*3 {
			break
		}
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// StoreVocab persists the TF-IDF engine's vocabulary and IDF to SQLite.
func (idx *SQLiteIndex) StoreVocab(engine *search.TFIDFEngine) error {
	vocabJSON, err := json.Marshal(engine.Vocabulary)
	if err != nil {
		return err
	}
	idfJSON, err := json.Marshal(engine.IDF)
	if err != nil {
		return err
	}

	_, err = idx.db.Exec(`
		INSERT OR REPLACE INTO semantic_vocab (id, vocabulary, idf, doc_count)
		VALUES (1, ?, ?, ?)
	`, string(vocabJSON), string(idfJSON), engine.DocCount)
	return err
}

// LoadVocab loads the TF-IDF engine from SQLite.
func (idx *SQLiteIndex) LoadVocab() (*search.TFIDFEngine, error) {
	var vocabStr, idfStr string
	var docCount int

	err := idx.db.QueryRow("SELECT vocabulary, idf, doc_count FROM semantic_vocab WHERE id = 1").
		Scan(&vocabStr, &idfStr, &docCount)
	if err != nil {
		return nil, err
	}

	engine := search.NewTFIDFEngine()
	engine.DocCount = docCount

	if err := json.Unmarshal([]byte(vocabStr), &engine.Vocabulary); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(idfStr), &engine.IDF); err != nil {
		return nil, err
	}

	return engine, nil
}

// Ensure types is used (compile guard)
var _ = types.Decision{}
