# 🎭 TeamContext Demo Script
> **The Brain That Remembers Everything. The Brain That Outlives Everyone.**

This script provides a step-by-step guide to demo TeamContext to your engineering team. It is designed to show maximum value in ~20 minutes.

**Setup:** Have your IDE (Cursor/VS Code) open with a project already initialized with TeamContext.

---

## 🏗 Before the Demo

Make sure TeamContext is already initialized in your demo project:

```bash
cd /path/to/[Project-Name]
teamcontext status
```

You should see:
```
Project: [Project-Name]
Files indexed: [Total-Files]
Decisions: [Total-Decisions]
```

---

## 🚀 Demo Flow

### Part 1: "What does this project look like?" (2 min)

Open your IDE chat and type:

> **"What is this project?"**

The AI calls `get_project` and returns a high-level overview: name, primary languages, architecture, and goals.

Then ask:

> **"Show me the code structure"**

The AI calls `get_code_map` and returns a hierarchical tree of all indexed files.

**Talking point:** "TeamContext indexed the entire project in one command. The AI didn't have to read every file individually—it already knows the full structure and purpose of our codebase."

---

### Part 2: "Who knows what?" (3 min)

Ask the AI about a specific module or directory:

> **"Who are the experts on the [Module-Name]?"**

The AI calls `find_experts` and returns a ranked list of contributors based on git history:

```
Alice — 60% ownership, active
Bob   — 25% ownership, active
Charlie — 15% ownership, INACTIVE
```

**Talking point:** "TeamContext automatically mines git history to find our 'knowledge owners'. It even flags when an expert is inactive, helping us spot knowledge silos before they become a problem."

---

### Part 3: "Where are the risks?" (3 min)

This is the business-value moment. Ask:

> **"What knowledge risks do we have?"**

The AI calls `get_knowledge_risks` and returns a list of critical areas:

```
CRITICAL: [Path/To/Critical/Module]
  → Charlie owns 100% and is inactive
  → Last activity: 6 months ago
  → Mitigation: Assign new maintainer to review and document
```

**Talking point:** "These are real-time risks derived from our repo. TeamContext finds them in seconds, showing us exactly where we have a code 'bus factor' risk."

---

### Part 4: "What changes together?" (2 min)

Ask:

> **"What files usually change together?"**

The AI calls `get_file_correlations` and returns correlated file pairs:

```
[config/staging.yaml] ↔ [config/production.yaml]
  → 100% correlation, changed together 12 times
```

**Talking point:** "TeamContext prevents partial changes. If someone modifies a staging config but forgets production, the AI agent will warn them because it knows these files are statistically linked."

---

### Part 5: "Understand code without reading it" (2 min)

Ask for a structural overview of a complex file:

> **"Show me the skeleton of [ComplexFile].go"** (or .ts, .py)

The AI calls `get_skeleton` and returns just the signatures:

```go
type Processor interface {
    Process(ctx context.Context, data []byte) error
}

func NewProcessor(cfg *Config) *processor { ... }
```

**Talking point:** "The AI understands the full structure—interfaces, methods, and parameters—without reading the implementation details. This saves up to 90% in tokens compared to traditional code-pasting."

---

### Part 6: "Decisions that persist" (2 min)

Ask:

> **"What architectural decisions have we recorded?"**

The AI calls `list_decisions` and returns the recorded history:

```
dec-001: Use PostgreSQL for [Reasoning]
  Alternatives: [Alternative-1], [Alternative-2]
  Related files: [file-1], [file-2]
```

Then show how to record a new one:

> **"Record a decision: We are switching our [Library/Framework] to [New-Library] because [Detailed Reason]"**

The AI calls `add_decision` and it's saved permanently.

**Talking point:** "Next time a new dev asks 'why didn't we use X?', the AI points them directly to this decision. We're capturing our 'technical memory' as we work."

---

### Part 7: "Zero-friction capture" (2 min)

Show the automated capture via git hooks:

```bash
teamcontext install-hooks
```

Explain the concept: "When we commit a revert or a large deletion, TeamContext watches in the background and auto-records a warning. No manual documentation required."

Ask:

> **"List recent warnings"**

The AI shows auto-detected reverts or pitfalls found in the history.

---

### Part 8: "The agent that follows rules" (1 min)

Show the rules generator:

```bash
teamcontext generate-rules
```

Open the generated `.cursorrules` or `CLAUDE.md`.

**Talking point:** "We've bridged the gap between documentation and execution. The AI agent now has our team's decisions and patterns as binding constraints for every line of code it writes."

---

### Part 9: "Day 1 onboarding" (1 min)

Ask:

> **"I just joined the team, onboard me"**

The AI calls `onboard` and returns a structured walkthrough of the project, key decisions, active warnings, and expert contacts.

**Talking point:** "A new developer gets the 'Team Brain' in one call. No hunting through stale wikis or digging through Slack archives."

---

## 🎯 Closing Pitch

> "TeamContext does four things:
>
> 1. **Remembers** — Decisions and patterns survive when people leave.
> 2. **Captures** — Automated hooks and checkpoints mean zero documentation overhead.
> 3. **Analyzes** — Identifies risk, ownership, and correlations automatically.
> 4. **Governs** — Ensures AI agents and developers follow the team's shared rules.
>
> It's the technical memory that outlives everyone."

---

## ❓ FAQ for Demo

| Question | Answer |
| :--- | :--- |
| "Does it need an API key?" | No. Everything runs locally on your machine. |
| "Does it work with [Service]?" | It reads local git history. No cloud-specific APIs required. |
| "What languages?" | Skeletons support 12+ languages; API extraction covers most modern frameworks. |
| "How do we share it?" | Knowledge is stored in `.teamcontext/` as JSON. Just `git push` it. |
