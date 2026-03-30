package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var relationshipDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a relationship",
	Long: `Delete a relationship from the knowledge graph.

WARNING: This performs a hard delete and removes the relationship from the database entirely.
For most use cases, you should use 'bd relationship update <id> --valid-until <time>' instead,
which preserves the historical record while marking the relationship as no longer active.

Hard deletion should only be used for:
- Correcting mistakes (wrong source/target entities)
- Removing test data
- Compliance requirements (data removal)

Examples:
  # Delete a relationship (hard delete)
  bd relationship delete rel-abc123

  # Delete with JSON output
  bd relationship delete rel-abc123 --json

  # RECOMMENDED: Close temporal window instead (preserves history)
  bd relationship update rel-abc123 --valid-until "2024-12-31 23:59"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		CheckReadonly("relationship delete")
		ctx := rootCtx
		relID := args[0]

		// Get confirmation flag
		force, _ := cmd.Flags().GetBool("force")

		// Retrieve relationship first to show what will be deleted
		store := getStore()
		rel, err := store.GetRelationship(ctx, relID)
		if err != nil {
			FatalErrorRespectJSON("failed to get relationship: %v", err)
		}

		// Confirm deletion (unless --force is specified)
		if !force && !jsonOutput {
			fmt.Printf("WARNING: About to permanently delete relationship:\n")
			fmt.Printf("  ID: %s\n", rel.ID)
			fmt.Printf("  %s -[%s]-> %s\n", rel.SourceEntityID, rel.RelationshipType, rel.TargetEntityID)
			fmt.Printf("\nThis operation cannot be undone.\n")
			fmt.Printf("Consider using 'bd relationship update %s --valid-until <time>' instead.\n\n", relID)
			fmt.Printf("Proceed with deletion? (y/N): ")

			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "yes" && response != "Y" && response != "YES" {
				fmt.Println("Deletion cancelled")
				return
			}
		}

		// Delete relationship
		if err := store.DeleteRelationship(ctx, relID); err != nil {
			FatalErrorRespectJSON("failed to delete relationship: %v", err)
		}

		// Mark that we wrote data
		commandDidWrite.Store(true)

		// Output result
		if jsonOutput {
			result := map[string]interface{}{
				"deleted": true,
				"id":      relID,
			}
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				FatalErrorRespectJSON("failed to marshal JSON: %v", err)
			}
			fmt.Println(string(data))
		} else {
			fmt.Printf("Deleted relationship %s\n", relID)
		}
	},
}

func init() {
	relationshipDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	relationshipCmd.AddCommand(relationshipDeleteCmd)
}
