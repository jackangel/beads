# Task 2-2: Extract similarity functions to reusable package

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Risk Level**: LOW
- **Phase**: 2
- **Depends On**: None

## Files
- `e:\Projects\BeadsMemory\beads\internal\similarity\similarity.go` (NEW)
- `e:\Projects\BeadsMemory\beads\internal\similarity\similarity_test.go` (NEW)

## Instructions

Extract tokenization and similarity algorithms from `cmd/bd/find_duplicates.go` into a reusable package for entity deduplication and semantic search.

**Outcome:** Tokenization, Jaccard similarity, and cosine similarity are available as standalone utilities for multiple features.

**File: `internal/similarity/similarity.go`**

```go
package similarity

import (
    "math"
    "strings"
    "unicode"
)

// Tokenize splits text into lowercase tokens, removing punctuation and single-char words.
// Returns a map of token -> count for frequency analysis.
func Tokenize(text string) map[string]int {
    tokens := make(map[string]int)
    
    // Split by whitespace and punctuation
    words := strings.FieldsFunc(text, func(r rune) bool {
        return unicode.IsSpace(r) || unicode.IsPunct(r)
    })
    
    for _, word := range words {
        word = strings.ToLower(word)
        if len(word) > 1 { // Exclude single-character tokens
            tokens[word]++
        }
    }
    
    return tokens
}

// JaccardSimilarity computes Jaccard similarity between two token frequency maps.
// Jaccard = |intersection| / |union|
// Range: [0.0, 1.0] where 1.0 = identical token sets
func JaccardSimilarity(a, b map[string]int) float64 {
    if len(a) == 0 && len(b) == 0 {
        return 1.0 // Both empty = identical
    }
    if len(a) == 0 || len(b) == 0 {
        return 0.0 // One empty = no overlap
    }
    
    intersection := 0
    union := 0
    
    allTokens := make(map[string]bool)
    for token := range a {
        allTokens[token] = true
    }
    for token := range b {
        allTokens[token] = true
    }
    
    for token := range allTokens {
        countA := a[token]
        countB := b[token]
        
        if countA > 0 && countB > 0 {
            intersection += min(countA, countB)
        }
        union += max(countA, countB)
    }
    
    if union == 0 {
        return 0.0
    }
    
    return float64(intersection) / float64(union)
}

// CosineSimilarity computes cosine similarity between two token frequency maps.
// Cosine = dot(a, b) / (||a|| * ||b||)
// Range: [0.0, 1.0] where 1.0 = identical frequency distribution
func CosineSimilarity(a, b map[string]int) float64 {
    if len(a) == 0 || len(b) == 0 {
        return 0.0
    }
    
    var dotProduct, magnitudeA, magnitudeB float64
    
    for token, countA := range a {
        countB := b[token]
        dotProduct += float64(countA * countB)
    }
    
    for _, count := range a {
        magnitudeA += float64(count * count)
    }
    for _, count := range b {
        magnitudeB += float64(count * count)
    }
    
    magnitudeA = math.Sqrt(magnitudeA)
    magnitudeB = math.Sqrt(magnitudeB)
    
    if magnitudeA == 0 || magnitudeB == 0 {
        return 0.0
    }
    
    return dotProduct / (magnitudeA * magnitudeB)
}

// NormalizeText prepares text for similarity comparison by trimming, lowercasing, and normalizing whitespace.
func NormalizeText(text string) string {
    text = strings.TrimSpace(text)
    text = strings.ToLower(text)
    text = strings.Join(strings.Fields(text), " ") // Normalize whitespace
    return text
}

func min(a, b int) int {
    if a < b { return a }
    return b
}

func max(a, b int) int {
    if a > b { return a }
    return b
}
```

**File: `internal/similarity/similarity_test.go`**

