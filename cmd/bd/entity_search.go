package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/ui"
)

var entitySearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Semantic search for entities using natural language",
	Long: `Search entities using natural language query.

Uses cosine similarity on entity names and summaries to find the most
relevant entities matching your search query.

Examples:
  # Search for entities related to "authentication"
  bd entity search --query "authentication"

  # Search with entity type filter
  bd entity search --query "API client" --entity-type component

  # Limit results and increase threshold
  bd entity search --query "user management" --top 5 --threshold 0.2

  # JSON output
  bd entity search --query "database schema" --json`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := rootCtx

		// Get filter flags
		query, _ := cmd.Flags().GetString("query")
		entityType, _ := cmd.Flags().GetString("entity-type")
		top, _ := cmd.Flags().GetInt("top")
		threshold, _ := cmd.Flags().GetFloat64("threshold")

		// Validate query is non-empty
		if query == "" {
			FatalErrorRespectJSON("--query flag is required and cannot be empty")
		}

		// Build filters
		filters := storage.EntityFilters{
			TextQuery:               query,
			TextSimilarityThreshold: threshold,
			EntityType:              entityType,
			Limit:                   top,
		}

		// Search entities
		entities, err := store.SearchEntities(ctx, filters)
		if err != nil {
			FatalErrorRespectJSON("failed to search entities: %v", err)
		}

		// Output results
		if jsonOutput {
			output := map[string]interface{}{
				"entities": entities,
				"count":    len(entities),
				"query":    query,
			}
			jsonBytes, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			if len(entities) == 0 {
				fmt.Println("No entities found")
				return
			}

			fmt.Printf("Found %d %s matching \"%s\"\n\n",
				len(entities),
				pluralizeEntity(len(entities), "entity", "entities"),
				query)

			for _, entity := range entities {
				// Format: ID | Name (Type)
				fmt.Printf("%s %s %s\n",
					ui.RenderID(entity.ID),
					ui.RenderBold(entity.Name),
					ui.RenderMuted(fmt.Sprintf("(%s)", entity.EntityType)))

				// Show summary if present
				if entity.Summary != "" {
					fmt.Printf("  %s\n", entity.Summary)
				}

				// Show metadata if present
				if len(entity.Metadata) > 0 {
					fmt.Printf("  %s ", ui.RenderMuted("Metadata:"))
					metaJSON, _ := json.Marshal(entity.Metadata)
					fmt.Printf("%s\n", ui.RenderMuted(string(metaJSON)))
				}

				fmt.Println()
			}
		}
	},
}

func init() {
	entityCmd.AddCommand(entitySearchCmd)

	entitySearchCmd.Flags().String("query", "", "Search query text (required)")
	entitySearchCmd.MarkFlagRequired("query")
	entitySearchCmd.Flags().String("entity-type", "", "Filter by entity type")
	entitySearchCmd.Flags().Int("top", 10, "Maximum number of results to return")
	entitySearchCmd.Flags().Float64("threshold", 0.1, "Minimum similarity threshold (0.0-1.0)")
}
