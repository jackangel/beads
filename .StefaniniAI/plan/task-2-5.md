# Task 2-5: Add CLI command for merging entities

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Risk Level**: MEDIUM
- **Phase**: 2
- **Depends On**: 1-3

## Files
- `e:\Projects\BeadsMemory\beads\cmd\bd\entity_merge.go` (NEW)

## Instructions
Create `bd entity merge <source> <target>` command that calls `store.MergeEntities` from task 1-3.

**Implementation:**
```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "github.com/spf13/cobra"
)

var entityMergeCmd = &cobra.Command{
    Use:   "merge <source-id> <target-id>",
    Short: "Merge source entity into target entity",
    Long: `Merge a source entity into a target entity.
    
- Moves all relationships from source to target
- Marks source as merged (soft delete)
- Preserves metadata and provenance`,
    Args: cobra.ExactArgs(2),
    RunE: runEntityMerge,
}

func init() {
    entityCmd.AddCommand(entityMergeCmd)
}

func runEntityMerge(cmd *cobra.Command, args []string) error {
    CheckReadonly("merge")
    
    sourceID := args[0]
    targetID := args[1]
    ctx := rootCtx
    
    if err := store.MergeEntities(ctx, sourceID, targetID, actor); err != nil {
        return fmt.Errorf("merging entities: %w", err)
    }
    
    if jsonOutput {
        return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
            "source":  sourceID,
            "target":  targetID,
            "message": "Entity merged successfully",
        })
    }
    
    fmt.Printf("✓ Merged %s into %s\n", sourceID, targetID)
    fmt.Printf("  %s is now marked as merged (soft-deleted)\n", sourceID)
    fmt.Printf("  All relationships transferred to %s\n", targetID)
    
    return nil
}
```

## Validation Criteria
- [ ] Command created: `bd entity merge <source> <target>`
- [ ] Calls `store.MergeEntities` from task 1-3
- [ ] `--json` output includes source, target, message
- [ ] Human-readable output confirms merge
- [ ] Checks readonly mode

## Impact Analysis
- **Direct impact**: New CLI command
- **Indirect impact**: Task 2-9 (MCP tool) wraps this
- **Dependencies**: Task 1-3 (MergeEntities storage method)

## Context
- Research Bundle: Task 1-3 design
- Pattern: `cmd/bd/close.go` (mutation command)

## User Feedback
*(Empty)*
