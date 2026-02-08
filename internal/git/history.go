package git

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// GIT HISTORY ANALYSIS - Mining the team's institutional memory
// =============================================================================

// CommitInfo represents a parsed git commit
type CommitInfo struct {
	Hash         string    `json:"hash"`
	ShortHash    string    `json:"short_hash"`
	Author       string    `json:"author"`
	AuthorEmail  string    `json:"author_email"`
	Date         time.Time `json:"date"`
	Message      string    `json:"message"`
	FilesChanged []string  `json:"files_changed"`
	Insertions   int       `json:"insertions"`
	Deletions    int       `json:"deletions"`
}

// FileExpertise represents who knows a file best
type FileExpertise struct {
	File           string           `json:"file"`
	TotalCommits   int              `json:"total_commits"`
	TotalLines     int              `json:"total_lines"`
	Contributors   []ContributorStat `json:"contributors"`
	LastModified   time.Time        `json:"last_modified"`
	LastModifiedBy string           `json:"last_modified_by"`
	ChurnRate      string           `json:"churn_rate"` // low, medium, high
}

// ContributorStat represents a contributor's involvement with a file
type ContributorStat struct {
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Commits      int       `json:"commits"`
	LinesAdded   int       `json:"lines_added"`
	LinesRemoved int       `json:"lines_removed"`
	FirstCommit  time.Time `json:"first_commit"`
	LastCommit   time.Time `json:"last_commit"`
	Ownership    float64   `json:"ownership"` // 0-1 percentage
	IsActive     bool      `json:"is_active"` // committed in last 3 months
}

// Expert represents a person with expertise in an area
type Expert struct {
	Name       string   `json:"name"`
	Email      string   `json:"email"`
	Score      float64  `json:"score"`      // 0-1 expertise score
	Commits    int      `json:"commits"`
	FilesTouched int    `json:"files_touched"`
	IsActive   bool     `json:"is_active"`
	Reasons    []string `json:"reasons"`
}

// KnowledgeRisk represents knowledge concentration risk
type KnowledgeRisk struct {
	Area           string   `json:"area"`
	RiskLevel      string   `json:"risk_level"` // LOW, MEDIUM, HIGH, CRITICAL
	Reason         string   `json:"reason"`
	Files          []string `json:"files"`
	PrimaryExpert  string   `json:"primary_expert,omitempty"`
	ExpertIsActive bool     `json:"expert_is_active"`
	LastActivity   string   `json:"last_activity"`
	Mitigation     string   `json:"mitigation"`
}

// FileCorrelation represents files that usually change together
type FileCorrelation struct {
	File1       string  `json:"file1"`
	File2       string  `json:"file2"`
	CoChanges   int     `json:"co_changes"`    // Times changed together
	Correlation float64 `json:"correlation"`   // 0-1
	Confidence  string  `json:"confidence"`    // low, medium, high
}

// CommitContext provides context for why code exists
type CommitContext struct {
	File         string       `json:"file"`
	Lines        []int        `json:"lines,omitempty"`
	Commits      []CommitInfo `json:"commits"`
	Summary      string       `json:"summary"`
	Contributors []string     `json:"contributors"`
	TimeSpan     string       `json:"time_span"`
}

// HistoryAnalyzer provides Git history analysis
type HistoryAnalyzer struct {
	repoPath string
}

// NewHistoryAnalyzer creates a new analyzer
func NewHistoryAnalyzer(repoPath string) *HistoryAnalyzer {
	return &HistoryAnalyzer{repoPath: repoPath}
}

