package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/idgen"
	"github.com/steveyegge/beads/internal/types"
)

var entityCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new entity",
	Long: `Create a new entity in the knowledge graph.

Entities represent any trackable object in the system, such as people, components,
documents, or domain concepts. Each entity has a type, name, and summary.

Examples:
  # Create a person entity
  bd entity create --entity-type person --name "Alice Smith" --summary "Senior engineer"

  # Create with custom metadata
  bd entity create --entity-type component --name "API Gateway" --summary "Central API router" --metadata '{"language":"Go","version":"1.0"}'

  # Create with JSON output
  bd entity create --entity-type document --name "Architecture Doc" --summary "System design overview" --json`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckReadonly("entity create")
		ctx := rootCtx

		// Get required flags
		entityType, _ := cmd.Flags().GetString("entity-type")
		if entityType == "" {
			FatalErrorRespectJSON("--entity-type is required")
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			FatalErrorRespectJSON("--name is required")
		}

		// Get optional flags
		summary, _ := cmd.Flags().GetString("summary")
		metadataStr, _ := cmd.Flags().GetString("metadata")
		createdBy, _ := cmd.Flags().GetString("created-by")
		explicitID, _ := cmd.Flags().GetString("id")

		// Parse metadata JSON if provided
		var metadata map[string]interface{}
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
				FatalErrorRespectJSON("invalid JSON in --metadata: %v", err)
			}
		}

		// Generate entity ID if not provided
		entityID := explicitID
		if entityID == "" {
			// Use hash-based ID generation
			entityID = idgen.GenerateHashID("ent", entityType+":"+name, summary, actor, time.Now(), 6, 0)
		}

		// Create entity
		entity := &types.Entity{
			ID:         entityID,
			EntityType: entityType,
			Name:       name,
			Summary:    summary,
			Metadata:   metadata,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
			CreatedBy:  createdBy,
			UpdatedBy:  createdBy,
		}

		err := store.CreateEntity(ctx, entity)
		if err != nil {
			FatalErrorRespectJSON("failed to create entity: %v", err)
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
			fmt.Fprintf(os.Stderr, "Created entity %s\n", entity.ID)
			fmt.Printf("%s (%s)\n", entity.Name, entity.EntityType)
			if entity.Summary != "" {
				fmt.Printf("Summary: %s\n", entity.Summary)
			}
		}
	},
}

func init() {
	entityCmd.AddCommand(entityCreateCmd)

	entityCreateCmd.Flags().String("entity-type", "", "Entity type (e.g., person, component, document)")
	entityCreateCmd.Flags().String("name", "", "Entity name")
	entityCreateCmd.Flags().String("summary", "", "Entity summary")
	entityCreateCmd.Flags().String("metadata", "", "Custom metadata as JSON")
	entityCreateCmd.Flags().String("created-by", "", "Creator name")
	entityCreateCmd.Flags().String("id", "", "Explicit entity ID (auto-generated if not provided)")
}
