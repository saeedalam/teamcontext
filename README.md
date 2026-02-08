# TeamContext

**The Technical Memory That Outlives Everyone**

TeamContext is a Go-based MCP server that captures, indexes, and serves everything your team knows about your codebase. It mines Git history, tracks decisions, maps dependencies, and provides token-efficient code analysis - so AI tools and developers never start from zero.

**TeamContext does NOT think** - it remembers. The IDE agent does all intelligence.

---

### 📖 Documentation Structure
*   **[Master Developer Guide](DOCS/MASTER_DEVELOPER_GUIDE.md)**: Your primary entry point for elite workflows and features.
*   **[DOCS/](DOCS/)**: Deep reference material, tool schemas, and scenario tutorials.

---

## Quick Start

```bash
# 1. Build
git clone git@github.com:saeedalam/teamcontext.git && cd teamcontext
go build -o teamcontext ./cmd/teamcontext/
sudo mv teamcontext /usr/local/bin/

# 2. Initialize in your project
cd /path/to/your/project
teamcontext init

# 3. Restart your IDE (Cursor, VS Code, Windsurf)
# TeamContext is ready - ask your AI: "What TeamContext tools are available?"
```

That's it. `teamcontext init` automatically:
- Creates `.teamcontext/` knowledge store
- Indexes all code files (skeletons, imports, dependency graph)
- Configures your IDE with MCP settings
- Starts background workers for git-connected auto-indexing

---

## Installation

### Prerequisites
- Go 1.22+
- Git (for history analysis)
- ripgrep (optional, for code search)

### Build & Install

```bash
cd teamcontext

# Option 1: Makefile (recommended)
make build                    # Build for current platform
make install-system           # Build + copy to /usr/local/bin
make release                  # Cross-compile for all platforms

# Option 2: Go directly
go build -o teamcontext ./cmd/teamcontext/
sudo mv teamcontext /usr/local/bin/        # Linux/Mac
go install ./cmd/teamcontext/               # Go way (adds to $GOPATH/bin)
```

### Version

```bash
teamcontext version             # Shows version, commit, build date
```

### IDE Setup

`teamcontext init` auto-configures these IDEs:

| IDE | Config Created | Notes |
|-----|---------------|-------|
| **Cursor** | `.cursor/mcp.json` | Project-level, auto-detected |
| **VS Code** | `.vscode/mcp.json` | Project-level, auto-detected |
| **Windsurf** | `.windsurf/mcp.json` | Project-level, auto-detected |
| **Claude Desktop** | Manual | Run `teamcontext install claude --global` |

Manual install/uninstall:
```bash
teamcontext install cursor            # Project-level
teamcontext install claude --global   # Global (Claude Desktop)
teamcontext uninstall cursor          # Remove
```

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `teamcontext init` | Initialize in project (index + IDE setup) |
| `teamcontext index` | Re-index all project files |
| `teamcontext serve` | Start MCP server (for IDE integration) |
| `teamcontext status` | Show current state |
| `teamcontext stats` | Show detailed statistics |
| `teamcontext start <name>` | Start a feature context |
| `teamcontext list` | List all features |
| `teamcontext archive <name>` | Archive a completed feature |
| `teamcontext resume <name>` | Resume a feature |
| `teamcontext recall <name>` | Recall from archive |
| `teamcontext search <query>` | Search knowledge base |
| `teamcontext rebuild` | Rebuild SQLite from JSON |
| `teamcontext install <ide>` | Configure IDE manually |
| `teamcontext uninstall <ide>` | Remove from IDE |
| `teamcontext generate-rules` | Generate .cursorrules / CLAUDE.md from knowledge |
| `teamcontext install-hooks` | Install git post-commit hook for auto-capture |
| `teamcontext uninstall-hooks` | Remove TeamContext git hooks |
| `teamcontext analyze-commit` | Analyze latest commit (called by hooks) |
| `teamcontext sync` | Sync team knowledge via git (pull + push) |
| `teamcontext feed` | Show recent team knowledge activity |
| `teamcontext version` | Print version, commit, build date |

---

## MCP Tools (54 Total)

### Search & Query (6 tools)

| Tool | What It Does |
|------|-------------|
| `query` | Natural language search across all knowledge |
| `get_context` | Get relevant context for a task/intent |
| `search` | Search decisions, warnings, patterns |
| `search_files` | Search indexed files by name/language |
| `search_code` | Search actual code content with regex |
| `get_related` | Traverse knowledge graph from a node to find connected items |