// GetCommitHistory retrieves commit history for a time period
func (h *HistoryAnalyzer) GetCommitHistory(since time.Time, limit int) ([]CommitInfo, error) {
	sinceStr := since.Format("2006-01-02")

	// Git log with stats
	args := []string{
		"log",
		"--since=" + sinceStr,
		"--pretty=format:%H|%h|%an|%ae|%aI|%s",
		"--numstat",
	}
	if limit > 0 {
		args = append(args, "-n", strconv.Itoa(limit))
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = h.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseCommitLog(string(output)), nil
}

// GetFileExpertise analyzes who knows a file best
func (h *HistoryAnalyzer) GetFileExpertise(filePath string) (*FileExpertise, error) {
	// Get blame stats
	cmd := exec.Command("git", "log", "--follow", "--pretty=format:%H|%an|%ae|%aI", "--numstat", "--", filePath)
	cmd.Dir = h.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	expertise := &FileExpertise{
		File:         filePath,
		Contributors: []ContributorStat{},
	}

	contributorMap := make(map[string]*ContributorStat)
	lines := strings.Split(string(output), "\n")

	var currentAuthor, currentEmail string
	var currentDate time.Time

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse commit line
		if strings.Contains(line, "|") && len(strings.Split(line, "|")) == 4 {
			parts := strings.Split(line, "|")
			currentAuthor = parts[1]
			currentEmail = parts[2]
			currentDate, _ = time.Parse(time.RFC3339, parts[3])

			// Initialize contributor if new
			if _, exists := contributorMap[currentEmail]; !exists {
				contributorMap[currentEmail] = &ContributorStat{
					Name:        currentAuthor,
					Email:       currentEmail,
					FirstCommit: currentDate,
					LastCommit:  currentDate,
				}
			}

			stat := contributorMap[currentEmail]
			stat.Commits++
			if currentDate.After(stat.LastCommit) {
				stat.LastCommit = currentDate
			}
			if currentDate.Before(stat.FirstCommit) {
				stat.FirstCommit = currentDate
			}

			expertise.TotalCommits++
			if currentDate.After(expertise.LastModified) {
				expertise.LastModified = currentDate
				expertise.LastModifiedBy = currentAuthor
			}
			continue
		}

		// Parse numstat line (additions deletions filename)
		numstatPattern := regexp.MustCompile(`^(\d+|-)\s+(\d+|-)\s+(.+)$`)
		if m := numstatPattern.FindStringSubmatch(line); m != nil {
			additions, _ := strconv.Atoi(m[1])
			deletions, _ := strconv.Atoi(m[2])

			if stat, exists := contributorMap[currentEmail]; exists {
				stat.LinesAdded += additions
				stat.LinesRemoved += deletions
			}
			expertise.TotalLines += additions
		}
	}

	// Calculate ownership and activity
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	for _, stat := range contributorMap {
		if expertise.TotalCommits > 0 {
			stat.Ownership = float64(stat.Commits) / float64(expertise.TotalCommits)
		}
		stat.IsActive = stat.LastCommit.After(threeMonthsAgo)
		expertise.Contributors = append(expertise.Contributors, *stat)
	}

	// Sort by ownership
	sort.Slice(expertise.Contributors, func(i, j int) bool {
		return expertise.Contributors[i].Ownership > expertise.Contributors[j].Ownership
	})

	// Calculate churn rate
	if expertise.TotalCommits > 20 {
		expertise.ChurnRate = "high"
	} else if expertise.TotalCommits > 5 {
		expertise.ChurnRate = "medium"
	} else {
		expertise.ChurnRate = "low"
	}

	return expertise, nil
}

// FindExperts finds people with expertise in given files or area
func (h *HistoryAnalyzer) FindExperts(files []string, area string) ([]Expert, error) {
	expertMap := make(map[string]*Expert)

	// If area is provided, find files matching that area
	if area != "" && len(files) == 0 {
		cmd := exec.Command("git", "ls-files", "*"+area+"*")
		cmd.Dir = h.repoPath
		output, _ := cmd.Output()
		files = strings.Split(strings.TrimSpace(string(output)), "\n")
	}

	for _, file := range files {
		if file == "" {
			continue
		}

		expertise, err := h.GetFileExpertise(file)
		if err != nil {
			continue
		}

		for _, contrib := range expertise.Contributors {
			if expert, exists := expertMap[contrib.Email]; exists {
				expert.Commits += contrib.Commits
				expert.FilesTouched++
				expert.Score += contrib.Ownership
				if contrib.IsActive {
					expert.IsActive = true
				}
			} else {
				expertMap[contrib.Email] = &Expert{
					Name:         contrib.Name,
					Email:        contrib.Email,
					Score:        contrib.Ownership,
					Commits:      contrib.Commits,
					FilesTouched: 1,
					IsActive:     contrib.IsActive,
					Reasons:      []string{},
				}
			}
		}
	}

	// Normalize scores and build reasons
	var experts []Expert
	for _, expert := range expertMap {
		// Normalize score
		if len(files) > 0 {
			expert.Score = expert.Score / float64(len(files))
		}

		// Build reasons
		expert.Reasons = append(expert.Reasons,
			formatReason(expert.Commits, "commits"),
			formatReason(expert.FilesTouched, "files touched"),
		)
		if !expert.IsActive {
			expert.Reasons = append(expert.Reasons, "⚠️ Not active in last 3 months")
		}

		experts = append(experts, *expert)
	}

	// Sort by score
	sort.Slice(experts, func(i, j int) bool {
		return experts[i].Score > experts[j].Score
	})

	// Limit to top 5
	if len(experts) > 5 {
		experts = experts[:5]
	}

	return experts, nil
}

