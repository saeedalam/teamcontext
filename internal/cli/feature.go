package cli

import (
	"fmt"

	"github.com/saeedalam/teamcontext/internal/storage"
	"github.com/saeedalam/teamcontext/pkg/types"
	"github.com/spf13/cobra"
)

var featureBranch string
var featureExtends string

var startCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a new feature context",
	Long: `Start a new feature context.

A feature context is a workspace for tracking progress, decisions, and conversations
related to a specific feature or task.

Example:
  teamcontext start payment-retry
  teamcontext start auth-v2 --branch feature/auth-v2
  teamcontext start notification-prefs --extends notification-system`,
	Args: cobra.ExactArgs(1),
	Run:  runStart,
}

var resumeCmd = &cobra.Command{
	Use:   "resume <name>",
	Short: "Resume an existing feature context",
	Long: `Resume an existing feature context.

This marks the feature as actively being worked on.

Example:
  teamcontext resume payment-retry`,
	Args: cobra.ExactArgs(1),
	Run:  runResume,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all features",
	Long: `List all feature contexts.

Shows both active and archived features.

Example:
  teamcontext list
  teamcontext list --active`,
	Run: runList,
}

var archiveCmd = &cobra.Command{
	Use:   "archive <name>",
	Short: "Archive a completed feature",
	Long: `Archive a completed feature context.

Archived features are preserved for future reference and can be inherited by new features.

Example:
  teamcontext archive payment-retry`,
	Args: cobra.ExactArgs(1),
	Run:  runArchive,
}

var recallCmd = &cobra.Command{
	Use:   "recall <name>",
	Short: "Recall an archived feature",
	Long: `Recall an archived feature back to active status.

Example:
  teamcontext recall payment-retry`,
	Args: cobra.ExactArgs(1),
	Run:  runRecall,
}

var activeOnly bool

func init() {
	startCmd.Flags().StringVarP(&featureBranch, "branch", "b", "", "Git branch name")
	startCmd.Flags().StringVarP(&featureExtends, "extends", "e", "", "Parent feature to inherit from")

	listCmd.Flags().BoolVarP(&activeOnly, "active", "a", false, "Show only active features")
}

func runStart(cmd *cobra.Command, args []string) {
	featureID := args[0]

	tcDir, err := findTeamContextDirFromCwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	jsonStore := storage.NewJSONStore(tcDir)

	// Check if feature already exists
	existing, _ := jsonStore.GetFeature(featureID)
	if existing != nil {
		fmt.Printf("Feature '%s' already exists.\n", featureID)
		fmt.Printf("Use 'teamcontext resume %s' to resume it.\n", featureID)
		return
	}

	feature := &types.Feature{
		ID:      featureID,
		Branch:  featureBranch,
		Extends: featureExtends,
	}

	if err := jsonStore.CreateFeature(feature); err != nil {
		fmt.Printf("Error creating feature: %v\n", err)
		return
	}

	// Update SQLite index
	sqliteIndex, err := storage.NewSQLiteIndex(tcDir)
	if err == nil {
		sqliteIndex.IndexFeature(feature)
		sqliteIndex.Close()
	}

	fmt.Printf("Started feature context: %s\n", featureID)

	if featureBranch != "" {
		fmt.Printf("  Branch: %s\n", featureBranch)
	}
	if featureExtends != "" {
		fmt.Printf("  Extends: %s\n", featureExtends)
	}

	fmt.Println()
	fmt.Println("The feature is now active. Your IDE agent can:")
	fmt.Println("  - Store decisions with add_decision")
	fmt.Println("  - Record warnings with add_warning")
	fmt.Println("  - Save conversations with save_conversation")
	fmt.Println("  - Update state with update_feature_state")
}

