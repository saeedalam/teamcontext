package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// IDEConfig represents configuration for a specific IDE
type IDEConfig struct {
	Name             string
	ConfigPaths      []string // Global config paths
	ProjectConfigDir string   // Directory name for project-level config (e.g., ".cursor")
}

var supportedIDEs = map[string]IDEConfig{
	"cursor": {
		Name: "Cursor",
		ConfigPaths: []string{
			"~/.cursor/mcp.json",
			"~/.config/cursor/mcp.json",
		},
		ProjectConfigDir: ".cursor",
	},
	"claude": {
		Name: "Claude Desktop",
		ConfigPaths: []string{
			"~/Library/Application Support/Claude/claude_desktop_config.json",           // macOS
			"~/.config/claude-desktop/claude_desktop_config.json",                        // Linux
			filepath.Join(os.Getenv("APPDATA"), "Claude", "claude_desktop_config.json"),  // Windows
		},
		ProjectConfigDir: "", // Claude Desktop doesn't support project-level configs
	},
	"claudecode": {
		Name: "Claude Code (CLI)",
		ConfigPaths: []string{
			"~/.claude/config.json",
		},
		ProjectConfigDir: ".teamcontext/idp", // Hidden to avoid cluttering root, but Claude Code looks for .mcp.json at root.
	},
	"windsurf": {
		Name: "Windsurf",
		ConfigPaths: []string{
			"~/.windsurf/mcp.json",
			"~/.config/windsurf/mcp.json",
		},
		ProjectConfigDir: ".windsurf",
	},
	"vscode": {
		Name: "VS Code",
		ConfigPaths: []string{
			"~/.vscode/mcp.json",
			"~/.config/Code/User/mcp.json",
		},
		ProjectConfigDir: ".vscode",
	},
}

// Special case for Claude Code: it prefers .mcp.json at the root of the project
var claudeCLIConfig = IDEConfig{
	Name:             "Claude Code (CLI)",
	ConfigPaths:      []string{"~/.claude/config.json"},
	ProjectConfigDir: ".", // Special marker for root-level .mcp.json
}

func init() {
	supportedIDEs["claudecode"] = claudeCLIConfig
}

var installGlobal bool

var installCmd = &cobra.Command{
	Use:   "install <ide>",
	Short: "Install TeamContext for an IDE",
	Long: `Install TeamContext MCP server configuration for your IDE.

Supported IDEs:
  cursor    - Cursor IDE
  claude    - Claude Desktop
  windsurf  - Windsurf (Codeium)
  vscode    - VS Code (if MCP extension installed)

When run from a project with .teamcontext, creates a PROJECT-LEVEL config
that includes the project path. This is the recommended approach.

Use --global to install to the global config instead (not recommended for
most IDEs, as the MCP server won't know which project to serve).

Examples:
  cd /path/to/project
  teamcontext init              # Initialize project first
  teamcontext install cursor    # Creates .cursor/mcp.json with project path

  teamcontext install cursor --global  # Install to ~/.cursor/mcp.json (not recommended)`,
	Args: cobra.ExactArgs(1),
	Run:  runInstall,
}

var uninstallGlobal bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <ide>",
	Short: "Remove TeamContext from an IDE",
	Long: `Remove TeamContext MCP server configuration from your IDE.

Supported IDEs: cursor, claude, windsurf, vscode

By default, removes from project-level config if in a project.
Use --global to remove from global config.

Examples:
  teamcontext uninstall cursor           # Remove from project config
  teamcontext uninstall cursor --global  # Remove from global config`,
	Args: cobra.ExactArgs(1),
	Run:  runUninstall,
}

func init() {
	installCmd.Flags().BoolVarP(&installGlobal, "global", "g", false, "Install to global config instead of project-level")
	uninstallCmd.Flags().BoolVarP(&uninstallGlobal, "global", "g", false, "Remove from global config instead of project-level")
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}

