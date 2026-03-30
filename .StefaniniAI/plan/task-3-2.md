# Task 3-2: Add CLI command for single episode extraction

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Risk Level**: MEDIUM
- **Phase**: 3
- **Depends On**: 3-1, 1-1

## Files
- `e:\Projects\BeadsMemory\beads\cmd\bd\episode_extract.go` (NEW)

## Instructions
Create `bd episode extract <episode-id>` command that processes an episode using LLM extraction.

**Implementation:**
```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "time"
    "github.com/spf13/cobra"
    "github.com/steveyegge/beads/internal/extraction"
)

var episodeExtractCmd = &cobra.Command{
    Use:   "extract <episode-id>",
    Short: "Extract entities and relationships from an episode using LLM",
    Long: `Process an episode's raw data and extract structured knowledge graph data.
    
Requires ANTHROPIC_API_KEY environment variable.`,
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
    
    // Extract entities and relationships
    result, err := extraction.ExtractFromEpisode(ctx, store, episodeID, apiKey)
    if err != nil {
        return fmt.Errorf("extracting from episode: %w", err)
    }
    
    // Create extracted entities
    entityIDs := []string{}
    for _, entity := range result.Entities {
        entity.CreatedBy = actor
        if err := store.CreateEntity(ctx, entity); err != nil {
            return fmt.Errorf("creating entity %s: %w", entity.Name, err)
        }
        entityIDs = append(entityIDs, entity.ID)
    }
    
    // Create extracted relationships (resolve names to IDs)
    for _, rel := range result.Relationships {
        // TODO: Resolve source/target names to entity IDs (simple: find by name)
        rel.CreatedBy = actor
        if err := store.CreateRelationship(ctx, rel); err != nil {
            return fmt.Errorf("creating relationship: %w", err)
        }
    }
    
    // Update episode extracted_at timestamp
    // (TBD: add UpdateEpisode method or direct SQL)
    
    if jsonOutput {
        return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
            "episode_id":         episodeID,
            "entities_created":   len(result.Entities),
            "relationships_created": len(result.Relationships),
            "entity_ids":         entityIDs,
        })
    }
    
    fmt.Printf("✓ Extracted %d entities and %d relationships from episode %s\n",
        len(result.Entities), len(result.Relationships), episodeID)
    
    return nil
}
```

**Edge case:** Name→ID resolution for relationships needs a helper function (search by name+type).

## Validation Criteria
- [ ] Command created: `bd episode extract <episode-id>`
- [ ] Reads ANTHROPIC_API_KEY from environment
- [ ] Calls `extraction.ExtractFromEpisode`
- [ ] Creates entities and relationships in storage
- [ ] `--json` output includes counts and entity IDs
- [ ] Checks readonly mode
- [ ] No compilation errors

## Impact Analysis
- **Direct impact**: New CLI command
- **Indirect impact**: Task 3-8 (MCP tool) wraps this
- **Dependencies**: Task 3-1 (extraction package), Task 1-1 (extracted_at column)

## Context
- Research Bundle: Task 3-1 design
- Pattern: `cmd/bd/create.go`

## User Feedback
*(Empty)*
