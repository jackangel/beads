package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/dedup"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/ui"
)

var (
	entityTypeFilter string
	dedupThreshold   float64
)

var entityFindDuplicatesCmd = &cobra.Command{
	Use:   "find-duplicates",
	Short: "Find potential duplicate entities",
	Long: `Find entities that may be duplicates based on name and summary similarity.

Uses Jaccard and cosine similarity to compare entity names and summaries.
Only entities with matching entity types are compared.

Examples:
  bd entity find-duplicates                         # All entity types, 0.8 threshold
  bd entity find-duplicates --threshold 0.6         # Lower threshold = more results
  bd entity find-duplicates --entity-type person    # Only check person entities
  bd entity find-duplicates --json                  # JSON output`,
	RunE: runEntityFindDuplicates,
}

func init() {
	entityFindDuplicatesCmd.Flags().StringVar(&entityTypeFilter, "entity-type", "", "Filter by entity type")
	entityFindDuplicatesCmd.Flags().Float64Var(&dedupThreshold, "threshold", 0.8, "Similarity threshold (0.0-1.0)")
	entityCmd.AddCommand(entityFindDuplicatesCmd)
}

func runEntityFindDuplicates(cmd *cobra.Command, args []string) error {
	ctx := rootCtx

	// Validate threshold
	if dedupThreshold < 0.0 || dedupThreshold > 1.0 {
		return fmt.Errorf("threshold must be between 0.0 and 1.0")
	}

	// Load entities
	filters := storage.EntityFilters{
		EntityType: entityTypeFilter,
		Limit:      0, // Load all entities
	}

	entities, err := store.SearchEntities(ctx, filters)
	if err != nil {
		return fmt.Errorf("loading entities: %w", err)
	}

	if len(entities) < 2 {
		if jsonOutput {
			outputJSON(map[string]interface{}{
				"duplicates": []interface{}{},
				"count":      0,
				"threshold":  dedupThreshold,
			})
		} else {
			fmt.Println("Not enough entities to compare (need at least 2)")
		}
		return nil
	}

	// Find duplicates
	duplicates := dedup.FindDuplicates(entities, dedupThreshold)

	// Output results
	if jsonOutput {
		type pairJSON struct {
			EntityAID   string  `json:"entity_a_id"`
			EntityBID   string  `json:"entity_b_id"`
			EntityAName string  `json:"entity_a_name"`
			EntityBName string  `json:"entity_b_name"`
			Score       float64 `json:"score"`
			Reason      string  `json:"reason"`
		}

		jsonPairs := make([]pairJSON, len(duplicates))
		for i, dup := range duplicates {
			jsonPairs[i] = pairJSON{
				EntityAID:   dup.EntityA.ID,
				EntityBID:   dup.EntityB.ID,
				EntityAName: dup.EntityA.Name,
				EntityBName: dup.EntityB.Name,
				Score:       dup.Score,
				Reason:      dup.Reason,
			}
		}

		outputJSON(map[string]interface{}{
			"duplicates": jsonPairs,
			"count":      len(jsonPairs),
			"threshold":  dedupThreshold,
		})
		return nil
	}

	// Human-readable output
	if len(duplicates) == 0 {
		fmt.Printf("No duplicates found (threshold: %.0f%%)\n", dedupThreshold*100)
		return nil
	}

	fmt.Printf("Found %d potential duplicate pair(s) (threshold: %.0f%%):\n\n", len(duplicates), dedupThreshold*100)

	for i, dup := range duplicates {
		pct := dup.Score * 100
		fmt.Printf("%s Pair %d (%.0f%% similar):\n",
			ui.RenderAccent("━━"), i+1, pct)
		fmt.Printf("  A: %s %s %s\n",
			ui.RenderID(dup.EntityA.ID),
			ui.RenderBold(dup.EntityA.Name),
			ui.RenderMuted(fmt.Sprintf("(%s)", dup.EntityA.EntityType)))
		if dup.EntityA.Summary != "" {
			fmt.Printf("     %s\n", dup.EntityA.Summary)
		}
		fmt.Printf("  B: %s %s %s\n",
			ui.RenderID(dup.EntityB.ID),
			ui.RenderBold(dup.EntityB.Name),
			ui.RenderMuted(fmt.Sprintf("(%s)", dup.EntityB.EntityType)))
		if dup.EntityB.Summary != "" {
			fmt.Printf("     %s\n", dup.EntityB.Summary)
		}
		fmt.Printf("  %s %s\n\n",
			ui.RenderMuted("Reason:"),
			dup.Reason)
	}

	return nil
}
