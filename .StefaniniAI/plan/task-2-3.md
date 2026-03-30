# Task 2-3: Create entity deduplication package with Jaccard/cosine algorithms

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Risk Level**: MEDIUM
- **Phase**: 2
- **Depends On**: 2-2

## Files
- `e:\Projects\BeadsMemory\beads\internal\dedup\entity.go` (NEW)
- `e:\Projects\BeadsMemory\beads\internal\dedup\entity_test.go` (NEW)

## Instructions

Create an entity deduplication package that finds potential duplicate entities using Jaccard and cosine similarity on entity names + summaries.

**Outcome:** A reusable algorithm for detecting duplicate entities across the knowledge graph.

**File: `internal/dedup/entity.go`**

```go
package dedup

import (
    "context"
    "github.com/steveyegge/beads/internal/similarity"
    "github.com/steveyegge/beads/internal/storage"
    "github.com/steveyegge/beads/internal/types"
)

// DuplicatePair represents two entities that may be duplicates.
type DuplicatePair struct {
    EntityA    *types.Entity `json:"entity_a"`
    EntityB    *types.Entity `json:"entity_b"`
    Similarity float64       `json:"similarity"`
    Method     string        `json:"method"` // "jaccard" or "cosine"
    Reason     string        `json:"reason,omitempty"`
}

// FindDuplicates searches for duplicate entities using mechanical similarity.
// Compares name + summary text using both Jaccard and cosine similarity.
// Returns pairs above the threshold, sorted by similarity (highest first).
func FindDuplicates(ctx context.Context, store storage.EntityStore, entityType string, threshold float64) ([]DuplicatePair, error) {
    // Get all entities of the specified type (or all types if empty)
    filters := storage.EntityFilters{
        EntityType: entityType,
        Limit:      1000, // Pagination TBD
    }
    
    entities, err := store.SearchEntities(ctx, filters)
    if err != nil {
        return nil, err
    }
    
    var duplicates []DuplicatePair
    
    // O(n^2) naive comparison (optimization TBD for large datasets)
    for i := 0; i < len(entities); i++ {
        for j := i + 1; j < len(entities); j++ {
            a := entities[i]
            b := entities[j]
            
            // Skip if already merged
            if a.MergedInto != nil || b.MergedInto != nil {
                continue
            }
            
            // Compare names + summaries
            textA := similarity.NormalizeText(a.Name + " " + a.Summary)
            textB := similarity.NormalizeText(b.Name + " " + b.Summary)
            
            tokensA := similarity.Tokenize(textA)
            tokensB := similarity.Tokenize(textB)
            
            jaccardSim := similarity.JaccardSimilarity(tokensA, tokensB)
            cosineSim := similarity.CosineSimilarity(tokensA, tokensB)
            
            // Use average of Jaccard and cosine
            avgSim := (jaccardSim + cosineSim) / 2.0
            
            if avgSim >= threshold {
                method := "jaccard+cosine"
                reason := ""
                if a.Name == b.Name {
                    reason = "Identical names"
                } else if jaccardSim > cosineSim {
                    method = "jaccard"
                } else {
                    method = "cosine"
                }
                
                duplicates = append(duplicates, DuplicatePair{
                    EntityA:    a,
                    EntityB:    b,
                    Similarity: avgSim,
                    Method:     method,
                    Reason:     reason,
                })
            }
        }
    }
    
    // Sort by similarity (descending)
    sortDuplicatesBySimil(duplicates)
    
    return duplicates, nil
}

func sortDuplicatesByimilarity(pairs []DuplicatePair) {
    // Implement sort (slice.Sort or custom)
    // For simplicity: bubble sort descending by Similarity
    for i := 0; i < len(pairs); i++ {
        for j := i + 1; j < len(pairs); j++ {
            if pairs[j].Similarity > pairs[i].Similarity {
                pairs[i], pairs[j] = pairs[j], pairs[i]
            }
        }
    }
}
```

**File: `internal/dedup/entity_test.go`**

```go
package dedup

import (
    "context"
    "testing"
    "github.com/steveyegge/beads/internal/types"
)

func TestFindDuplicates(t *testing.T) {
    // Mock store (TBD: use testutil or in-memory mock)
    // For now, just validate the sorting/filtering logic
    
    pairs := []DuplicatePair{
        {Similarity: 0.5},
        {Similarity: 0.9},
        {Similarity: 0.7},
    }
    
    sortDuplicatesByimilarity(pairs)
    
    if pairs[0].Similarity != 0.9 {
        t.Errorf("Expected highest similarity first, got %.2f", pairs[0].Similarity)
    }
}
```

**Notes:**
- O(n²) performance: optimize in future with LSH (Locality-Sensitive Hashing) or clustering
- Pagination: handle large entity sets (>1000) with batched processing
- Merged entity filter: skip entities where `merged_into IS NOT NULL`

## Architecture Pattern

**Deduplication Algorithm**:
- Combine name + summary for comparison text
- Use both Jaccard (set overlap) and cosine (frequency distribution)
- Average scores for balanced results
- Sort by similarity descending

**Storage Integration**:
- Use `EntityStore.SearchEntities` to fetch entities
- Filter out already-merged entities (check `MergedInto` field)
- Return `DuplicatePair` structs (not raw entities)

## Validation Criteria
- [ ] `internal/dedup/entity.go` created with `FindDuplicates` function
- [ ] `DuplicatePair` struct matches Research Bundle spec
- [ ] Entities filtered by type (if specified)
- [ ] Merged entities excluded (check MergedInto != nil)
- [ ] Results sorted by similarity descending
- [ ] Uses `similarity` package from task 2-2
- [ ] No compilation errors
- [ ] Basic test validates sorting logic

## Impact Analysis
- **Direct impact**: New dedup package
- **Indirect impact**: Task 2-4 (CLI command) calls this function
- **Dependencies**: Task 2-2 (similarity package)

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Feature 3: Entity Deduplication" for algorithm design)
- Similar code: `cmd/bd/find_duplicates.go` (issue dedup, copy pattern)
- Similarity package: Task 2-2 provides Tokenize, Jaccard, Cosine

## User Feedback
*(Empty)