### Token-Saving Tools (7 tools) - Save 60-95% tokens

| Tool | Savings | What It Does |
|------|---------|-------------|
| `get_skeleton` | ~90% | Code structure without bodies (functions, classes, signatures) |
| `get_types` | ~70% | Type definitions, interfaces, enums only |
| `search_snippets` | ~80% | Search and return only matching code chunks |
| `get_recent_changes` | ~70% | Git history with impact analysis |
| `resume_context` | ~95% | Compressed context from previous sessions |
| `list_conversations` | ~90% | Browse saved conversation history across features |
| `get_task_context` | ~80% | Pre-built context bundle for common tasks |

### High-Impact Extraction (4 tools) - Multi-language

| Tool | Languages | What It Does |
|------|-----------|-------------|
| `get_blueprint` | NestJS, Express, Go/Gin/Echo, Python/FastAPI/Flask/Django, Rust/Actix/Axum | **THE MAGIC TOOL** - Complete task blueprint: file patterns, code snippets, imports, conventions, decisions, warnings, checklist. One call replaces 20+ exploration calls. |
| `get_api_surface` | TS/NestJS, Express, Go, Python/Flask/FastAPI/Django, Java/Spring, C#/ASP.NET | Extract all REST endpoints and Kafka handlers |
| `get_schema_models` | Prisma, Go/GORM, Python/SQLAlchemy/Django, Java/JPA, TS/TypeORM | Extract database models, fields, relations, enums |
| `get_config_map` | All | Extract env vars and config usage across project |

#### `get_blueprint` - Framework Support

```bash
get_blueprint(task: "add-endpoint", app: "my-app")
```

| Framework | Detection | File Patterns | Checklists |
|-----------|-----------|---------------|------------|
| **NestJS** | package.json | controller/service/module/schemas/types | ✅ Full + conventions |
| **Express** | package.json | route/controller/service | ✅ Basic |
| **Go + Gin** | go.mod | handler/service/repository/dto | ✅ Full |
| **Go + Echo** | go.mod | handler/service/repository/model | ✅ Full |
| **FastAPI** | requirements.txt | router/schemas/services/models | ✅ Full |
| **Flask** | requirements.txt | route/service/model | ✅ Basic |
| **Django** | requirements.txt | views/serializers/models/urls | ✅ Full |
| **Actix** | Cargo.toml | handler/service/model/mod | ✅ Full |
| **Axum** | Cargo.toml | handlers/models/router | ✅ Full |

Task types: `add-endpoint`, `add-feature`, `add-service`, `fix-bug`, `refactor`, `add-test`

### Code Analysis (4 tools)

| Tool | What It Does |
|------|-------------|
| `scan_imports` | Scan file/directory imports (TS, Go, Python) |
| `get_code_map` | Hierarchical view of indexed files |
| `get_dependencies` | What a file depends on / what depends on it |
| `trace_flow` | Trace data flow through import chain |

### Git Intelligence (5 tools) - Mine team history

| Tool | What It Does |
|------|-------------|
| `find_experts` | Who knows this code best? Ranked by ownership, activity |
| `get_file_history` | Full expertise analysis: contributors, churn, ownership % |
| `get_knowledge_risks` | Find areas where experts left or knowledge is concentrated |
| `get_file_correlations` | Files that usually change together (prevent incomplete changes) |
| `get_commit_context` | Why does this code exist? Git history for file/lines |

### Compliance, Onboarding & Team (3 tools)

| Tool | What It Does |
|------|-------------|
| `check_compliance` | Validate code against recorded decisions and patterns. Returns violations with severity and references. |
| `onboard` | Structured project walkthrough: architecture, decisions, warnings, patterns, experts, risks. One call for full project understanding. |
| `get_feed` | Recent team activity timeline: decisions, warnings, patterns, conversations. Filter by type, time range, or limit. |

**Cross-repo activity:** Configure `linked_repos` in `.teamcontext/config.json` to track contributor activity across sibling repositories. A contributor marked inactive in repo A will be marked active if they have recent commits in linked repo B, with the `active_in_repo` field indicating where.

```json
{
  "linked_repos": [
    "/path/to/sibling-repo-1",
    "/path/to/sibling-repo-2"
  ]
}
```

### Indexing & Graph (3 tools)

