package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/saeedalam/teamcontext/internal/storage"
)

var feedLimit int
var feedSince string
var feedJSON bool

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Show recent team knowledge activity",
	Long: `Display a feed of recent knowledge activity across the project.

Shows decisions, warnings, patterns, insights, and conversations
sorted by most recent first.

Examples:
  teamcontext feed                    # Last 20 entries
  teamcontext feed --limit 50         # Last 50 entries
  teamcontext feed --since 7d         # Last 7 days
  teamcontext feed --since 2025-01-01 # Since a specific date
  teamcontext feed --json             # JSON output for piping`,
	Run: runFeed,
}

func init() {
	feedCmd.Flags().IntVar(&feedLimit, "limit", 20, "Maximum entries to show")
	feedCmd.Flags().StringVar(&feedSince, "since", "", "Show entries since (e.g., 7d, 24h, 2025-01-01)")
	feedCmd.Flags().BoolVar(&feedJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(feedCmd)
}

// FeedEntry represents a single feed item
type FeedEntry struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Detail    string    `json:"detail,omitempty"`
	Author    string    `json:"author,omitempty"`
	Feature   string    `json:"feature,omitempty"`
	Severity  string    `json:"severity,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func runFeed(cmd *cobra.Command, args []string) {
	_, tcDir, found := findProjectRoot()
	if !found {
		fmt.Println("Error: No .teamcontext directory found.")
		fmt.Println("Run 'teamcontext init' first.")
		return
	}

	store := storage.NewJSONStore(tcDir)

	// Parse since
	var sinceTime time.Time
	if feedSince != "" {
		sinceTime = parseSince(feedSince)
	}

	entries := collectFeedEntries(store, sinceTime)

	// Sort by most recent
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})

	// Apply limit
	if feedLimit > 0 && len(entries) > feedLimit {
		entries = entries[:feedLimit]
	}

	if feedJSON {
		data, _ := json.MarshalIndent(entries, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Pretty print
	if len(entries) == 0 {
		fmt.Println("No recent activity.")
		return
	}

	fmt.Printf("Recent Activity (%d entries)\n", len(entries))
	fmt.Println(strings.Repeat("-", 60))

	for _, e := range entries {
		age := formatAge(e.CreatedAt)
		icon := feedIcon(e.Type)
		fmt.Printf("\n%s  %s  %s\n", icon, e.Type, age)
		fmt.Printf("   %s\n", e.Title)
		if e.Detail != "" {
			fmt.Printf("   %s\n", e.Detail)
		}
		if e.Author != "" {
			fmt.Printf("   by %s\n", e.Author)
		}
	}
	fmt.Println()
}

func collectFeedEntries(store *storage.JSONStore, since time.Time) []FeedEntry {
	var entries []FeedEntry

	// Decisions
	decisions, _ := store.GetDecisions()
	for _, d := range decisions {
		if !since.IsZero() && d.CreatedAt.Before(since) {
			continue
		}
		entries = append(entries, FeedEntry{
			Type:      "decision",
			ID:        d.ID,
			Title:     d.Content,
			Detail:    d.Reason,
			Author:    d.Author,
			Feature:   d.Feature,
			CreatedAt: d.CreatedAt,
		})
	}

	// Warnings
	warnings, _ := store.GetWarnings()
	for _, w := range warnings {
		if !since.IsZero() && w.CreatedAt.Before(since) {
			continue
		}
		entries = append(entries, FeedEntry{
			Type:      "warning",
			ID:        w.ID,
			Title:     w.Content,
			Detail:    w.Reason,
			Author:    w.Author,
			Feature:   w.Feature,
			Severity:  w.Severity,
			CreatedAt: w.CreatedAt,
		})
	}

	// Patterns
	patterns, _ := store.GetPatterns()
	for _, p := range patterns {
		if !since.IsZero() && p.CreatedAt.Before(since) {
			continue
		}
		entries = append(entries, FeedEntry{
			Type:      "pattern",
			ID:        p.ID,
			Title:     p.Name,
			Detail:    p.Description,
			CreatedAt: p.CreatedAt,
		})
	}

	// Insights
	insights, _ := store.GetInsights()
	for _, i := range insights {
		if !since.IsZero() && i.CreatedAt.Before(since) {
			continue
		}
		entries = append(entries, FeedEntry{
			Type:      "insight",
			ID:        i.ID,
			Title:     i.Content,
			Author:    i.Author,
			Feature:   i.Feature,
			CreatedAt: i.CreatedAt,
		})
	}

	// Conversations
	convs, _ := store.GetAllConversations()
	for _, c := range convs {
		if !since.IsZero() && c.CreatedAt.Before(since) {
			continue
		}
		entries = append(entries, FeedEntry{
			Type:      "conversation",
			ID:        c.ID,
			Title:     c.Summary,
			Feature:   c.Feature,
			CreatedAt: c.CreatedAt,
		})
	}

	// Evolution events
	timeline, _ := store.GetEvolutionTimeline()
	if timeline != nil {
		for _, ev := range timeline.Events {
			if !since.IsZero() && ev.Timestamp.Before(since) {
				continue
			}
			entries = append(entries, FeedEntry{
				Type:      "event",
				ID:        ev.ID,
				Title:     ev.Title,
				Detail:    ev.Description,
				Author:    ev.Author,
				CreatedAt: ev.Timestamp,
			})
		}
	}

	return entries
}

func parseSince(s string) time.Time {
	// Try duration format: 7d, 24h, 30m
	if len(s) > 1 {
		unit := s[len(s)-1]
		numStr := s[:len(s)-1]
		var num int
		if _, err := fmt.Sscanf(numStr, "%d", &num); err == nil {
			switch unit {
			case 'd':
				return time.Now().Add(-time.Duration(num) * 24 * time.Hour)
			case 'h':
				return time.Now().Add(-time.Duration(num) * time.Hour)
			case 'm':
				return time.Now().Add(-time.Duration(num) * time.Minute)
			}
		}
	}

	// Try ISO date
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}

	fmt.Printf("Warning: could not parse --since=%q, showing all entries\n", s)
	return time.Time{}
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

func feedIcon(entryType string) string {
	switch entryType {
	case "decision":
		return "[DEC]"
	case "warning":
		return "[WRN]"
	case "pattern":
		return "[PAT]"
	case "insight":
		return "[INS]"
	case "conversation":
		return "[CON]"
	case "event":
		return "[EVT]"
	default:
		return "[???]"
	}
}
