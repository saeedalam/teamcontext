package mcp

import "testing"

func TestContainsAny(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		text     string
		query    string
		expected bool
	}{
		{
			name:     "returns false for empty query",
			text:     "alpha beta gamma",
			query:    "   ",
			expected: false,
		},
		{
			name:     "matches full phrase when present",
			text:     "project contains auth timeout handling",
			query:    "auth timeout",
			expected: true,
		},
		{
			name:     "matches any query word when phrase missing",
			text:     "project contains auth module only",
			query:    "payment auth",
			expected: true,
		},
		{
			name:     "returns false when no token overlaps",
			text:     "project contains storage adapters",
			query:    "network queue",
			expected: false,
		},
		{
			name:     "matches case insensitively",
			text:     "TeamContext MCP Server",
			query:    "teamcontext",
			expected: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			actual := containsAny(testCase.text, testCase.query)
			if actual != testCase.expected {
				t.Errorf("containsAny(%q, %q) = %t; want %t", testCase.text, testCase.query, actual, testCase.expected)
			}
		})
	}
}
