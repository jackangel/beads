package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage"
)

var (
	retrieveQuery   string
	retrieveMaxHops int
	retrieveTopK    int
	retrieveMinConf float64
)

var memoryRetrieveCmd = &cobra.Command{
	Use:   "retrieve",
	Short: "Retrieve memory context from knowledge graph",
	Long: `Query the knowledge graph and retrieve relevant entities, relationships, and provenance.
    
Combines semantic search + graph traversal + temporal filtering.`,
	RunE: runMemoryRetrieve,
}

func init() {
	memoryRetrieveCmd.Flags().StringVarP(&retrieveQuery, "query", "q", "", "Natural language query (required)")
	memoryRetrieveCmd.MarkFlagRequired("query")
	memoryRetrieveCmd.Flags().IntVar(&retrieveMaxHops, "hops", 2, "Graph traversal depth")
	memoryRetrieveCmd.Flags().IntVar(&retrieveTopK, "top", 5, "Max initial entities from semantic search")
	memoryRetrieveCmd.Flags().Float64Var(&retrieveMinConf, "min-confidence", 0.5, "Minimum relationship confidence")
	rootCmd.AddCommand(memoryRetrieveCmd)
}

func runMemoryRetrieve(cmd *cobra.Command, args []string) error {
	ctx := rootCtx

	query := storage.MemoryQuery{
		TextQuery:     retrieveQuery,
		MaxHops:       retrieveMaxHops,
		TopK:          retrieveTopK,
		MinConfidence: retrieveMinConf,
	}

	result, err := store.RetrieveMemory(ctx, query)
	if err != nil {
		return fmt.Errorf("retrieving memory: %w", err)
	}

	if jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	// Human-readable output
	fmt.Printf("Memory Context for: \"%s\"\n\n", retrieveQuery)
	fmt.Printf("Entities (%d):\n", len(result.Entities))
	for _, e := range result.Entities {
		score := result.RelevanceScores[e.ID]
		fmt.Printf("  - %s (%s) [%.2f relevance]\n", e.Name, e.ID, score)
	}
	fmt.Printf("\nRelationships (%d):\n", len(result.Relationships))
	for _, rel := range result.Relationships {
		fmt.Printf("  - %s -[%s]-> %s\n", rel.SourceEntityID, rel.RelationshipType, rel.TargetEntityID)
	}

	return nil
}
