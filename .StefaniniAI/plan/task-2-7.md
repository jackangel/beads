# Task 2-7: Add CLI command for semantic entity search

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Risk Level**: LOW
- **Phase**: 2
- **Depends On**: 2-6

## Files
- `e:\Projects\BeadsMemory\beads\cmd\bd\entity_search.go` (NEW)

## Instructions
Create `bd entity search --query "<text>"` command for natural language entity search using semantic similarity.

**Implementation:**
```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "github.com/spf13/cobra"
    "github.com/steveyegge/beads/internal/storage"
)

var (
    searchQuery      string
    searchEntityType string
    searchTopK       int
)

var entitySearchCmd = &cobra.Command{
    Use:   "search",
    Short: "Semantic search for entities",
    Long: `Search entities using natural language query.
    
Uses cosine similarity on entity names and summaries.
Returns top-K most relevant results.`,
    RunE: runEntitySearch,
}

func init() {
    entitySearchCmd.Flags().StringVarP(&searchQuery, "query", "q", "", "Search query (required)")
    entitySearchCmd.MarkFlagRequired("query")
    entitySearchCmd.Flags().StringVar(&searchEntityType, "entity-type", "", "Filter by entity type")
    entitySearchCmd.Flags().IntVar(&searchTopK, "top", 10, "Maximum results")
    entityCmd.AddCommand(entitySearchCmd)
}

func runEntitySearch(cmd *cobra.Command, args []string) error {
    ctx := rootCtx
    
    filters := storage.EntityFilters{
        SemanticQuery: searchQuery,
        EntityType:    searchEntityType,
        TopK:          searchTopK,
    }
    
    entities, err := store.SearchEntities(ctx, filters)
    if err != nil {
        return fmt.Errorf("searching entities: %w", err)
    }
    
    if jsonOutput {
        return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
            "entities": entities,
            "count":    len(entities),
            "query":    searchQuery,
        })
    }
    
    // Human-readable output
    if len(entities) == 0 {
        fmt.Println("No entities found.")
        return nil
    }
    
    fmt.Printf("Found %d entit(ies) matching \"%s\":\n\n", len(entities), searchQuery)
    for _, e := range entities {
        fmt.Printf("- %s (%s) [%s]\n", e.Name, e.ID, e.EntityType)
        if e.Summary != "" {
            fmt.Printf("  %s\n", e.Summary)
        }
        fmt.Println()
    }
    
    return nil
}
```

## Validation Criteria
- [ ] Command created: `bd entity search --query "<text>"`
- [ ] `--entity-type` and `--top` flags work
- [ ] `--json` output includes entities, count, query
- [ ] Human-readable output shows entity names, types, summaries
- [ ] No compilation errors

## Impact Analysis
- **Direct impact**: New CLI command
- **Indirect impact**: Task 2-8 (MCP tool) wraps this
- **Dependencies**: Task 2-6 (semantic search in storage)

## Context
- Research Bundle: Task 2-6 design
- Pattern: `cmd/bd/list.go`

## User Feedback
*(Empty)*
