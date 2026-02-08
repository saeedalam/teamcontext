package search

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	"github.com/saeedalam/teamcontext/pkg/types"
)

// RipgrepMatch represents a single ripgrep match
type RipgrepMatch struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		Lines struct {
			Text string `json:"text"`
		} `json:"lines"`
		LineNumber int `json:"line_number"`
	} `json:"data"`
}

// SearchCode searches code content using ripgrep
func SearchCode(pattern string, path string, glob string, limit int) ([]types.CodeMatch, error) {
	if limit <= 0 {
		limit = 50
	}

	args := []string{
		"--json",
		"--max-count", strconv.Itoa(limit),
	}

	if glob != "" {
		args = append(args, "--glob", glob)
	}

	args = append(args, pattern, path)

	cmd := exec.Command("rg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		// ripgrep not found, fallback to basic grep
		return searchCodeFallback(pattern, path, limit)
	}

	var matches []types.CodeMatch
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() && len(matches) < limit {
		line := scanner.Text()

		var match RipgrepMatch
		if err := json.Unmarshal([]byte(line), &match); err != nil {
			continue
		}

		if match.Type == "match" {
			matches = append(matches, types.CodeMatch{
				Path:    match.Data.Path.Text,
				Line:    match.Data.LineNumber,
				Content: strings.TrimSpace(match.Data.Lines.Text),
			})
		}
	}

	cmd.Wait()

	return matches, nil
}

// searchCodeFallback uses grep when ripgrep is not available
func searchCodeFallback(pattern string, path string, limit int) ([]types.CodeMatch, error) {
	args := []string{
		"-r",             // recursive
		"-n",             // line numbers
		"-H",             // filename
		"--include=*.go", // default to go files
		"--include=*.ts",
		"--include=*.js",
		"--include=*.py",
		"--include=*.java",
		"--include=*.rs",
		pattern,
		path,
	}

	cmd := exec.Command("grep", args...)
	output, err := cmd.Output()
	if err != nil {
		// grep returns 1 if no matches
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []types.CodeMatch{}, nil
		}
		return nil, err
	}

	var matches []types.CodeMatch
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if line == "" || len(matches) >= limit {
			continue
		}

		// Parse grep output: filename:linenum:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			lineNum, _ := strconv.Atoi(parts[1])
			matches = append(matches, types.CodeMatch{
				Path:    parts[0],
				Line:    lineNum,
				Content: strings.TrimSpace(parts[2]),
			})
		}
	}

	return matches, nil
}

// CheckRipgrep checks if ripgrep is installed
func CheckRipgrep() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}
