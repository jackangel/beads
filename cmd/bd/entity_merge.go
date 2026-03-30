package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage"
)

var entityMergeCmd = &cobra.Command{
	Use:   "merge <source-id> <target-id>",
	Short: "Merge source entity into target entity",
	Long: `Merge a source entity into a target entity.

This operation:
- Moves all relationships from source to target
- Marks source as merged (soft delete)
- Preserves metadata and provenance

Examples:
  # Merge two entities
  bd entity merge bd-a3f8e9 bd-b2c4d1

  # Merge with JSON output
  bd entity merge bd-a3f8e9 bd-b2c4d1 --json`,
	Args: cobra.ExactArgs(2),
	RunE: runEntityMerge,
}

func init() {
	entityCmd.AddCommand(entityMergeCmd)
}

func runEntityMerge(cmd *cobra.Command, args []string) error {
	CheckReadonly("entity merge")

	sourceID := args[0]
	targetID := args[1]
	ctx := rootCtx

	// Verify both entities exist before merging
	sourceEntity, err := store.GetEntity(ctx, sourceID)
	if err != nil {
		if err.Error() == storage.ErrNotFound.Error() || 
		   (err != nil && err.Error() == "not found: entity "+sourceID) {
			FatalErrorRespectJSON("source entity %s not found", sourceID)
		}
		return fmt.Errorf("failed to get source entity: %w", err)
	}

	targetEntity, err := store.GetEntity(ctx, targetID)
	if err != nil {
		if err.Error() == storage.ErrNotFound.Error() || 
		   (err != nil && err.Error() == "not found: entity "+targetID) {
			FatalErrorRespectJSON("target entity %s not found", targetID)
		}
		return fmt.Errorf("failed to get target entity: %w", err)
	}

	// Perform the merge
	if err := store.MergeEntities(ctx, sourceID, targetID, actor); err != nil {
		return fmt.Errorf("merging entities: %w", err)
	}

	// Output result
	if jsonOutput {
		output := map[string]interface{}{
			"merged":    true,
			"source_id": sourceID,
			"target_id": targetID,
			"message":   fmt.Sprintf("Entity %s merged into %s", sourceID, targetID),
		}
		jsonBytes, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "✓ Merged %s into %s\n", sourceID, targetID)
		fmt.Fprintf(os.Stderr, "  Source: %s (%s) - now marked as merged\n", 
			sourceEntity.Name, sourceEntity.EntityType)
		fmt.Fprintf(os.Stderr, "  Target: %s (%s) - received all relationships\n", 
			targetEntity.Name, targetEntity.EntityType)
	}

	return nil
}
