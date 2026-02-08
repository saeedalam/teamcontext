package cli

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
	"github.com/saeedalam/teamcontext/internal/git"
	"github.com/saeedalam/teamcontext/internal/storage"
	"github.com/saeedalam/teamcontext/internal/worker"
)

var reindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Re-index the codebase while preserving knowledge",
	Long: `Re-index the codebase while preserving all accumulated knowledge.

This command:
- Re-scans all files for skeletons, imports, and graph edges
- Re-processes git history for experts, risks, and correlations
- PRESERVES decisions, warnings, insights, patterns, and conversations

Use this when:
- You've made significant code changes
- The index seems out of date
- You want to refresh git intelligence

Example:
  teamcontext reindex`,
	Run: runReindex,
}

func runReindex(cmd *cobra.Command, args []string) {
	tcDir, err := findTeamContextDirFromCwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Run 'teamcontext init' to initialize TeamContext first.")
		return
	}

	// Get project root (parent of .teamcontext)
	projectRoot := filepath.Dir(tcDir)

	fmt.Println("Re-indexing codebase (preserving knowledge)...")
	fmt.Println("")

	var wg sync.WaitGroup
	var indexErr, gitErr error
	var indexed int

	// Re-index files
	wg.Add(1)
	go func() {
		defer wg.Done()
		jsonStore := storage.NewJSONStore(tcDir)
		sqliteIndex, err := storage.NewSQLiteIndex(tcDir)
		if err != nil {
			indexErr = err
			return
		}
		defer sqliteIndex.Close()

		workerMgr := worker.NewManager(tcDir, jsonStore, sqliteIndex)
		indexed, indexErr = workerMgr.InitProject()
	}()

	// Re-process git history
	wg.Add(1)
	go func() {
		defer wg.Done()
		report, err := git.ProcessGitHistory(projectRoot)
		if err != nil {
			gitErr = err
			return
		}

		knowledgeDir := filepath.Join(tcDir, "knowledge")
		if err := git.WriteReportFiles(report, knowledgeDir); err != nil {
			gitErr = err
			return
		}
		fmt.Printf("  ✓ Processed git history (%d commits, %d contributors)\n", report.CommitCount, report.Contributors)
	}()

	wg.Wait()

	if indexErr != nil {
		fmt.Printf("  Warning: File indexing had errors: %v\n", indexErr)
	} else {
		fmt.Printf("  ✓ Indexed %d files (skeletons, imports, graph edges)\n", indexed)
	}

	if gitErr != nil {
		fmt.Printf("  Warning: Git processing had errors: %v\n", gitErr)
	}

	fmt.Println("")
	fmt.Println("Re-indexing complete!")
	fmt.Println("Knowledge files (decisions, warnings, insights) preserved.")
}
