package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage"
)

var entityDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete an entity",
	Long: `Delete an entity from the knowledge graph.

This is a destructive operation that permanently removes the entity.
Use with caution.

Examples:
  # Delete an entity
  bd entity delete bd-a3f8e9

  # Delete with JSON output
  bd entity delete bd-a3f8e9 --json

  # Force delete without confirmation
  bd entity delete bd-a3f8e9 --force`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		CheckReadonly("entity delete")
		ctx := rootCtx
		entityID := args[0]

		// Get entity details before deletion (for confirmation/output)
		entity, err := store.GetEntity(ctx, entityID)
		if err != nil {
			if err.Error() == storage.ErrNotFound.Error() || 
			   (err != nil && err.Error() == "not found: entity "+entityID) {
				FatalErrorRespectJSON("entity %s not found", entityID)
			}
			FatalErrorRespectJSON("failed to get entity: %v", err)
		}

		// Confirmation prompt (unless --force or --json)
		force, _ := cmd.Flags().GetBool("force")
		if !force && !jsonOutput {
			fmt.Fprintf(os.Stderr, "Delete entity %s (%s - %s)? [y/N]: ", 
				entity.ID, entity.Name, entity.EntityType)
			
			var response string
			fmt.Scanln(&response)
			
			if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
				fmt.Fprintf(os.Stderr, "Deletion cancelled\n")
				os.Exit(0)
			}
		}

		// Delete entity
		err = store.DeleteEntity(ctx, entityID)
		if err != nil {
			FatalErrorRespectJSON("failed to delete entity: %v", err)
		}

		// Output result
		if jsonOutput {
			output := map[string]interface{}{
				"deleted": true,
				"id":      entityID,
				"name":    entity.Name,
				"type":    entity.EntityType,
			}
			jsonBytes, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Fprintf(os.Stderr, "Deleted entity %s\n", entityID)
			fmt.Printf("%s (%s)\n", entity.Name, entity.EntityType)
		}
	},
}

func init() {
	entityCmd.AddCommand(entityDeleteCmd)

	entityDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}