| Tool | What It Does |
|------|-------------|
| `index` | Trigger full project re-index (files, skeletons, imports, graph) |
| `index_status` | Get current index status (files indexed, last run, stale count) |
| `get_graph` | View knowledge graph edges and relationships between all entities |

### Knowledge Management (9 read + 11 write tools)

**Read:**

| Tool | What It Does |
|------|-------------|
| `get_project` | Project overview and metadata |
| `get_feature` | Feature context with decisions, warnings, conversations |
| `list_features` | All features with status |
| `list_decisions` | All architectural decisions |
| `list_warnings` | All known pitfalls |
| `list_patterns` | All established patterns |
| `get_stats` | System statistics |
| `get_architecture` | High-level architecture description |
| `get_evolution_timeline` | How knowledge evolved over time |

**Write:**

| Tool | What It Does |
|------|-------------|
| `index_file` | Index a file (auto-indexes content, skeleton, imports, graph) |
| `add_decision` | Record an architectural decision with reasoning |
| `add_warning` | Record a pitfall/gotcha to avoid |
| `add_insight` | Record a discovered behavior or pattern |
| `add_pattern` | Define an established pattern |
| `add_evolution_event` | Add a milestone or event |
| `save_conversation` | Save compressed conversation memory |
| `compact_conversation` | Compact long conversation into dense summary |
| `update_feature_state` | Update feature progress |
| `update_architecture` | Update architecture description |
| `update_project` | Update project metadata |

**Feature Lifecycle (3 tools):**
`start_feature`, `archive_feature`, `recall_feature`

**Auto-Capture Conversations:**
Sessions are automatically checkpointed every 25 tool calls, after knowledge-creation events (`add_decision`, `add_warning`), and on feature lifecycle changes. No AI cooperation required.

**Graph:**
`get_graph` and `get_related` — see Indexing & Graph and Search & Query sections above.

---

## Intelligence Features

TeamContext includes 5 intelligence features that make context retrieval smarter.

### 1. TF-IDF Semantic Search (Enhanced)

Keyword search (`query`, `search`) now includes **semantic matching** alongside FTS5 full-text search. This means searching for "authentication flow" will also find documents about "login process" — without requiring an LLM or external API.

**How it works:**
- During `teamcontext init`, a TF-IDF vocabulary is built from all decisions, warnings, patterns, insights, file summaries, **git experts, and git risks**
- Vocabulary selection uses **IDF-based ranking** — rare-but-not-unique terms are kept (most discriminative), while common terms and hapax legomena are filtered out
- **Stemming** strips common suffixes (`-ation`, `-tion`, `-ing`, `-ed`, `-ly`, etc.) so "authentication" and "authenticat" match
- **Synonym expansion** maps ~30 common abbreviations (`auth→authentication`, `db→database`, `api→endpoint`, `config→configuration`, etc.)
- Each document is vectorized and stored as a compressed sparse vector in SQLite
- Similarity threshold is **0.15** (filters noise), with **early termination** when enough high-quality results are found

**Usage:** No changes needed — `query` and `search` automatically use semantic search when the index is available. New decisions/warnings/patterns/insights are automatically vectorized when added.

```
# These will find semantically related results, not just exact keyword matches
query: "authentication flow"     → finds "login process", "auth middleware", etc.
query: "error handling strategy" → finds "exception management", "fault tolerance", etc.
query: "who knows the API?"      → finds git experts with ownership in api/ directories
query: "risky areas"             → finds git risk entries for knowledge concentration
```

**Rebuilding the index:** Run `teamcontext init` or `teamcontext rebuild` to regenerate the semantic index.

### 2. Smart Context Loading (Token Budgeting)

`get_context` now accepts a `max_tokens` parameter and returns results ranked by relevance, fitting within your token budget.

**How it works:**
1. All matching context items (decisions, warnings, patterns, files) are scored:
   - File path match to `target_files`: **1.0**
   - Critical warnings: **+0.5 boost**
   - Feature match: **0.9**
   - Knowledge graph connection: **0.7**
   - Ancestor feature context: **0.6**
   - Semantic similarity: **0.0–1.0**
   - Keyword match: **0.5**
2. Items are sorted by score descending
3. Items are added until the token budget is filled (estimated at `len(text)/4`)
4. Response includes a `token_budget` field showing what fit

**Usage:**
```json
{
  "tool": "get_context",
  "arguments": {
    "intent": "fix authentication bug",
    "target_files": ["src/auth/login.ts"],
    "max_tokens": 4000
  }
}
```

