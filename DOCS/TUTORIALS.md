# TeamContext Use Cases

Practical examples of how to use TeamContext in your daily workflow.

---

## 1. Onboard to a New Codebase

**Scenario:** You just joined a team and need to understand the project fast.

```bash
# Step 1: Initialize TeamContext
cd /path/to/project
teamcontext init

# Step 2: Ask your AI agent these questions (via MCP tools)
```

| Question | MCP Tool | What You Get |
|----------|----------|-------------|
| "What does this project do?" | `get_project` | Project name, languages, architecture overview |
| "Show me the code structure" | `get_code_map` | Hierarchical tree of all indexed files |
| "What are the key decisions?" | `list_decisions` | All architectural decisions with reasoning |
| "What should I avoid?" | `list_warnings` | Known pitfalls and gotchas |
| "Who knows the auth code?" | `find_experts` with `path: "src/auth"` | Top contributors ranked by ownership % |

---

## 2. Fix a Bug You've Never Seen Before

**Scenario:** A bug report mentions "payment webhook fails for refunds."

```
Step 1: Find relevant code
  → search_code: "webhook" + "refund"
  → Returns file paths and matching lines

Step 2: Understand without reading full files
  → get_skeleton: "src/payments/webhook.ts"
  → Returns function signatures, classes, types — no bodies (~90% token savings)

Step 3: Check for known pitfalls
  → get_context: intent="payment webhook refund bug"
  → Returns related decisions, warnings, patterns

Step 4: Find who to ask
  → find_experts: path="src/payments"
  → Returns: "Alice (72% ownership, active), Bob (18%, inactive)"

Step 5: Check what usually changes together
  → get_file_correlations
  → "webhook.ts and payment.service.ts change together 85% of the time"
  → Don't forget to check payment.service.ts too!
```

---

## 3. Resume Yesterday's Work

**Scenario:** You were debugging an auth issue yesterday and want to pick up where you left off.

```
Step 1: Resume context
  → resume_context: feature="auth-bug-fix"
  → Returns: compressed summary, files discussed, open questions, next steps

Step 2: Get current feature state
  → get_feature: id="auth-bug-fix"
  → Returns: current state, relevant files, decisions made so far
```

**Tokens saved:** ~95% compared to re-reading the full conversation history.

---

## 4. Add a New API Endpoint

**Scenario:** You need to add a `POST /api/invoices` endpoint.

```
Step 1: See how existing endpoints are built
  → get_api_surface
  → Returns all existing endpoints with their handlers, files, decorators

Step 2: Get the pattern to follow
  → get_task_context: task="add-api-endpoint", domain="invoices"
  → Returns: patterns, example code, types needed, warnings, checklist

Step 3: Check what types/models exist
  → get_schema_models
  → Returns all database models, fields, relations

Step 4: Understand the dependency chain
  → get_dependencies: path="src/controllers/payments.controller.ts", direction="both"
  → See what the existing payment controller imports and what imports it
```

---

## 5. Review a Pull Request

**Scenario:** Someone submitted a PR touching 8 files. You need to understand the impact.

```
Step 1: Check recent changes
  → get_recent_changes: limit=5
  → See what changed, who changed it, impact level

Step 2: For each changed file, understand context
  → get_skeleton: "path/to/changed/file.ts"
  → Quick structural overview without reading every line

Step 3: Check knowledge risks
  → get_knowledge_risks
  → "WARNING: src/billing/ has CRITICAL risk — primary expert (Charlie) is inactive"

Step 4: Check correlated files
  → get_file_correlations
  → "billing.service.ts and invoice.template.ts change together 90% of the time"
  → Verify the PR includes invoice.template.ts changes too
```

---

## 6. Understand Why Code Exists

**Scenario:** You see a weird `setTimeout(0)` in the codebase and wonder why.

