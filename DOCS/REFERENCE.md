# TeamContext Complete Guide

Everything you need to use every TeamContext feature. From first install to full team deployment.

---

## Table of Contents

1. [Setup](#1-setup)
2. [Daily Workflow](#2-daily-workflow)
3. [All CLI Commands](#3-all-cli-commands)
4. [All MCP Tools (54)](#4-all-mcp-tools-54)
5. [Team Workflows](#5-team-workflows)
6. [VS Code Extension](#6-vs-code-extension)
7. [Advanced Features](#7-advanced-features)
8. [Troubleshooting](#8-troubleshooting)

---

## 1. Setup

### First-Time Install

```bash
# Clone and build
git clone <repo> && cd teamcontext
make build                        # or: go build -o teamcontext ./cmd/teamcontext/

# Install to PATH
make install-system               # copies to /usr/local/bin
# or: go install ./cmd/teamcontext/ # copies to $GOPATH/bin

# Verify
teamcontext version
```

### Initialize a Project

```bash
cd /path/to/your/project
teamcontext init
```

This does everything:
- Creates `.teamcontext/` directory (knowledge store)
- Indexes all code files (skeletons, imports, dependency graph)
- Builds SQLite FTS5 search index + TF-IDF semantic vectors
- Mines git history (experts, risks, correlations, file history)
- Auto-detects and configures your IDE (Cursor, VS Code, Windsurf)
- Starts background workers

### Configure Your IDE

`teamcontext init` auto-configures these IDEs if detected:

| IDE | Auto-creates |
|-----|-------------|
| Cursor | `.cursor/mcp.json` |
| VS Code | `.vscode/mcp.json` |
| Windsurf | `.windsurf/mcp.json` |

For Claude Desktop (no project-level config):
```bash
teamcontext install claude --global
```

Manual configuration:
```bash
teamcontext install cursor           # Project-level (recommended)
teamcontext install cursor --global  # Global (not recommended)
teamcontext uninstall cursor         # Remove
```

### Set Up Team Features

```bash
# Make the AI agent follow team rules
teamcontext generate-rules            # Creates .cursorrules
teamcontext generate-rules --format claude  # Creates CLAUDE.md
teamcontext generate-rules --format all     # All formats

# Auto-capture knowledge from commits
teamcontext install-hooks

# Link sibling repos for cross-repo activity tracking
# Edit .teamcontext/config.json:
{
  "linked_repos": ["/path/to/other-repo-1", "/path/to/other-repo-2"]
}
```

### Verify Everything Works

```bash
teamcontext status     # Project name, files indexed, knowledge count
teamcontext stats      # Detailed statistics
```

Then restart your IDE and ask the AI: **"What TeamContext tools are available?"**

---

## 2. Daily Workflow

### Morning: Catch Up

```bash
# What did the team do while I was away?
teamcontext feed --since 24h

# Or in the IDE, ask:
# "Show me recent team activity" → AI calls get_feed
```

### Pull Latest Knowledge

```bash
teamcontext sync --pull              # Get team's latest knowledge from git
```

### Start Working on a Feature

Ask the AI:

> "Start a feature called auth-refactor"

The AI calls `start_feature` with `id: "auth-refactor"`. All subsequent decisions, warnings, and conversations are scoped to this feature.

To inherit context from a previous feature:
> "Start auth-v2 extending auth-v1"

### Before Writing Code

Ask the AI:

> "Check context for modifying src/auth/login.ts"

The AI calls `get_context` with your intent and target files. Returns relevant decisions, warnings, patterns, and expert info — all within your token budget.

### While Writing Code

The AI already has your team's rules from `.cursorrules` or `CLAUDE.md`. It will:
- Check decisions before suggesting code
- Warn about known pitfalls
- Follow established patterns
- Auto-capture new decisions you make

### After Writing Code

> "Check compliance on src/auth/session.ts"

The AI calls `check_compliance` and reports any violations of recorded decisions or patterns.

### Record Knowledge

> "Record a decision: We use bcrypt for password hashing because argon2 has poor Node.js support"

> "Record a warning: Never call the payment API without idempotency keys"

> "Record a pattern: All NestJS controllers must use the standard error interceptor"

### End of Day: Push Knowledge

```bash
teamcontext sync --push              # Share your knowledge additions
# or
teamcontext sync                     # Pull + push in one command
```

### Resume Next Day

Ask the AI:

> "Resume context for auth-refactor"

The AI calls `resume_context` and gets your compressed session history: files discussed, decisions made, open questions, next steps.

---

## 3. All CLI Commands

### Project Setup

| Command | What It Does | When to Use |
|---------|-------------|-------------|
| `teamcontext init` | Initialize project (index + IDE setup + git analysis) | First time in a project |
| `teamcontext index` | Re-index all files | After large codebase changes |
| `teamcontext rebuild` | Rebuild SQLite from JSON | If search seems stale |
| `teamcontext install <ide>` | Configure IDE MCP settings | Manual IDE setup |
| `teamcontext uninstall <ide>` | Remove IDE config | Cleanup |
| `teamcontext status` | Show project state | Quick check |
| `teamcontext stats` | Detailed statistics | Deep check |
| `teamcontext version` | Show version, commit, build date | Verify install |

### Server & IDE

| Command | What It Does | When to Use |
|---------|-------------|-------------|
| `teamcontext serve` | Start MCP server (JSON-RPC over stdio) | IDE uses this automatically |
| `teamcontext serve /path/to/project` | Serve specific project | When IDE needs explicit path |

### Feature Management

| Command | What It Does | When to Use |
|---------|-------------|-------------|
| `teamcontext start <name>` | Start a feature context | Beginning a task |
| `teamcontext list` | List all features | See what's active |
| `teamcontext resume <name>` | Resume a paused feature | Coming back to work |
| `teamcontext archive <name>` | Archive completed feature | Done with a feature |
| `teamcontext recall <name>` | Recall from archive | Revisiting old work |

### Knowledge Search

| Command | What It Does | When to Use |
|---------|-------------|-------------|
| `teamcontext search <query>` | Search all knowledge | Quick lookup |

### Team Collaboration

| Command | What It Does | When to Use |
|---------|-------------|-------------|
| `teamcontext sync` | Pull + push knowledge via git | Share with team |
| `teamcontext sync --pull` | Only pull latest | Get team updates |
| `teamcontext sync --push` | Only push local changes | Share your additions |
| `teamcontext feed` | Show recent activity | Catch up |
| `teamcontext feed --since 7d` | Activity in last 7 days | Weekly review |
| `teamcontext feed --limit 50` | More entries | Deeper review |
| `teamcontext feed --json` | JSON output | Scripting/piping |

### Intelligence & Automation

| Command | What It Does | When to Use |
|---------|-------------|-------------|
| `teamcontext generate-rules` | Generate .cursorrules | After adding decisions |
| `teamcontext generate-rules --format claude` | Generate CLAUDE.md | For Claude Code |
| `teamcontext generate-rules --format windsurf` | Generate .windsurfrules | For Windsurf |
| `teamcontext generate-rules --format all` | All formats | Multi-IDE team |
| `teamcontext install-hooks` | Install post-commit hook | Auto-capture from commits |
| `teamcontext uninstall-hooks` | Remove hooks | Disable auto-capture |
| `teamcontext analyze-commit` | Analyze latest commit | Called by hooks automatically |
| `teamcontext analyze-commit --quiet` | Silent analysis | Background use |

---

## 4. All MCP Tools (54)

These are the tools the AI agent calls when you interact through your IDE. You don't call them directly — you ask the AI in natural language and it picks the right tool.

### Search & Discovery (6 tools)

**`query`** — Natural language search across all knowledge
```
"Search for everything about authentication"
→ Finds decisions, warnings, patterns, files, git experts, conversations
→ Uses TF-IDF semantic matching + FTS5 keyword search
```

**`get_context`** — Get relevant context for an intent
```
"Get context for fixing the payment webhook"
→ intent: "fix payment webhook"
→ target_files: ["src/payments/webhook.ts"]
→ max_tokens: 4000
→ Returns ranked decisions, warnings, patterns within token budget
```

**`search`** — Search decisions, warnings, patterns by keyword
```
"Search for decisions about database"
→ Returns matching decisions, warnings, patterns with relevance scores
```

**`search_files`** — Find indexed files by name, path, or language
```
"Find all TypeScript files in the auth directory"
→ query: "auth", language: "typescript"
```

**`search_code`** — Search actual code content with regex
```
"Search for uses of localStorage"
→ pattern: "localStorage"
→ Returns file:line matches with surrounding context
```

**`get_related`** — Traverse knowledge graph from any node
```
"What's related to decision dec-001?"
→ node_type: "decision", node_id: "dec-001", max_depth: 2
→ Returns connected decisions, warnings, files, patterns via BFS
```

### Token-Saving Tools (7 tools)

**`get_skeleton`** — Code structure without bodies (~90% token savings)
```
"Show me the skeleton of payment.service.ts"
→ Returns classes, methods, signatures, types — no implementation
```

**`get_types`** — Type definitions only (~70% savings)
```
"What types are defined in the user module?"
→ Returns interfaces, type aliases, enums — no functions
```

**`search_snippets`** — Search and return matching code chunks (~80% savings)
```
"Find code snippets about error handling"
→ Returns relevant code fragments with context, ranked by relevance
```

**`get_recent_changes`** — Git history with impact analysis (~70% savings)
```
"What changed recently?"
→ limit: 10
→ Returns commits with files changed, impact level, AI summary
```

**`resume_context`** — Compressed session context (~95% savings)
```
"Resume where I left off on auth-refactor"
→ feature: "auth-refactor"
→ Returns compressed timeline of all sessions, files, decisions, next steps
```

**`list_conversations`** — Browse saved conversations
```
"Show me past sessions for auth-v2"
→ feature: "auth-v2", limit: 10
→ Returns chronological conversation summaries with key points
```

**`get_task_context`** — Pre-built context for common tasks
```
"I'm about to add a new API endpoint for invoices"
→ task: "add-api-endpoint", domain: "invoices"
→ Returns patterns, example code, types needed, warnings, checklist
```

### High-Impact Extraction (3 tools)

**`get_api_surface`** — Extract all API endpoints
```
"Show me all API endpoints in the notification app"
→ path: "apps/notification", app: "notification"
→ Supports: NestJS, Express, Go/gin/echo, Flask, FastAPI, Django, Spring, ASP.NET
→ Also extracts Kafka consumers/producers
```

**`get_schema_models`** — Extract database models
```
"What database models exist?"
→ path: "prisma/schema.prisma"
→ Supports: Prisma, GORM, SQLAlchemy, Django, JPA/Hibernate, TypeORM
→ Returns models, fields, relations, enums
```

**`get_config_map`** — Extract all config/env vars
```
"What environment variables does this project use?"
→ path: "src/"
→ Scans for process.env, os.Getenv, config files
```

### Code Analysis (4 tools)

**`scan_imports`** — Scan imports in a file or directory
```
"What does this file import?"
→ path: "src/auth/login.ts"
→ Returns: source, imported name, type (relative/package/builtin)
→ Supports: TypeScript, Go, Python
```

**`get_code_map`** — Hierarchical file tree
```
"Show me the project structure"
→ Returns tree of all indexed files with summaries, grouped by directory
```

**`get_dependencies`** — Dependency analysis
```
"What depends on auth.service.ts?"
→ path: "src/auth/auth.service.ts", direction: "both"
→ Returns imports (what it uses) and importers (what uses it)
```

**`trace_flow`** — Trace data flow through imports
```
"Trace the flow from the payment controller"
→ entry: "src/controllers/payment.controller.ts", depth: 3
→ Returns dependency chain with summaries and relevance scores
```

### Git Intelligence (5 tools)

**`find_experts`** — Who owns this code?
```
"Who knows the notification service?"
→ path: "apps/notification" OR files: ["src/sms/sms.service.ts"]
→ Returns developers ranked by ownership %, with activity status
→ Cross-repo activity detected via linked_repos config
```

**`get_file_history`** — Full expertise analysis for a file
```
"Who touched webhook.ts and when?"
→ path: "src/payments/webhook.ts"
→ Returns all contributors, ownership %, last commit, churn metrics
```

**`get_knowledge_risks`** — Bus factor analysis
```
"Where are our knowledge risks?"
→ Returns CRITICAL/HIGH/MEDIUM risk areas where experts are inactive
→ Includes mitigation suggestions
```

**`get_file_correlations`** — Files that change together
```
"What files always change together?"
→ Returns file pairs with correlation % and co-change count
→ Prevents partial changes (e.g., changing staging but forgetting production)
```

**`get_commit_context`** — Why does code exist?
```
"Why is there a setTimeout(0) on line 42 of queue.ts?"
→ file: "src/utils/queue.ts", start_line: 42, end_line: 45
→ Returns the commit that introduced it, author, message, and date
```

### Compliance, Onboarding & Team (3 tools)

**`check_compliance`** — Validate code against team rules
```
"Check if session.ts follows our decisions"
→ file_path: "src/auth/session.ts"
→ Returns violations: decision blockers, warning matches, anti-pattern hits
→ Also works with: diff (paste git diff), code (paste code)
```

**`onboard`** — Full project walkthrough
```
"I just joined, onboard me"
→ role: "backend" (optional), focus: "patterns" (optional)
→ Returns: project overview, top 15 decisions, warnings, patterns,
  code map, active features, expert contacts, knowledge risks, next steps
```

**`get_feed`** — Recent team activity
```
"What happened this week?"
→ since: "7d", limit: 20, type: "decision" (optional filter)
→ Returns timeline of decisions, warnings, patterns, conversations, events
```

### Indexing & Graph (3 tools)

**`index`** — Trigger full project re-index
```
"Re-index the project"
→ Scans all files, rebuilds skeletons, imports, dependency graph
```

**`index_status`** — Current index status
```
"How many files are indexed?"
→ Returns: files indexed, last run time, stale entries count
```

**`get_graph`** — View knowledge graph
```
"Show the knowledge graph"
→ Returns all edges: decision→file, warning→decision, pattern→file, etc.
```

### Knowledge Read (9 tools)

**`get_project`** — Project overview
```
"What is this project?"
→ Returns name, description, languages, architecture, goals, team mindset
```

**`get_feature`** — Feature context
```
"What's the status of auth-refactor?"
→ id: "auth-refactor"
→ Returns status, description, relevant files, decisions, warnings, conversations
```

**`list_features`** — All features
```
"What features are active?"
→ Returns all features with status (active/paused/archived)
```

**`list_decisions`** — All decisions
```
"What architectural decisions do we have?"
→ Returns all decisions with reasoning, alternatives, status
→ Optional: feature filter
```

**`list_warnings`** — All warnings
```
"What pitfalls should I know about?"
→ Returns all warnings with severity, evidence, related files
```

**`list_patterns`** — All patterns
```
"What coding patterns does the team follow?"
→ Returns patterns with examples, rules, anti-patterns
```

**`get_stats`** — System statistics
```
"How much knowledge does TeamContext have?"
→ Returns: files indexed, decisions, warnings, patterns, features, conversations
```

**`get_architecture`** — Architecture description
```
"What's the system architecture?"
→ Returns description, diagram, services, data flows
```

**`get_evolution_timeline`** — Knowledge evolution
```
"How has the project evolved?"
→ Returns chronological events: decisions, architecture changes, milestones
```

### Knowledge Write (11 tools)

**`index_file`** — Index a single file
```
"Index src/new-service.ts"
→ path: "src/new-service.ts"
→ Auto-generates skeleton, parses imports, creates graph edges, updates FTS index
```

**`add_decision`** — Record a decision
```
"Record: We chose PostgreSQL over MongoDB for ACID compliance"
→ content, reason, alternatives, feature, related_files, tags
→ Auto-creates evolution event + semantic vector + graph edges
```

**`add_warning`** — Record a pitfall
```
"Record warning: The legacy API returns 200 for errors with error in body"
→ content, reason, evidence, severity (info/warning/critical), feature
```

**`add_insight`** — Record a discovery
```
"Record insight: The auth module uses a custom token rotation strategy"
→ content, context, feature, related_files
```

**`add_pattern`** — Define a pattern
```
"Record pattern: All services must use dependency injection via constructor"
→ name, description, examples (file paths), rules, anti_patterns
```

**`add_evolution_event`** — Record a milestone
```
"Record: We migrated from Express to NestJS"
→ title, description, impact (low/medium/high)
```

**`save_conversation`** — Save session memory
```
→ summary, key_points, files_discussed, decisions_made, feature
→ Note: Auto-captured every 25 tool calls — manual save rarely needed
```

**`compact_conversation`** — Compress long sessions
```
→ Summarizes a conversation into a dense format preserving key decisions
```

**`update_feature_state`** — Update feature progress
```
"We finished the database migration for auth-refactor"
→ id: "auth-refactor", state: "DB migration complete, starting API layer"
```

**`update_architecture`** — Update architecture
```
"Update architecture: We now have a separate auth microservice"
→ description, diagram (ASCII/mermaid), services, data_flows
```

**`update_project`** — Update project metadata
```
"Update project description"
→ name, description, languages, goals, team_mindset
```

### Feature Lifecycle (3 tools)

**`start_feature`** — Create a feature context
```
→ id: "payment-v2", description: "Payment system rewrite"
→ Optional: extends: "payment-v1" (inherit parent context)
```

**`archive_feature`** — Archive when done
```
→ id: "payment-v2", summary: "Completed migration to Stripe"
→ Preserved forever, searchable, recallable
```

**`recall_feature`** — Bring back from archive
```
→ id: "payment-v2"
→ Reactivates with full history intact
```

---

## 5. Team Workflows

### Sharing Knowledge via Git

TeamContext stores knowledge in JSON files inside `.teamcontext/`. These files are git-tracked, so sharing is just git:

```bash
# Developer A records a decision, then shares:
teamcontext sync --push

# Developer B picks it up:
teamcontext sync --pull

# Or both at once:
teamcontext sync
```

**Merge conflicts** in knowledge files are auto-resolved using a union strategy. Both sides' additions are kept.

### Cross-Repo Activity Tracking

If your team has multiple repos, contributors might be active in repo B but look inactive in repo A:

```json
// .teamcontext/config.json in repo A
{
  "linked_repos": ["/path/to/repo-b", "/path/to/repo-c"]
}
```

After `teamcontext init` or `teamcontext rebuild`, cross-repo activity is detected. A contributor shows as "active (in repo-b)" instead of "INACTIVE".

### Team Onboarding Checklist

For a new developer joining:

1. `teamcontext init` in the project directory
2. Ask the AI: **"Onboard me"** → calls `onboard`, returns full project context
3. Ask: **"Who are the experts?"** → calls `find_experts`, returns ownership map
4. Ask: **"What should I avoid?"** → calls `list_warnings`, returns pitfalls
5. Ask: **"What patterns should I follow?"** → calls `list_patterns`, returns conventions

### Weekly Knowledge Review

```bash
# See what the team added this week
teamcontext feed --since 7d

# Check knowledge risks (are any experts leaving?)
# Ask the AI: "What are our knowledge risks?"

# Regenerate agent rules with latest knowledge
teamcontext generate-rules
```

### CI/CD Integration (Manual)

You can add TeamContext checks to CI:

```bash
# In your CI pipeline:
teamcontext init                     # Index the codebase
teamcontext analyze-commit           # Check latest commit
teamcontext feed --json --since 1d   # Get recent knowledge as JSON
```

---

## 6. VS Code Extension

A VS Code extension scaffold is included in `vscode-extension/`.

### Features

- **Activity Bar:** TeamContext sidebar with 5 panels
  - Knowledge Risks — CRITICAL/HIGH/MEDIUM risk areas
  - Decisions — All active architectural decisions
  - Warnings — All known pitfalls
  - Experts — Code ownership for the current file
  - Activity Feed — Recent team knowledge activity
- **Commands:**
  - `TeamContext: Refresh` — Refresh all panels
  - `TeamContext: Check Compliance` — Check current file against team rules
  - `TeamContext: Show Experts` — Show experts for current file
  - `TeamContext: Sync Knowledge` — Run `teamcontext sync` in terminal
  - `TeamContext: Show Activity Feed` — Refresh feed panel
- **Real-time Compliance:** On every file save, compliance is checked and violations appear in VS Code's Problems panel

### Setup

```bash
cd vscode-extension
npm install
npm run compile
# Then install the extension in VS Code via the VSIX or dev mode
```

The extension activates automatically when it detects a `.teamcontext` directory in the workspace.

---

## 7. Advanced Features

### Semantic Search (No LLM Required)

TeamContext uses TF-IDF vectorization with stemming and synonym expansion:

- **Stemming:** `-ation`, `-tion`, `-ing`, `-ed`, `-ly` stripped → "authentication" matches "authenticat"
- **Synonyms:** `auth→authentication`, `db→database`, `api→endpoint`, `config→configuration`, etc. (~30 mappings)
- **IDF ranking:** Rare-but-not-unique terms score highest (most discriminative)
- **Early termination:** Search stops when enough high-quality results are found

No API keys, no LLM calls, no network. Everything runs locally.

### Token Budgeting

`get_context` accepts `max_tokens` and returns results ranked by relevance:

| Score | Source |
|-------|--------|
| 1.0 | Direct file path match |
| 0.9 | Same feature |
| 0.7 | Knowledge graph connection |
| 0.6 | Ancestor feature context |
| 0.5+ | Critical warning boost |
| 0.5 | Keyword match |
| 0.0-1.0 | Semantic similarity |

Items are added until the budget is filled. Response includes `token_budget.used` and `token_budget.remaining`.

### Knowledge Graph

Every `add_decision`, `add_warning`, `add_pattern`, and `index_file` creates graph edges:

- `decision → file` (affects)
- `warning → decision` (warns)
- `pattern → file` (follows)
- `feature → decision` (belongs_to)

Use `get_graph` to see all edges. Use `get_related` to BFS-traverse from any node.

### Auto-Capture

TeamContext captures knowledge automatically from three sources:

**1. Session checkpoints** (every 25 tool calls):
- Summary of tools used, files touched, decisions made
- Semantically indexed for future search

**2. Knowledge triggers** (after `add_decision`/`add_warning`):
- If 5+ tool calls accumulated, auto-save before the knowledge is lost

**3. Git commit analysis** (via `install-hooks`):
- Reverts → auto-warning
- Large deletions → auto-warning
- Dependency/config changes → evolution event

### IDE Agent Rules

`teamcontext generate-rules` creates a system prompt that instructs the AI to:

1. Call `get_context` before modifying code
2. Treat all decisions as binding constraints
3. Heed all warnings as pitfalls
4. Follow all patterns as conventions
5. Auto-capture decisions when the user makes them
6. Mention code experts when modifying their files

The rules include the full text of every active decision, warning, and pattern. Regenerate after adding new knowledge.

### Context Inheritance

Features can inherit from parent features:

```json
{"tool": "start_feature", "arguments": {"id": "auth-v2", "extends": "auth-v1"}}
```

The child inherits `relevant_files`, `decisions`, and `warnings` from the parent. `get_context` walks up to 5 ancestor levels.

### Atomic Storage

JSON writes use atomic rename (write to `.tmp`, then rename). This prevents corruption from crashes or concurrent access. All JSON files end with a trailing newline for clean git diffs.

---

## 8. Troubleshooting

### "teamcontext: command not found"

```bash
# Check if it's installed
which teamcontext
ls $GOPATH/bin/teamcontext
ls /usr/local/bin/teamcontext

# Rebuild and install
cd teamcontext
make install-system
```

### "No .teamcontext directory found"

Run `teamcontext init` in your project root directory.

### Search returns no results

```bash
# Rebuild the search index
teamcontext rebuild
```

### IDE doesn't see TeamContext

1. Check the MCP config exists: `cat .cursor/mcp.json` (or `.vscode/mcp.json`)
2. Restart the IDE
3. Verify the binary path in the config is correct
4. Try reinstalling: `teamcontext install cursor`

### Stale index

```bash
teamcontext index     # Re-index all files
teamcontext rebuild   # Rebuild SQLite from JSON
```

### Hook not firing

```bash
# Check if hook exists
cat .git/hooks/post-commit

# Reinstall
teamcontext install-hooks
```

### Sync conflicts

```bash
# teamcontext sync auto-resolves JSON conflicts
# If it fails, manually resolve and retry:
git add .teamcontext/
git commit -m "resolve knowledge merge"
teamcontext sync --push
```

### Too many tokens

Use token-saving tools:
- `get_skeleton` instead of reading full files (~90% savings)
- `get_types` for just type definitions (~70% savings)
- `search_snippets` for relevant fragments (~80% savings)
- `get_context` with `max_tokens` parameter for budgeted responses

---

## Complete Feature Checklist

Use this to verify you've set up everything:

- [ ] `teamcontext init` — Project initialized
- [ ] `teamcontext status` — Shows correct file count
- [ ] `teamcontext install <ide>` — IDE configured (or auto-detected)
- [ ] `teamcontext generate-rules` — Agent rules generated
- [ ] `teamcontext install-hooks` — Git hooks installed
- [ ] Record first decision — `add_decision` via AI
- [ ] Record first warning — `add_warning` via AI
- [ ] Record first pattern — `add_pattern` via AI
- [ ] Test search — `query` or `search` via AI
- [ ] Test compliance — `check_compliance` via AI
- [ ] Test onboarding — `onboard` via AI
- [ ] Test experts — `find_experts` via AI
- [ ] Test risks — `get_knowledge_risks` via AI
- [ ] Test feed — `teamcontext feed` or `get_feed` via AI
- [ ] Test sync — `teamcontext sync` (needs remote)
- [ ] Linked repos — Configure `linked_repos` if multi-repo
- [ ] Team sharing — Push `.teamcontext/` via git
