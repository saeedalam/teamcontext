package cli

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/saeedalam/teamcontext/internal/storage"
	"github.com/saeedalam/teamcontext/pkg/types"
	"github.com/spf13/cobra"
)

var analyzeQuiet bool

var analyzeCommitCmd = &cobra.Command{
	Use:   "analyze-commit",
	Short: "Analyze the latest commit for knowledge extraction",
	Long: `Analyze the latest git commit and auto-extract knowledge.

This command is designed to be called by git hooks (post-commit).
It inspects the commit diff and message to detect and auto-capture:

  DECISIONS (from commit message):
    - Lines starting with "Decision:", "DECISION:", "Why:", "Reason:"
    - Conventional commits: "feat!:", "fix!:", or body with "BREAKING CHANGE:"
    - PR merge commits: extracts from PR description if available

  WARNINGS (from commit message):
    - Lines starting with "Warning:", "CAUTION:", "Gotcha:", "Pitfall:"
    - "BREAKING CHANGE:" sections
    - Reverts (auto-creates a warning)

  AUTOMATIC DETECTION:
    - Large deletions (>100 lines removed)
    - Dependency changes (package.json, go.mod, etc.)
    - Config file changes
    - TODO/FIXME/HACK additions

Use --quiet to suppress output (recommended for git hooks).

Examples:
  teamcontext analyze-commit
  teamcontext analyze-commit --quiet`,
	Run: runAnalyzeCommit,
}

func init() {
	println("[DEBUG] cli/analyze_commit.go: init")
	analyzeCommitCmd.Flags().BoolVarP(&analyzeQuiet, "quiet", "q", false, "Suppress output (for git hooks)")
	rootCmd.AddCommand(analyzeCommitCmd)
}