```
Step 1: Get commit context
  → get_commit_context: file="src/utils/queue.ts", start_line=42, end_line=45
  → Returns: "Added in commit abc123 by Alice: 'Fix race condition in event loop processing'"

Step 2: Check related decisions
  → search: query="race condition event loop"
  → Returns: Decision dec-007: "Use setTimeout(0) for deferred execution to avoid Zalgo"
```

---

## 7. Track Knowledge Risks Across Repos

**Scenario:** Your team has 3 microservices. Contributors move between them.

```bash
# Step 1: Link repos in each project's config
# In /path/to/service-a/.teamcontext/config.json:
{
  "linked_repos": [
    "/path/to/service-b",
    "/path/to/service-c"
  ]
}
```

```
Step 2: Run init or rebuild
  → teamcontext init
  → Cross-references contributor activity across all linked repos

Step 3: Check risks
  → get_knowledge_risks
  → Alice shows as "active" even though she hasn't committed to service-a recently,
    because she's active in service-b (shown via active_in_repo field)
  → Risk level drops from CRITICAL to MEDIUM
```

---

## 8. Search With Natural Language

**Scenario:** You want to find everything related to "user authentication" but the code uses different terms.

```
Step 1: Semantic search
  → query: "user authentication"
  → Finds results mentioning:
    - "auth middleware" (synonym: auth → authentication)
    - "login process" (semantic similarity)
    - "JWT token validation" (related context)
    - Expert: "Alice in src/auth with 65% ownership" (git expert vectorized)
    - Risk: "HIGH in src/auth - single contributor" (git risk vectorized)

Step 2: Narrow with context
  → get_context: intent="fix user authentication", target_files=["src/auth/login.ts"], max_tokens=4000
  → Returns decisions, warnings, patterns — all within your token budget
```

---

## 9. Record Decisions As You Go

**Scenario:** You and the AI decide to use JWT instead of sessions. Record it so future sessions know.

```
→ add_decision:
    content: "Use JWT for authentication instead of server-side sessions"
    reason: "Stateless authentication scales better with our microservice architecture"
    alternatives: ["Server-side sessions with Redis", "OAuth2 only"]
    feature: "auth-v2"

→ add_warning:
    content: "JWT tokens cannot be revoked — use short expiry + refresh tokens"
    severity: "warning"
    feature: "auth-v2"
```

Next time anyone works on auth, `get_context` automatically surfaces these.

---

## 10. Migrate or Refactor Safely

**Scenario:** You're migrating from Express to NestJS.

```
Step 1: See all current endpoints
  → get_api_surface
  → Returns every route, handler, middleware

Step 2: Start a feature with context inheritance
  → start_feature: id="nestjs-migration", extends="express-api"
  → Inherits all relevant files, decisions, warnings from the parent feature

Step 3: Track progress
  → update_feature_state: id="nestjs-migration", state="Migrated 5/12 controllers"

Step 4: Record migration patterns
  → add_pattern:
      name: "Express to NestJS controller migration"
      description: "Replace app.get() with @Get() decorator on controller class"
      examples: ["src/controllers/users.controller.ts"]

Step 5: When done, archive with summary
  → archive_feature: id="nestjs-migration"
  → Preserved forever, recallable anytime
```

---

## 11. Make Your IDE Agent Follow Team Rules Automatically

**Scenario:** You want Cursor/Claude Code to automatically check TeamContext before writing code and capture decisions without being asked.

```bash
# Generate rules from your knowledge base
teamcontext generate-rules                    # Creates .cursorrules
teamcontext generate-rules --format claude    # Creates CLAUDE.md
teamcontext generate-rules --format all       # Creates all formats
```

**What it generates:**
- A system prompt that instructs the IDE agent to call `get_context` before modifying code
- All active decisions listed as binding constraints
- All warnings listed as pitfalls to avoid
- All patterns listed as conventions to follow
- Instructions to auto-capture decisions and warnings from conversations

