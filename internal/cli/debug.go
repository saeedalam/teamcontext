package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/saeedalam/teamcontext/internal/mcp"
	"github.com/spf13/cobra"
)

var debugMcpCmd = &cobra.Command{
	Use:   "debug-mcp <tool_name> <json_params>",
	Short: "Debug MCP tools directly from CLI",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		toolName := args[0]
		paramsStr := "{}"
		if len(args) > 1 {
			paramsStr = args[1]
		}

		// Setup server context
		cwd, _ := os.Getwd()
		tcDir := filepath.Join(cwd, ".teamcontext")

		server, err := mcp.NewServer(tcDir)
		if err != nil {
			return err
		}
		
		var params json.RawMessage = []byte(paramsStr)
		var resp interface{}

		switch toolName {
		case "get_tree":
			resp, err = server.HandleToolCall("get_tree", params)
		case "get_signature":
			resp, err = server.HandleToolCall("get_signature", params)
		default:
			return fmt.Errorf("unknown tool: %s", toolName)
		}

		if err != nil {
			return err
		}

		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(debugMcpCmd)
}
