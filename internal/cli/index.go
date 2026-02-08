package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/saeedalam/teamcontext/internal/storage"
	"github.com/saeedalam/teamcontext/internal/worker"
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index or re-index project files",
	Long: `Scan and index all code files in the project.

This extracts:
- File skeletons (functions, classes, interfaces)
- Import relationships (builds dependency graph)
- Code content (for full-text search)
- Exports and patterns

Run this:
- After initial setup
- After major changes
- To refresh the knowledge base

Example:
  teamcontext index           # Index entire project
  teamcontext index src/      # Index specific directory`,
	Run: runIndex,
}

func runIndex(cmd *cobra.Command, args []string) {
	tcDir, err := findTeamContextDirFromCwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Run 'teamcontext init' first to initialize TeamContext.")
		return
	}

	fmt.Println("Indexing project files...")
	fmt.Println("")

	// Initialize storage
	jsonStore := storage.NewJSONStore(tcDir)
	sqliteIndex, err := storage.NewSQLiteIndex(tcDir)
	if err != nil {
		fmt.Printf("Error initializing SQLite: %v\n", err)
		return
	}
	defer sqliteIndex.Close()

	// Create worker manager and run init
	workerMgr := worker.NewManager(tcDir, jsonStore, sqliteIndex)

	indexed, err := workerMgr.InitProject()
	if err != nil {
		fmt.Printf("Warning: Indexing completed with errors: %v\n", err)
	}

	// Get stats
	stats := workerMgr.GetStats()

	fmt.Println("Indexing complete!")
	fmt.Println("")
	fmt.Printf("  Files indexed:     %d\n", indexed)
	fmt.Printf("  Skeletons cached:  %d\n", stats.SkeletonsCached)
	fmt.Printf("  Git checks:        %d\n", stats.GitChecks)
	fmt.Println("")
	fmt.Println("The background worker will keep the index updated as files change.")
}

func init() {
	rootCmd.AddCommand(indexCmd)
}
