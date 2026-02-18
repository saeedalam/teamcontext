package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "teamcontext",
	Short: "The Technical Memory That Outlives Everyone",
	Long: `TeamContext - Your Company's Technical Memory

TeamContext captures, preserves, and serves everything your team learns about
your codebase, architecture, and development history. It's the knowledge
layer that sits UNDER all AI tools.

Key Features:
  - Knowledge Graph: Connected decisions, warnings, patterns, files
  - Feature Contexts: Working memory with full lifecycle
  - Evolution Timeline: Track how knowledge changes over time
  - MCP Protocol: Works with any MCP-compatible AI tool

TeamContext does NOT think - the IDE agent (Claude, Cursor) does all intelligence.

Quick Start:
  teamcontext init          Initialize in current directory
  teamcontext serve         Start MCP server for IDE integration
  teamcontext start <name>  Start a new feature context
  teamcontext status        Show current status`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(recallCmd)
	rootCmd.AddCommand(rebuildCmd)
	rootCmd.AddCommand(reindexCmd)
	rootCmd.AddCommand(searchCmd)
	// installCmd and uninstallCmd are registered in install.go
}
