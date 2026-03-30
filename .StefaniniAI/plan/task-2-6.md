# Task 2-6: Add semantic search to EntityFilters (in-memory cosine fallback)

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: MEDIUM
- **Phase**: 2
- **Depends On**: 2-2

## Files
- `e:\Projects\BeadsMemory\beads\internal\storage\storage.go` (EXISTING)
- `e:\Projects\BeadsMemory\beads\internal\storage\dolt\entities.go` (EXISTING)

## Instructions
 Add semantic search capability to `EntityFilters` and implement in-memory cosine similarity fallback (no LLM embeddings required).

**Outcome:** Users can search entities by natural language query using term frequency similarity.

**Changes to `internal/storage/storage.go`:**
```go
type EntityFilters struct {
    EntityType string
    Name       string  // SQL LIKE search
    SemanticQuery string // NEW: Natural language search (in-memory cosine)
    Metadata   map[string]interface{}
    CreatedBy  string
    TopK       int     // NEW: Limit for semantic search results
    Limit      int
    Offset     int
}
```

**Changes to `internal/storage/dolt/entities.go` in `SearchEntities`:**
```go
func (d *DoltStore) SearchEntities(ctx context.Context, filters EntityFilters) ([]*types.Entity, error) {
    // If SemanticQuery set, use two-phase search:
    // 1. Fetch all entities (or filtered by EntityType)
    // 2. Score by cosine similarity
    // 3. Return top-K results
    
    if filters.SemanticQuery != "" {
        return d.searchEntitiesSemantic(ctx, filters)
    }
    
    // Existing SQL-based search unchanged
    // ...
}

func (d *DoltStore) searchEntitiesSemantic(ctx context.Context, filters EntityFilters) ([]*types.Entity, error) {
    // Fetch all entities (optionally filtered by type)
    baseFilters := EntityFilters{
        EntityType: filters.EntityType,
        Limit:      10000, // Large limit for scoring
    }
    
    entities, err := d.fetchEntitiesSQL(ctx, baseFilters)
    if err != nil {
        return nil, err
    }
    
    // Tokenize query
    queryTokens := similarity.Tokenize(similarity.NormalizeText(filters.SemanticQuery))
    
    // Score each entity
    type scored struct {
        entity *types.Entity
        score  float64
    }
    
    var scores []scored
    for _, e := range entities {
        text := similarity.NormalizeText(e.Name + " " + e.Summary)
        tokens := similarity.Tokenize(text)
        score := similarity.CosineSimilarity(queryTokens, tokens)
        
        if score > 0.1 { // Minimum relevance threshold
            scores = append(scores, scored{e, score})
        }
    }
    
    // Sort by score descending
    sortScores(scores)
    
    // Return top-K
    topK := filters.TopK
    if topK == 0 {
        topK = 10 // Default
    }
    if topK > len(scores) {
        topK = len(scores)
    }
    
    result := make([]*types.Entity, topK)
    for i := 0; i < topK; i++ {
        result[i] = scores[i].entity
    }
    
    return result, nil
}
```

## Architecture Pattern
Two-phase search: SQL fetch + in-memory scoring. No external vector DB. Fallback suitable for small-medium datasets (<10k entities).

## Validation Criteria
- [ ] `EntityFilters` has `SemanticQuery` and `TopK` fields
- [ ] `SearchEntities` detects `SemanticQuery` and routes to semantic path
- [ ] Cosine similarity scores entities
- [ ] Results sorted by relevance
- [ ] Top-K limit applied
- [ ] Minimum threshold (0.1) filters irrelevant results
- [ ] No compilation errors

## Impact Analysis
- **Direct impact**: Storage interface and DoltStore implementation
- **Indirect impact**: Task 2-7 (CLI), Task 2-8 (MCP) use this
- **Dependencies**: Task 2-2 (similarity package)

## Context
- Research Bundle: Task 2-2 cosine similarity
- Pattern: Two-tier query (SQL + in-memory) from `internal/query/`

## User Feedback
*(Empty)*