**Response includes:**
```json
{
  "token_budget": {
    "requested": 4000,
    "used": 3800,
    "remaining": 200
  },
  "decisions": [...],
  "warnings": [...],
  "patterns": [...]
}
```

### 3. Evolution Timeline (Auto-Capture)

Every write operation now automatically records an evolution event. Previously only `add_decision` and `add_warning` did this.

**Auto-captured events:**

| Action | Event Type | Impact |
|--------|-----------|--------|
| `add_pattern` | `pattern_adopted` | medium |
| `add_insight` | `insight` | low |
| `update_architecture` | `architecture_change` | high |
| `start_feature` | `milestone` | medium |
| `archive_feature` | `milestone` | medium |
| `update_project` | `architecture_change` | medium |
| `add_decision` | `decision` | high |
| `add_warning` | `warning` | medium |

**Usage:** View the timeline with `get_evolution_timeline`. No action needed — events are recorded automatically.

### 4. Knowledge Graph Traversal (`get_related`)

New tool that performs BFS traversal of the knowledge graph starting from any node.

**Usage:**
```json
{
  "tool": "get_related",
  "arguments": {
    "node_type": "decision",
    "node_id": "dec-001",
    "max_depth": 2
  }
}
```

**Parameters:**
- `node_type` (required): `"decision"`, `"warning"`, `"pattern"`, `"file"`, `"feature"`
- `node_id` (required): The ID of the starting node
- `max_depth` (optional, default 2): How many hops to traverse (max 5)

**Response:**
```json
{
  "start": { "type": "decision", "id": "dec-001" },
  "depth": 2,
  "edges_found": 4,
  "connected": {
    "decisions": ["dec-003"],
    "warnings": ["warn-002"],
    "files": ["src/auth/login.ts", "src/middleware/jwt.ts"],
    "patterns": ["pat-001"]
  },
  "edges": [...]
}
```

### 5. Auto-Capture Conversations (Server-Side)

Sessions are automatically checkpointed without any AI cooperation. The MCP server tracks tool calls and triggers saves:

| Trigger | Condition | Threshold |
|---------|-----------|-----------|
| **Checkpoint** | Every N tool calls | 25 calls since last save |
| **Knowledge Created** | After `add_decision` or `add_warning` | 5+ calls accumulated |
| **Feature Lifecycle** | After `start_feature` or `archive_feature` | 3+ calls accumulated |
| **Session End** | Server shutdown | 5+ total calls |

**What's captured:** Session summary, tool usage, files touched, decisions made, features started/archived. Each checkpoint is semantically indexed for search.

**No save instruction in responses:** The previous approach appended ~200 tokens to every tool response asking the AI to save. This is removed — auto-capture handles it server-side.

### 6. Compliance Checking (`check_compliance`)

Validate code against all recorded decisions and patterns before committing.

**Usage:**
```json
{
  "tool": "check_compliance",
  "arguments": {
    "file_path": "src/auth/session.ts"
  }
}
```

**Response:**
```json
{
  "compliant": false,
  "blockers": 1,
  "violations": [
    {
      "type": "decision",
      "id": "dec-007",
      "severity": "blocker",
      "message": "Code contains 'localStorage' which conflicts with decision: Ban localStorage for tokens",
      "reference": "Ban localStorage for tokens — Security: use httpOnly cookies"
    }
  ]
}
```

**What it checks:**
- Prohibited terms from decisions ("avoid X", "don't use Y", "ban Z")
- Related-file governance (decisions that mention specific files)
- Warning pattern matches (critical/warning severity items)
- Pattern anti-patterns (things that violate established conventions)

### 7. IDE Agent Rules Generation

Generate `.cursorrules`, `CLAUDE.md`, or `.windsurfrules` from your knowledge base:

```bash
teamcontext generate-rules                  # .cursorrules (default)
teamcontext generate-rules --format claude  # CLAUDE.md
teamcontext generate-rules --format all     # All formats
```

The generated rules instruct the IDE agent to:
1. Call `get_context` before writing code
2. Respect all recorded decisions
3. Heed all active warnings
4. Follow established patterns
5. Auto-capture decisions and warnings from conversations
6. Mention code experts when modifying their files

Regenerate after adding new knowledge to keep rules current.

### 8. Git Commit Auto-Analysis (Auto-Capture)

