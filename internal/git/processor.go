package git

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GitHistoryReport contains all computed git history analysis
type GitHistoryReport struct {
	Summary      GitSummary        `json:"summary"`
	Experts      []DirectoryExpert `json:"experts"`
	Risks        []KnowledgeRisk   `json:"risks"`
	Correlations []FileCorrelation `json:"correlations"`

	// Metadata
	ProcessedAt time.Time `json:"processed_at"`
	CommitCount int       `json:"commit_count"`
	Contributors int      `json:"contributors"`
}

// GitSummary contains high-level repo stats
type GitSummary struct {
	TotalCommits    int               `json:"total_commits"`
	Contributors    []ContributorInfo `json:"contributors"`
	FirstCommitDate string            `json:"first_commit_date"`
	LastCommitDate  string            `json:"last_commit_date"`
	ActiveBranch    string            `json:"active_branch"`
	TopFiles        []FileActivity    `json:"top_files"`
}

// ContributorInfo is a summary-level contributor record
type ContributorInfo struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Commits      int    `json:"commits"`
	Added        int    `json:"lines_added"`
	Removed      int    `json:"lines_removed"`
	Active       bool   `json:"active"`
	ActiveInRepo string `json:"active_in_repo,omitempty"` // which linked repo they're active in
}

// FileActivity tracks how active a file is
type FileActivity struct {
	Path    string `json:"path"`
	Changes int    `json:"changes"`
}

// DirectoryExpert maps a directory to its top contributors
type DirectoryExpert struct {
	Directory    string          `json:"directory"`
	FileCount    int             `json:"file_count"`
	TotalCommits int            `json:"total_commits"`
	TopExperts   []ExpertEntry   `json:"top_experts"`
}

// ExpertEntry is a contributor's expertise in a directory
type ExpertEntry struct {
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Commits      int       `json:"commits"`
	Ownership    float64   `json:"ownership"`
	Active       bool      `json:"active"`
	ActiveInRepo string    `json:"active_in_repo,omitempty"` // which linked repo they're active in
	LastCommit   time.Time `json:"last_commit,omitempty"`
}