func runAnalyzeCommit(cmd *cobra.Command, args []string) {
	tcDir, err := findTeamContextDirFromCwd()
	if err != nil {
		if !analyzeQuiet {
			fmt.Printf("Not a TeamContext project, skipping analysis.\n")
		}
		return
	}

	jsonStore := storage.NewJSONStore(tcDir)

	// Get latest commit info
	commitHash, err := gitOutput("rev-parse", "HEAD")
	if err != nil {
		return
	}
	commitMsg, err := gitOutput("log", "-1", "--pretty=%B")
	if err != nil {
		return
	}
	commitAuthor, _ := gitOutput("log", "-1", "--pretty=%an")

	// Get diff stats
	diffStat, _ := gitOutput("diff", "--stat", "HEAD~1..HEAD")
	diffContent, _ := gitOutput("diff", "HEAD~1..HEAD")

	var findings []string
	msgLower := strings.ToLower(commitMsg)

	// =========================================================================
	// AUTO-EXTRACT DECISIONS FROM COMMIT MESSAGE
	// =========================================================================
	decisions := extractDecisionsFromMessage(commitMsg)
	for _, dec := range decisions {
		decision := &types.Decision{
			Content:  dec.title,
			Reason:   dec.reason,
			Context:  fmt.Sprintf("Commit %s by %s", commitHash[:8], commitAuthor),
			Author:   commitAuthor,
			Status:   "active",
			Tags:     []string{"auto-captured", "git-commit"},
		}
		if err := jsonStore.AddDecision(decision); err == nil {
			findings = append(findings, fmt.Sprintf("Decision captured: %s → %s", dec.title, decision.ID))
		}
	}

	// =========================================================================
	// AUTO-EXTRACT WARNINGS FROM COMMIT MESSAGE
	// =========================================================================
	warnings := extractWarningsFromMessage(commitMsg)
	for _, warn := range warnings {
		warning := &types.Warning{
			Content:  warn.title,
			Reason:   warn.reason,
			Evidence: fmt.Sprintf("Commit %s by %s", commitHash[:8], commitAuthor),
			Severity: warn.severity,
			Author:   commitAuthor,
			Tags:     []string{"auto-captured", "git-commit"},
		}
		if err := jsonStore.AddWarning(warning); err == nil {
			findings = append(findings, fmt.Sprintf("Warning captured: %s → %s", warn.title, warning.ID))
		}
	}

	// =========================================================================
	// TRY TO EXTRACT FROM PR DESCRIPTION (if this is a merge commit)
	// =========================================================================
	if strings.Contains(msgLower, "merge pull request") || strings.Contains(msgLower, "merge pr") {
		prDecisions, prWarnings := extractFromPRDescription(commitMsg)
		for _, dec := range prDecisions {
			decision := &types.Decision{
				Content: dec.title,
				Reason:  dec.reason,
				Context: fmt.Sprintf("PR merged in commit %s", commitHash[:8]),
				Author:  commitAuthor,
				Status:  "active",
				Tags:    []string{"auto-captured", "pull-request"},
			}
			if err := jsonStore.AddDecision(decision); err == nil {
				findings = append(findings, fmt.Sprintf("PR Decision: %s → %s", dec.title, decision.ID))
			}
		}
		for _, warn := range prWarnings {
			warning := &types.Warning{
				Content:  warn.title,
				Reason:   warn.reason,
				Evidence: fmt.Sprintf("PR merged in commit %s", commitHash[:8]),
				Severity: warn.severity,
				Author:   commitAuthor,
				Tags:     []string{"auto-captured", "pull-request"},
			}
			if err := jsonStore.AddWarning(warning); err == nil {
				findings = append(findings, fmt.Sprintf("PR Warning: %s → %s", warn.title, warning.ID))
			}
		}
	}

	// =========================================================================
	// EXISTING DETECTION LOGIC
	// =========================================================================

	// 1. Detect reverts
	if strings.Contains(msgLower, "revert") {
		warning := &types.Warning{
			Content:  fmt.Sprintf("Commit reverted: %s", strings.TrimSpace(commitMsg)),
			Reason:   "A previous approach was reverted. Check the original commit for context on what failed.",
			Evidence: fmt.Sprintf("Commit %s by %s", commitHash[:8], commitAuthor),
			Severity: "warning",
			Author:   "teamcontext-auto",
			Tags:     []string{"auto-detected", "revert"},
		}
		if err := jsonStore.AddWarning(warning); err == nil {
			findings = append(findings, fmt.Sprintf("Warning: Revert detected → %s", warning.ID))
		}
	}

	// 2. Detect large deletions
	deletions := countDeletions(diffStat)
	if deletions > 100 {
		warning := &types.Warning{
			Content:  fmt.Sprintf("Large deletion: %d lines removed in commit %s", deletions, commitHash[:8]),
			Reason:   fmt.Sprintf("Significant code removal by %s. Verify the deleted logic is no longer needed.", commitAuthor),
			Evidence: fmt.Sprintf("Commit: %s — %s", commitHash[:8], strings.TrimSpace(commitMsg)),
			Severity: "info",
			Author:   "teamcontext-auto",
			Tags:     []string{"auto-detected", "large-deletion"},
		}
		if err := jsonStore.AddWarning(warning); err == nil {
			findings = append(findings, fmt.Sprintf("Warning: Large deletion (%d lines) → %s", deletions, warning.ID))
		}
	}

	// 3. Detect dependency changes
	depFiles := []string{
		"package.json", "go.mod", "go.sum", "Cargo.toml", "Cargo.lock",
		"requirements.txt", "Pipfile", "pyproject.toml", "Gemfile", "Gemfile.lock",
		"pom.xml", "build.gradle", "composer.json", "pubspec.yaml",
	}
	changedFiles := extractChangedFiles(diffStat)
	for _, cf := range changedFiles {
		base := filepath.Base(cf)
		for _, df := range depFiles {
			if base == df {
				findings = append(findings, fmt.Sprintf("Dependency change: %s modified", cf))
				break
			}
		}
	}

	// 4. Detect config changes
	configPatterns := []string{
		".env", "Dockerfile", "docker-compose", ".yml", ".yaml",
		"tsconfig", "webpack", "vite.config", "next.config", ".eslintrc",
		"Makefile", "CMakeLists",
	}
	for _, cf := range changedFiles {
		base := strings.ToLower(filepath.Base(cf))
		for _, cp := range configPatterns {
			if strings.Contains(base, cp) {
				findings = append(findings, fmt.Sprintf("Config change: %s modified", cf))
				break
			}
		}
	}

	// 5. Detect TODO/FIXME/HACK additions in diff
	if diffContent != "" {
		for _, line := range strings.Split(diffContent, "\n") {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				lineLower := strings.ToLower(line)
				if strings.Contains(lineLower, "todo") || strings.Contains(lineLower, "fixme") || strings.Contains(lineLower, "hack") {
					findings = append(findings, fmt.Sprintf("Code debt marker added: %s", strings.TrimSpace(line[:min(len(line), 80)])))
					break // Only report once per commit
				}
			}
		}
	}

	// Save analysis as an evolution event if there are findings
	if len(findings) > 0 {
		event := &types.EvolutionEvent{
			EventType:   "commit_analysis",
			Title:       fmt.Sprintf("Auto-analyzed commit %s", commitHash[:8]),
			Description: fmt.Sprintf("%s (commit: %s, author: %s)", strings.Join(findings, "; "), commitHash[:8], commitAuthor),
			Author:      "teamcontext-auto",
			Impact:      assessImpact(findings),
		}
		jsonStore.AddEvolutionEvent(event)
	}

	if !analyzeQuiet {
		if len(findings) > 0 {
			fmt.Printf("TeamContext analyzed commit %s:\n", commitHash[:8])
			for _, f := range findings {
				fmt.Printf("  - %s\n", f)
			}
		} else {
			fmt.Printf("TeamContext analyzed commit %s: no notable patterns detected.\n", commitHash[:8])
		}
	}
}

