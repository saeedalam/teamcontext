package cli

import (
	"fmt"
	"strings"

	"github.com/saeedalam/teamcontext/internal/storage"
	"github.com/spf13/cobra"
)

var searchType string
var searchLimit int

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search knowledge base",
	Long: `Search the TeamContext knowledge base.

Searches across files, decisions, and warnings using full-text search.

Example:
  teamcontext search "payment retry"
  teamcontext search "authentication" --type decisions
  teamcontext search "error handling" --limit 10`,
	Args: cobra.ExactArgs(1),
	Run:  runSearch,
}

func init() {
	searchCmd.Flags().StringVarP(&searchType, "type", "t", "all", "Search type: all, files, decisions, warnings")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 20, "Max results per category")
}

func runSearch(cmd *cobra.Command, args []string) {
	query := args[0]

	tcDir, err := findTeamContextDirFromCwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	sqliteIndex, err := storage.NewSQLiteIndex(tcDir)
	if err != nil {
		fmt.Printf("Error opening index: %v\n", err)
		fmt.Println("Run 'teamcontext rebuild' to create the search index.")
		return
	}
	defer sqliteIndex.Close()

	fmt.Printf("Searching for: %s\n", query)
	fmt.Println(strings.Repeat("-", 40))

	totalResults := 0

	// Search files
	if searchType == "all" || searchType == "files" {
		files, _ := sqliteIndex.SearchFiles(query, "", searchLimit)
		if len(files) > 0 {
			fmt.Println()
			fmt.Printf("Files (%d results):\n", len(files))
			for _, f := range files {
				summary := f.Summary
				if len(summary) > 60 {
					summary = summary[:57] + "..."
				}
				fmt.Printf("  %s\n", f.Path)
				fmt.Printf("    %s\n", summary)
			}
			totalResults += len(files)
		}
	}

	// Search decisions
	if searchType == "all" || searchType == "decisions" {
		decisions, _ := sqliteIndex.SearchDecisions(query, "", "", searchLimit)
		if len(decisions) > 0 {
			fmt.Println()
			fmt.Printf("Decisions (%d results):\n", len(decisions))
			for _, d := range decisions {
				content := d.Content
				if len(content) > 60 {
					content = content[:57] + "..."
				}
				fmt.Printf("  [%s] %s\n", d.ID, content)
				if d.Feature != "" {
					fmt.Printf("    Feature: %s\n", d.Feature)
				}
			}
			totalResults += len(decisions)
		}
	}

	// Search warnings
	if searchType == "all" || searchType == "warnings" {
		warnings, _ := sqliteIndex.SearchWarnings(query, "", "", searchLimit)
		if len(warnings) > 0 {
			fmt.Println()
			fmt.Printf("Warnings (%d results):\n", len(warnings))
			for _, w := range warnings {
				content := w.Content
				if len(content) > 60 {
					content = content[:57] + "..."
				}
				severity := w.Severity
				if severity == "" {
					severity = "warning"
				}
				fmt.Printf("  [%s] [%s] %s\n", w.ID, severity, content)
			}
			totalResults += len(warnings)
		}
	}

	if totalResults == 0 {
		fmt.Println()
		fmt.Println("No results found.")
	} else {
		fmt.Println()
		fmt.Printf("Total: %d results\n", totalResults)
	}
}