// ProcessGitHistory runs a single git log pass and computes all reports.
// Reads all commits on the main branch (main or master) for full ownership
// history, capped at 2000 to avoid OOM on very large repos.
func ProcessGitHistory(repoPath string) (*GitHistoryReport, error) {
	// Detect default branch: try main, then master, then HEAD
	defaultBranch := detectDefaultBranch(repoPath)

	// Full history on default branch only — gives accurate ownership without
	// feature-branch noise. Capped at 2000 commits for safety.
	args := []string{
		"log",
		defaultBranch,
		"--pretty=format:%H|%h|%an|%ae|%aI|%s",
		"--numstat",
		"-n", "2000",
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		// Fallback: if the branch doesn't exist, try --all with the old approach
		args = []string{
			"log",
			"--all",
			"--pretty=format:%H|%h|%an|%ae|%aI|%s",
			"--numstat",
			"-n", "2000",
		}
		cmd = exec.Command("git", args...)
		cmd.Dir = repoPath
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git log failed: %w", err)
		}
	}

	// Parse all commits
	commits := parseProcessorLog(string(output))
	if len(commits) == 0 {
		return &GitHistoryReport{
			ProcessedAt: time.Now(),
		}, nil
	}

	// Get current branch
	branch := ""
	branchCmd := exec.Command("git", "branch", "--show-current")
	branchCmd.Dir = repoPath
	if branchOut, err := branchCmd.Output(); err == nil {
		branch = strings.TrimSpace(string(branchOut))
	}

	// Build in-memory structures from the single pass
	contributorMap := make(map[string]*ContributorInfo)
	fileChangeCount := make(map[string]int)
	dirContributors := make(map[string]map[string]*ExpertEntry) // dir -> email -> expert
	commitFiles := make([][]string, 0, len(commits))           // for correlation

	threeMonthsAgo := time.Now().AddDate(0, -3, 0)

	for _, c := range commits {
		// Contributor stats
		ci, exists := contributorMap[c.AuthorEmail]
		if !exists {
			ci = &ContributorInfo{
				Name:  c.Author,
				Email: c.AuthorEmail,
			}
			contributorMap[c.AuthorEmail] = ci
		}
		ci.Commits++
		ci.Added += c.Insertions
		ci.Removed += c.Deletions
		if c.Date.After(threeMonthsAgo) {
			ci.Active = true
		}

		// File and directory stats
		for _, f := range c.FilesChanged {
			fileChangeCount[f]++

			dir := filepath.Dir(f)
			if dirContributors[dir] == nil {
				dirContributors[dir] = make(map[string]*ExpertEntry)
			}
			entry, exists := dirContributors[dir][c.AuthorEmail]
			if !exists {
				entry = &ExpertEntry{
					Name:  c.Author,
					Email: c.AuthorEmail,
				}
				dirContributors[dir][c.AuthorEmail] = entry
			}
			entry.Commits++
			if c.Date.After(entry.LastCommit) {
				entry.LastCommit = c.Date
			}
			if c.Date.After(threeMonthsAgo) {
				entry.Active = true
			}
		}

		commitFiles = append(commitFiles, c.FilesChanged)
	}

	// === Build Summary ===
	summary := GitSummary{
		TotalCommits:    len(commits),
		FirstCommitDate: commits[len(commits)-1].Date.Format(time.RFC3339),
		LastCommitDate:  commits[0].Date.Format(time.RFC3339),
		ActiveBranch:    branch,
	}

	for _, ci := range contributorMap {
		summary.Contributors = append(summary.Contributors, *ci)
	}
	sort.Slice(summary.Contributors, func(i, j int) bool {
		return summary.Contributors[i].Commits > summary.Contributors[j].Commits
	})

	// Top 20 most-changed files
	type fileStat struct {
		path    string
		changes int
	}
	var fileStats []fileStat
	for path, count := range fileChangeCount {
		fileStats = append(fileStats, fileStat{path, count})
	}
	sort.Slice(fileStats, func(i, j int) bool {
		return fileStats[i].changes > fileStats[j].changes
	})
	limit := 20
	if len(fileStats) < limit {
		limit = len(fileStats)
	}
	for _, fs := range fileStats[:limit] {
		summary.TopFiles = append(summary.TopFiles, FileActivity{
			Path:    fs.path,
			Changes: fs.changes,
		})
	}

	// === Build Experts ===
	var experts []DirectoryExpert
	for dir, contributors := range dirContributors {
		de := DirectoryExpert{
			Directory: dir,
		}

		totalDirCommits := 0
		for _, entry := range contributors {
			totalDirCommits += entry.Commits
		}
		de.TotalCommits = totalDirCommits

		// Count unique files in this dir
		for f := range fileChangeCount {
			if filepath.Dir(f) == dir {
				de.FileCount++
			}
		}

		// Build sorted expert list with ownership
		for _, entry := range contributors {
			if totalDirCommits > 0 {
				entry.Ownership = float64(entry.Commits) / float64(totalDirCommits)
			}
			de.TopExperts = append(de.TopExperts, *entry)
		}
		sort.Slice(de.TopExperts, func(i, j int) bool {
			return de.TopExperts[i].Ownership > de.TopExperts[j].Ownership
		})
		// Keep top 5 per directory
		if len(de.TopExperts) > 5 {
			de.TopExperts = de.TopExperts[:5]
		}

		experts = append(experts, de)
	}
	sort.Slice(experts, func(i, j int) bool {
		return experts[i].TotalCommits > experts[j].TotalCommits
	})

	// === Build Risks ===
	var risks []KnowledgeRisk
	for dir, contributors := range dirContributors {
		// Skip trivial directories
		fileCount := 0
		for f := range fileChangeCount {
			if filepath.Dir(f) == dir {
				fileCount++
			}
		}
		if fileCount < 2 {
			continue
		}

		totalDirCommits := 0
		for _, entry := range contributors {
			totalDirCommits += entry.Commits
		}

		// Find top contributor
		var topEntry ExpertEntry
		found := false
		for _, entry := range contributors {
			if !found || entry.Commits > topEntry.Commits {
				topEntry = *entry
				found = true
			}
		}
		if !found {
			continue
		}

		ownership := float64(topEntry.Commits) / float64(totalDirCommits)

		// Cross-reference: check if expert is globally active even if directory-inactive
		globallyActive := false
		if ci, exists := contributorMap[topEntry.Email]; exists {
			globallyActive = ci.Active
		}

		// Use global activity to determine expert active status
		expertActive := topEntry.Active || globallyActive

		risk := KnowledgeRisk{
			Area:           dir,
			PrimaryExpert:  topEntry.Name,
			ExpertIsActive: expertActive,
		}

		// Populate LastActivity from expert's last commit in this directory
		if !topEntry.LastCommit.IsZero() {
			daysSince := int(time.Since(topEntry.LastCommit).Hours() / 24)
			if daysSince <= 7 {
				risk.LastActivity = "Within the last week"
			} else if daysSince <= 30 {
				risk.LastActivity = fmt.Sprintf("%d days ago", daysSince)
			} else if daysSince <= 90 {
				risk.LastActivity = fmt.Sprintf("%d months ago", daysSince/30)
			} else if daysSince <= 365 {
				risk.LastActivity = fmt.Sprintf("%d months ago", daysSince/30)
			} else {
				risk.LastActivity = "Over a year ago"
			}
		}

		if ownership > 0.8 {
			if !expertActive {
				risk.RiskLevel = "CRITICAL"
				risk.Reason = fmt.Sprintf("Primary contributor (%s) owns %.0f%% and is inactive", topEntry.Name, ownership*100)
				risk.Mitigation = "Urgent: Assign new maintainer, schedule knowledge transfer"
			} else if !topEntry.Active && globallyActive {
				// Globally active but hasn't touched this directory recently
				risk.RiskLevel = "MEDIUM"
				risk.Reason = fmt.Sprintf("Single contributor (%s) owns %.0f%%; active elsewhere but not in this area recently", topEntry.Name, ownership*100)
				risk.Mitigation = "Consider pairing to spread knowledge"
			} else {
				risk.RiskLevel = "MEDIUM"
				risk.Reason = fmt.Sprintf("Single contributor (%s) owns %.0f%% of this area", topEntry.Name, ownership*100)
				risk.Mitigation = "Consider pairing to spread knowledge"
			}
		} else if !expertActive && len(contributors) < 3 {
			risk.RiskLevel = "HIGH"
			risk.Reason = "Top contributor is no longer active and few contributors"
			risk.Mitigation = "Identify current maintainer or assign one"
		} else {
			risk.RiskLevel = "LOW"
			risk.Reason = "Knowledge is distributed among active contributors"
			risk.Mitigation = "None needed"
		}

		// Only include medium+ risk
		if risk.RiskLevel != "LOW" {
			risks = append(risks, risk)
		}
	}

	riskOrder := map[string]int{"CRITICAL": 0, "HIGH": 1, "MEDIUM": 2, "LOW": 3}
	sort.Slice(risks, func(i, j int) bool {
		return riskOrder[risks[i].RiskLevel] < riskOrder[risks[j].RiskLevel]
	})

	// === Build Correlations ===
	// Count co-changes: for each commit, every pair of files incremented
	coChangeCount := make(map[string]int) // "fileA\x00fileB" -> count
	fileCommitCount := make(map[string]int)

	for _, files := range commitFiles {
		if len(files) > 30 {
			continue // skip huge commits (merges, bulk changes)
		}
		for _, f := range files {
			fileCommitCount[f]++
		}
		for i := 0; i < len(files); i++ {
			for j := i + 1; j < len(files); j++ {
				a, b := files[i], files[j]
				if a > b {
					a, b = b, a
				}
				coChangeCount[a+"\x00"+b]++
			}
		}
	}

	var correlations []FileCorrelation
	for key, count := range coChangeCount {
		if count < 5 {
			continue
		}
		parts := strings.SplitN(key, "\x00", 2)
		if len(parts) != 2 {
			continue
		}
		a, b := parts[0], parts[1]

		// Correlation = coChanges / min(commits(a), commits(b))
		minCommits := fileCommitCount[a]
		if fileCommitCount[b] < minCommits {
			minCommits = fileCommitCount[b]
		}
		if minCommits == 0 {
			continue
		}

		corr := float64(count) / float64(minCommits)
		if corr < 0.5 {
			continue
		}

		conf := "medium"
		if count > 15 {
			conf = "high"
		} else if count <= 8 {
			// Below medium threshold — skip (only store medium+ confidence)
			continue
		}

		correlations = append(correlations, FileCorrelation{
			File1:       a,
			File2:       b,
			CoChanges:   count,
			Correlation: corr,
			Confidence:  conf,
		})
	}

	sort.Slice(correlations, func(i, j int) bool {
		return correlations[i].Correlation > correlations[j].Correlation
	})
	if len(correlations) > 50 {
		correlations = correlations[:50]
	}

	return &GitHistoryReport{
		Summary:      summary,
		Experts:      experts,
		Risks:        risks,
		Correlations: correlations,
		ProcessedAt:  time.Now(),
		CommitCount:  len(commits),
		Contributors: len(contributorMap),
	}, nil
}