func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func countDeletions(diffStat string) int {
	total := 0
	for _, line := range strings.Split(diffStat, "\n") {
		// Format: " file.go | 25 +---" or " file.go | 10 -"
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		changePart := strings.TrimSpace(parts[1])
		// Count minus signs or parse "X deletions(-)"
		total += strings.Count(changePart, "-")
	}
	return total
}

func extractChangedFiles(diffStat string) []string {
	var files []string
	for _, line := range strings.Split(diffStat, "\n") {
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		file := strings.TrimSpace(parts[0])
		if file != "" && !strings.Contains(file, "changed") {
			files = append(files, file)
		}
	}
	return files
}

func assessImpact(findings []string) string {
	for _, f := range findings {
		if strings.Contains(f, "Revert") || strings.Contains(f, "Large deletion") {
			return "high"
		}
	}
	if len(findings) > 3 {
		return "medium"
	}
	return "low"
}

// min returns the smaller of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =========================================================================
// DECISION & WARNING EXTRACTION FROM COMMIT MESSAGES
// =========================================================================

type extractedDecision struct {
	title  string
	reason string
}

type extractedWarning struct {
	title    string
	reason   string
	severity string
}

// extractDecisionsFromMessage parses commit messages for decision markers.
// Supports multiple formats:
//   - "Decision: <title>" or "DECISION: <title>"
//   - "Why: <reason>" or "Reason: <reason>"
//   - Conventional commits with "!" (breaking change implies a decision)
func extractDecisionsFromMessage(msg string) []extractedDecision {
	var decisions []extractedDecision
	lines := strings.Split(msg, "\n")

	var currentDecision *extractedDecision

	// Decision markers (case-insensitive)
	decisionPrefixes := []string{
		"decision:", "Decision:", "DECISION:",
		"architectural decision:", "arch decision:",
		"design decision:", "tech decision:",
		"chose:", "decided:", "we decided:",
		"going with:", "using:", "switched to:",
	}

	reasonPrefixes := []string{
		"why:", "Why:", "WHY:",
		"reason:", "Reason:", "REASON:",
		"because:", "Because:",
		"rationale:", "Rationale:",
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check for decision markers
		for _, prefix := range decisionPrefixes {
			if strings.HasPrefix(trimmed, prefix) || strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(prefix)) {
				content := strings.TrimSpace(trimmed[len(prefix):])
				if content != "" {
					currentDecision = &extractedDecision{title: content}
				}
				break
			}
		}

		// Check for reason (attaches to current decision)
		if currentDecision != nil {
			for _, prefix := range reasonPrefixes {
				if strings.HasPrefix(trimmed, prefix) || strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(prefix)) {
					content := strings.TrimSpace(trimmed[len(prefix):])
					if content != "" {
						currentDecision.reason = content
						decisions = append(decisions, *currentDecision)
						currentDecision = nil
					}
					break
				}
			}
		}
	}

	// If we have a decision without a reason, still capture it
	if currentDecision != nil && currentDecision.title != "" {
		decisions = append(decisions, *currentDecision)
	}

	// Check for conventional commit with breaking change indicator
	firstLine := strings.TrimSpace(lines[0])
	if strings.Contains(firstLine, "!:") {
		// e.g., "feat!: migrate to Zod validation"
		parts := strings.SplitN(firstLine, "!:", 2)
		if len(parts) == 2 {
			title := strings.TrimSpace(parts[1])
			if title != "" {
				// Check if this wasn't already captured
				alreadyCaptured := false
				for _, d := range decisions {
					if d.title == title {
						alreadyCaptured = true
						break
					}
				}
				if !alreadyCaptured {
					decisions = append(decisions, extractedDecision{
						title:  title,
						reason: "Breaking change - see commit for details",
					})
				}
			}
		}
	}

	return decisions
}

