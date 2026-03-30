package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/ui"
)

var relationshipShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show relationship details",
	Long: `Show detailed information about a specific relationship.

Displays the relationship's source entity, target entity, type, temporal validity
window, metadata, and attribution information.

Examples:
  # Show relationship details
  bd relationship show rel-abc123

  # Show in JSON format
  bd relationship show rel-abc123 --json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := rootCtx
		relID := args[0]

		// Retrieve relationship
		store := getStore()
		rel, err := store.GetRelationship(ctx, relID)
		if err != nil {
			FatalErrorRespectJSON("failed to get relationship: %v", err)
		}

		// Output result
		if jsonOutput {
			data, err := json.MarshalIndent(rel, "", "  ")
			if err != nil {
				FatalErrorRespectJSON("failed to marshal JSON: %v", err)
			}
			fmt.Println(string(data))
		} else {
			// Human-readable format
			fmt.Printf("%s Relationship %s\n\n", ui.RenderInfoIcon(), ui.RenderID(rel.ID))

			// Core relationship
			fmt.Printf("  %s -[%s]-> %s\n\n",
				ui.RenderID(rel.SourceEntityID),
				ui.RenderAccent(rel.RelationshipType),
				ui.RenderID(rel.TargetEntityID))

			// Temporal validity
			fmt.Printf("Temporal Validity:\n")
			if rel.ValidUntil != nil {
				fmt.Printf("  From: %s\n", rel.ValidFrom.Format("2006-01-02 15:04:05 MST"))
				fmt.Printf("  Until: %s\n", rel.ValidUntil.Format("2006-01-02 15:04:05 MST"))
				if rel.IsValidAt(rel.ValidFrom) {
					fmt.Printf("  Status: %s\n", ui.StatusClosedStyle.Render("Expired"))
				} else {
					fmt.Printf("  Status: %s\n", ui.StatusInProgressStyle.Render("Active"))
				}
			} else {
				fmt.Printf("  From: %s\n", rel.ValidFrom.Format("2006-01-02 15:04:05 MST"))
				fmt.Printf("  Until: (ongoing)\n")
				fmt.Printf("  Status: %s\n", ui.StatusInProgressStyle.Render("Active"))
			}
			fmt.Println()

			// Metadata
			if len(rel.Metadata) > 0 {
				fmt.Printf("Metadata:\n")
				metadataJSON, _ := json.MarshalIndent(rel.Metadata, "  ", "  ")
				fmt.Printf("  %s\n\n", string(metadataJSON))
			}

			// Attribution
			fmt.Printf("Attribution:\n")
			fmt.Printf("  Created: %s\n", rel.CreatedAt.Format("2006-01-02 15:04:05 MST"))
			if rel.CreatedBy != "" {
				fmt.Printf("  Created by: %s\n", rel.CreatedBy)
			}
		}
	},
}

func init() {
	relationshipCmd.AddCommand(relationshipShowCmd)
}
