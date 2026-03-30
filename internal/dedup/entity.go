// Package dedup provides entity deduplication and resolution algorithms.
package dedup

import (
	"sort"

	"github.com/steveyegge/beads/internal/similarity"
	"github.com/steveyegge/beads/internal/types"
)

// MergeStrategy defines how to merge duplicate entities.
type MergeStrategy int

const (
	// KeepTarget keeps the target entity unchanged, discards source.
	KeepTarget MergeStrategy = iota
	// MergeMetadata merges metadata fields from both entities.
	MergeMetadata
	// MergeSummaries combines summaries from both entities.
	MergeSummaries
)

// DuplicateCandidate represents two entities that may be duplicates.
type DuplicateCandidate struct {
	EntityA *types.Entity `json:"entity_a"`
	EntityB *types.Entity `json:"entity_b"`
	Score   float64       `json:"score"`   // Similarity score [0.0, 1.0]
	Reason  string        `json:"reason"`  // Human-readable explanation
}

// FindDuplicates searches for duplicate entities using text similarity.
// Compares name + summary text using both Jaccard and cosine similarity.
// Only entities with matching entity_type are compared.
// Returns pairs above the threshold, sorted by score (highest first).
//
// This function operates on already-loaded entity slices and does NOT call storage.
func FindDuplicates(entities []*types.Entity, threshold float64) []DuplicateCandidate {
	if len(entities) == 0 {
		return nil
	}

	var duplicates []DuplicateCandidate

	// O(n²) pairwise comparison
	for i := 0; i < len(entities); i++ {
		for j := i + 1; j < len(entities); j++ {
			a := entities[i]
			b := entities[j]

			// Skip if different entity types
			if a.EntityType != b.EntityType {
				continue
			}

			// Compare names (primary signal)
			nameScore := compareText(a.Name, b.Name)

			// Compare summaries (secondary signal)
			summaryScore := compareText(a.Summary, b.Summary)

			// Weighted average: name is more important than summary
			// Name: 70%, Summary: 30%
			finalScore := nameScore*0.7 + summaryScore*0.3

			if finalScore >= threshold {
				reason := buildReason(a, b, nameScore, summaryScore)
				duplicates = append(duplicates, DuplicateCandidate{
					EntityA: a,
					EntityB: b,
					Score:   finalScore,
					Reason:  reason,
				})
			}
		}
	}

	// Sort by score descending (highest similarity first)
	sort.Slice(duplicates, func(i, j int) bool {
		return duplicates[i].Score > duplicates[j].Score
	})

	return duplicates
}

// compareText computes similarity between two text strings using Jaccard and cosine.
// Returns the average of both metrics.
func compareText(textA, textB string) float64 {
	if textA == "" && textB == "" {
		return 1.0 // Both empty = identical
	}
	if textA == "" || textB == "" {
		return 0.0 // One empty = no match
	}

	// Normalize and tokenize
	normalizedA := similarity.NormalizeText(textA)
	normalizedB := similarity.NormalizeText(textB)

	tokensA := similarity.Tokenize(normalizedA)
	tokensB := similarity.Tokenize(normalizedB)

	// Compute both metrics
	jaccard := similarity.JaccardSimilarity(tokensA, tokensB)
	cosine := similarity.CosineSimilarity(tokensA, tokensB)

	// Average for balanced results
	return (jaccard + cosine) / 2.0
}

// buildReason generates a human-readable explanation for why entities match.
func buildReason(a, b *types.Entity, nameScore, summaryScore float64) string {
	if a.Name == b.Name {
		return "Identical names"
	}
	if nameScore >= 0.9 {
		return "Very similar names"
	}
	if nameScore >= 0.7 && summaryScore >= 0.7 {
		return "Similar names and summaries"
	}
	if nameScore >= 0.7 {
		return "Similar names"
	}
	if summaryScore >= 0.7 {
		return "Similar summaries"
	}
	return "Moderate similarity"
}
