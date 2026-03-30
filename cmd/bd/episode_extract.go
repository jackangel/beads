package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/extraction"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
)

var episodeExtractCmd = &cobra.Command{
	Use:   "extract <episode-id>",
	Short: "Extract entities and relationships from an episode using LLM",
	Long: `Process an episode's raw data and extract structured knowledge graph data.

This command uses an LLM (Claude) to analyze the raw data stored in an episode
and extract entities (people, concepts, components) and their relationships.

Requires ANTHROPIC_API_KEY environment variable.

Examples:
  bd episode extract ep-abc123
  bd episode extract ep-abc123 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runEpisodeExtract,
}

func init() {
	episodeCmd.AddCommand(episodeExtractCmd)
}

func runEpisodeExtract(cmd *cobra.Command, args []string) error {
	CheckReadonly("extract")

	episodeID := args[0]
	ctx := rootCtx

	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	// Ensure direct mode for database access
	if err := ensureDirectMode("episode extract requires direct database access"); err != nil {
		return fmt.Errorf("database access error: %w", err)
	}

	// Extract entities and relationships
	result, err := extraction.ExtractFromEpisode(ctx, store, episodeID, apiKey)
	if err != nil {
		return fmt.Errorf("extracting from episode: %w", err)
	}

	// Create extracted entities, converting from extraction types to storage types
	entityIDs := []string{}
	entityNameToID := make(map[string]string) // For relationship resolution
	for _, extractedEntity := range result.Entities {
		// Convert to types.Entity using helper
		entity := extraction.ConvertToTypesEntity(extractedEntity, actor)
		
		if err := store.CreateEntity(ctx, entity); err != nil {
			return fmt.Errorf("creating entity %s: %w", entity.Name, err)
		}
		entityIDs = append(entityIDs, entity.ID)
		entityNameToID[entity.Name] = entity.ID
	}

	// Create extracted relationships, resolving names to IDs
	relationshipIDs := []string{}
	for _, extractedRel := range result.Relationships {
		// Resolve entity names to IDs
		sourceID, sourceOK := entityNameToID[extractedRel.SourceName]
		targetID, targetOK := entityNameToID[extractedRel.TargetName]
		
		if !sourceOK || !targetOK {
			// Skip relationships where we can't resolve both entities
			continue
		}

		// Convert to types.Relationship using helper
		rel := extraction.ConvertToTypesRelationship(extractedRel, sourceID, targetID, actor)
		
		if err := store.CreateRelationship(ctx, rel); err != nil {
			return fmt.Errorf("creating relationship: %w", err)
		}
		relationshipIDs = append(relationshipIDs, rel.ID)
	}

	// Update episode extracted_at timestamp
	// Note: This requires UpdateEpisode method which may not exist yet
	// For now, we skip this step - it will be added when episode updates are implemented

	if jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"episode_id":            episodeID,
			"entities_created":      len(entityIDs),
			"relationships_created": len(relationshipIDs),
			"entity_ids":            entityIDs,
			"relationship_ids":      relationshipIDs,
		})
	}

	fmt.Printf("✓ Extracted %d entities and %d relationships from episode %s\n",
		len(entityIDs), len(relationshipIDs), episodeID)

	return nil
}

// findEntityByName searches for an entity by name and type.
// This is a helper for resolving relationship source/target names to IDs.
func findEntityByName(ctx context.Context, store storage.Storage, name, entityType string) (*types.Entity, error) {
	filters := storage.EntityFilters{
		Name:       name,
		EntityType: entityType,
		Limit:      1,
	}
	entities, err := store.SearchEntities(ctx, filters)
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return nil, fmt.Errorf("entity not found: name=%s, type=%s", name, entityType)
	}
	return entities[0], nil
}
