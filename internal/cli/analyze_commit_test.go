package cli

import (
	"testing"
)

// =============================================================================
// DECISION EXTRACTION TESTS
// =============================================================================

func TestExtractDecisionFromMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected int
		content  string
	}{
		{
			name: "Simple decision marker",
			message: `feat: add user validation

Decision: Use Zod for validation
Why: Better TypeScript inference than class-validator`,
			expected: 1,
			content:  "Use Zod for validation",
		},
		{
			name: "Decision with DECISION uppercase",
			message: `DECISION: Switch to PostgreSQL
Reason: Better JSON support than MySQL`,
			expected: 1,
			content:  "Switch to PostgreSQL",
		},
		{
			name: "Multiple decisions",
			message: `Decision: Use JWT for auth
Why: Stateless authentication

Decision: Use Redis for caching
Reason: Fast in-memory storage`,
			expected: 2,
			content:  "Use JWT for auth",
		},
		{
			name: "Breaking change conventional commit",
			message: `feat!: migrate to Zod validation

BREAKING CHANGE: All DTOs now use Zod schemas`,
			expected: 1,
			content:  "migrate to Zod validation",
		},
		{
			name: "No decisions",
			message: `fix: correct typo in readme

This is just a minor fix.`,
			expected: 0,
			content:  "",
		},
		{
			name: "Decision without reason",
			message: `Decision: Use monorepo structure`,
			expected: 1,
			content:  "Use monorepo structure",
		},
		{
			name: "Decided marker",
			message: `decided: go with Option A for the database

We evaluated several options and Option A fits best.`,
			expected: 1,
			content:  "go with Option A for the database",
		},
		{
			name: "Tech decision marker",
			message: `tech decision: Use GraphQL instead of REST
Because: Better type safety and flexible queries`,
			expected: 1,
			content:  "Use GraphQL instead of REST",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decisions := extractDecisionsFromMessage(tt.message)
			
			if len(decisions) != tt.expected {
				t.Errorf("Expected %d decisions, got %d", tt.expected, len(decisions))
				for i, d := range decisions {
					t.Logf("Decision %d: %s", i, d.title)
				}
			}
			
			if tt.expected > 0 && len(decisions) > 0 {
				if decisions[0].title != tt.content {
					t.Errorf("Expected decision content '%s', got '%s'", tt.content, decisions[0].title)
				}
			}
		})
	}
}

func TestExtractDecisionWithReason(t *testing.T) {
	message := `Decision: Use dependency injection
Why: Better testability and decoupling`
	
	decisions := extractDecisionsFromMessage(message)
	
	if len(decisions) != 1 {
		t.Fatalf("Expected 1 decision, got %d", len(decisions))
	}
	
	if decisions[0].title != "Use dependency injection" {
		t.Errorf("Expected title 'Use dependency injection', got '%s'", decisions[0].title)
	}
	
	if decisions[0].reason != "Better testability and decoupling" {
		t.Errorf("Expected reason 'Better testability and decoupling', got '%s'", decisions[0].reason)
	}
}

// =============================================================================
// WARNING EXTRACTION TESTS
// =============================================================================

func TestExtractWarningFromMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected int
		content  string
		severity string
	}{
		{
			name: "Simple warning marker",
			message: `fix: handle edge case

Warning: Don't call this API without auth token`,
			expected: 1,
			content:  "Don't call this API without auth token",
			severity: "warning",
		},
		{
			name: "Caution marker",
			message: `CAUTION: This changes the response format`,
			expected: 1,
			content:  "This changes the response format",
			severity: "warning",
		},
		{
			name: "Breaking change warning",
			message: `BREAKING CHANGE: API v1 endpoints are now deprecated`,
			expected: 1,
			content:  "API v1 endpoints are now deprecated",
			severity: "critical",
		},
		{
			name: "Danger marker",
			message: `Danger: Never expose this endpoint publicly`,
			expected: 1,
			content:  "Never expose this endpoint publicly",
			severity: "critical",
		},
		{
			name: "Note marker",
			message: `Note: This requires environment variable X to be set`,
			expected: 1,
			content:  "This requires environment variable X to be set",
			severity: "info",
		},
		{
			name: "Multiple warnings",
			message: `Warning: Don't use in production
Caution: Requires manual migration`,
			expected: 2,
			content:  "Don't use in production",
			severity: "warning",
		},
		{
			name: "No warnings",
			message: `feat: add new feature

This is a regular commit.`,
			expected: 0,
			content:  "",
			severity: "",
		},
		{
			name: "Gotcha marker",
			message: `Gotcha: The cache doesn't invalidate automatically`,
			expected: 1,
			content:  "The cache doesn't invalidate automatically",
			severity: "warning",
		},
		{
			name: "Pitfall marker",
			message: `Pitfall: Don't call this synchronously`,
			expected: 1,
			content:  "Don't call this synchronously",
			severity: "warning",
		},
		{
			name: "Important marker",
			message: `Important: Run migrations before deploying`,
			expected: 1,
			content:  "Run migrations before deploying",
			severity: "critical",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := extractWarningsFromMessage(tt.message)
			
			if len(warnings) != tt.expected {
				t.Errorf("Expected %d warnings, got %d", tt.expected, len(warnings))
				for i, w := range warnings {
					t.Logf("Warning %d: %s (severity: %s)", i, w.title, w.severity)
				}
			}
			
			if tt.expected > 0 && len(warnings) > 0 {
				if warnings[0].title != tt.content {
					t.Errorf("Expected warning content '%s', got '%s'", tt.content, warnings[0].title)
				}
				if warnings[0].severity != tt.severity {
					t.Errorf("Expected severity '%s', got '%s'", tt.severity, warnings[0].severity)
				}
			}
		})
	}
}

// =============================================================================
// PR DESCRIPTION EXTRACTION TESTS
// =============================================================================

func TestExtractFromPRDescription(t *testing.T) {
	// Note: This test doesn't actually call GitHub, it tests the parsing logic
	
	prBody := `## Summary
This PR adds user authentication.

## Decisions
- Switched from session-based to JWT authentication
- Using bcrypt for password hashing

## Warnings
- JWT tokens cannot be revoked individually
- Must use HTTPS in production

## Breaking Changes
The old session endpoints are deprecated.
`
	
	// Test decision extraction from PR body
	decisions := extractDecisionsFromMessage(prBody)
	
	// Should find decisions from the PR body markers
	t.Logf("Found %d decisions from PR body", len(decisions))
	for i, d := range decisions {
		t.Logf("Decision %d: %s", i, d.title)
	}
	
	// Test warning extraction from PR body
	warnings := extractWarningsFromMessage(prBody)
	
	t.Logf("Found %d warnings from PR body", len(warnings))
	for i, w := range warnings {
		t.Logf("Warning %d: %s (severity: %s)", i, w.title, w.severity)
	}
}

// =============================================================================
// EDGE CASES
// =============================================================================

func TestEmptyMessage(t *testing.T) {
	decisions := extractDecisionsFromMessage("")
	if len(decisions) != 0 {
		t.Errorf("Expected 0 decisions from empty message, got %d", len(decisions))
	}
	
	warnings := extractWarningsFromMessage("")
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings from empty message, got %d", len(warnings))
	}
}

func TestMessageWithOnlyWhitespace(t *testing.T) {
	message := "   \n\n   \t\t   \n"
	
	decisions := extractDecisionsFromMessage(message)
	if len(decisions) != 0 {
		t.Errorf("Expected 0 decisions from whitespace message, got %d", len(decisions))
	}
	
	warnings := extractWarningsFromMessage(message)
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings from whitespace message, got %d", len(warnings))
	}
}

