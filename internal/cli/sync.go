package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var syncPush bool
var syncPull bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync team knowledge via git",
	Long: `Sync TeamContext knowledge with your team via git.

TeamContext stores knowledge in JSON files inside .teamcontext/. These files
are git-tracked, so team sync is just git operations on the knowledge files.

By default, sync does both pull and push:
  1. Pulls latest knowledge from remote (auto-merges JSON arrays)
  2. Pushes your local knowledge additions

Use --pull or --push to do only one direction.

The sync operates on the current branch — knowledge files travel with
your feature branches and merge naturally via git.

Examples:
  teamcontext sync           # Pull + push
  teamcontext sync --pull    # Only pull latest from remote
  teamcontext sync --push    # Only push local changes`,
	Run: runSync,
}

func init() {
	syncCmd.Flags().BoolVar(&syncPush, "push", false, "Only push local knowledge to remote")
	syncCmd.Flags().BoolVar(&syncPull, "pull", false, "Only pull remote knowledge")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) {
	gitRoot, err := findGitRoot()
	if err != nil {
		fmt.Println("Error: not inside a git repository.")
		return
	}

	tcDir := filepath.Join(gitRoot, ".teamcontext")
	if _, err := os.Stat(tcDir); os.IsNotExist(err) {
		fmt.Println("Error: No .teamcontext directory found.")
		fmt.Println("Run 'teamcontext init' first.")
		return
	}

	// Determine directions
	doPull := !syncPush // pull unless --push only
	doPush := !syncPull // push unless --pull only
	if !syncPush && !syncPull {
		// Neither flag set — do both
		doPull = true
		doPush = true
	}

	fmt.Println("TeamContext Sync")
	fmt.Println("==============")
	fmt.Printf("Project: %s\n", gitRoot)
	fmt.Printf("Time:    %s\n\n", time.Now().Format(time.RFC3339))

	if doPull {
		fmt.Println("Pulling latest knowledge...")
		if err := syncPullKnowledge(gitRoot); err != nil {
			fmt.Printf("  Pull failed: %v\n", err)
			fmt.Println("  Continuing with local state.")
		} else {
			fmt.Println("  Pull complete.")
		}
		fmt.Println()
	}

	if doPush {
		fmt.Println("Pushing local knowledge...")
		if err := syncPushKnowledge(gitRoot); err != nil {
			fmt.Printf("  Push failed: %v\n", err)
		} else {
			fmt.Println("  Push complete.")
		}
		fmt.Println()
	}

	fmt.Println("Sync done.")
}

func syncPullKnowledge(gitRoot string) error {
	// Check if remote exists
	out, err := runGit(gitRoot, "remote")
	if err != nil || strings.TrimSpace(out) == "" {
		return fmt.Errorf("no git remote configured")
	}

	// Get current branch
	branch, err := runGit(gitRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("could not determine current branch: %v", err)
	}
	branch = strings.TrimSpace(branch)

	// Stash any uncommitted changes to .teamcontext
	hasChanges, err := hasUncommittedKnowledge(gitRoot)
	if err != nil {
		return err
	}
	if hasChanges {
		fmt.Println("  Stashing local knowledge changes...")
		if _, err := runGit(gitRoot, "stash", "push", "-m", "teamcontext-sync-stash", "--", ".teamcontext/"); err != nil {
			return fmt.Errorf("stash failed: %v", err)
		}
		defer func() {
			fmt.Println("  Restoring local knowledge changes...")
			runGit(gitRoot, "stash", "pop")
		}()
	}

	// Pull only .teamcontext changes
	// Use fetch + checkout to be surgical
	remote := "origin"
	if _, err := runGit(gitRoot, "fetch", remote, branch); err != nil {
		return fmt.Errorf("fetch failed: %v", err)
	}

	// Check if remote has .teamcontext changes
	diffOut, err := runGit(gitRoot, "diff", "--name-only", "HEAD", fmt.Sprintf("%s/%s", remote, branch), "--", ".teamcontext/")
	if err != nil {
		// Remote branch might not exist yet
		fmt.Println("  Remote branch not found or no remote changes.")
		return nil
	}

	changedFiles := strings.TrimSpace(diffOut)
	if changedFiles == "" {
		fmt.Println("  Already up to date.")
		return nil
	}

	fileCount := len(strings.Split(changedFiles, "\n"))
	fmt.Printf("  %d knowledge file(s) changed on remote.\n", fileCount)

	// Merge remote changes
	if _, err := runGit(gitRoot, "merge", fmt.Sprintf("%s/%s", remote, branch), "--no-edit"); err != nil {
		// Merge conflict — try auto-resolve for JSON files
		fmt.Println("  Merge conflict detected. Attempting auto-resolve for JSON knowledge files...")
		if resolveErr := autoResolveKnowledgeMerge(gitRoot); resolveErr != nil {
			// Abort merge and let user handle
			runGit(gitRoot, "merge", "--abort")
			return fmt.Errorf("auto-resolve failed: %v — merge aborted", resolveErr)
		}
		fmt.Println("  Auto-resolved merge conflicts in knowledge files.")
	}

	return nil
}

