package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/ui"
)

var entityShowCmd = &cobra.Command{
	Use:     "show <id>",
	Aliases: []string{"view"},
	Short:   "Show entity details",
	Long: `Show detailed information about an entity.

Entities are nodes in the knowledge graph representing trackable objects
such as people, components, documents, or domain concepts.

Examples:
  # Show entity details
  bd entity show bd-a3f8e9

  # Show with JSON output
  bd entity show bd-a3f8e9 --json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := rootCtx
		entityID := args[0]

		// Get entity
		entity, err := store.GetEntity(ctx, entityID)
		if err != nil {
			if err.Error() == storage.ErrNotFound.Error() || 
			   (err != nil && err.Error() == "not found: entity "+entityID) {
				FatalErrorRespectJSON("entity %s not found", entityID)
			}
			FatalErrorRespectJSON("failed to get entity: %v", err)
		}

		// Output result
		if jsonOutput {
			output := map[string]interface{}{
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
			jsonBytes, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			// Header: ID | Name (Type)
			fmt.Printf("%s %s %s\n\n",
				ui.RenderID(entity.ID),
				ui.RenderBold(entity.Name),
				ui.RenderMuted(fmt.Sprintf("(%s)", entity.EntityType)))

			// Summary
			if entity.Summary != "" {
				fmt.Printf("%s\n%s\n\n", ui.RenderBold("SUMMARY"), entity.Summary)
			}

			// Metadata
			if len(entity.Metadata) > 0 {
				fmt.Printf("%s\n", ui.RenderBold("METADATA"))
				metaJSON, _ := json.MarshalIndent(entity.Metadata, "", "  ")
				fmt.Printf("%s\n\n", string(metaJSON))
			}

			// Timestamps
			fmt.Printf("%s\n", ui.RenderBold("TIMESTAMPS"))
			fmt.Printf("Created: %s", entity.CreatedAt.Format("2006-01-02 15:04"))
			if entity.CreatedBy != "" {
				fmt.Printf(" by %s", entity.CreatedBy)
			}
			fmt.Println()

			fmt.Printf("Updated: %s", entity.UpdatedAt.Format("2006-01-02 15:04"))
			if entity.UpdatedBy != "" {
				fmt.Printf(" by %s", entity.UpdatedBy)
			}
			fmt.Println()
		}
	},
}

func init() {
	entityCmd.AddCommand(entityShowCmd)
}
