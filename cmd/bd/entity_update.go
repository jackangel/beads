package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
)

var entityUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an existing entity",
	Long: `Update an existing entity in the knowledge graph.

Only the specified fields will be updated. All other fields remain unchanged.

Examples:
  # Update entity name
  bd entity update bd-a3f8e9 --name "Alice Jones"

  # Update summary
  bd entity update bd-a3f8e9 --summary "Lead architect"

  # Update multiple fields
  bd entity update bd-a3f8e9 --name "Alice Jones" --summary "Lead architect"

  # Update with custom metadata
  bd entity update bd-a3f8e9 --metadata '{"role":"architect","team":"platform"}'

  # Update with JSON output
  bd entity update bd-a3f8e9 --name "Alice Jones" --json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		CheckReadonly("entity update")
		ctx := rootCtx
		entityID := args[0]

		// Check if entity exists first
		existingEntity, err := store.GetEntity(ctx, entityID)
		if err != nil {
			if err.Error() == storage.ErrNotFound.Error() || 
			   (err != nil && err.Error() == "not found: entity "+entityID) {
				FatalErrorRespectJSON("entity %s not found", entityID)
			}
			FatalErrorRespectJSON("failed to get entity: %v", err)
		}

		// Get update flags
		name, _ := cmd.Flags().GetString("name")
		summary, _ := cmd.Flags().GetString("summary")
		entityType, _ := cmd.Flags().GetString("entity-type")
		metadataStr, _ := cmd.Flags().GetString("metadata")
		updatedBy, _ := cmd.Flags().GetString("updated-by")

		// Check if at least one field is specified
		if name == "" && summary == "" && entityType == "" && metadataStr == "" && updatedBy == "" {
			FatalErrorRespectJSON("at least one field must be specified for update")
		}

		// Parse metadata JSON if provided
		var metadata map[string]interface{}
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
				FatalErrorRespectJSON("invalid JSON in --metadata: %v", err)
			}
		}

		// Build update entity (only set non-empty fields)
		updateEntity := &types.Entity{
			ID: entityID,
		}

		if name != "" {
			updateEntity.Name = name
		}
		if summary != "" {
			updateEntity.Summary = summary
		}
		if entityType != "" {
			updateEntity.EntityType = entityType
		}
		if metadataStr != "" {
			updateEntity.Metadata = metadata
		}
		if updatedBy != "" {
			updateEntity.UpdatedBy = updatedBy
		}

		// Update entity
		err = store.UpdateEntity(ctx, updateEntity)
		if err != nil {
			FatalErrorRespectJSON("failed to update entity: %v", err)
		}

		// Fetch updated entity for display
		updatedEntity, err := store.GetEntity(ctx, entityID)
		if err != nil {
			FatalErrorRespectJSON("failed to get updated entity: %v", err)
		}

		// Output result
		if jsonOutput {
			output := map[string]interface{}{
				"id":          updatedEntity.ID,
				"entity_type": updatedEntity.EntityType,
				"name":        updatedEntity.Name,
				"summary":     updatedEntity.Summary,
				"metadata":    updatedEntity.Metadata,
				"created_at":  updatedEntity.CreatedAt,
				"updated_at":  updatedEntity.UpdatedAt,
				"created_by":  updatedEntity.CreatedBy,
				"updated_by":  updatedEntity.UpdatedBy,
			}
			jsonBytes, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Fprintf(os.Stderr, "Updated entity %s\n", entityID)
			fmt.Printf("%s (%s)\n", updatedEntity.Name, updatedEntity.EntityType)
			if updatedEntity.Summary != "" {
				fmt.Printf("Summary: %s\n", updatedEntity.Summary)
			}

			// Show what changed
			changed := []string{}
			if name != "" && name != existingEntity.Name {
				changed = append(changed, fmt.Sprintf("name: %q → %q", existingEntity.Name, name))
			}
			if summary != "" && summary != existingEntity.Summary {
				changed = append(changed, fmt.Sprintf("summary: %q → %q", existingEntity.Summary, summary))
			}
			if entityType != "" && entityType != existingEntity.EntityType {
				changed = append(changed, fmt.Sprintf("type: %q → %q", existingEntity.EntityType, entityType))
			}
			if metadataStr != "" {
				changed = append(changed, "metadata updated")
			}

			if len(changed) > 0 {
				fmt.Fprintf(os.Stderr, "\nChanged: %v\n", changed)
			}
		}
	},
}

func init() {
	entityCmd.AddCommand(entityUpdateCmd)

	entityUpdateCmd.Flags().String("name", "", "New entity name")
	entityUpdateCmd.Flags().String("summary", "", "New entity summary")
	entityUpdateCmd.Flags().String("entity-type", "", "New entity type")
	entityUpdateCmd.Flags().String("metadata", "", "New custom metadata as JSON")
	entityUpdateCmd.Flags().String("updated-by", "", "Updater name")
}
