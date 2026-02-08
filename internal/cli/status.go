package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/saeedalam/teamcontext/internal/search"
	"github.com/saeedalam/teamcontext/internal/storage"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current TeamContext status",
	Long: `Show the current status of TeamContext.

Displays:
- Project information
- Knowledge base statistics
- Active features
- System status`,
	Run: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) {
	tcDir, err := findTeamContextDirFromCwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Run 'teamcontext init' to initialize.")
		return
	}

	jsonStore := storage.NewJSONStore(tcDir)

	// Get config
	config, err := jsonStore.GetConfig()
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	fmt.Println("TeamContext Status")
	fmt.Println("================")
	fmt.Println()

	// Project info
	fmt.Printf("Project: %s\n", config.Name)
	fmt.Printf("Version: %s\n", config.Version)
	fmt.Printf("Location: %s\n", tcDir)
	fmt.Println()

	// Get stats
	files, _ := jsonStore.GetFilesIndex()
	decisions, _ := jsonStore.GetDecisions()
	warnings, _ := jsonStore.GetWarnings()
	insights, _ := jsonStore.GetInsights()
	features, _ := jsonStore.GetFeatures()
	patterns, _ := jsonStore.GetPatterns()

	fmt.Println("Knowledge Base:")
	fmt.Printf("  Files indexed:    %d\n", len(files))
	fmt.Printf("  Patterns:         %d\n", len(patterns))
	fmt.Printf("  Decisions:        %d\n", len(decisions))
	fmt.Printf("  Warnings:         %d\n", len(warnings))
	fmt.Printf("  Insights:         %d\n", len(insights))
	fmt.Println()

	// Active features
	var activeFeatures []string
	for _, f := range features {
		if f.Status == "active" {
			activeFeatures = append(activeFeatures, f.ID)
		}
	}

	fmt.Println("Features:")
	fmt.Printf("  Active:           %d\n", len(activeFeatures))
	fmt.Printf("  Total:            %d\n", len(features))

	if len(activeFeatures) > 0 {
		fmt.Println()
		fmt.Println("Active Features:")
		for _, id := range activeFeatures {
			for _, f := range features {
				if f.ID == id {
					state := f.CurrentState
					if state == "" {
						state = "(no state)"
					}
					if len(state) > 50 {
						state = state[:47] + "..."
					}
					fmt.Printf("  - %s: %s\n", id, state)
					break
				}
			}
		}
	}

	fmt.Println()

	// System status
	fmt.Println("System:")
	if search.CheckRipgrep() {
		fmt.Println("  ripgrep:          installed")
	} else {
		fmt.Println("  ripgrep:          NOT FOUND (code search will use fallback)")
	}

	// Check if SQLite index exists
	sqliteIndex, err := storage.NewSQLiteIndex(tcDir)
	if err != nil {
		fmt.Println("  SQLite index:     NOT INITIALIZED")
		fmt.Println("                    Run 'teamcontext rebuild' to create")
	} else {
		stats, _ := sqliteIndex.GetStats()
		fmt.Printf("  SQLite index:     OK (%d files, %d decisions)\n",
			stats["files"], stats["decisions"])
		sqliteIndex.Close()
	}

	fmt.Println()
}
