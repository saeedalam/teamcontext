# TeamContext Cheat Sheet 🚀

**The Brain That Remembers. The Tools That Save Tokens.**

---

## 🏎️ The Quick Start
```bash
teamcontext init --force    # Start fresh (index + IDE setup)
teamcontext reindex         # Clean re-index (preserves knowledge)
teamcontext status          # Check system health
```

---

## 🛠️ The Power Tools (MCP)
*Ask your AI to use these specific tools for maximum impact.*

| Tool | When to Use | Secret Sauce |
|------|-------------|--------------|
| `get_blueprint` | **Starting a Task** | Returns file patterns, snippets, and conventions in ONE call. |
| `get_tree` | **Navigating** | Ultra-compact hierarchical view. Flattens single-child dirs. |
| `get_context` | **Before Coding** | Ranked decisions, warnings, and expert info within token budget. |
| `get_skeleton` | **Understanding** | Classes and signatures ONLY. Saves 90% tokens. |
| `find_experts` | **Collaboration** | Who owns this code? Ranked by ownership % and activity. |

---

## 🧠 Knowledge Capture
*Don't just code. Build the team's memory.*

- **Record a Decision**: "Record: We use bcrypt because argon2 lacks stable Node bindings."
- **Record a Warning**: "Warning: The legacy API returns 200 for internal errors."
- **Start a Feature**: "Start feature auth-v2 extending auth-v1" (Inherits context!)
- **Check Compliance**: "Check if this code violates any team decisions."

---

## 📈 Real-World Workflows

### 🆕 The "First Day" Onboarding
1. Run `teamcontext init`
2. Ask AI: **"Onboard me"**
3. Ask AI: **"Who are the experts?"**
4. Result: Full project mental map in under 2 minutes.

### 🕵️ The "Bug Hunt"
1. Ask AI: **"What warnings are relevant to the payment service?"**
2. Ask AI: **"Get history for src/payments/webhook.ts"**
3. Result: Pitfalls revealed and experts identified instantly.

### 🏗️ The "Big Refactor"
1. Run `teamcontext start refactor-v1`
2. Document every key decision as you go via `add_decision`.
3. At the end, run `teamcontext generate-rules`.
4. Result: The Next Dev (or AI) follows your rules automatically.

---

## 📡 Team Sync
```bash
teamcontext sync     # Share knowledge with the team via Git
teamcontext feed     # See what changed in the knowledge base recently
```

---
*"Memory is the ultimate leverage."*