// extractWarningsFromMessage parses commit messages for warning markers.
func extractWarningsFromMessage(msg string) []extractedWarning {
	var warnings []extractedWarning
	lines := strings.Split(msg, "\n")

	warningPrefixes := []string{
		"warning:", "Warning:", "WARNING:",
		"caution:", "Caution:", "CAUTION:",
		"gotcha:", "Gotcha:", "GOTCHA:",
		"pitfall:", "Pitfall:", "PITFALL:",
		"danger:", "Danger:", "DANGER:",
		"note:", "Note:", "NOTE:",
		"important:", "Important:", "IMPORTANT:",
		"beware:", "Beware:", "BEWARE:",
		"don't:", "Don't:", "DON'T:",
		"avoid:", "Avoid:", "AVOID:",
	}

	breakingPrefixes := []string{
		"breaking change:", "Breaking Change:", "BREAKING CHANGE:",
		"breaking:", "Breaking:", "BREAKING:",
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check for warning markers
		for _, prefix := range warningPrefixes {
			if strings.HasPrefix(trimmed, prefix) {
				content := strings.TrimSpace(trimmed[len(prefix):])
				if content != "" {
					severity := "warning"
					if strings.Contains(strings.ToLower(prefix), "danger") ||
						strings.Contains(strings.ToLower(prefix), "important") {
						severity = "critical"
					} else if strings.Contains(strings.ToLower(prefix), "note") {
						severity = "info"
					}
					warnings = append(warnings, extractedWarning{
						title:    content,
						severity: severity,
					})
				}
				break
			}
		}

		// Check for breaking change markers (always critical)
		for _, prefix := range breakingPrefixes {
			if strings.HasPrefix(trimmed, prefix) {
				content := strings.TrimSpace(trimmed[len(prefix):])
				if content != "" {
					warnings = append(warnings, extractedWarning{
						title:    content,
						reason:   "Breaking change introduced in commit",
						severity: "critical",
					})
				}
				break
			}
		}
	}

	return warnings
}

// extractFromPRDescription attempts to fetch and parse PR description.
// Uses GitHub CLI (gh) if available.
func extractFromPRDescription(commitMsg string) ([]extractedDecision, []extractedWarning) {
	var decisions []extractedDecision
	var warnings []extractedWarning

	// Try to extract PR number from merge commit message
	// Format: "Merge pull request #123 from branch" or "Merge PR #123"
	prNumber := ""
	msgLower := strings.ToLower(commitMsg)

	if idx := strings.Index(msgLower, "#"); idx != -1 {
		// Extract the number after #
		numStart := idx + 1
		numEnd := numStart
		for numEnd < len(commitMsg) && commitMsg[numEnd] >= '0' && commitMsg[numEnd] <= '9' {
			numEnd++
		}
		if numEnd > numStart {
			prNumber = commitMsg[numStart:numEnd]
		}
	}

	if prNumber == "" {
		return decisions, warnings
	}

	// Try to fetch PR description using gh CLI
	out, err := exec.Command("gh", "pr", "view", prNumber, "--json", "body", "--jq", ".body").Output()
	if err != nil {
		// gh not available or not authenticated - silently skip
		return decisions, warnings
	}

	prBody := strings.TrimSpace(string(out))
	if prBody == "" || prBody == "null" {
		return decisions, warnings
	}

	// Parse PR body for decisions and warnings
	decisions = append(decisions, extractDecisionsFromMessage(prBody)...)
	warnings = append(warnings, extractWarningsFromMessage(prBody)...)

	// Also look for common PR template sections
	sections := map[string]string{
		"## decisions":        "decision",
		"## architectural":    "decision",
		"### decisions":       "decision",
		"## warnings":         "warning",
		"## breaking changes": "warning",
		"### breaking":        "warning",
		"## notes":            "warning",
	}

	lines := strings.Split(prBody, "\n")
	currentSection := ""
	sectionContent := []string{}

	for _, line := range lines {
		lineLower := strings.ToLower(strings.TrimSpace(line))

		// Check if this is a section header
		newSection := ""
		for header, sType := range sections {
			if strings.HasPrefix(lineLower, header) {
				newSection = sType
				break
			}
		}

		if newSection != "" {
			// Save previous section
			if currentSection != "" && len(sectionContent) > 0 {
				content := strings.TrimSpace(strings.Join(sectionContent, " "))
				if content != "" {
					if currentSection == "decision" {
						decisions = append(decisions, extractedDecision{
							title:  content,
							reason: "From PR description",
						})
					} else {
						warnings = append(warnings, extractedWarning{
							title:    content,
							reason:   "From PR description",
							severity: "warning",
						})
					}
				}
			}
			currentSection = newSection
			sectionContent = []string{}
		} else if currentSection != "" && strings.TrimSpace(line) != "" {
			// Accumulate content for current section
			// Skip bullet points marker
			content := strings.TrimSpace(line)
			content = strings.TrimPrefix(content, "- ")
			content = strings.TrimPrefix(content, "* ")
			if content != "" && !strings.HasPrefix(content, "#") {
				sectionContent = append(sectionContent, content)
			}
		}
	}

	// Don't forget last section
	if currentSection != "" && len(sectionContent) > 0 {
		content := strings.TrimSpace(strings.Join(sectionContent, " "))
		if content != "" {
			if currentSection == "decision" {
				decisions = append(decisions, extractedDecision{
					title:  content,
					reason: "From PR description",
				})
			} else {
				warnings = append(warnings, extractedWarning{
					title:    content,
					reason:   "From PR description",
					severity: "warning",
				})
			}
		}
	}

	return decisions, warnings
}

// The git hook calls `teamcontext analyze-commit --quiet` which runs this function.

