# Contributing to TeamContext

Thank you for your interest in improving the project! This guide covers the technical landscape for those developing **on** TeamContext itself.

---

## 🛠 Build System

The project uses a standard Go toolchain with a `Makefile` for convenience.

```bash
make build          # Local build (binary: ./teamcontext)
make test           # Run suite (unit + integration)
make lint           # Vet + Staticcheck
make install-system # Build and install to /usr/local/bin
```

---

## ⚙️ Core Architecture

- **`internal/mcp`**: The JSON-RPC server layer. This is where tools are registered and handled.
- **`internal/storage`**: Dual-layer storage. JSON (Source of Truth) + SQLite (FTS5/TF-IDF caching).
- **`internal/extractor`**: Static analysis for "High Impact" data like APIs and DB Models.
- **`internal/skeleton`**: The cross-language parser that computes code structure without intent (token-efficient).
- **`internal/search`**: Ripgrep integration for code search and the custom TFIDF engine for semantic search.
- **`internal/worker`**: Background goroutines for Git watching and incremental reindexing.

---

## 🚀 Extending TeamContext

### 1. Adding a New Tool
1. Define the handler in `internal/mcp/` (e.g., `tools_analysis.go`).
2. Register it in `internal/mcp/server.go` in the `registerTools()` function.
3. Add input schema documentation in `handleToolsList` (found in `internal/mcp/tools_index.go`).

### 2. Supporting a New Language
1. Update `internal/skeleton/parser.go` with regex patterns for the new language's classes and functions.
2. Add file extension mapping in `detectLanguageFromPath`.

### 3. Adding a New Framework
1. Update `internal/extractor/api.go` with the framework's routing patterns.
2. Add framework detection logic (lookup in `package.json`, `go.mod`, etc.).

---

## 🎨 Design Principles

1.  **Safety First**: Always use atomic writes for storage (write to `.tmp`, then rename).
2.  **Token Efficiency**: Tools should return the absolute minimum data required for the AI to understand the context.
3.  **Local Only**: No tool should ever require an external API key or network access for its core logic.