// AnalyzeKnowledgeRisk identifies areas with knowledge concentration risk
func (h *HistoryAnalyzer) AnalyzeKnowledgeRisk() ([]KnowledgeRisk, error) {
	// Get all files
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = h.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Group by directory
	dirFiles := make(map[string][]string)
	for _, file := range files {
		dir := filepath.Dir(file)
		dirFiles[dir] = append(dirFiles[dir], file)
	}

	var risks []KnowledgeRisk
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)

	for dir, dirFileList := range dirFiles {
		if len(dirFileList) < 2 {
			continue
		}

		// Analyze expertise for directory
		experts, _ := h.FindExperts(dirFileList, "")

		risk := KnowledgeRisk{
			Area:  dir,
			Files: dirFileList,
		}

		if len(experts) == 0 {
			risk.RiskLevel = "CRITICAL"
			risk.Reason = "No contributors found in history"
			risk.Mitigation = "Investigate - files may be untracked or very old"
		} else {
			topExpert := experts[0]
			risk.PrimaryExpert = topExpert.Name
			risk.ExpertIsActive = topExpert.IsActive

			// Check for single-contributor risk
			if topExpert.Score > 0.8 {
				if !topExpert.IsActive {
					risk.RiskLevel = "CRITICAL"
					risk.Reason = "Primary contributor (" + topExpert.Name + ") owns 80%+ and is inactive"
					risk.Mitigation = "Urgent: Assign new maintainer, schedule knowledge transfer"
				} else {
					risk.RiskLevel = "MEDIUM"
					risk.Reason = "Single contributor owns 80%+ of this area"
					risk.Mitigation = "Consider pairing to spread knowledge"
				}
			} else if !topExpert.IsActive {
				risk.RiskLevel = "HIGH"
				risk.Reason = "Top contributor is no longer active"
				risk.Mitigation = "Identify current maintainer or assign one"
			} else {
				risk.RiskLevel = "LOW"
				risk.Reason = "Knowledge is distributed among active contributors"
				risk.Mitigation = "None needed"
			}

			// Check last activity
			latestExpertise, _ := h.GetFileExpertise(dirFileList[0])
			if latestExpertise != nil {
				if latestExpertise.LastModified.Before(sixMonthsAgo) {
					risk.LastActivity = "Over 6 months ago"
					if risk.RiskLevel != "CRITICAL" {
						risk.RiskLevel = "HIGH"
						risk.Reason += "; No recent activity"
					}
				} else if latestExpertise.LastModified.Before(threeMonthsAgo) {
					risk.LastActivity = "3-6 months ago"
				} else {
					risk.LastActivity = "Within 3 months"
				}
			}
		}

		// Only include medium+ risk
		if risk.RiskLevel != "LOW" {
			risks = append(risks, risk)
		}
	}

	// Sort by risk level
	riskOrder := map[string]int{"CRITICAL": 0, "HIGH": 1, "MEDIUM": 2, "LOW": 3}
	sort.Slice(risks, func(i, j int) bool {
		return riskOrder[risks[i].RiskLevel] < riskOrder[risks[j].RiskLevel]
	})

	return risks, nil
}

// GetFileCorrelations finds files that frequently change together
func (h *HistoryAnalyzer) GetFileCorrelations(filePath string, minCorrelation float64) ([]FileCorrelation, error) {
	// Get commits that touched this file
	cmd := exec.Command("git", "log", "--pretty=format:%H", "--follow", "--", filePath)
	cmd.Dir = h.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(commits) == 0 {
		return nil, nil
	}

	// Count co-changes with other files
	coChangeCount := make(map[string]int)

	for _, commit := range commits {
		if commit == "" {
			continue
		}

		// Get files in this commit
		cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", commit)
		cmd.Dir = h.repoPath
		filesOutput, err := cmd.Output()
		if err != nil {
			continue
		}

		files := strings.Split(strings.TrimSpace(string(filesOutput)), "\n")
		for _, f := range files {
			if f != "" && f != filePath {
				coChangeCount[f]++
			}
		}
	}

	// Calculate correlations
	var correlations []FileCorrelation
	totalCommits := len(commits)

	for otherFile, count := range coChangeCount {
		correlation := float64(count) / float64(totalCommits)
		if correlation >= minCorrelation {
			conf := "low"
			if count > 10 {
				conf = "high"
			} else if count > 3 {
				conf = "medium"
			}

			correlations = append(correlations, FileCorrelation{
				File1:       filePath,
				File2:       otherFile,
				CoChanges:   count,
				Correlation: correlation,
				Confidence:  conf,
			})
		}
	}

	// Sort by correlation
	sort.Slice(correlations, func(i, j int) bool {
		return correlations[i].Correlation > correlations[j].Correlation
	})

	// Limit to top 10
	if len(correlations) > 10 {
		correlations = correlations[:10]
	}

	return correlations, nil
}