func runInstall(cmd *cobra.Command, args []string) {
	ideName := strings.ToLower(args[0])

	ide, ok := supportedIDEs[ideName]
	if !ok {
		fmt.Printf("Unknown IDE: %s\n\n", ideName)
		fmt.Println("Supported IDEs:")
		for name, config := range supportedIDEs {
			fmt.Printf("  %-10s - %s\n", name, config.Name)
		}
		return
	}

	// Find the teamcontext binary
	binaryPath, err := findBinaryPath()
	if err != nil {
		fmt.Printf("Error finding TeamContext binary: %v\n", err)
		fmt.Println("\nPlease ensure teamcontext is in your PATH or run from its directory.")
		return
	}

	// Check if we're in a project with .teamcontext
	projectRoot, tcDir, inProject := findProjectRoot()

	// Determine if we should do project-level or global install
	useProjectLevel := inProject && !installGlobal && ide.ProjectConfigDir != ""

	if useProjectLevel {
		fmt.Printf("Installing TeamContext for %s (project-level)...\n\n", ide.Name)
		fmt.Printf("Project: %s\n", projectRoot)
		fmt.Printf("Binary: %s\n", binaryPath)

		// Create project-level config
		configFileName := "mcp.json"
		if ideName == "claudecode" && ide.ProjectConfigDir == "." {
			configFileName = ".mcp.json"
		}
		configPath := filepath.Join(projectRoot, ide.ProjectConfigDir, configFileName)

		// Create directory if needed
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Printf("Error creating config directory: %v\n", err)
			return
		}

		// Read or create config
		config, err := readOrCreateConfig(configPath)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			return
		}

		// Add TeamContext with project path
		if err := addTeamContextToConfigWithPath(config, binaryPath, projectRoot); err != nil {
			fmt.Printf("Error updating config: %v\n", err)
			return
		}

		// Write config back
		if err := writeConfig(configPath, config); err != nil {
			fmt.Printf("Error writing config: %v\n", err)
			return
		}

		fmt.Println()
		fmt.Printf("TeamContext installed for %s!\n", ide.Name)
		fmt.Printf("Config created: %s\n", configPath)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Printf("  1. Restart %s\n", ide.Name)
		fmt.Printf("  2. Open this project in %s\n", ide.Name)
		fmt.Println("  3. Ask the AI: \"What TeamContext tools are available?\"")

		// Warn about global config conflict
		globalPath, err := findConfigPath(ide.ConfigPaths)
		if err == nil {
			globalConfig, err := readOrCreateConfig(globalPath)
			if err == nil {
				if mcpServers, ok := globalConfig["mcpServers"].(map[string]interface{}); ok {
					if _, exists := mcpServers["teamcontext"]; exists {
						fmt.Println()
						fmt.Println("Note: TeamContext is also in your global config.")
						fmt.Println("The project-level config will take precedence.")
						fmt.Printf("Consider removing it: teamcontext uninstall %s --global\n", ideName)
					}
				}
			}
		}
	} else {
		// Global installation
		if inProject && ide.ProjectConfigDir != "" && !installGlobal {
			fmt.Println("Warning: Installing globally, but you're in a project.")
			fmt.Println("The MCP server won't know which project to serve.")
			fmt.Printf("Consider: teamcontext install %s (without --global)\n\n", ideName)
		} else if !inProject && ide.ProjectConfigDir != "" {
			fmt.Println("Note: Not in a TeamContext project.")
			fmt.Println("Run 'teamcontext init' first, then 'teamcontext install' from the project.")
			fmt.Println()
		}

		fmt.Printf("Installing TeamContext for %s (global)...\n\n", ide.Name)
		fmt.Printf("Binary: %s\n", binaryPath)

		// Find the config file
		configPath, err := findConfigPath(ide.ConfigPaths)
		if err != nil {
			fmt.Printf("\nCould not find %s config file.\n", ide.Name)
			fmt.Println("Checked locations:")
			for _, p := range ide.ConfigPaths {
				fmt.Printf("  - %s\n", expandPath(p))
			}
			fmt.Println("\nCreating config at:", expandPath(ide.ConfigPaths[0]))
			configPath = expandPath(ide.ConfigPaths[0])
		}

		// Read or create config
		config, err := readOrCreateConfig(configPath)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			return
		}

		// Add TeamContext to MCP servers (without project path - user must run from project)
		if err := addTeamContextToConfig(config, binaryPath); err != nil {
			fmt.Printf("Error updating config: %v\n", err)
			return
		}

		// Write config back
		if err := writeConfig(configPath, config); err != nil {
			fmt.Printf("Error writing config: %v\n", err)
			return
		}

		fmt.Println()
		fmt.Printf("TeamContext installed for %s!\n", ide.Name)
		fmt.Printf("Config updated: %s\n", configPath)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Printf("  1. Restart %s\n", ide.Name)
		fmt.Println("  2. Initialize TeamContext in your project: teamcontext init")
		fmt.Println("  3. Ask the AI: \"What TeamContext tools are available?\"")

		// For global installs without project path, warn about limitations
		if ide.ProjectConfigDir != "" && !inProject {
			fmt.Println()
			fmt.Println("Important: Global install requires running the IDE from the project directory.")
			fmt.Println("For better results, run 'teamcontext install' from within an initialized project.")
		}

		// If we found a .teamcontext but still doing global, note that
		if inProject && tcDir != "" {
			fmt.Println()
			fmt.Printf("Detected project at: %s\n", projectRoot)
			fmt.Println("Consider project-level install for better experience.")
		}
	}
}

// findProjectRoot looks for .teamcontext directory and returns the project root
func findProjectRoot() (projectRoot string, tcDir string, found bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", false
	}

	// Check current directory
	tc := filepath.Join(cwd, ".teamcontext")
	if _, err := os.Stat(tc); err == nil {
		return cwd, tc, true
	}

	// Walk up the directory tree
	dir := cwd
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent

		tc = filepath.Join(dir, ".teamcontext")
		if _, err := os.Stat(tc); err == nil {
			return dir, tc, true
		}
	}

	return "", "", false
}

