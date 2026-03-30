package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
)

var (
	visualizeFormat  string
	visualizeDepth   int
	visualizeValidAt string
)

var graphVisualizeCmd = &cobra.Command{
	Use:   "visualize <entity-id>",
	Short: "Generate graph visualization in DOT format",
	Long: `Generate a graph visualization starting from an entity.

Outputs in Graphviz DOT format for rendering with tools like:
  dot -Tsvg output.dot -o graph.svg
  dot -Tpng output.dot -o graph.png

Also supports JSON format for raw graph data.

Examples:
  # Generate DOT format (pipe to graphviz)
  bd graph visualize bd-a3f8e9 > graph.dot
  bd graph visualize bd-a3f8e9 | dot -Tsvg > graph.svg

  # Explore deeper (default depth is 2)
  bd graph visualize bd-a3f8e9 --depth 3

  # JSON format for raw graph data
  bd graph visualize bd-a3f8e9 --format json

  # Visualize relationships valid at a specific time
  bd graph visualize bd-a3f8e9 --valid-at 2024-01-15`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := rootCtx
		entityID := args[0]

		// Parse validation time
		var validAt time.Time
		if visualizeValidAt != "" {
			var err error
			validAt, err = time.Parse("2006-01-02", visualizeValidAt)
			if err != nil {
				FatalErrorRespectJSON("invalid --valid-at format (use YYYY-MM-DD): %v", err)
			}
		} else {
			validAt = time.Now()
		}

		// Validate format
		if visualizeFormat != "dot" && visualizeFormat != "json" {
			FatalErrorRespectJSON("invalid format (must be 'dot' or 'json'): %s", visualizeFormat)
		}

		// Explore graph for visualization
		graphData, err := exploreForVisualization(ctx, store, entityID, visualizeDepth, validAt)
		if err != nil {
			FatalErrorRespectJSON("failed to explore graph: %v", err)
		}

		// Output in requested format
		if visualizeFormat == "json" || jsonOutput {
			jsonBytes, _ := json.MarshalIndent(graphData, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			renderKnowledgeGraphDOT(graphData)
		}
	},
}

// GraphVisualizationData holds the graph data for visualization
type GraphVisualizationData struct {
	StartEntity   string                `json:"start_entity"`
	Depth         int                   `json:"depth"`
	ValidAt       time.Time             `json:"valid_at"`
	Entities      []*types.Entity       `json:"entities"`
	Relationships []*types.Relationship `json:"relationships"`
}

// exploreForVisualization explores the graph to gather visualization data
func exploreForVisualization(ctx context.Context, st storage.Storage, startID string, maxDepth int, validAt time.Time) (*GraphVisualizationData, error) {
	result := &GraphVisualizationData{
		StartEntity:   startID,
		Depth:         maxDepth,
		ValidAt:       validAt,
		Entities:      make([]*types.Entity, 0),
		Relationships: make([]*types.Relationship, 0),
	}

	// Get start entity
	startEntity, err := st.GetEntity(ctx, startID)
	if err != nil {
		return nil, fmt.Errorf("start entity not found: %v", err)
	}
	result.Entities = append(result.Entities, startEntity)

	// Use BFS to explore the graph
	type queueItem struct {
		entityID string
		depth    int
	}
	queue := []queueItem{{entityID: startID, depth: 0}}
	visited := make(map[string]bool)
	visited[startID] = true
	relSeen := make(map[string]bool)

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
			// Avoid duplicate relationships
			if !relSeen[rel.ID] {
				result.Relationships = append(result.Relationships, rel)
				relSeen[rel.ID] = true
			}

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

			// Add to queue
			queue = append(queue, queueItem{
				entityID: nextID,
				depth:    current.depth + 1,
			})
		}
	}

	return result, nil
}

// renderKnowledgeGraphDOT renders the graph in Graphviz DOT format
func renderKnowledgeGraphDOT(data *GraphVisualizationData) {
	var sb strings.Builder

	// DOT header
	sb.WriteString("digraph knowledge_graph {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=rounded];\n")
	sb.WriteString("  edge [fontsize=10];\n\n")

	// Add nodes
	sb.WriteString("  // Entities\n")
	for _, entity := range data.Entities {
		label := fmt.Sprintf("%s\\n%s", entity.Name, entity.EntityType)
		// Escape quotes in label
		label = strings.ReplaceAll(label, `"`, `\"`)
		
		// Highlight start entity
		if entity.ID == data.StartEntity {
			sb.WriteString(fmt.Sprintf(`  "%s" [label="%s", style="rounded,filled", fillcolor=lightblue];`, 
				entity.ID, label))
		} else {
			sb.WriteString(fmt.Sprintf(`  "%s" [label="%s"];`, entity.ID, label))
		}
		sb.WriteString("\n")
	}

	// Add edges
	sb.WriteString("\n  // Relationships\n")
	for _, rel := range data.Relationships {
		label := rel.RelationshipType
		
		// Add temporal info if relationship has ended
		if rel.ValidUntil != nil {
			label += fmt.Sprintf("\\n(until %s)", rel.ValidUntil.Format("2006-01-02"))
		}
		
		// Escape quotes
		label = strings.ReplaceAll(label, `"`, `\"`)
		
		sb.WriteString(fmt.Sprintf(`  "%s" -> "%s" [label="%s"];`,
			rel.SourceEntityID, rel.TargetEntityID, label))
		sb.WriteString("\n")
	}

	// DOT footer
	sb.WriteString("}\n")

	fmt.Print(sb.String())
}

func init() {
	graphVisualizeCmd.Flags().StringVar(&visualizeFormat, "format", "dot", "Output format (dot or json)")
	graphVisualizeCmd.Flags().IntVar(&visualizeDepth, "depth", 2, "Maximum traversal depth")
	graphVisualizeCmd.Flags().StringVar(&visualizeValidAt, "valid-at", "", "Temporal validity filter (YYYY-MM-DD)")
	graphCmd.AddCommand(graphVisualizeCmd)
}