func runResume(cmd *cobra.Command, args []string) {
	featureID := args[0]

	tcDir, err := findTeamContextDirFromCwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	jsonStore := storage.NewJSONStore(tcDir)

	feature, err := jsonStore.GetFeature(featureID)
	if err != nil {
		fmt.Printf("Feature '%s' not found.\n", featureID)
		fmt.Println("Use 'teamcontext list' to see available features.")
		return
	}

	// Update last accessed
	if err := jsonStore.UpdateFeature(feature); err != nil {
		fmt.Printf("Error updating feature: %v\n", err)
		return
	}

	fmt.Printf("Resumed feature: %s\n", featureID)
	fmt.Printf("  Status: %s\n", feature.Status)

	if feature.CurrentState != "" {
		fmt.Printf("  State: %s\n", feature.CurrentState)
	}

	if len(feature.RelevantFiles) > 0 {
		fmt.Printf("  Files: %d\n", len(feature.RelevantFiles))
	}

	// Get decisions and conversations count
	decisions, _ := jsonStore.GetDecisionsByFeature(featureID)
	conversations, _ := jsonStore.GetConversations(featureID)

	fmt.Printf("  Decisions: %d\n", len(decisions))
	fmt.Printf("  Conversations: %d\n", len(conversations))
}

func runList(cmd *cobra.Command, args []string) {
	tcDir, err := findTeamContextDirFromCwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	jsonStore := storage.NewJSONStore(tcDir)
	features, err := jsonStore.GetFeatures()
	if err != nil {
		fmt.Printf("Error listing features: %v\n", err)
		return
	}

	if len(features) == 0 {
		fmt.Println("No features found.")
		fmt.Println("Use 'teamcontext start <name>' to create one.")
		return
	}

	// Separate active and archived
	var active, archived []types.Feature
	for _, f := range features {
		if f.Status == "active" {
			active = append(active, f)
		} else {
			archived = append(archived, f)
		}
	}

	if len(active) > 0 {
		fmt.Println("Active Features:")
		for _, f := range active {
			state := f.CurrentState
			if state == "" {
				state = "(no state)"
			}
			if len(state) > 50 {
				state = state[:47] + "..."
			}
			fmt.Printf("  - %s\n", f.ID)
			fmt.Printf("    State: %s\n", state)
			fmt.Printf("    Last accessed: %s\n", f.LastAccessed.Format("2006-01-02 15:04"))
		}
	}

	if !activeOnly && len(archived) > 0 {
		fmt.Println()
		fmt.Println("Archived Features:")
		for _, f := range archived {
			fmt.Printf("  - %s (archived %s)\n", f.ID, f.ArchivedAt.Format("2006-01-02"))
		}
	}
}

func runArchive(cmd *cobra.Command, args []string) {
	featureID := args[0]

	tcDir, err := findTeamContextDirFromCwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	jsonStore := storage.NewJSONStore(tcDir)

	// Get feature to check it exists
	feature, err := jsonStore.GetFeature(featureID)
	if err != nil {
		fmt.Printf("Feature '%s' not found.\n", featureID)
		return
	}

	if feature.Status == "archived" {
		fmt.Printf("Feature '%s' is already archived.\n", featureID)
		return
	}

	// Get counts before archiving
	decisions, _ := jsonStore.GetDecisionsByFeature(featureID)
	conversations, _ := jsonStore.GetConversations(featureID)

	if err := jsonStore.ArchiveFeature(featureID); err != nil {
		fmt.Printf("Error archiving feature: %v\n", err)
		return
	}

	fmt.Printf("Archived feature: %s\n", featureID)
	fmt.Printf("  Decisions preserved: %d\n", len(decisions))
	fmt.Printf("  Conversations preserved: %d\n", len(conversations))
	fmt.Println()
	fmt.Println("The feature is now available for inheritance by new features:")
	fmt.Printf("  teamcontext start new-feature --extends %s\n", featureID)
}

func runRecall(cmd *cobra.Command, args []string) {
	featureID := args[0]

	tcDir, err := findTeamContextDirFromCwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	jsonStore := storage.NewJSONStore(tcDir)

	if err := jsonStore.RecallFeature(featureID); err != nil {
		fmt.Printf("Error recalling feature: %v\n", err)
		fmt.Println("Make sure the feature exists in the archive.")
		return
	}

	fmt.Printf("Recalled feature: %s\n", featureID)
	fmt.Println("The feature is now active again.")
}
