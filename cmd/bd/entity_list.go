package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/ui"
)

var entityListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List entities",
	Long: `List entities matching the provided filters.

Entities are nodes in the knowledge graph representing trackable objects
such as people, components, documents, or domain concepts.

Examples:
  # List all entities
  bd entity list

  # List entities by type
  bd entity list --entity-type person

  # List entities with name filter
  bd entity list --name "API"

  # List with JSON output
  bd entity list --entity-type component --json

  # List with pagination
  bd entity list --limit 10 --offset 0`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := rootCtx

		// Get filter flags
		entityType, _ := cmd.Flags().GetString("entity-type")
		name, _ := cmd.Flags().GetString("name")
		createdBy, _ := cmd.Flags().GetString("created-by")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		// Build filters
		filters := storage.EntityFilters{
			EntityType: entityType,
			Name:       name,
			CreatedBy:  createdBy,
			Limit:      limit,
			Offset:     offset,
		}

		// Search entities
		entities, err := store.SearchEntities(ctx, filters)
		if err != nil {
			FatalErrorRespectJSON("failed to list entities: %v", err)
		}

		// Output results
		if jsonOutput {
			output := make([]map[string]interface{}, len(entities))
			for i, entity := range entities {
				output[i] = map[string]interface{}{
					"id":          entity.ID,
					"entity_type": entity.EntityType,
					"name":        entity.Name,
					"summary":     entity.Summary,
					"metadata":    entity.Metadata,
					"created_at":  entity.CreatedAt,
					"updated_at":  entity.UpdatedAt,
					"created_by":  entity.CreatedBy,
					"updated_by":  entity.UpdatedBy,
				}
			}
			jsonBytes, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			if len(entities) == 0 {
				fmt.Println("No entities found")
				return
			}

			fmt.Printf("Found %d %s\n\n", len(entities), pluralizeEntity(len(entities), "entity", "entities"))

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
	entityCmd.AddCommand(entityListCmd)

	entityListCmd.Flags().String("entity-type", "", "Filter by entity type")
	entityListCmd.Flags().String("name", "", "Filter by name (partial match)")
	entityListCmd.Flags().String("created-by", "", "Filter by creator")
	entityListCmd.Flags().Int("limit", 0, "Maximum number of results (0 = no limit)")
	entityListCmd.Flags().Int("offset", 0, "Number of results to skip (for pagination)")
}

// pluralizeEntity returns singular or plural form based on count
func pluralizeEntity(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
