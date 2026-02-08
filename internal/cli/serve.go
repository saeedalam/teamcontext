package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/saeedalam/teamcontext/internal/mcp"
	"github.com/spf13/cobra"
)

var serveDir string

var serveCmd = &cobra.Command{
	Use:   "serve [path]",
	Short: "Start MCP server for IDE integration",
	Long: `Start the MCP (Model Context Protocol) server.

This allows IDE agents like Claude Desktop, Cursor, or other MCP-compatible
tools to interact with TeamContext.

The server communicates via stdio (standard input/output) using JSON-RPC.

You can specify the project directory in three ways:
  1. Pass it as an argument: teamcontext serve /path/to/project
  2. Use the --dir flag: teamcontext serve --dir /path/to/project
  3. Run from within the project directory (uses current directory)

Examples:
  teamcontext serve
  teamcontext serve /path/to/project
  teamcontext serve --dir /path/to/project`,
	Args: cobra.MaximumNArgs(1),
	Run:  runServe,
}

func init() {
	serveCmd.Flags().StringVarP(&serveDir, "dir", "d", "", "Project directory containing .teamcontext")
}

func runServe(cmd *cobra.Command, args []string) {
	// Determine the starting directory for finding .teamcontext
	var startDir string

	// Priority: 1. --dir flag, 2. positional argument, 3. current directory
	if serveDir != "" {
		startDir = serveDir
	} else if len(args) > 0 {
		startDir = args[0]
	} else {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	tcDir, err := findTeamContextDir(startDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'teamcontext init' first.\n")
		os.Exit(1)
	}

	server, err := mcp.NewServer(tcDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}

	// Run the server (blocks until stdin closes)
	server.Run()
}

// findTeamContextDirFromCwd finds the .teamcontext directory from current working directory
// This is the original behavior used by most commands
func findTeamContextDirFromCwd() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findTeamContextDir(cwd)
}

// findTeamContextDir finds the .teamcontext directory starting from the given path
func findTeamContextDir(startDir string) (string, error) {
	// Make path absolute if it isn't already
	absDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	// Check starting directory
	tcDir := filepath.Join(absDir, ".teamcontext")
	if _, err := os.Stat(tcDir); err == nil {
		return tcDir, nil
	}

	// Walk up the directory tree
	dir := absDir
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent

		tcDir = filepath.Join(dir, ".teamcontext")
		if _, err := os.Stat(tcDir); err == nil {
			return tcDir, nil
		}
	}

	return "", fmt.Errorf("not a TeamContext project (no .teamcontext directory found)")
}