Install a post-commit hook to auto-extract knowledge from every commit:

```bash
teamcontext install-hooks
```

**Decisions are auto-captured from commit messages containing:**
```
Decision: Use Zod instead of class-validator
Why: Better type inference and smaller bundle size

# Or conventional commits with breaking change:
feat!: migrate validation to Zod

# Or in PR descriptions (merged via GitHub):
## Decisions
- Switched from class-validator to Zod for validation
```

**Warnings are auto-captured from:**
```
Warning: Don't use class-validator anymore
BREAKING CHANGE: Validation schemas moved to /schemas
Caution: This changes the API response format
```

**Automatic detection (no markers needed):**
- Reverted commits → Warning created
- Large deletions (>100 lines) → Warning created  
- Dependency file changes → Evolution event logged
- Config file changes → Evolution event logged
- TODO/FIXME/HACK additions → Evolution event logged

**PR Description extraction:** If you merge via GitHub and have `gh` CLI installed, TeamContext will also parse PR descriptions for decisions/warnings sections.

Remove with `teamcontext uninstall-hooks`.

### 9. Context Inheritance (`start_feature --extends`)

Features can now inherit context from a parent feature. When you start a feature with `extends`, it automatically copies the parent's relevant files, decisions, and warnings.

**Usage:**
```json
{
  "tool": "start_feature",
  "arguments": {
    "id": "auth-v2",
    "description": "Rewrite authentication with OAuth",
    "extends": "auth-v1"
  }
}
```

**What gets inherited:**
- `relevant_files` from the parent feature
- `decisions` from the parent feature
- `warnings` from the parent feature

**Ancestor chain:** `get_context` also walks the ancestor chain (up to 5 levels) and merges ancestor decisions and warnings into the response, scored at 0.6 relevance.

---

## Architecture

```
                    ┌─────────────────────────────┐
                    │     IDE Agent (Claude)       │
                    │  Reads code, makes decisions │
                    └──────────┬──────────────────┘
                               │ MCP Protocol (JSON-RPC)
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                        TEAMCONTEXT                             │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │  54 MCP      │  │  Background  │  │  Git History     │   │
│  │  Tools       │  │  Workers     │  │  Analyzer        │   │
│  │              │  │              │  │                   │   │
│  │  Query       │  │  Git Watch   │  │  Expertise       │   │
│  │  Extract     │  │  Auto-Index  │  │  Correlations    │   │
│  │  Analyze     │  │  Auto-Graph  │  │  Risk Analysis   │   │
│  │  Store       │  │  Discovery   │  │  Commit Context  │   │
│  └──────┬───────┘  └──────┬───────┘  └────────┬─────────┘   │
│         │                 │                    │              │
│         ▼                 ▼                    ▼              │
│  ┌─────────────────────────────────────────────────────┐     │
│  │              Storage Layer                          │     │
│  │                                                     │     │
│  │  JSON Files (git-tracked)    SQLite FTS5 (cache)   │     │
│  │  ├── decisions.json          ├── Full-text search  │     │
│  │  ├── warnings.json           ├── Code chunks       │     │
│  │  ├── patterns.json           ├── File index        │     │
│  │  ├── graph.json              ├── Semantic vectors  │     │
│  │  ├── files.json              └── TF-IDF vocabulary │     │
│  │  └── features/*.json                               │     │
│  └─────────────────────────────────────────────────────┘     │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐     │
│  │              Extractors (Static Analysis)           │     │
│  │                                                     │     │
│  │  Skeleton   Imports   API Surface   Schema   Config │     │
│  │  12 langs   3 langs   6 frameworks  5 ORMs   All   │     │
│  └─────────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────────┘
```

### Background Workers (Auto, Git-Connected)

| Worker | Interval | What It Does |
|--------|----------|-------------|
| Git Watcher | 30 sec | Detects commits, auto-indexes changed files |
| Auto-Discovery | 10 min | Finds new unindexed files |
| Periodic Reindex | 5 min | Validates and updates stale entries |

On every git change: re-index file, update skeleton, rebuild imports, create graph edges.

### Storage Design

```
JSON (Source of Truth, Git-tracked)  →  SQLite FTS5 + TF-IDF (Search Cache, .gitignored)
```

JSON is the canonical store. SQLite holds both the FTS5 keyword index and TF-IDF semantic vectors. Both are rebuilt from JSON on demand (`teamcontext rebuild`). Team members get shared knowledge through git push/pull.