// GetCommitContext gets context for why specific code exists
func (h *HistoryAnalyzer) GetCommitContext(filePath string, lines []int) (*CommitContext, error) {
	ctx := &CommitContext{
		File:  filePath,
		Lines: lines,
	}

	var args []string
	if len(lines) > 0 {
		// Get blame for specific lines
		lineRange := strconv.Itoa(lines[0])
		if len(lines) > 1 {
			lineRange += "," + strconv.Itoa(lines[len(lines)-1])
		}
		args = []string{"log", "-L", lineRange + ":" + filePath, "--pretty=format:%H|%h|%an|%ae|%aI|%s", "-n", "10"}
	} else {
		// Get full file history
		args = []string{"log", "--follow", "--pretty=format:%H|%h|%an|%ae|%aI|%s", "-n", "10", "--", filePath}
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = h.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	contributorSet := make(map[string]bool)
	var oldestDate, newestDate time.Time

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "|") {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}

		date, _ := time.Parse(time.RFC3339, parts[4])
		commit := CommitInfo{
			Hash:        parts[0],
			ShortHash:   parts[1],
			Author:      parts[2],
			AuthorEmail: parts[3],
			Date:        date,
			Message:     parts[5],
		}

		ctx.Commits = append(ctx.Commits, commit)
		contributorSet[commit.Author] = true

		if oldestDate.IsZero() || date.Before(oldestDate) {
			oldestDate = date
		}
		if date.After(newestDate) {
			newestDate = date
		}
	}

	// Build summary
	for author := range contributorSet {
		ctx.Contributors = append(ctx.Contributors, author)
	}

	if !oldestDate.IsZero() {
		ctx.TimeSpan = formatTimeSpan(oldestDate, newestDate)
	}

	if len(ctx.Commits) > 0 {
		ctx.Summary = "Code evolved through " + strconv.Itoa(len(ctx.Commits)) +
			" commits by " + strconv.Itoa(len(ctx.Contributors)) + " contributors"
	}

	return ctx, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func parseCommitLog(output string) []CommitInfo {
	var commits []CommitInfo
	var current *CommitInfo

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if it's a commit header line
		if strings.Contains(line, "|") && len(strings.Split(line, "|")) == 6 {
			parts := strings.Split(line, "|")
			date, _ := time.Parse(time.RFC3339, parts[4])

			if current != nil {
				commits = append(commits, *current)
			}

			current = &CommitInfo{
				Hash:        parts[0],
				ShortHash:   parts[1],
				Author:      parts[2],
				AuthorEmail: parts[3],
				Date:        date,
				Message:     parts[5],
			}
			continue
		}

		// Check if it's a numstat line
		if current != nil {
			numstatPattern := regexp.MustCompile(`^(\d+|-)\s+(\d+|-)\s+(.+)$`)
			if m := numstatPattern.FindStringSubmatch(line); m != nil {
				additions, _ := strconv.Atoi(m[1])
				deletions, _ := strconv.Atoi(m[2])
				current.Insertions += additions
				current.Deletions += deletions
				current.FilesChanged = append(current.FilesChanged, m[3])
			}
		}
	}

	if current != nil {
		commits = append(commits, *current)
	}

	return commits
}

func formatReason(count int, thing string) string {
	return strconv.Itoa(count) + " " + thing
}

func formatTimeSpan(oldest, newest time.Time) string {
	duration := newest.Sub(oldest)
	days := int(duration.Hours() / 24)

	if days > 365 {
		return strconv.Itoa(days/365) + " years"
	} else if days > 30 {
		return strconv.Itoa(days/30) + " months"
	} else if days > 0 {
		return strconv.Itoa(days) + " days"
	}
	return "same day"
}
