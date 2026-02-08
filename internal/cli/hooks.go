package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var installHooksCmd = &cobra.Command{
	Use:   "install-hooks",
	Short: "Install git hooks for automatic knowledge capture",
	Long: `Install git hooks that automatically capture knowledge from your workflow.

Currently installs:
  post-commit  — Analyzes each commit for reverts, large deletions, dependency
                 changes, and config modifications. Auto-records warnings and
                 updates the project knowledge base.

The hooks run teamcontext's analysis on commit diffs without requiring an LLM.
They detect patterns like:
  - Reverted commits → auto-warning
  - Large file deletions (>100 lines) → auto-warning
  - Dependency file changes → auto-logged
  - Config file modifications → auto-logged

Examples:
  teamcontext install-hooks
  teamcontext uninstall-hooks`,
	Run: runInstallHooks,
}

var uninstallHooksCmd = &cobra.Command{
	Use:   "uninstall-hooks",
	Short: "Remove TeamContext git hooks",
	Long:  `Remove TeamContext git hooks from the repository.`,
	Run:   runUninstallHooks,
}

func init() {
	println("[DEBUG] cli/hooks.go: init")
	rootCmd.AddCommand(installHooksCmd)
	rootCmd.AddCommand(uninstallHooksCmd)
}

const hookMarker = "# === TEAMCONTEXT HOOK ==="

// postCommitHookScript is the shell script installed as .git/hooks/post-commit.
// It runs teamcontext analyze-commit after every commit.
const postCommitHookScript = `#!/bin/sh
` + hookMarker + `
# Auto-analyze commits for knowledge extraction.
# Installed by: teamcontext install-hooks
# Remove with:  teamcontext uninstall-hooks

# Find the teamcontext binary
TEAMCONTEXT=$(command -v teamcontext 2>/dev/null)
if [ -z "$TEAMCONTEXT" ]; then
    # Try common install locations
    for p in "$HOME/go/bin/teamcontext" "/usr/local/bin/teamcontext"; do
        if [ -x "$p" ]; then
            TEAMCONTEXT="$p"
            break
        fi
    done
fi

if [ -z "$TEAMCONTEXT" ]; then
    exit 0  # Silently skip if teamcontext not found
fi

# Run analysis in background so it doesn't slow down commits
"$TEAMCONTEXT" analyze-commit --quiet &
`

func runInstallHooks(cmd *cobra.Command, args []string) {
	// Find git root
	gitRoot, err := findGitRoot()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("This command must be run inside a git repository.")
		return
	}

	// Check that .teamcontext exists
	tcDir := filepath.Join(gitRoot, ".teamcontext")
	if _, err := os.Stat(tcDir); os.IsNotExist(err) {
		fmt.Println("Error: No .teamcontext directory found.")
		fmt.Println("Run 'teamcontext init' first.")
		return
	}

	hooksDir := filepath.Join(gitRoot, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		fmt.Printf("Error creating hooks directory: %v\n", err)
		return
	}

	hookPath := filepath.Join(hooksDir, "post-commit")

	// Check if hook already exists
	if existing, err := os.ReadFile(hookPath); err == nil {
		content := string(existing)
		if strings.Contains(content, hookMarker) {
			fmt.Println("TeamContext post-commit hook is already installed.")
			return
		}
		// Hook exists but isn't ours — append
		fmt.Println("Existing post-commit hook found. Appending TeamContext hook.")
		combined := content + "\n\n" + postCommitHookScript
		if err := os.WriteFile(hookPath, []byte(combined), 0755); err != nil {
			fmt.Printf("Error writing hook: %v\n", err)
			return
		}
	} else {
		// No existing hook — create new
		if err := os.WriteFile(hookPath, []byte(postCommitHookScript), 0755); err != nil {
			fmt.Printf("Error writing hook: %v\n", err)
			return
		}
	}

	fmt.Println("Installed git hooks:")
	fmt.Println("  post-commit — auto-analyze commits for knowledge extraction")
	fmt.Println()
	fmt.Println("Hooks will run 'teamcontext analyze-commit' after each commit.")
	fmt.Println("Remove with: teamcontext uninstall-hooks")
}

func runUninstallHooks(cmd *cobra.Command, args []string) {
	gitRoot, err := findGitRoot()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	hookPath := filepath.Join(gitRoot, ".git", "hooks", "post-commit")
	existing, err := os.ReadFile(hookPath)
	if err != nil {
		fmt.Println("No post-commit hook found.")
		return
	}

	content := string(existing)
	if !strings.Contains(content, hookMarker) {
		fmt.Println("No TeamContext hook found in post-commit.")
		return
	}

	// Remove the TeamContext section
	lines := strings.Split(content, "\n")
	var cleaned []string
	inTeamContextSection := false
	for _, line := range lines {
		if strings.Contains(line, hookMarker) {
			inTeamContextSection = true
			continue
		}
		if inTeamContextSection {
			// Skip until we hit another shebang or end of file
			if strings.HasPrefix(line, "#!/") {
				inTeamContextSection = false
				cleaned = append(cleaned, line)
			}
			continue
		}
		cleaned = append(cleaned, line)
	}

	result := strings.TrimSpace(strings.Join(cleaned, "\n"))
	if result == "" || result == "#!/bin/sh" {
		// Nothing left — remove the file
		os.Remove(hookPath)
		fmt.Println("Removed TeamContext post-commit hook.")
	} else {
		os.WriteFile(hookPath, []byte(result+"\n"), 0755)
		fmt.Println("Removed TeamContext section from post-commit hook.")
	}
}

func findGitRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}
