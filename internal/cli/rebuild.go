package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/saeedalam/teamcontext/internal/storage"
)

var rebuildCmd = &cobra.Command{
	Use:   "rebuild",
	Short: "Rebuild SQLite index from JSON files",
	Long: `Rebuild the SQLite search index from JSON files.

The SQLite database is a cache for fast searching. It can be rebuilt
at any time from the JSON files (which are the source of truth).

This is useful after:
- Cloning a repository
- Manually editing JSON files
- If the SQLite index gets corrupted

Example:
  teamcontext rebuild`,
	Run: runRebuild,
}

func runRebuild(cmd *cobra.Command, args []string) {
	tcDir, err := findTeamContextDirFromCwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Rebuilding SQLite index from JSON files...")

	jsonStore := storage.NewJSONStore(tcDir)
	sqliteIndex, err := storage.NewSQLiteIndex(tcDir)
	if err != nil {
		fmt.Printf("Error creating SQLite index: %v\n", err)
		return
	}
	defer sqliteIndex.Close()

	if err := sqliteIndex.RebuildFromJSON(jsonStore); err != nil {
		fmt.Printf("Error rebuilding index: %v\n", err)
		return
	}

	// Get stats
	stats, _ := sqliteIndex.GetStats()

	fmt.Println("Rebuild complete!")
	fmt.Println()
	fmt.Println("Indexed:")
	fmt.Printf("  Files:      %d\n", stats["files"])
	fmt.Printf("  Decisions:  %d\n", stats["decisions"])
	fmt.Printf("  Warnings:   %d\n", stats["warnings"])
	fmt.Printf("  Features:   %d\n", stats["features"])
}