func TestCaseInsensitivity(t *testing.T) {
	testCases := []string{
		"decision: Use X",
		"Decision: Use X",
		"DECISION: Use X",
	}
	
	for _, msg := range testCases {
		decisions := extractDecisionsFromMessage(msg)
		if len(decisions) != 1 {
			t.Errorf("Failed to extract decision from '%s'", msg)
		}
	}
}

func TestDecisionMarkerInMiddleOfLine(t *testing.T) {
	// Decision marker should only match at start of line
	message := `This is not a decision: just some text
Decision: This is a real decision`
	
	decisions := extractDecisionsFromMessage(message)
	
	if len(decisions) != 1 {
		t.Errorf("Expected 1 decision, got %d", len(decisions))
	}
	
	if len(decisions) > 0 && decisions[0].title != "This is a real decision" {
		t.Errorf("Expected 'This is a real decision', got '%s'", decisions[0].title)
	}
}

func TestConventionalCommitBreakingChange(t *testing.T) {
	// Test feat! format
	message := `feat!: remove deprecated API endpoints

All v1 endpoints have been removed.`
	
	decisions := extractDecisionsFromMessage(message)
	
	if len(decisions) != 1 {
		t.Errorf("Expected 1 decision from feat! commit, got %d", len(decisions))
	}
	
	if len(decisions) > 0 && decisions[0].title != "remove deprecated API endpoints" {
		t.Errorf("Expected 'remove deprecated API endpoints', got '%s'", decisions[0].title)
	}
}

// =============================================================================
// HELPER FUNCTION TESTS
// =============================================================================

func TestCountDeletions(t *testing.T) {
	tests := []struct {
		name     string
		diffStat string
		expected int
	}{
		{
			name: "Simple deletion",
			diffStat: ` file.go | 10 ----`,
			expected: 4,
		},
		{
			name: "Mixed changes",
			diffStat: ` file1.go | 5 ++---
 file2.go | 10 ++++++----`,
			expected: 7, // 3 + 4
		},
		{
			name: "No deletions",
			diffStat: ` file.go | 10 ++++++++++`,
			expected: 0,
		},
		{
			name: "Empty diff",
			diffStat: ``,
			expected: 0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countDeletions(tt.diffStat)
			if result != tt.expected {
				t.Errorf("Expected %d deletions, got %d", tt.expected, result)
			}
		})
	}
}

func TestExtractChangedFiles(t *testing.T) {
	diffStat := ` src/auth/login.ts | 25 +++---
 src/auth/logout.ts | 10 ++
 package.json | 5 +-
 3 files changed, 30 insertions(+), 10 deletions(-)`
	
	files := extractChangedFiles(diffStat)
	
	expected := []string{"src/auth/login.ts", "src/auth/logout.ts", "package.json"}
	
	if len(files) != len(expected) {
		t.Errorf("Expected %d files, got %d", len(expected), len(files))
	}
	
	for i, exp := range expected {
		if i < len(files) && files[i] != exp {
			t.Errorf("Expected file '%s', got '%s'", exp, files[i])
		}
	}
}

func TestAssessImpact(t *testing.T) {
	tests := []struct {
		name     string
		findings []string
		expected string
	}{
		{
			name:     "High impact - revert",
			findings: []string{"Revert detected"},
			expected: "high",
		},
		{
			name:     "High impact - large deletion",
			findings: []string{"Large deletion: 150 lines removed"},
			expected: "high",
		},
		{
			name:     "Medium impact - multiple findings",
			findings: []string{"Config change", "Dep change", "Another finding", "Fourth finding"},
			expected: "medium",
		},
		{
			name:     "Low impact - few findings",
			findings: []string{"Config change"},
			expected: "low",
		},
		{
			name:     "Low impact - empty",
			findings: []string{},
			expected: "low",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assessImpact(tt.findings)
			if result != tt.expected {
				t.Errorf("Expected impact '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
