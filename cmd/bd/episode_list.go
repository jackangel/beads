package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/timeparsing"
	"github.com/steveyegge/beads/internal/ui"
)

// episodeListCmd lists episodes matching search criteria.
var episodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List episodes matching search criteria",
	Long: `List episodes (immutable provenance logs) matching search criteria.

Episodes are ordered by timestamp descending (newest first) by default.
All filters are combined with AND logic.

Flags:
  --source <source>        Filter by data source (e.g., "github", "jira")
  --since <time>           Filter episodes from this timestamp (inclusive)
  --until <time>           Filter episodes to this timestamp (inclusive)
  --entities <id1,id2,...> Filter by extracted entity IDs
  --limit <n>              Maximum number of results (default: 50)
  --offset <n>             Skip first N results for pagination
  --json                   Output in JSON format

Examples:
  bd episode list --source github
  bd episode list --since "2024-01-01" --until "2024-01-31"
  bd episode list --entities ent-abc123 --json
  bd episode list --limit 10 --offset 20`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := rootCtx

		// Get filter flags
		source, _ := cmd.Flags().GetString("source")
		sinceStr, _ := cmd.Flags().GetString("since")
		untilStr, _ := cmd.Flags().GetString("until")
		entities, _ := cmd.Flags().GetStringSlice("entities")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		// Parse time filters
		var sinceTime, untilTime *time.Time
		var err error

		if sinceStr != "" {
			t, err := timeparsing.ParseRelativeTime(sinceStr, time.Now())
			if err != nil {
				FatalError("invalid --since time: %v", err)
			}
			sinceTime = &t
		}

		if untilStr != "" {
			t, err := timeparsing.ParseRelativeTime(untilStr, time.Now())
			if err != nil {
				FatalError("invalid --until time: %v", err)
			}
			untilTime = &t
		}

		// Build filters
		filters := storage.EpisodeFilters{
			Source:            source,
			TimestampStart:    sinceTime,
			TimestampEnd:      untilTime,
			EntitiesExtracted: entities,
			Limit:             limit,
			Offset:            offset,
		}

		// Query episodes
		if err := ensureDirectMode("episode list requires direct database access"); err != nil {
			FatalError("%v", err)
		}
		episodes, err := store.SearchEpisodes(ctx, filters)
		if err != nil {
			FatalError("failed to search episodes: %v", err)
		}

		// Output results
		if jsonOutput {
			result := make([]map[string]interface{}, len(episodes))
			for i, ep := range episodes {
				result[i] = map[string]interface{}{
					"id":                 ep.ID,
					"timestamp":          ep.Timestamp.Format(time.RFC3339),
					"source":             ep.Source,
					"raw_data_size":      len(ep.RawData),
					"entities_extracted": ep.EntitiesExtracted,
					"created_at":         ep.CreatedAt.Format(time.RFC3339),
				}
			}
			outputJSON(result)
		} else {
			if len(episodes) == 0 {
				fmt.Println("No episodes found")
				return
			}

			// Sort by timestamp descending (newest first)
			sort.Slice(episodes, func(i, j int) bool {
				return episodes[i].Timestamp.After(episodes[j].Timestamp)
			})

			// Display table header
			fmt.Printf("%-15s %-20s %-12s %-10s %s\n",
				"ID", "TIMESTAMP", "SOURCE", "SIZE", "ENTITIES")
			fmt.Println(ui.RenderMuted("─────────────────────────────────────────────────────────────────────────────"))

			// Display episodes
			for _, ep := range episodes {
				timestamp := ep.Timestamp.Format("2006-01-02 15:04:05")
				size := formatBytes(int64(len(ep.RawData)))
				entitiesStr := formatEntities(ep.EntitiesExtracted)

				fmt.Printf("%-15s %-20s %-12s %-10s %s\n",
					ep.ID, timestamp, ep.Source, size, entitiesStr)
			}

			fmt.Printf("\n%d episodes found", len(episodes))
			if limit > 0 && len(episodes) == limit {
				fmt.Printf(" (limit: %d, use --offset to paginate)", limit)
			}
			fmt.Println()
		}
	},
}

func init() {
	episodeListCmd.Flags().String("source", "", "Filter by data source")
	episodeListCmd.Flags().String("since", "", "Filter episodes from this timestamp (inclusive)")
	episodeListCmd.Flags().String("until", "", "Filter episodes to this timestamp (inclusive)")
	episodeListCmd.Flags().StringSlice("entities", nil, "Filter by extracted entity IDs")
	episodeListCmd.Flags().Int("limit", 50, "Maximum number of results")
	episodeListCmd.Flags().Int("offset", 0, "Skip first N results for pagination")
}

// formatEntities formats entity list for display
func formatEntities(entities []string) string {
	if len(entities) == 0 {
		return ui.RenderMuted("(none)")
	}
	if len(entities) <= 3 {
		return fmt.Sprintf("%v", entities)
	}
	return fmt.Sprintf("%s, %s, ... (%d total)", entities[0], entities[1], len(entities))
}
