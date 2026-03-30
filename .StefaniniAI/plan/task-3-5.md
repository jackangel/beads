# Task 3-5: Create memory retrieval package with graph traversal

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Risk Level**: HIGH
- **Phase**: 3
- **Depends On**: 2-6, 1-2

## Files
- `e:\Projects\BeadsMemory\beads\internal\retrieval\context.go` (NEW)
- `e:\Projects\BeadsMemory\beads\internal\retrieval\types.go` (NEW)
- `e:\Projects\BeadsMemory\beads\internal\retrieval\retrieval_test.go` (NEW)

## Instructions
Create a memory retrieval package that combines semantic search + graph traversal + temporal filtering to assemble relevant context.

**Outcome:** AI agents can query "Alice's role in auth service" and get entities, relationships, episodes, and relevance scores.

**File: `internal/retrieval/types.go`**
```go
package retrieval

import (
    "time"
    "github.com/steveyegge/beads/internal/types"
)

// MemoryQuery specifies what context to retrieve.
type MemoryQuery struct {
    TextQuery      string    // Natural language query
    EntityIDs      []string  // Specific entities to include
    ValidAt        *time.Time // Temporal filter (default: now)
    MaxHops        int       // Graph traversal depth (default: 2)
    TopK           int       // Max entities from semantic search (default: 5)
    MinConfidence  float64   // Filter relationships (default: 0.5)
}

// MemoryContext holds the retrieved context.
type MemoryContext struct {
    Entities         []*types.Entity         `json:"entities"`
    Relationships    []*types.Relationship   `json:"relationships"`
    SourceEpisodes   []*types.Episode        `json:"source_episodes"`
    RelevanceScores  map[string]float64      `json:"relevance_scores"` // entity ID -> score
}
```

**File: `internal/retrieval/context.go`**
```go
package retrieval

import (
    "context"
    "github.com/steveyegge/beads/internal/storage"
    "github.com/steveyegge/beads/internal/types"
    "time"
)

// RetrieveMemory assembles relevant context from knowledge graph.
// 1. Semantic search for initial entities (TextQuery)
// 2. Graph traversal from initial entities (MaxHops)
// 3. Temporal filtering on relationships (ValidAt)
// 4. Episode lookup for provenance
func RetrieveMemory(ctx context.Context, store storage.Storage, query MemoryQuery) (*MemoryContext, error) {
    validAt := query.ValidAt
    if validAt == nil {
        now := time.Now()
        validAt = &now
    }
    
    // Step 1: Semantic search for initial entities
    searchFilters := storage.EntityFilters{
        SemanticQuery: query.TextQuery,
        TopK:          query.TopK,
    }
    seedEntities, err := store.SearchEntities(ctx, searchFilters)
    if err != nil {
        return nil, err
    }
    
    // Step 2: Graph traversal (BFS from seed entities)
    entities := make(map[string]*types.Entity)
    relationships := make(map[string]*types.Relationship)
    
    for _, e := range seedEntities {
        entities[e.ID] = e
    }
    
    visited := make(map[string]bool)
    frontier := make([]string, 0)
    for _, e := range seedEntities {
        frontier = append(frontier, e.ID)
    }
    
    for hop := 0; hop < query.MaxHops && len(frontier) > 0; hop++ {
        nextFrontier := []string{}
        
        for _, entityID := range frontier {
            if visited[entityID] {
                continue
            }
            visited[entityID] = true
            
            // Get all relationships (outgoing and incoming)
            relFilters := storage.RelationshipFilters{
                ValidAt:       validAt,
                MinConfidence: &query.MinConfidence,
            }
            
            // Outgoing
            relFilters.SourceEntityID = entityID
            rels, _ := store.SearchRelationships(ctx, relFilters)
            for _, rel := range rels {
                relationships[rel.ID] = rel
                if !visited[rel.TargetEntityID] {
                    nextFrontier = append(nextFrontier, rel.TargetEntityID)
                }
            }
            
            // Incoming
            relFilters.SourceEntityID = ""
            relFilters.TargetEntityID = entityID
            rels, _ = store.SearchRelationships(ctx, relFilters)
            for _, rel := range rels {
                relationships[rel.ID] = rel
                if !visited[rel.SourceEntityID] {
                    nextFrontier = append(nextFrontier, rel.SourceEntityID)
                }
            }
        }
        
        // Fetch entities for next frontier
        for _, id := range nextFrontier {
            if entities[id] == nil {
                e, _ := store.GetEntity(ctx, id)
                if e != nil {
                    entities[id] = e
                }
            }
        }
        
        frontier = nextFrontier
    }
    
    // Step 3: Lookup source episodes (entities_extracted includes entity IDs)
    // TBD: Add GetEpisodesByEntityID method or filter SearchEpisodes
    
    // Assemble result
    result := &MemoryContext{
        Entities:      make([]*types.Entity, 0, len(entities)),
        Relationships: make([]*types.Relationship, 0, len(relationships)),
        RelevanceScores: make(map[string]float64),
    }
    
    for _, e := range entities {
        result.Entities = append(result.Entities, e)
    }
    for _, rel := range relationships {
        result.Relationships = append(result.Relationships, rel)
    }
    
    // Relevance: seed entities score 1.0, others decay by hop distance
    for _, e := range seedEntities {
        result.RelevanceScores[e.ID] = 1.0
    }
    
    return result, nil
}
```

## Architecture Pattern
**Graph Traversal**: BFS (Breadth-First Search) from seed entities. **Temporal Filtering**: Only include relationships valid at `ValidAt` timestamp. **Relevance Scoring**: Seed entities = 1.0, decays by hop distance.

## Validation Criteria
- [ ] `internal/retrieval/types.go` created with MemoryQuery and MemoryContext
- [ ] `internal/retrieval/context.go` created with RetrieveMemory function
- [ ] Semantic search seeds the traversal
- [ ] BFS graph traversal respects MaxHops
- [ ] Temporal filtering on relationships (ValidAt)
- [ ] Confidence filtering applied
- [ ] Relevance scores computed
- [ ] No compilation errors

## Impact Analysis
- **Direct impact**: New retrieval package (core intelligence feature)
- **Indirect impact**: Task 3-6 (storage interface), Task 3-7 (CLI) use this
- **Dependencies**: Task 2-6 (semantic search), Task 1-2 (confidence filtering)

## Context
- Research Bundle: "Feature 5: Memory Retrieval Interface"
## User Feedback
*(Empty)*
