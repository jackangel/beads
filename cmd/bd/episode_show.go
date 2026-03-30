package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/ui"
)

// episodeShowCmd shows details of a specific episode.
var episodeShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show episode details",
	Long: `Show detailed information about a specific episode.

Episodes are immutable provenance logs. This command displays the episode's
metadata and raw data content.

Flags:
  --json            Output in JSON format
  --raw             Output only raw data (binary-safe)
  --pretty          Pretty-print raw data if JSON

Examples:
  bd episode show ep-abc123
  bd episode show ep-abc123 --json
  bd episode show ep-abc123 --raw > raw-data.json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := rootCtx
		episodeID := args[0]

		// Get flags
		rawOnly, _ := cmd.Flags().GetBool("raw")
		pretty, _ := cmd.Flags().GetBool("pretty")

		// Fetch episode
		if err := ensureDirectMode("episode show requires direct database access"); err != nil {
			FatalError("%v", err)
		}
		episode, err := store.GetEpisode(ctx, episodeID)
		if err != nil {
			FatalError("failed to get episode: %v", err)
		}
		if episode == nil {
			FatalError("episode %s not found", episodeID)
		}

		// Handle --raw flag (output only raw data)
		if rawOnly {
			fmt.Print(string(episode.RawData))
			return
		}

		// Output full details
		if jsonOutput {
			result := map[string]interface{}{
				"id":                 episode.ID,
				"timestamp":          episode.Timestamp.Format(time.RFC3339),
				"source":             episode.Source,
				"raw_data":           string(episode.RawData),
				"raw_data_size":      len(episode.RawData),
				"entities_extracted": episode.EntitiesExtracted,
				"created_at":         episode.CreatedAt.Format(time.RFC3339),
			}
			if episode.Metadata != nil {
				result["metadata"] = episode.Metadata
			}
			outputJSON(result)
		} else {
			// Human-readable output
			fmt.Println(ui.RenderBold("Episode Details"))
			fmt.Println(strings.Repeat("─", 80))
			fmt.Printf("ID:               %s\n", episode.ID)
			fmt.Printf("Source:           %s\n", episode.Source)
			fmt.Printf("Timestamp:        %s\n", episode.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Printf("Created:          %s\n", episode.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Raw data size:    %d bytes\n", len(episode.RawData))

			if len(episode.EntitiesExtracted) > 0 {
				fmt.Printf("Entities:         %v\n", episode.EntitiesExtracted)
			} else {
				fmt.Printf("Entities:         %s\n", ui.RenderMuted("(none)"))
			}

			if episode.Metadata != nil && len(episode.Metadata) > 0 {
				fmt.Println("\nMetadata:")
				for key, value := range episode.Metadata {
					fmt.Printf("  %s: %v\n", key, value)
				}
			}

			// Display raw data
			fmt.Println("\nRaw Data:")
			fmt.Println(strings.Repeat("─", 80))

			rawDataStr := string(episode.RawData)

			// If --pretty flag and raw data is JSON, pretty-print it
			if pretty && isJSON(rawDataStr) {
				var prettyData interface{}
				if err := json.Unmarshal(episode.RawData, &prettyData); err == nil {
					prettyBytes, err := json.MarshalIndent(prettyData, "", "  ")
					if err == nil {
						rawDataStr = string(prettyBytes)
					}
				}
			}

			// Truncate if too long
			const maxDisplayBytes = 2000
			if len(rawDataStr) > maxDisplayBytes {
				fmt.Println(rawDataStr[:maxDisplayBytes])
				fmt.Printf("\n%s\n", ui.RenderMuted(fmt.Sprintf("... (truncated, %d bytes total, use --raw to see full content)", len(episode.RawData))))
			} else {
				fmt.Println(rawDataStr)
			}
		}
	},
}

func init() {
	episodeShowCmd.Flags().Bool("raw", false, "Output only raw data (binary-safe)")
	episodeShowCmd.Flags().Bool("pretty", false, "Pretty-print raw data if JSON")
}

// isJSON checks if a string is valid JSON
func isJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
