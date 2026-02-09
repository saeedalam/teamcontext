# TeamContext Cheat Sheet 🚀

**The brain that remembers everything. The brain that outlives everyone.**

TeamContext is a technical memory layer for your codebase. It saves you from re-explaining context to AI and hunting for "Why was this done?" in old PRs.

---

## ⚡️ Real-World Scenarios

### 🆕 The "First Day" Onboarding
**Scenario**: You just joined a complex project and don't know where to start.
1.  In IDE ask: `"Onboard me to the backend"`
2.  **Result**: You get a high-level architecture view, top 10 architectural decisions, active warnings, and who the experts are for the key directories.

### 🔍 The "Bug Hunt"
**Scenario**: A bug is reported in the payment webhook.
1.  In IDE ask: `"Trace the flow of payment webhooks"`
2.  **Result**: TeamContext uses the dependency graph to show you exactly which files handle the request, plus any `warnings` recorded by teammates about that specific logic.

### 🏗 The "Big Refactor"
**Scenario**: You need to migrate from Express to Fastify.
1.  In IDE ask: `"Get context for migrating Express to Fastify"`
2.  **Result**: TeamContext pulls all decisions related to "Express", finds all files importing "express", and gives you a ranked list of files that will be affected.

---

## 🛠 Power Tools (Use These First!)

| Tool | When to use | Why it's elite |
| :--- | :--- | :--- |
| **`get_blueprint`** | Starting a feature | Generates a checklist, file patterns, and examples in 1 call. |
| **`get_tree`** | Exploring folders | **Ultra-compact**. Hides noise (node_modules), flattens paths. |
| **`get_context`** | Before writing code | Pulls *just* the right knowledge within your token budget. |
| **`query`** | You have a question | Uses **Semantic Search** (TF-IDF) to find "Why" and "How". |

---

## ⚙️ How it works (The 30-Second Pitch)

- **JSON + SQLite Hybrid**: Your "Knowledge" (Decisions/Warnings) is stored in **JSON** (so you can Git-track it). The "Search Index" is in **SQLite** (so it's instant).
- **100% Local**: No credit card required. No tokens sent to an external embedding API. All indexing happens on your machine.
- **Fast Init**: v0.2.1 indexes **2,700+ files in ~10 seconds** using bulk transactions.

---

## 📤 Forward this to your team
> "Hey team, I've set up **TeamContext** for this repo. It's an MCP server that tracks our architectural decisions and git expertise.
> 
> **To join in:**
> 1. `brew install teamcontext` (or build from source)
> 2. `teamcontext init` in this repo.
> 3. Restart Cursor/Claude/Windsurf.
> 
> **Try asking the AI:** 'Show me the project architecture' or 'Who knows the auth module best?'"

---
*For the full manual, see [REFERENCE.md](./REFERENCE.md)*