**Regenerate anytime:**
```bash
# After adding new decisions or patterns, regenerate
teamcontext generate-rules
```

---

## 12. Auto-Capture Knowledge from Git Commits

**Scenario:** You want TeamContext to learn from every commit without manual input.

```bash
# Install the post-commit hook
teamcontext install-hooks

# Now every commit is automatically analyzed:
git commit -m "Revert bad auth migration"
# → TeamContext auto-creates Warning: "Commit reverted: Revert bad auth migration"

git commit -m "Remove legacy payment gateway"
# → TeamContext auto-creates Warning: "Large deletion: 250 lines removed"
```

**What the hook detects:**
- Reverted commits → auto-warning
- Large deletions (>100 lines) → auto-warning
- Dependency changes (package.json, go.mod, etc.) → logged
- Config changes (Dockerfile, .env, etc.) → logged
- TODO/FIXME/HACK additions → logged

**Remove anytime:**
```bash
teamcontext uninstall-hooks
```

---

## 13. Validate Code Against Team Conventions

**Scenario:** You wrote code and want to check it follows recorded decisions before committing.

```
Step 1: Check compliance
  → check_compliance: file_path="src/auth/session.ts"
  → Returns:
    BLOCKER: Code contains 'localStorage' which conflicts with decision:
             "Ban localStorage for tokens" — Security: tokens must use httpOnly cookies
    WARNING: Code may trigger known pitfall: "JWT tokens cannot be revoked"
    INFO: This file is governed by decision: "Use JWT for authentication"

Step 2: Fix violations, then re-check
  → check_compliance: file_path="src/auth/session.ts"
  → { "compliant": true, "violations": [], "blockers": 0 }
```

**Works with diffs too:**
```
→ check_compliance: diff="<paste git diff output>"
```

---

## 14. Onboard a New Team Member in One Call

**Scenario:** A new developer joins and needs to understand the entire project immediately.

```
→ onboard
→ Returns:
  - Project overview (name, languages, architecture)
  - Top 15 active decisions (with reasoning and alternatives)
  - Active warnings and pitfalls
  - Established patterns (with anti-patterns to avoid)
  - Code map (top 30 key files with summaries)
  - Active features being worked on
  - Expert contacts (who to ask about what)
  - Knowledge risks (bus factor areas)
  - Next steps (tools to use)

# Tailor to role:
→ onboard: role="frontend"
→ Adds frontend-specific tips and focus areas

→ onboard: role="backend", focus="patterns"
→ Shows only backend patterns in detail
```

---

## 15. Review Past Sessions

**Scenario:** You want to see what was discussed in previous sessions.

```
Step 1: List recent conversations
  → list_conversations
  → Returns conversations sorted by most recent, with summaries and key points

Step 2: Filter to a specific feature
  → list_conversations: feature="auth-v2"

Step 3: Filter by date
  → list_conversations: since="2025-06-01T00:00:00Z"
```

**Note:** Conversations are auto-saved as checkpoints every 25 tool calls, after knowledge-creation events, and on feature lifecycle changes. No manual save needed.

---

## 16. Sync Knowledge With Your Team

**Scenario:** You recorded decisions and warnings. You want your teammates to have them too.

```bash
# Push your local knowledge to the team
teamcontext sync --push

# Pull the team's latest knowledge
teamcontext sync --pull

# Do both (default)
teamcontext sync
```

**What happens:**
- `.teamcontext/` JSON files are committed and pushed via git
- Merge conflicts in knowledge files are auto-resolved (union strategy)
- Knowledge travels with your branches — merge naturally via git

**No extra infrastructure:** No database, no API server, no Slack. Just git.

---

## 17. Check What the Team Did Recently

**Scenario:** You've been away for a week and want to catch up.

```bash
# CLI: See what changed this week
teamcontext feed --since 7d

# CLI: Just decisions
teamcontext feed --since 7d --limit 10

# CLI: JSON output for scripting
teamcontext feed --json
```