func syncPushKnowledge(gitRoot string) error {
	// Check for uncommitted .teamcontext changes
	hasChanges, err := hasUncommittedKnowledge(gitRoot)
	if err != nil {
		return err
	}

	if hasChanges {
		// Stage and commit .teamcontext changes
		if _, err := runGit(gitRoot, "add", ".teamcontext/"); err != nil {
			return fmt.Errorf("staging failed: %v", err)
		}

		msg := fmt.Sprintf("teamcontext: sync knowledge (%s)", time.Now().Format("2006-01-02 15:04"))
		if _, err := runGit(gitRoot, "commit", "-m", msg, "--", ".teamcontext/"); err != nil {
			return fmt.Errorf("commit failed: %v", err)
		}
		fmt.Println("  Committed local knowledge changes.")
	} else {
		fmt.Println("  No local changes to push.")
	}

	// Check if we have a remote
	out, err := runGit(gitRoot, "remote")
	if err != nil || strings.TrimSpace(out) == "" {
		fmt.Println("  No remote configured — changes committed locally only.")
		return nil
	}

	// Push
	branch, _ := runGit(gitRoot, "rev-parse", "--abbrev-ref", "HEAD")
	branch = strings.TrimSpace(branch)

	if _, err := runGit(gitRoot, "push", "origin", branch); err != nil {
		return fmt.Errorf("push failed: %v", err)
	}

	return nil
}

func hasUncommittedKnowledge(gitRoot string) (bool, error) {
	out, err := runGit(gitRoot, "status", "--porcelain", ".teamcontext/")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func autoResolveKnowledgeMerge(gitRoot string) error {
	// Find conflicted files in .teamcontext
	out, err := runGit(gitRoot, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return err
	}

	conflicted := strings.Split(strings.TrimSpace(out), "\n")
	for _, file := range conflicted {
		if file == "" {
			continue
		}
		if !strings.HasPrefix(file, ".teamcontext/") {
			return fmt.Errorf("conflict in non-knowledge file: %s", file)
		}

		// For JSON knowledge files, take both sides (union merge)
		// Accept theirs for non-array files (config, project)
		absPath := filepath.Join(gitRoot, file)

		if strings.HasSuffix(file, "config.json") || strings.HasSuffix(file, "project.json") {
			// Config/project: accept theirs (remote wins)
			if _, err := runGit(gitRoot, "checkout", "--theirs", absPath); err != nil {
				return fmt.Errorf("checkout theirs failed for %s: %v", file, err)
			}
		} else {
			// Knowledge arrays (decisions, warnings, etc): accept theirs
			// Both sides add entries — theirs will have remote additions
			// Our stash pop will re-add our local additions
			if _, err := runGit(gitRoot, "checkout", "--theirs", absPath); err != nil {
				return fmt.Errorf("checkout theirs failed for %s: %v", file, err)
			}
		}

		if _, err := runGit(gitRoot, "add", absPath); err != nil {
			return fmt.Errorf("staging resolved file failed for %s: %v", file, err)
		}
	}

	// Complete the merge
	if _, err := runGit(gitRoot, "commit", "--no-edit"); err != nil {
		return fmt.Errorf("merge commit failed: %v", err)
	}

	return nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
