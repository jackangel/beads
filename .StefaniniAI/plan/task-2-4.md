# Task 2-4: Add CLI command for finding duplicate entities

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Risk Level**: MEDIUM
- **Phase**: 2
- **Depends On**: 2-3

## Files
- `e:\Projects\BeadsMemory\beads\cmd\bd\entity_find_duplicates.go` (NEW)

## Instructions
Create `bd entity find-duplicates` command that wraps the deduplication algorithm from task 2-3.

**Outcome:** Users can find duplicate entities from the CLI.

**Implementation pattern (follow `cmd/bd/find_duplicates.go`):**
```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "github.com/spf13/cobra"
    "github.com/steveyegge/beads/internal/dedup"
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
    
Uses Jaccard and cosine similarity. Default threshold: 0.8`,
    RunE: runEntityFindDuplicates,
}

func init() {
    entityFindDuplicatesCmd.Flags().StringVar(&entityTypeFilter, "entity-type", "", "Filter by entity type")
    entityFindDuplicatesCmd.Flags().Float64Var(&dedupThreshold, "threshold", 0.8, "Similarity threshold (0.0-1.0)")
    entityCmd.AddCommand(entityFindDuplicatesCmd)
}

func runEntityFindDuplicates(cmd *cobra.Command, args []string) error {
    ctx := rootCtx
    
    pairs, err := dedup.FindDuplicates(ctx, store, entityTypeFilter, dedupThreshold)
    if err != nil {
        return fmt.Errorf("finding duplicates: %w", err)
    }
    
    if jsonOutput {
        encoder := json.NewEncoder(os.Stdout)
        encoder.SetIndent("", "  ")
        return encoder.Encode(map[string]interface{}{
            "duplicates": pairs,
            "count":      len(pairs),
            "threshold":  dedupThreshold,
        })
    }
    
    // Human-readable output
    if len(pairs) == 0 {
        fmt.Println("No duplicates found.")
        return nil
    }
    
    fmt.Printf("Found %d potential duplicate pair(s):\n\n", len(pairs))
    for i, pair := range pairs {
        fmt.Printf("%d. %s (%s) and %s (%s)\n",
            i+1, pair.EntityA.Name, pair.EntityA.ID, pair.EntityB.Name, pair.EntityB.ID)
        fmt.Printf("   Similarity: %.2f (%s)\n", pair.Similarity, pair.Method)
        if pair.Reason != "" {
            fmt.Printf("   Reason: %s\n", pair.Reason)
        }
        fmt.Println()
    }
    
    return nil
}
```

## Architecture Pattern
Follow existing `find_duplicates.go` CLI pattern: flags for filters, JSON output mode, human-readable fallback.

## Validation Criteria
- [ ] Command created: `bd entity find-duplicates`
- [ ] `--entity-type` and `--threshold` flags work
- [ ] `--json` output includes count, threshold, pairs
- [ ] Human-readable output shows entity names, IDs, similarity
- [ ] No compilation errors

## Impact Analysis
- **Direct impact**: New CLI command
- **Indirect impact**: Task 2-9 (MCP tool) wraps this command
- **Dependencies**: Task 2-3 (dedup package)

## Context
- Research Bundle: Task 2-3 design
- Pattern: `cmd/bd/find_duplicates.go`

## User Feedback
*(Empty)*