```
# MCP tool: Ask the AI
→ get_feed: since="7d", limit=20
→ Returns timeline of decisions, warnings, patterns, conversations, events
```

**Talking point:** "Like a team standup but for knowledge. No meetings needed."

---

## 18. Trace Data Flow Across Files

**Scenario:** You need to understand how a request flows through the system.

```
Step 1: Trace from an entry point
  → trace_flow: entry="handleWebhook"
  → Returns:
    handleWebhook (src/webhooks/handler.ts)
      → validateSignature (src/webhooks/auth.ts)
      → processPayment (src/payments/processor.ts)
        → updateOrder (src/orders/order.service.ts)
        → sendNotification (src/notifications/notify.ts)

Step 2: See dependencies in both directions
  → get_dependencies: path="src/payments/processor.ts", direction="both"
  → Imports: order.service.ts, notification.service.ts, payment.types.ts
  → Imported by: webhook/handler.ts, api/payment.controller.ts
```

---

## 19. Explore the Knowledge Graph

**Scenario:** You want to see how files, people, and decisions are connected.

```
Step 1: Get the full graph
  → get_graph
  → Returns nodes (files, people, decisions, warnings) and edges (owns, relates_to, authored)

Step 2: Navigate visually
  → The graph shows:
    Alice ──owns──→ src/auth/login.ts
    src/auth/login.ts ──relates_to──→ dec-005 ("Use JWT for auth")
    dec-005 ──relates_to──→ warn-012 ("JWT cannot be revoked")
```

**Use when:** You want to understand the big picture — how everything connects.

---

## 20. Extract Types Without Reading Files

**Scenario:** You need to know what TypeScript types exist before writing new code.

```
Step 1: Get all types in a directory
  → get_types: path="src/payments"
  → Returns:
    PaymentStatus (enum): PENDING, COMPLETED, FAILED, REFUNDED
    PaymentRequest (interface): { amount: number, currency: string, ... }
    PaymentResult (type): { success: boolean, transactionId: string }

Step 2: Use in your new code without reading the source files
```

**Tokens saved:** ~95% compared to reading all type definition files.

---

## 21. Manage the Search Index

**Scenario:** Files changed outside of TeamContext's watch (e.g., branch switch, git pull).

```bash
# CLI: Rebuild the index
teamcontext rebuild

# CLI: Check index status
teamcontext status
```

```
# MCP: Trigger re-index
  → index
  → Returns: { "files_indexed": 2636, "duration_ms": 1200 }

# MCP: Check index health
  → index_status
  → Returns: { "indexed_files": 2636, "stale_files": 12, "last_indexed": "2026-01-30T10:00:00Z" }
```

**When to use:** After large git operations (rebase, merge, branch switch) or when search results seem stale.

---

## 22. Get Project Statistics

**Scenario:** You want a quick health check of the project's knowledge base.

```
→ get_stats
→ Returns:
  Files indexed: 2636
  Decisions: 12 (8 active, 4 superseded)
  Warnings: 7 (2 critical, 3 warning, 2 info)
  Patterns: 5
  Features: 3 (2 active, 1 archived)
  Conversations: 14
  Contributors: 8 (5 active, 3 inactive)
  Risk areas: 4 (2 critical, 2 high)
```

**Use when:** Starting a session, preparing a status update, or checking knowledge base health.

---

## 23. Recall an Archived Feature

**Scenario:** A feature was archived months ago, but now you need to revisit it.

```
Step 1: Find the feature
  → list_features: include_archived=true
  → Returns: "nestjs-migration" (archived 2025-11-15)

Step 2: Recall its full context
  → recall_feature: id="nestjs-migration"
  → Returns:
    - All decisions made during the migration
    - Warnings that were raised
    - Patterns established
    - Files that were involved
    - Conversations from that period
    - Final state summary

Step 3: Use the context for your new work
  → The AI now knows everything about the original migration
    without you explaining any of it
```

