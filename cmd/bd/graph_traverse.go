package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
	"github.com/steveyegge/beads/internal/ui"
)

var (
	traverseValidAt string
)

var graphTraverseCmd = &cobra.Command{
	Use:   "traverse <from-id> <to-id>",
	Short: "Find shortest path between two entities",
	Long: `Find the shortest path between two entities in the knowledge graph.

Uses Breadth-First Search (BFS) to find the shortest path, considering
only relationships that are temporally valid at the specified time.

Examples:
  # Find shortest path between two entities
  bd graph traverse bd-a3f8e9 bd-b7c2d1

  # Find path valid at a specific time
  bd graph traverse bd-a3f8e9 bd-b7c2d1 --valid-at 2024-01-15

  # JSON output for programmatic use
  bd graph traverse bd-a3f8e9 bd-b7c2d1 --json`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := rootCtx
		fromID := args[0]
		toID := args[1]

		// Parse validation time
		var validAt time.Time
		if traverseValidAt != "" {
			var err error
			validAt, err = time.Parse("2006-01-02", traverseValidAt)
			if err != nil {
				FatalErrorRespectJSON("invalid --valid-at format (use YYYY-MM-DD): %v", err)
			}
		} else {
			validAt = time.Now()
		}

		// Find shortest path
		result, err := findShortestPath(ctx, store, fromID, toID, validAt)
		if err != nil {
			FatalErrorRespectJSON("failed to find path: %v", err)
		}

		// Output result
		if jsonOutput {
			jsonBytes, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			renderTraverseResult(result)
		}
	},
}

// GraphTraverseResult holds the shortest path result
type GraphTraverseResult struct {
	FromEntity    string                `json:"from_entity"`
	ToEntity      string                `json:"to_entity"`
	ValidAt       time.Time             `json:"valid_at"`
	PathFound     bool                  `json:"path_found"`
	PathLength    int                   `json:"path_length"`
	Path          []string              `json:"path"`           // Entity IDs in order
	Entities      []*types.Entity       `json:"entities"`       // Full entity details
	Relationships []*types.Relationship `json:"relationships"`  // Relationships in path
}

// findShortestPath uses BFS to find the shortest path between two entities
func findShortestPath(ctx context.Context, st storage.Storage, fromID, toID string, validAt time.Time) (*GraphTraverseResult, error) {
	result := &GraphTraverseResult{
		FromEntity:    fromID,
		ToEntity:      toID,
		ValidAt:       validAt,
		PathFound:     false,
		Entities:      make([]*types.Entity, 0),
		Relationships: make([]*types.Relationship, 0),
	}

	// Verify both entities exist
	fromEntity, err := st.GetEntity(ctx, fromID)
	if err != nil {
		return nil, fmt.Errorf("from entity not found: %v", err)
	}
	toEntity, err := st.GetEntity(ctx, toID)
	if err != nil {
		return nil, fmt.Errorf("to entity not found: %v", err)
	}

	// Handle trivial case
	if fromID == toID {
		result.PathFound = true
		result.PathLength = 0
		result.Path = []string{fromID}
		result.Entities = append(result.Entities, fromEntity)
		return result, nil
	}

	// BFS to find shortest path
	type queueItem struct {
		entityID string
		path     []string
		rels     []*types.Relationship
	}
	queue := []queueItem{{entityID: fromID, path: []string{fromID}, rels: []*types.Relationship{}}}
	visited := make(map[string]bool)
	visited[fromID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Get relationships in both directions
		rels, err := st.GetRelationshipsWithTemporalFilter(ctx, current.entityID, validAt, storage.RelationshipDirectionBoth)
		if err != nil {
			return nil, fmt.Errorf("failed to get relationships for %s: %v", current.entityID, err)
		}

		for _, rel := range rels {
			// Determine the connected entity ID
			var nextID string
			if rel.SourceEntityID == current.entityID {
				nextID = rel.TargetEntityID
			} else {
				nextID = rel.SourceEntityID
			}

			// Skip if already visited
			if visited[nextID] {
				continue
			}
			visited[nextID] = true

			// Build new path
			newPath := append(current.path, nextID)
			newRels := append(current.rels, rel)

			// Check if we found the target
			if nextID == toID {
				result.PathFound = true
				result.PathLength = len(newPath) - 1
				result.Path = newPath
				result.Relationships = newRels

				// Collect all entities in the path
				entityMap := make(map[string]*types.Entity)
				entityMap[fromID] = fromEntity
				entityMap[toID] = toEntity

				for _, id := range newPath {
					if _, exists := entityMap[id]; !exists {
						entity, err := st.GetEntity(ctx, id)
						if err == nil {
							entityMap[id] = entity
						}
					}
				}

				// Build entities list in path order
				for _, id := range newPath {
					if entity, exists := entityMap[id]; exists {
						result.Entities = append(result.Entities, entity)
					}
				}

				return result, nil
			}

			// Add to queue for further exploration
			queue = append(queue, queueItem{
				entityID: nextID,
				path:     newPath,
				rels:     newRels,
			})
		}
	}

	// No path found
	return result, nil
}

// renderTraverseResult renders the path result in human-readable format
func renderTraverseResult(result *GraphTraverseResult) {
	fmt.Printf("%s\n", ui.RenderBold("SHORTEST PATH"))
	fmt.Printf("From: %s\n", ui.RenderID(result.FromEntity))
	fmt.Printf("To: %s\n", ui.RenderID(result.ToEntity))
	fmt.Printf("Valid At: %s\n\n", result.ValidAt.Format("2006-01-02 15:04"))

	if !result.PathFound {
		fmt.Printf("%s\n", ui.RenderFail("No path found between entities"))
		return
	}

	fmt.Printf("Path Length: %d hops\n\n", result.PathLength)

	// Render path with entities and relationships
	fmt.Printf("%s\n", ui.RenderBold("PATH"))
	for i, entity := range result.Entities {
		fmt.Printf("%s %s (%s)\n",
			ui.RenderID(entity.ID),
			ui.RenderBold(entity.Name),
			ui.RenderMuted(entity.EntityType))

		// Show relationship to next entity
		if i < len(result.Relationships) {
			rel := result.Relationships[i]
			arrow := "→"
			relType := rel.RelationshipType
			
			// Determine direction
			if rel.SourceEntityID == entity.ID {
				fmt.Printf("    %s -[%s]-> %s\n", arrow, ui.RenderMuted(relType), "")
			} else {
				fmt.Printf("    %s <-[%s]- %s\n", arrow, ui.RenderMuted(relType), "")
			}
		}
	}
}

func init() {
	graphTraverseCmd.Flags().StringVar(&traverseValidAt, "valid-at", "", "Temporal validity filter (YYYY-MM-DD)")
	graphCmd.AddCommand(graphTraverseCmd)
}
