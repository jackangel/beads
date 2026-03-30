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
	exploreDepth     int
	exploreAlgorithm string
	exploreValidAt   string
)

var graphExploreCmd = &cobra.Command{
	Use:   "explore <entity-id>",
	Short: "Explore entities connected to a starting entity",
	Long: `Explore the knowledge graph starting from an entity using BFS or DFS traversal.

Traverses relationships up to a specified depth, respecting temporal validity.
By default, explores relationships valid at the current time.

Algorithms:
  bfs   Breadth-First Search (default) - explores layer by layer
  dfs   Depth-First Search - explores deep paths first

Examples:
  # Explore 2 hops from an entity using BFS
  bd graph explore bd-a3f8e9 --depth 2

  # Explore 3 hops using DFS
  bd graph explore bd-a3f8e9 --depth 3 --algorithm dfs

  # Explore relationships valid at a specific time
  bd graph explore bd-a3f8e9 --depth 2 --valid-at 2024-01-15

  # JSON output for programmatic use
  bd graph explore bd-a3f8e9 --depth 2 --json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := rootCtx
		entityID := args[0]

		// Parse validation time
		var validAt time.Time
		if exploreValidAt != "" {
			var err error
			validAt, err = time.Parse("2006-01-02", exploreValidAt)
			if err != nil {
				FatalErrorRespectJSON("invalid --valid-at format (use YYYY-MM-DD): %v", err)
			}
		} else {
			validAt = time.Now()
		}

		// Validate algorithm
		if exploreAlgorithm != "bfs" && exploreAlgorithm != "dfs" {
			FatalErrorRespectJSON("invalid algorithm (must be 'bfs' or 'dfs'): %s", exploreAlgorithm)
		}

		// Explore graph
		var result *GraphExploreResult
		var err error
		if exploreAlgorithm == "bfs" {
			result, err = exploreGraphBFS(ctx, store, entityID, exploreDepth, validAt)
		} else {
			result, err = exploreGraphDFS(ctx, store, entityID, exploreDepth, validAt)
		}
		if err != nil {
			FatalErrorRespectJSON("failed to explore graph: %v", err)
		}

		// Output result
		if jsonOutput {
			jsonBytes, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			renderGraphExploreResult(result)
		}
	},
}

// GraphExploreResult holds the exploration results
type GraphExploreResult struct {
	StartEntity  string                  `json:"start_entity"`
	Algorithm    string                  `json:"algorithm"`
	Depth        int                     `json:"depth"`
	ValidAt      time.Time               `json:"valid_at"`
	Entities     []*types.Entity         `json:"entities"`
	Relationships []*types.Relationship  `json:"relationships"`
	Paths        map[string][]string     `json:"paths"` // entityID -> path from start
}

// exploreGraphBFS performs breadth-first search exploration
func exploreGraphBFS(ctx context.Context, st storage.Storage, startID string, maxDepth int, validAt time.Time) (*GraphExploreResult, error) {
	result := &GraphExploreResult{
		StartEntity:   startID,
		Algorithm:     "bfs",
		Depth:         maxDepth,
		ValidAt:       validAt,
		Entities:      make([]*types.Entity, 0),
		Relationships: make([]*types.Relationship, 0),
		Paths:         make(map[string][]string),
	}

	// Get start entity
	startEntity, err := st.GetEntity(ctx, startID)
	if err != nil {
		return nil, fmt.Errorf("start entity not found: %v", err)
	}
	result.Entities = append(result.Entities, startEntity)
	result.Paths[startID] = []string{startID}

	// BFS queue: (entityID, depth)
	type queueItem struct {
		entityID string
		depth    int
		path     []string
	}
	queue := []queueItem{{entityID: startID, depth: 0, path: []string{startID}}}
	visited := make(map[string]bool)
	visited[startID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Stop if we've reached max depth
		if current.depth >= maxDepth {
			continue
		}

		// Get relationships in both directions
		rels, err := st.GetRelationshipsWithTemporalFilter(ctx, current.entityID, validAt, storage.RelationshipDirectionBoth)
		if err != nil {
			return nil, fmt.Errorf("failed to get relationships for %s: %v", current.entityID, err)
		}

		for _, rel := range rels {
			result.Relationships = append(result.Relationships, rel)

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

			// Get the entity
			entity, err := st.GetEntity(ctx, nextID)
			if err != nil {
				continue // Skip entities that can't be loaded
			}
			result.Entities = append(result.Entities, entity)

			// Record path
			newPath := append(current.path, nextID)
			result.Paths[nextID] = newPath

			// Add to queue
			queue = append(queue, queueItem{
				entityID: nextID,
				depth:    current.depth + 1,
				path:     newPath,
			})
		}
	}

	return result, nil
}

// exploreGraphDFS performs depth-first search exploration
func exploreGraphDFS(ctx context.Context, st storage.Storage, startID string, maxDepth int, validAt time.Time) (*GraphExploreResult, error) {
	result := &GraphExploreResult{
		StartEntity:   startID,
		Algorithm:     "dfs",
		Depth:         maxDepth,
		ValidAt:       validAt,
		Entities:      make([]*types.Entity, 0),
		Relationships: make([]*types.Relationship, 0),
		Paths:         make(map[string][]string),
	}

	// Get start entity
	startEntity, err := st.GetEntity(ctx, startID)
	if err != nil {
		return nil, fmt.Errorf("start entity not found: %v", err)
	}
	result.Entities = append(result.Entities, startEntity)
	result.Paths[startID] = []string{startID}

	visited := make(map[string]bool)
	visited[startID] = true

	// Recursive DFS helper
	var dfs func(entityID string, depth int, path []string) error
	dfs = func(entityID string, depth int, path []string) error {
		if depth >= maxDepth {
			return nil
		}

		// Get relationships in both directions
		rels, err := st.GetRelationshipsWithTemporalFilter(ctx, entityID, validAt, storage.RelationshipDirectionBoth)
		if err != nil {
			return fmt.Errorf("failed to get relationships for %s: %v", entityID, err)
		}

		for _, rel := range rels {
			result.Relationships = append(result.Relationships, rel)

			// Determine the connected entity ID
			var nextID string
			if rel.SourceEntityID == entityID {
				nextID = rel.TargetEntityID
			} else {
				nextID = rel.SourceEntityID
			}

			// Skip if already visited
			if visited[nextID] {
				continue
			}
			visited[nextID] = true

			// Get the entity
			entity, err := st.GetEntity(ctx, nextID)
			if err != nil {
				continue // Skip entities that can't be loaded
			}
			result.Entities = append(result.Entities, entity)

			// Record path
			newPath := append(path, nextID)
			result.Paths[nextID] = newPath

			// Recurse
			if err := dfs(nextID, depth+1, newPath); err != nil {
				return err
			}
		}

		return nil
	}

	if err := dfs(startID, 0, []string{startID}); err != nil {
		return nil, err
	}

	return result, nil
}

// renderGraphExploreResult renders the exploration result in human-readable format
func renderGraphExploreResult(result *GraphExploreResult) {
	fmt.Printf("%s\n", ui.RenderBold("GRAPH EXPLORATION"))
	fmt.Printf("Start Entity: %s\n", ui.RenderID(result.StartEntity))
	fmt.Printf("Algorithm: %s\n", result.Algorithm)
	fmt.Printf("Max Depth: %d\n", result.Depth)
	fmt.Printf("Valid At: %s\n", result.ValidAt.Format("2006-01-02 15:04"))
	fmt.Printf("Entities Found: %d\n", len(result.Entities))
	fmt.Printf("Relationships Found: %d\n\n", len(result.Relationships))

	fmt.Printf("%s\n", ui.RenderBold("DISCOVERED ENTITIES"))
	for _, entity := range result.Entities {
		path := result.Paths[entity.ID]
		depth := len(path) - 1
		
		// Indent based on depth
		indent := ""
		for i := 0; i < depth; i++ {
			indent += "  "
		}
		
		fmt.Printf("%s%s %s (%s) - depth %d\n",
			indent,
			ui.RenderID(entity.ID),
			ui.RenderBold(entity.Name),
			ui.RenderMuted(entity.EntityType),
			depth)
	}
	
	if len(result.Relationships) > 0 {
		fmt.Printf("\n%s\n", ui.RenderBold("RELATIONSHIPS"))
		for _, rel := range result.Relationships {
			fmt.Printf("%s -[%s]-> %s\n",
				ui.RenderID(rel.SourceEntityID),
				ui.RenderMuted(rel.RelationshipType),
				ui.RenderID(rel.TargetEntityID))
		}
	}
}

func init() {
	graphExploreCmd.Flags().IntVar(&exploreDepth, "depth", 2, "Maximum traversal depth")
	graphExploreCmd.Flags().StringVar(&exploreAlgorithm, "algorithm", "bfs", "Traversal algorithm (bfs or dfs)")
	graphExploreCmd.Flags().StringVar(&exploreValidAt, "valid-at", "", "Temporal validity filter (YYYY-MM-DD)")
	graphCmd.AddCommand(graphExploreCmd)
}