**Tokens saved:** ~99% compared to re-reading old chat logs.

---

## 24. Update Architecture Documentation

**Scenario:** The project architecture changed (new service, new database, new pattern).

```
→ update_architecture:
    description: "Added Redis cache layer between API gateway and database"
    components: ["api-gateway", "redis-cache", "postgres"]
    reason: "Reduce database load for read-heavy endpoints"

# The architecture info is now:
# - Returned by get_project
# - Included in onboard results
# - Surfaced by get_context when relevant
# - Embedded in generated rules
```

---

## 25. Compact a Long Conversation

**Scenario:** You've been working for hours. The conversation is getting long and expensive.

```
Step 1: Save current progress
  → save_conversation
  → Saves: summary, key points, files discussed, open questions, next steps

Step 2: Compact to reduce tokens
  → compact_conversation
  → Returns a compressed summary of the full conversation
  → The AI can continue with this summary instead of the full history

Step 3: Later, resume where you left off
  → resume_context: feature="my-feature"
  → Restores the compressed context from the saved conversation
```

**Token savings:** ~95% reduction in context size while preserving all important information.

---

## 26. Full Workflow: Zero to Productive

**Scenario:** Complete workflow from first install to daily use.

```bash
# Day 0: Setup (once)
teamcontext init                    # Index everything
teamcontext install-hooks           # Auto-capture from commits
teamcontext generate-rules          # Make AI follow team conventions

# Day 1: Morning routine
teamcontext feed --since 24h        # What happened since yesterday?
teamcontext sync                    # Pull team knowledge
```

```
# During work: AI-assisted coding
  → get_context: intent="add payment retry logic"     # Before coding
  → get_skeleton: "src/payments/service.ts"            # Understand structure
  → find_experts: path="src/payments"                  # Who to ask
  → check_compliance: file_path="src/payments/retry.ts" # After coding
```

```
# Record what you learned
  → add_decision: content="Use exponential backoff for retries"
  → add_warning: content="Max 3 retries — vendor rate limits at 5/min"
```

```bash
# End of day
teamcontext sync --push             # Share with team
teamcontext generate-rules          # Update agent rules with new knowledge
```

---

## Quick Reference: When to Use What

| I want to... | Use this tool |
|--------------|--------------|
| Understand a file fast | `get_skeleton` |
| Find code by content | `search_code` |
| Find files by name/language | `search_files` |
| Find related knowledge | `query` or `get_context` |
| Know who wrote something | `find_experts` |
| Check what's risky | `get_knowledge_risks` |
| See what changes together | `get_file_correlations` |
| Resume previous work | `resume_context` |
| Record a decision | `add_decision` |
| Record a pitfall | `add_warning` |
| See all endpoints | `get_api_surface` |
| See database models | `get_schema_models` |
| See env vars | `get_config_map` |
| Trace data flow | `trace_flow` |
| See how knowledge evolved | `get_evolution_timeline` |
| Validate code compliance | `check_compliance` |
| Onboard to a project | `onboard` |
| Browse session history | `list_conversations` |
| See recent team activity | `get_feed` or CLI: `teamcontext feed` |
| Sync knowledge with team | CLI: `teamcontext sync` |
| Generate agent rules | CLI: `teamcontext generate-rules` |
| Auto-capture from commits | CLI: `teamcontext install-hooks` |
| Explore the knowledge graph | `get_graph` |
| Extract types from code | `get_types` |
| Rebuild the search index | `index` or CLI: `teamcontext rebuild` |
| Check index health | `index_status` or CLI: `teamcontext status` |
| Get project statistics | `get_stats` |
| Recall an old feature | `recall_feature` |
| Update architecture info | `update_architecture` |
| Compress a long session | `compact_conversation` |
| Save session manually | `save_conversation` |
| See file dependencies | `get_dependencies` |
