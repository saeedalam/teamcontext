package git

import (
	"bytes"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/saeedalam/teamcontext/pkg/types"
)

// GetRecentChanges returns recent git commits with metadata
func GetRecentChanges(repoPath string, since string, limit int) ([]types.GitChange, error) {
	if limit <= 0 {
		limit = 20
	}

	args := []string{
		"log",
		"--pretty=format:%H|%h|%s|%an|%ae|%aI",
		"--numstat",
		"-n", strconv.Itoa(limit),
	}

	if since != "" {
		args = append(args, "--since="+since)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseGitLog(string(output))
}

// GetRecentChangesForPath returns changes for a specific path
func GetRecentChangesForPath(repoPath, filePath string, limit int) ([]types.GitChange, error) {
	if limit <= 0 {
		limit = 10
	}

	args := []string{
		"log",
		"--pretty=format:%H|%h|%s|%an|%ae|%aI",
		"--numstat",
		"-n", strconv.Itoa(limit),
		"--", filePath,
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseGitLog(string(output))
}

// GetFileDiff returns the diff for a specific file
func GetFileDiff(repoPath, filePath string, compareWith string) (*types.GitDiff, error) {
	if compareWith == "" {
		compareWith = "HEAD~1"
	}

	args := []string{"diff", compareWith, "--", filePath}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		// No diff available (maybe new file or no changes)
		return &types.GitDiff{Path: filePath, Status: "unchanged"}, nil
	}

	diff := &types.GitDiff{
		Path: filePath,
	}

	diffStr := string(output)
	if diffStr == "" {
		diff.Status = "unchanged"
		return diff, nil
	}

	diff.Status = "modified"

	// Count insertions and deletions
	lines := strings.Split(diffStr, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			diff.Insertions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			diff.Deletions++
		}
	}

	// Extract hunks
	hunkPattern := regexp.MustCompile(`(?m)^@@.+@@`)
	hunkMatches := hunkPattern.FindAllStringIndex(diffStr, -1)
	for i, match := range hunkMatches {
		end := len(diffStr)
		if i < len(hunkMatches)-1 {
			end = hunkMatches[i+1][0]
		}
		hunk := diffStr[match[0]:end]
		if len(hunk) > 2000 {
			hunk = hunk[:2000] + "\n... (truncated)"
		}
		diff.Hunks = append(diff.Hunks, hunk)
	}

	return diff, nil
}

// GetUncommittedChanges returns list of uncommitted file changes
func GetUncommittedChanges(repoPath string) ([]types.GitDiff, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var diffs []types.GitDiff
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		status := strings.TrimSpace(line[:2])
		filePath := strings.TrimSpace(line[3:])

		diff := types.GitDiff{
			Path: filePath,
		}

		switch {
		case strings.Contains(status, "A"):
			diff.Status = "added"
		case strings.Contains(status, "M"):
			diff.Status = "modified"
		case strings.Contains(status, "D"):
			diff.Status = "deleted"
		case strings.Contains(status, "R"):
			diff.Status = "renamed"
			parts := strings.Split(filePath, " -> ")
			if len(parts) == 2 {
				diff.OldPath = parts[0]
				diff.Path = parts[1]
			}
		case strings.Contains(status, "?"):
			diff.Status = "untracked"
		default:
			diff.Status = "unknown"
		}

		diffs = append(diffs, diff)
	}

	return diffs, nil
}

// GetBranch returns the current branch name
func GetBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GetFileHistory returns the history of a specific file
func GetFileHistory(repoPath, filePath string, limit int) ([]types.GitChange, error) {
	if limit <= 0 {
		limit = 10
	}

	args := []string{
		"log",
		"--follow",
		"--pretty=format:%H|%h|%s|%an|%ae|%aI",
		"-n", strconv.Itoa(limit),
		"--", filePath,
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseGitLogSimple(string(output))
}

// Helper functions

func parseGitLog(output string) ([]types.GitChange, error) {
	var changes []types.GitChange

	// Split by blank lines (commits are separated by blank lines after numstat)
	blocks := splitCommitBlocks(output)

	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		if len(lines) == 0 {
			continue
		}

		// First line is the commit info
		parts := strings.Split(lines[0], "|")
		if len(parts) < 6 {
			continue
		}

		change := types.GitChange{
			Hash:        parts[0],
			ShortHash:   parts[1],
			Message:     parts[2],
			Author:      parts[3],
			AuthorEmail: parts[4],
		}

		// Parse date
		if t, err := time.Parse(time.RFC3339, parts[5]); err == nil {
			change.Date = t
		}

		// Parse numstat lines (remaining lines)
		for i := 1; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}

			statParts := strings.Fields(line)
			if len(statParts) >= 3 {
				if ins, err := strconv.Atoi(statParts[0]); err == nil {
					change.Insertions += ins
				}
				if del, err := strconv.Atoi(statParts[1]); err == nil {
					change.Deletions += del
				}
				change.FilesChanged = append(change.FilesChanged, statParts[2])
			}
		}

		// Estimate impact level
		change.ImpactLevel = estimateImpact(change)

		changes = append(changes, change)
	}

	return changes, nil
}

func parseGitLogSimple(output string) ([]types.GitChange, error) {
	var changes []types.GitChange

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}

		change := types.GitChange{
			Hash:        parts[0],
			ShortHash:   parts[1],
			Message:     parts[2],
			Author:      parts[3],
			AuthorEmail: parts[4],
		}

		if t, err := time.Parse(time.RFC3339, parts[5]); err == nil {
			change.Date = t
		}

		changes = append(changes, change)
	}

	return changes, nil
}

func splitCommitBlocks(output string) []string {
	var blocks []string
	var current bytes.Buffer

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// New commit starts with a hash (40 hex chars followed by |)
		if len(line) > 41 && line[40] == '|' && isHexString(line[:40]) {
			if current.Len() > 0 {
				blocks = append(blocks, current.String())
				current.Reset()
			}
		}
		current.WriteString(line)
		current.WriteString("\n")
	}

	if current.Len() > 0 {
		blocks = append(blocks, current.String())
	}

	return blocks
}

func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func estimateImpact(change types.GitChange) string {
	totalChanges := change.Insertions + change.Deletions
	fileCount := len(change.FilesChanged)

	if totalChanges > 500 || fileCount > 20 {
		return "high"
	}
	if totalChanges > 100 || fileCount > 5 {
		return "medium"
	}
	return "low"
}