---

## Project Structure

```
teamcontext/
├── cmd/teamcontext/              # Entry point
│   └── main.go
├── internal/
│   ├── cli/                    # CLI commands (22 commands)
│   │   ├── root.go             # Root command registration
│   │   ├── init.go             # teamcontext init (+ auto-index + IDE setup)
│   │   ├── index.go            # teamcontext index (full project scan)
│   │   ├── install.go          # teamcontext install/uninstall
│   │   ├── serve.go            # teamcontext serve (MCP server)
│   │   ├── status.go           # teamcontext status
│   │   ├── stats.go            # teamcontext stats
│   │   ├── feature.go          # start/list/archive/resume/recall
│   │   ├── rebuild.go          # teamcontext rebuild
│   │   ├── search.go           # teamcontext search
│   │   ├── generate_rules.go   # teamcontext generate-rules
│   │   ├── hooks.go            # teamcontext install-hooks/uninstall-hooks
│   │   ├── analyze_commit.go   # teamcontext analyze-commit
│   │   ├── sync.go             # teamcontext sync (git-based team sync)
│   │   ├── feed.go             # teamcontext feed (activity timeline)
│   │   └── version.go          # teamcontext version
│   ├── mcp/                    # MCP server (54 tools)
│   │   └── server.go           # JSON-RPC handler
│   ├── storage/                # Storage layer
│   │   ├── json.go             # JSON file operations
│   │   └── sqlite.go           # SQLite FTS5 index
│   ├── git/                    # Git utilities
│   │   ├── diff.go             # Git diff/changes
│   │   ├── history.go          # Git history analysis (expertise, risk)
│   │   └── processor.go        # Single-pass git log processor, cross-repo linking
│   ├── extractor/              # Multi-language extractors
│   │   ├── api.go              # API surface (6 frameworks)
│   │   ├── schema.go           # Prisma schema
│   │   ├── schema_multi.go     # Multi-lang schema (5 ORMs)
│   │   └── config.go           # Config/env var extraction
│   ├── skeleton/               # Code skeleton (12 languages)
│   │   └── parser.go
│   ├── imports/                # Import scanner (TS, Go, Python)
│   │   └── scanner.go
│   ├── typeregistry/           # Type extraction
│   │   └── registry.go
│   ├── search/                 # Code search + semantic search
│   │   ├── code.go             # ripgrep integration
│   │   └── tfidf.go            # TF-IDF semantic search engine
│   └── worker/                 # Background workers
│       └── watcher.go          # Git watcher, auto-index, discovery
├── pkg/types/                  # Shared types
│   └── types.go                # All data models
├── vscode-extension/           # VS Code extension scaffold
│   ├── src/extension.ts        # Entry point, command registration
│   ├── src/client.ts           # MCP client (JSON-RPC over stdio)
│   ├── src/compliance.ts       # Real-time compliance diagnostics
│   └── src/providers/          # Sidebar tree data providers
├── Makefile                    # Build, install, cross-compile, release
├── .goreleaser.yml             # GoReleaser config for automated releases
└── .gitignore
```

---

## Feature Map

### Built (Current)