// WriteReportFiles writes the report to individual JSON files in the knowledge dir
func WriteReportFiles(report *GitHistoryReport, knowledgeDir string) error {
	if err := os.MkdirAll(knowledgeDir, 0755); err != nil {
		return fmt.Errorf("creating knowledge dir: %w", err)
	}

	files := map[string]interface{}{
		"git-summary.json":      report.Summary,
		"git-experts.json":      report.Experts,
		"git-risks.json":        report.Risks,
		"git-correlations.json": report.Correlations,
	}

	for name, data := range files {
		content, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling %s: %w", name, err)
		}
		path := filepath.Join(knowledgeDir, name)
		if err := os.WriteFile(path, content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", name, err)
		}
	}

	return nil
}

// parseProcessorLog parses git log output with numstat
func parseProcessorLog(output string) []processedCommit {
	var commits []processedCommit
	var current *processedCommit

	numstatPattern := regexp.MustCompile(`^(\d+|-)\s+(\d+|-)\s+(.+)$`)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if it's a commit header: hash|shorthash|author|email|date|message
		parts := strings.SplitN(line, "|", 6)
		if len(parts) == 6 && len(parts[0]) == 40 && isAllHex(parts[0]) {
			if current != nil {
				commits = append(commits, *current)
			}
			date, _ := time.Parse(time.RFC3339, parts[4])
			current = &processedCommit{
				Hash:        parts[0],
				Author:      parts[2],
				AuthorEmail: parts[3],
				Date:        date,
				Message:     parts[5],
			}
			continue
		}

		// Check numstat
		if current != nil {
			if m := numstatPattern.FindStringSubmatch(line); m != nil {
				add, _ := strconv.Atoi(m[1])
				del, _ := strconv.Atoi(m[2])
				current.Insertions += add
				current.Deletions += del
				current.FilesChanged = append(current.FilesChanged, m[3])
			}
		}
	}

	if current != nil {
		commits = append(commits, *current)
	}

	return commits
}

