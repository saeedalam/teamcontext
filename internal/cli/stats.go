package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/saeedalam/teamcontext/internal/storage"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show detailed statistics",
	Long: `Show detailed statistics about the TeamContext knowledge base.

Displays counts of all knowledge types and system metrics.`,
	Run: runStats,
}

func runStats(cmd *cobra.Command, args []string) {
	tcDir, err := findTeamContextDirFromCwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Run 'teamcontext init' to initialize.")
		return
	}

	jsonStore := storage.NewJSONStore(tcDir)

	stats, err := jsonStore.GetStats()
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
		return
	}

	fmt.Println("┌─────────────────────────────────────────────┐")
	fmt.Println("│           TeamContext Statistics              │")
	fmt.Println("├─────────────────────────────────────────────┤")
	fmt.Println("│ Codebase Index                              │")
	fmt.Printf("│   Files indexed:     %-20d │\n", stats.FilesIndexed)
	fmt.Printf("│   Patterns:          %-20d │\n", stats.Patterns)
	fmt.Println("│                                             │")
	fmt.Println("│ Knowledge Graph                             │")
	fmt.Printf("│   Decisions:         %-20d │\n", stats.Decisions)
	fmt.Printf("│   Warnings:          %-20d │\n", stats.Warnings)
	fmt.Printf("│   Insights:          %-20d │\n", stats.Insights)
	fmt.Println("│                                             │")
	fmt.Println("│ Features                                    │")
	fmt.Printf("│   Active:            %-20d │\n", stats.ActiveFeatures)
	fmt.Printf("│   Archived:          %-20d │\n", stats.ArchivedFeatures)
	fmt.Printf("│   Total:             %-20d │\n", stats.Features)
	fmt.Println("│                                             │")
	fmt.Println("│ Conversations                               │")
	fmt.Printf("│   Total:             %-20d │\n", stats.Conversations)
	fmt.Println("└─────────────────────────────────────────────┘")
}
