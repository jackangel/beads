package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/extraction"
	"github.com/steveyegge/beads/internal/idgen"
	"github.com/steveyegge/beads/internal/storage/dolt"
	"github.com/steveyegge/beads/internal/types"
)

var autoExtract bool

// episodeCreateCmd creates a new episode from raw data.
var episodeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new episode from raw data",
	Long: `Create a new episode (immutable provenance log) from raw data.

Episodes track data ingestion from external sources. The raw data is stored
as a BLOB and can be used to extract entities and relationships later.

Flags:
  --source <source>        Data source (e.g., "github", "jira", "manual") [REQUIRED]
  --file <path>            Path to raw data file [REQUIRED]
  --timestamp <time>       Override ingestion timestamp (default: now)
  --entities <id1,id2,...> Entity IDs extracted from this episode
  --json                   Output in JSON format

Examples:
  bd episode create --source github --file webhook.json
  bd episode create --source jira --file issue-export.json --entities ent-abc,ent-xyz
  bd episode create --source manual --file notes.txt --timestamp "2024-01-15 14:30"`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckReadonly("episode create")
		ctx := rootCtx

		// Check if v8 tables exist (episode command requires v8)
		if err := ensureDirectMode("episode create requires direct database access"); err != nil {
			FatalError("%v", err)
		}
		// Verify v8 schema is available before attempting operations
		store := getStore()
		if err := dolt.CheckV8TablesExist(ctx, store.DB()); err != nil {
			FatalError("%v", err)
		}

		// Get required flags
		source, _ := cmd.Flags().GetString("source")
		filePath, _ := cmd.Flags().GetString("file")

		// Validate required fields
		if source == "" {
			FatalError("--source is required (e.g., 'github', 'jira', 'manual')")
		}
		if filePath == "" {
			FatalError("--file is required (path to raw data file)")
		}

		// Read raw data from file
		rawData, err := os.ReadFile(filePath)
		if err != nil {
			FatalError("failed to read file %q: %v", filePath, err)
		}

		// Get optional flags
		timestampStr, _ := cmd.Flags().GetString("timestamp")
		entitiesExtracted, _ := cmd.Flags().GetStringSlice("entities")

		// Parse timestamp or use current time
		var timestamp time.Time
		if timestampStr != "" {
			timestamp, err = time.Parse(time.RFC3339, timestampStr)
			if err != nil {
				// Try alternative format
				timestamp, err = time.Parse("2006-01-02 15:04", timestampStr)
				if err != nil {
					FatalError("invalid timestamp format: %v (use RFC3339 or 'YYYY-MM-DD HH:MM')", err)
				}
			}
		} else {
			timestamp = time.Now()
		}

		// Generate episode ID
		episodeID := idgen.GenerateHashID("ep", source, filepath.Base(filePath), "", timestamp, 6, 0)

		// Create episode
		episode := &types.Episode{
			ID:                episodeID,
			Timestamp:         timestamp,
			Source:            source,
			RawData:           rawData,
			EntitiesExtracted: entitiesExtracted,
			CreatedAt:         time.Now(),
		}

		// Store episode
		if err := ensureDirectMode("episode create requires direct database access"); err != nil {
			FatalError("%v", err)
		}
		if err := store.CreateEpisode(ctx, episode); err != nil {
			FatalError("failed to create episode: %v", err)
		}

		// Auto-extract if flag is set
		if autoExtract {
			apiKey := os.Getenv("ANTHROPIC_API_KEY")
			if apiKey == "" {
				fmt.Fprintln(os.Stderr, "Warning: --extract requires ANTHROPIC_API_KEY, skipping extraction")
			} else {
				_, err := extraction.ExtractFromEpisode(ctx, store, episode.ID, apiKey)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: extraction failed: %v\n", err)
				}
			}
		}

		// Output result
		if jsonOutput {
			result := map[string]interface{}{
				"id":                 episode.ID,
				"timestamp":          episode.Timestamp.Format(time.RFC3339),
				"source":             episode.Source,
				"raw_data_size":      len(episode.RawData),
				"entities_extracted": episode.EntitiesExtracted,
				"created_at":         episode.CreatedAt.Format(time.RFC3339),
			}
			outputJSON(result)
		} else {
			fmt.Printf("Created episode %s\n", episode.ID)
			fmt.Printf("  Source:       %s\n", episode.Source)
			fmt.Printf("  Timestamp:    %s\n", episode.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Raw data:     %s (%d bytes)\n", filepath.Base(filePath), len(episode.RawData))
			if len(episode.EntitiesExtracted) > 0 {
				fmt.Printf("  Entities:     %v\n", episode.EntitiesExtracted)
			}
		}
	},
}

func init() {
	episodeCreateCmd.Flags().String("source", "", "Data source (e.g., 'github', 'jira', 'manual') [REQUIRED]")
	episodeCreateCmd.Flags().String("file", "", "Path to raw data file [REQUIRED]")
	episodeCreateCmd.Flags().String("timestamp", "", "Override ingestion timestamp (default: now)")
	episodeCreateCmd.Flags().StringSlice("entities", nil, "Entity IDs extracted from this episode")
	episodeCreateCmd.Flags().BoolVar(&autoExtract, "extract", false, "Auto-extract entities after creation")
}