type processedCommit struct {
	Hash         string
	Author       string
	AuthorEmail  string
	Date         time.Time
	Message      string
	FilesChanged []string
	Insertions   int
	Deletions    int
}

func isAllHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// CrossReferenceLinkedRepos checks linked sibling repos for contributor activity.
// If a contributor is inactive in the current repo but active in a linked repo,
// it updates their Active flag and sets ActiveInRepo to the linked repo name.
func CrossReferenceLinkedRepos(report *GitHistoryReport, linkedRepoPaths []string) {
	if report == nil || len(linkedRepoPaths) == 0 {
		return
	}

	// Build email→index map for contributors
	emailToIdx := make(map[string]int)
	for i, c := range report.Summary.Contributors {
		emailToIdx[c.Email] = i
	}

	for _, repoPath := range linkedRepoPaths {
		// Read the linked repo's git-summary.json
		summaryPath := filepath.Join(repoPath, ".teamcontext", "knowledge", "git-summary.json")
		data, err := os.ReadFile(summaryPath)
		if err != nil {
			continue
		}

		var linkedSummary GitSummary
		if err := json.Unmarshal(data, &linkedSummary); err != nil {
			continue
		}

		repoName := filepath.Base(repoPath)

		// Check each linked contributor against our inactive ones
		for _, linkedContrib := range linkedSummary.Contributors {
			if !linkedContrib.Active {
				continue
			}
			idx, exists := emailToIdx[linkedContrib.Email]
			if !exists {
				continue
			}
			if report.Summary.Contributors[idx].Active {
				continue // already active locally, skip
			}
			// Mark as active via linked repo
			report.Summary.Contributors[idx].Active = true
			report.Summary.Contributors[idx].ActiveInRepo = repoName
		}
	}

	// Propagate to experts: build updated active map from contributors
	activeInRepo := make(map[string]string) // email → repo name
	for _, c := range report.Summary.Contributors {
		if c.ActiveInRepo != "" {
			activeInRepo[c.Email] = c.ActiveInRepo
		}
	}

	for i, de := range report.Experts {
		for j, exp := range de.TopExperts {
			if !exp.Active {
				if repoName, ok := activeInRepo[exp.Email]; ok {
					report.Experts[i].TopExperts[j].Active = true
					report.Experts[i].TopExperts[j].ActiveInRepo = repoName
				}
			}
		}
	}
}

// detectDefaultBranch finds the default branch name (main, master, or HEAD)
func detectDefaultBranch(repoPath string) string {
	// Try to read the symbolic ref for origin/HEAD
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
	cmd.Dir = repoPath
	if out, err := cmd.Output(); err == nil {
		branch := strings.TrimSpace(string(out))
		// "origin/main" -> "main"
		if parts := strings.SplitN(branch, "/", 2); len(parts) == 2 {
			return parts[1]
		}
		return branch
	}

	// Fallback: check if "main" branch exists
	cmd = exec.Command("git", "rev-parse", "--verify", "main")
	cmd.Dir = repoPath
	if err := cmd.Run(); err == nil {
		return "main"
	}

	// Fallback: check if "master" branch exists
	cmd = exec.Command("git", "rev-parse", "--verify", "master")
	cmd.Dir = repoPath
	if err := cmd.Run(); err == nil {
		return "master"
	}

	// Last resort
	return "HEAD"
}