| Category | Feature | Status |
|----------|---------|--------|
| **Core** | JSON + SQLite storage | Done |
| **Core** | Knowledge graph (edges, relations) | Done |
| **Core** | Feature lifecycle (start/archive/recall) | Done |
| **Core** | Evolution timeline | Done |
| **Indexing** | Auto-index on init | Done |
| **Indexing** | Git-connected auto-reindex | Done |
| **Indexing** | Auto-discovery of new files | Done |
| **Indexing** | Import graph auto-building | Done |
| **Extraction** | Code skeleton (12 languages) | Done |
| **Extraction** | API surface (6 frameworks) | Done |
| **Extraction** | Schema models (5 ORMs) | Done |
| **Extraction** | Config/env vars | Done |
| **Extraction** | Type definitions | Done |
| **Extraction** | Import scanning | Done |
| **Git Intel** | Expert finder | Done |
| **Git Intel** | File expertise analysis | Done |
| **Git Intel** | Knowledge risk detection | Done |
| **Git Intel** | File correlation analysis | Done |
| **Git Intel** | Commit context (why code exists) | Done |
| **Search** | Full-text search (FTS5) | Done |
| **Search** | Code content search | Done |
| **Search** | File search | Done |
| **Token Saving** | Skeleton extraction (~90% savings) | Done |
| **Token Saving** | Type extraction (~70% savings) | Done |
| **Token Saving** | Code snippets (~80% savings) | Done |
| **Token Saving** | Resume context (~95% savings) | Done |
| **Conversation** | Save conversation memory | Done |
| **Conversation** | Compact long conversations | Done |
| **Conversation** | Conversation hooks (auto-save hints) | Done |
| **Conversation** | Auto-capture checkpoints (every 25 calls) | Done |
| **Conversation** | Knowledge-triggered saves (decisions/warnings) | Done |
| **Conversation** | Conversation semantic search | Done |
| **Conversation** | List conversations tool | Done |
| **Compliance** | Check code against decisions/patterns | Done |
| **Onboarding** | Structured project walkthrough | Done |
| **Agentic** | Generate IDE agent rules (.cursorrules / CLAUDE.md) | Done |
| **Agentic** | Git post-commit hook (auto-extract knowledge) | Done |
| **Agentic** | Commit analysis (reverts, deletions, deps) | Done |
| **IDE** | Auto-configure Cursor/VS Code/Windsurf | Done |
| **IDE** | Manual install/uninstall | Done |
| **CLI** | 22 commands | Done |
| **Intelligence** | TF-IDF semantic search (no LLM) | Done |
| **Intelligence** | TF-IDF stemming + synonym expansion | Done |
| **Intelligence** | IDF-ranked vocabulary (rare terms prioritized) | Done |
| **Intelligence** | Smart context loading (token budgeting) | Done |
| **Intelligence** | Auto-capture evolution timeline | Done |
| **Intelligence** | Knowledge graph traversal (BFS) | Done |
| **Intelligence** | Context inheritance (feature extends) | Done |
| **Intelligence** | Git knowledge vectorization (experts + risks) | Done |
| **Intelligence** | Cross-repo contributor activity | Done |
| **Intelligence** | Deterministic semantic file selection | Done |
| **Intelligence** | Early-termination semantic search | Done |
| **Team** | Git-based sync (`teamcontext sync`) | Done |
| **Team** | Activity feed CLI (`teamcontext feed`) | Done |
| **Team** | Activity feed MCP tool (`get_feed`) | Done |
| **Distribution** | Makefile (cross-compile, install) | Done |
| **Distribution** | GoReleaser config | Done |
| **Distribution** | Version info (`teamcontext version`) | Done |
| **Storage** | Atomic writes (corruption-safe) | Done |
| **Storage** | Trailing newline (clean git diffs) | Done |
| **IDE** | VS Code extension scaffold | Done |
| **MCP** | 54 tools | Done |

### Planned (Roadmap)

| Category | Feature | Priority | Value |
|----------|---------|----------|-------|
| **Git Intel** | PR context enrichment | High | Auto-generate PR descriptions with history |
| **Git Intel** | Change risk scoring | High | Predict risk before committing |
| **Git Intel** | Decision archaeology | High | "Why did we do this?" from git + PRs |
| **Team** | Contributor tracking | Medium | Who made which decisions |
| **Analysis** | Test coverage mapping | Medium | Map tests to source files |
| **Analysis** | Complexity scoring | Medium | Identify complex code |
| **Analysis** | Hot file detection | Medium | Files with most churn/bugs |
| **Integration** | CI/CD hooks | Medium | Auto-enrich PRs, block risky changes |
| **Integration** | REST API | Low | HTTP endpoints for dashboards |
| **Integration** | Web dashboard | Low | Visualize knowledge, team stats |
| **Feature** | Conflict resolution | Low | Handle concurrent modifications |

---

## How It Creates Value

### For Individual Developers
- Resume any conversation in seconds (not minutes of re-explaining)
- Understand code without reading entire files (skeleton = 90% token savings)
- Know who to ask about code (expert finder)
- Get warned about pitfalls before hitting them

### For Teams
- New developers get full context from day one
- Knowledge survives when people leave
- Consistent decisions across the team
- Identify knowledge risks before they become problems

### For AI Tools
- Token-efficient tools reduce costs by 60-95%
- Persistent memory across sessions
- Pre-built context for common tasks
- Git history as additional intelligence

---

## Requirements

- Go 1.22+
- Git
- ripgrep (optional, for `search_code`)

## License

MIT

---

*"The brain that remembers everything. The brain that outlives everyone."*