func runUninstall(cmd *cobra.Command, args []string) {
	ideName := strings.ToLower(args[0])

	ide, ok := supportedIDEs[ideName]
	if !ok {
		fmt.Printf("Unknown IDE: %s\n", ideName)
		return
	}

	// Check if we're in a project
	projectRoot, _, inProject := findProjectRoot()

	// Determine which config to modify
	useProjectLevel := inProject && !uninstallGlobal && ide.ProjectConfigDir != ""

	var configPath string
	var configType string

	if useProjectLevel {
		configPath = filepath.Join(projectRoot, ide.ProjectConfigDir, "mcp.json")
		configType = "project"
	} else {
		var err error
		configPath, err = findConfigPath(ide.ConfigPaths)
		if err != nil {
			fmt.Printf("Global config file not found for %s\n", ide.Name)
			return
		}
		configType = "global"
	}

	fmt.Printf("Removing TeamContext from %s (%s config)...\n\n", ide.Name, configType)

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Config file not found: %s\n", configPath)
		return
	}

	// Read config
	config, err := readOrCreateConfig(configPath)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	// Remove TeamContext from MCP servers
	if err := removeTeamContextFromConfig(config); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Write config back
	if err := writeConfig(configPath, config); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
		return
	}

	fmt.Printf("TeamContext removed from %s!\n", ide.Name)
	fmt.Printf("Config updated: %s\n", configPath)
	fmt.Printf("Restart %s for changes to take effect.\n", ide.Name)
}

func findBinaryPath() (string, error) {
	// First, try to find in PATH
	path, err := exec.LookPath("teamcontext")
	if err == nil {
		return path, nil
	}

	// Try current executable path
	executable, err := os.Executable()
	if err == nil {
		return executable, nil
	}

	// Try current working directory
	cwd, err := os.Getwd()
	if err == nil {
		localPath := filepath.Join(cwd, "teamcontext")
		if runtime.GOOS == "windows" {
			localPath += ".exe"
		}
		if _, err := os.Stat(localPath); err == nil {
			return localPath, nil
		}
	}

	return "", fmt.Errorf("teamcontext binary not found")
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func findConfigPath(paths []string) (string, error) {
	for _, path := range paths {
		expanded := expandPath(path)
		if _, err := os.Stat(expanded); err == nil {
			return expanded, nil
		}
	}
	return "", fmt.Errorf("config file not found")
}

func readOrCreateConfig(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create directory if needed
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, err
			}
			// Return empty config
			return map[string]interface{}{
				"mcpServers": map[string]interface{}{},
			}, nil
		}
		return nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Ensure mcpServers exists
	if _, ok := config["mcpServers"]; !ok {
		config["mcpServers"] = map[string]interface{}{}
	}

	return config, nil
}

func addTeamContextToConfig(config map[string]interface{}, binaryPath string) error {
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = map[string]interface{}{}
		config["mcpServers"] = mcpServers
	}

	// Clean up legacy teambrain entry
	if _, exists := mcpServers["teambrain"]; exists {
		fmt.Println("Removing legacy 'teambrain' entry...")
		delete(mcpServers, "teambrain")
	}

	// Check if already installed
	if _, exists := mcpServers["teamcontext"]; exists {
		fmt.Println("TeamContext is already configured. Updating...")
	}

	// Add TeamContext server config (no project path - for global installs)
	mcpServers["teamcontext"] = map[string]interface{}{
		"command": binaryPath,
		"args":    []string{"serve"},
	}

	return nil
}

func addTeamContextToConfigWithPath(config map[string]interface{}, binaryPath, projectPath string) error {
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = map[string]interface{}{}
		config["mcpServers"] = mcpServers
	}

	// Clean up legacy teambrain entry
	if _, exists := mcpServers["teambrain"]; exists {
		fmt.Println("Removing legacy 'teambrain' entry...")
		delete(mcpServers, "teambrain")
	}

	// Check if already installed
	if _, exists := mcpServers["teamcontext"]; exists {
		fmt.Println("TeamContext is already configured. Updating...")
	}

	// Add TeamContext server config with project path
	mcpServers["teamcontext"] = map[string]interface{}{
		"command": binaryPath,
		"args":    []string{"serve", projectPath},
	}

	return nil
}

func removeTeamContextFromConfig(config map[string]interface{}) error {
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no MCP servers configured")
	}

	if _, exists := mcpServers["teamcontext"]; !exists {
		return fmt.Errorf("TeamContext is not installed")
	}

	delete(mcpServers, "teamcontext")
	return nil
}

func writeConfig(path string, config map[string]interface{}) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
