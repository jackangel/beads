package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/extraction"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/timeparsing"
	"github.com/steveyegge/beads/internal/types"
)

var (
	extractSince string
	extractLimit int
)

// episodeExtractAllCmd processes multiple unextracted episodes in batch.
var episodeExtractAllCmd = &cobra.Command{
	Use:   "extract-all",
	Short: "Batch extract entities from unprocessed episodes",
	Long: `Process all episodes where extracted_at is NULL.

This command finds episodes that haven't been processed yet and extracts
entities and relationships from their raw data using an LLM (Claude).

Optionally filter by episodes created after --since timestamp.

Requires ANTHROPIC_API_KEY environment variable.

Flags:
  --since <time>    Process episodes since timestamp (ISO 8601 or relative like "7d")
  --limit <n>       Maximum episodes to process (default: 10)
  --json            Output in JSON format

Examples:
  bd episode extract-all
  bd episode extract-all --since "2024-01-01"
  bd episode extract-all --since "7d" --limit 5
  bd episode extract-all --json`,
	RunE: runEpisodeExtractAll,
}

func init() {
	episodeExtractAllCmd.Flags().StringVar(&extractSince, "since", "", "Process episodes since timestamp (ISO 8601 or relative)")
	episodeExtractAllCmd.Flags().IntVar(&extractLimit, "limit", 10, "Maximum episodes to process")
	episodeCmd.AddCommand(episodeExtractAllCmd)
}

func runEpisodeExtractAll(cmd *cobra.Command, args []string) error {
	CheckReadonly("extract-all")
	ctx := rootCtx

	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	// Ensure direct mode for database access
	if err := ensureDirectMode("episode extract-all requires direct database access"); err != nil {
		return fmt.Errorf("database access error: %w", err)
	}

	// Parse --since filter if provided
	var sinceTime *time.Time
	if extractSince != "" {
		t, err := timeparsing.ParseRelativeTime(extractSince, time.Now())
		if err != nil {
			return fmt.Errorf("invalid --since time: %w", err)
		}
		sinceTime = &t
	}

	// Fetch all episodes (we'll filter for unextracted ones in-memory)
	// Note: Ideally we'd use a filter like ExtractedAt: nil, but that's not
	// in EpisodeFilters yet. This requires task 1-1 (schema + filter addition).
	filters := storage.EpisodeFilters{
		TimestampStart: sinceTime,
		Limit:          0, // Fetch all, we'll limit after filtering
		Offset:         0,
	}

	episodes, err := store.SearchEpisodes(ctx, filters)
	if err != nil {
		return fmt.Errorf("searching episodes: %w", err)
	}

	// Filter for unextracted episodes (where EntitiesExtracted is empty)
	// TODO: Once extracted_at column exists, replace this with:
	//   - Add ExtractedAt field to EpisodeFilters
	//   - Filter in SQL: WHERE extracted_at IS NULL
	var unextractedEpisodes []*types.Episode
	for _, ep := range episodes {
		if len(ep.EntitiesExtracted) == 0 {
			unextractedEpisodes = append(unextractedEpisodes, ep)
		}
	}

	// Apply limit
	if extractLimit > 0 && len(unextractedEpisodes) > extractLimit {
		unextractedEpisodes = unextractedEpisodes[:extractLimit]
	}

	if len(unextractedEpisodes) == 0 {
		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"episodes_processed": 0,
				"message":            "no unextracted episodes found",
			})
		}
		fmt.Println("No unextracted episodes found")
		return nil
	}

	// Process each episode
	results := []map[string]interface{}{}
	totalEntities := 0
	totalRelationships := 0

	for _, episode := range unextractedEpisodes {
		episodeResult, err := processEpisode(ctx, store, episode, apiKey)
		if err != nil {
			// Log error but continue processing other episodes
			if jsonOutput {
				results = append(results, map[string]interface{}{
					"episode_id": episode.ID,
					"error":      err.Error(),
					"success":    false,
				})
			} else {
				fmt.Fprintf(os.Stderr, "✗ Failed to process episode %s: %v\n", episode.ID, err)
			}
			continue
		}

		totalEntities += episodeResult["entities_created"].(int)
		totalRelationships += episodeResult["relationships_created"].(int)
		results = append(results, episodeResult)

		if !jsonOutput {
			fmt.Printf("✓ Episode %s: extracted %d entities, %d relationships\n",
				episode.ID,
				episodeResult["entities_created"].(int),
				episodeResult["relationships_created"].(int))
		}
	}

	// Output summary
	if jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"episodes_processed":      len(results),
			"total_entities_created":  totalEntities,
			"total_relationships_created": totalRelationships,
			"results":                 results,
		})
	}

	fmt.Printf("\n✓ Processed %d episodes\n", len(results))
	fmt.Printf("  Total entities created: %d\n", totalEntities)
	fmt.Printf("  Total relationships created: %d\n", totalRelationships)

	return nil
}

// processEpisode extracts entities and relationships from a single episode.
// This function wraps the extraction logic and entity/relationship creation.
func processEpisode(ctx context.Context, store storage.Storage, episode *types.Episode, apiKey string) (map[string]interface{}, error) {
	// Call extraction.ExtractFromEpisode
	result, err := extraction.ExtractFromEpisode(ctx, store, episode.ID, apiKey)
	if err != nil {
		return nil, fmt.Errorf("extracting from episode: %w", err)
	}

	// Create extracted entities, converting from extraction types to storage types
	entityIDs := []string{}
	entityNameToID := make(map[string]string) // For relationship resolution
	for _, extractedEntity := range result.Entities {
		// Convert to types.Entity using helper
		entity := extraction.ConvertToTypesEntity(extractedEntity, actor)

		if err := store.CreateEntity(ctx, entity); err != nil {
			return nil, fmt.Errorf("creating entity %s: %w", entity.Name, err)
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
			return nil, fmt.Errorf("creating relationship: %w", err)
		}
		relationshipIDs = append(relationshipIDs, rel.ID)
	}

	// Update episode extracted_at timestamp
	// TODO: This requires UpdateEpisode method which doesn't exist yet (episodes are immutable).
	// Options:
	//   1. Add UpdateEpisode to storage interface (exception to immutability for extraction tracking)
	//   2. Store extraction status in metadata
	//   3. Add extracted_at to Episode struct and update it via direct SQL
	// For now, we skip this step and document the concern.

	return map[string]interface{}{
		"episode_id":            episode.ID,
		"entities_created":      len(entityIDs),
		"relationships_created": len(relationshipIDs),
		"entity_ids":            entityIDs,
		"relationship_ids":      relationshipIDs,
		"success":               true,
	}, nil
}
