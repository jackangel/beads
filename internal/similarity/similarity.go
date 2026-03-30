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

	result := dotProduct / (magnitudeA * magnitudeB)

	// Clamp to [0, 1] to handle floating point precision errors
	if result > 1.0 {
		return 1.0
	}
	if result < 0.0 {
		return 0.0
	}

	return result
}

// NormalizeText prepares text for similarity comparison by trimming, lowercasing, and normalizing whitespace.
func NormalizeText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)
	text = strings.Join(strings.Fields(text), " ") // Normalize whitespace
	return text
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
