package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/timeparsing"
	"github.com/steveyegge/beads/internal/ui"
)

var relationshipListCmd = &cobra.Command{
	Use:   "list",
	Short: "List relationships for an entity",
	Long: `List relationships connected to an entity.

Use --from to list outgoing relationships (where the entity is the source).
Use --to to list incoming relationships (where the entity is the target).
Specify --type to filter by relationship type.
Use --valid-at to filter relationships valid at a specific point in time.

One of --from or --to is required. Both can be specified to find relationships between two entities.

Examples:
  # List all outgoing relationships
  bd relationship list --from entity-1

  # List all incoming relationships
  bd relationship list --to entity-2

  # List relationships valid at a specific time
  bd relationship list --from entity-1 --valid-at "2024-06-01"

  # Filter by relationship type
  bd relationship list --from entity-1 --type uses

  # Find relationships between two entities
  bd relationship list --from entity-1 --to entity-2`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := rootCtx

		// Get flag values
		fromEntity, _ := cmd.Flags().GetString("from")
		toEntity, _ := cmd.Flags().GetString("to")
		relType, _ := cmd.Flags().GetString("type")
		validAtStr, _ := cmd.Flags().GetString("valid-at")

		// Validate that at least one of --from or --to is provided
		if fromEntity == "" && toEntity == "" {
			FatalErrorRespectJSON("at least one of --from or --to is required")
		}

		// Parse valid-at time (optional)
		var validAt *time.Time
		if validAtStr != "" {
		parsed, err := timeparsing.ParseRelativeTime(validAtStr, time.Now())
			if err != nil {
				FatalErrorRespectJSON("invalid --valid-at: %v", err)
			}
			validAt = &parsed
		}

		// Build search filters
		filters := storage.RelationshipFilters{
			SourceEntityID:   fromEntity,
			TargetEntityID:   toEntity,
			RelationshipType: relType,
			ValidAt:          validAt,
		}

		// Search relationships
		store := getStore()
		relationships, err := store.SearchRelationships(ctx, filters)
		if err != nil {
			FatalErrorRespectJSON("failed to search relationships: %v", err)
		}

		// Output results
		if jsonOutput {
			data, err := json.MarshalIndent(relationships, "", "  ")
			if err != nil {
				FatalErrorRespectJSON("failed to marshal JSON: %v", err)
			}
			fmt.Println(string(data))
		} else {
			if len(relationships) == 0 {
				fmt.Println("No relationships found")
				return
			}

			fmt.Printf("Found %d relationship(s):\n\n", len(relationships))
			for _, rel := range relationships {
				// Format: [ID] source -[type]-> target
				fmt.Printf("%s %s -[%s]-> %s\n",
				ui.RenderID(rel.ID),
				ui.RenderID(rel.SourceEntityID),
				ui.RenderAccent(rel.RelationshipType),
				ui.RenderID(rel.TargetEntityID))
			
			// Show validity period
			if !rel.ValidUntil.IsZero() {
				fmt.Printf("  Valid: %s to %s\n",
						rel.ValidFrom.Format("2006-01-02 15:04"))
				}

				// Show metadata if present
				if len(rel.Metadata) > 0 {
					metadataJSON, _ := json.Marshal(rel.Metadata)
					fmt.Printf("  Metadata: %s\n", string(metadataJSON))
				}

				fmt.Println()
			}
		}
	},
}

func init() {
	relationshipListCmd.Flags().String("from", "", "Source entity ID (list outgoing relationships)")
	relationshipListCmd.Flags().String("to", "", "Target entity ID (list incoming relationships)")
	relationshipListCmd.Flags().String("type", "", "Filter by relationship type")
	relationshipListCmd.Flags().String("valid-at", "", "Filter to relationships valid at this time")

	relationshipCmd.AddCommand(relationshipListCmd)
}