```go
package similarity

import (
    "testing"
)

func TestTokenize(t *testing.T) {
    tests := []struct {
        name   string
        input  string
        expect map[string]int
    }{
        {
            name:  "simple text",
            input: "hello world",
            expect: map[string]int{"hello": 1, "world": 1},
        },
        {
            name:  "repeated words",
            input: "test test test",
            expect: map[string]int{"test": 3},
        },
        {
            name:  "punctuation removal",
            input: "hello, world!",
            expect: map[string]int{"hello": 1, "world": 1},
        },
        {
            name:  "single-char exclusion",
            input: "a b c hello",
            expect: map[string]int{"hello": 1},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Tokenize(tt.input)
            if len(got) != len(tt.expect) {
                t.Errorf("Tokenize() len = %d, want %d", len(got), len(tt.expect))
            }
            for k, v := range tt.expect {
                if got[k] != v {
                    t.Errorf("Tokenize()[%s] = %d, want %d", k, got[k], v)
                }
            }
        })
    }
}

func TestJaccardSimilarity(t *testing.T) {
    a := map[string]int{"hello": 1, "world": 1}
    b := map[string]int{"hello": 1, "universe": 1}
    
    sim := JaccardSimilarity(a, b)
    if sim < 0.3 || sim > 0.4 {
        t.Errorf("JaccardSimilarity() = %.2f, want ~0.33", sim)
    }
    
    // Identical maps
    identical := JaccardSimilarity(a, a)
    if identical != 1.0 {
        t.Errorf("JaccardSimilarity(identical) = %.2f, want 1.0", identical)
    }
}

func TestCosineSimilarity(t *testing.T) {
    a := map[string]int{"hello": 1, "world": 1}
    b := map[string]int{"hello": 1, "universe": 1}
    
    sim := CosineSimilarity(a, b)
    if sim < 0.4 || sim > 0.6 {
        t.Errorf("CosineSimilarity() = %.2f, want ~0.5", sim)
    }
    
    // Identical maps
    identical := CosineSimilarity(a, a)
    if identical != 1.0 {
        t.Errorf("CosineSimilarity(identical) = %.2f, want 1.0", identical)
    }
}
```

**Migration from find_duplicates.go:**
After creating this package, update `cmd/bd/find_duplicates.go` to import and use `similarity.Tokenize`, `similarity.JaccardSimilarity`, `similarity.CosineSimilarity`. Remove duplicate function definitions from find_duplicates.go.

## Architecture Pattern

**Reusable Utility Package**:
- Pure functions (no state, no dependencies)
- Well-tested (table-driven tests)
- Used by multiple features (dedup, semantic search)
- Located in `internal/` (not exported outside beads)

**Algorithm Choice**:
- **Jaccard**: Set similarity (good for duplicate detection)
- **Cosine**: Vector similarity (good for semantic search, respects term frequency)
- **Tokenize**: Common preprocessing (lowercase, remove punctuation, filter single-char)

## Validation Criteria
- [ ] `internal/similarity/similarity.go` created with 5 functions (Tokenize, JaccardSimilarity, CosineSimilarity, NormalizeText, min/max)
- [ ] `internal/similarity/similarity_test.go` created with 3 test functions
- [ ] All tests pass (`go test ./internal/similarity/...`)
- [ ] Functions are pure (no side effects, no global state)
- [ ] Tokenize excludes single-char words
- [ ] Jaccard returns [0.0, 1.0] range
- [ ] Cosine returns [0.0, 1.0] range
- [ ] No compilation errors

## Impact Analysis
- **Direct impact**: New package (foundation for dedup and search)
- **Indirect impact**: `cmd/bd/find_duplicates.go` will be refactored to use this package
- **Dependencies**: Task 2-3 (entity dedup), task 2-6 (semantic search) use these functions

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Issue Duplicate Detection" section for algorithm details)
- Source: `cmd/bd/find_duplicates.go` (copy tokenize, jaccardSimilarity, cosineSimilarity functions)

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
