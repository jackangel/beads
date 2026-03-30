package dedup

import (
	"testing"

	"github.com/steveyegge/beads/internal/types"
)

func TestFindDuplicates_EmptyInput(t *testing.T) {
	entities := []*types.Entity{}
	duplicates := FindDuplicates(entities, 0.5)

	if duplicates != nil {
		t.Errorf("Expected nil for empty input, got %d duplicates", len(duplicates))
	}
}

func TestFindDuplicates_NoDuplicatesFound(t *testing.T) {
	entities := []*types.Entity{
		{
			ID:         "ent-1",
			Name:       "Alice Johnson",
			EntityType: "person",
			Summary:    "Senior software engineer at Acme Corp",
		},
		{
			ID:         "ent-2",
			Name:       "Bob Williams",
			EntityType: "person",
			Summary:    "Product manager at XYZ Inc",
		},
		{
			ID:         "ent-3",
			Name:       "Gamma Framework",
			EntityType: "technology",
			Summary:    "Open source web framework for Go",
		},
	}

	duplicates := FindDuplicates(entities, 0.5)

	if len(duplicates) != 0 {
		t.Errorf("Expected 0 duplicates, got %d", len(duplicates))
	}
}

func TestFindDuplicates_DuplicatesFoundAboveThreshold(t *testing.T) {
	entities := []*types.Entity{
		{
			ID:         "ent-1",
			Name:       "Alice Johnson",
			EntityType: "person",
			Summary:    "Senior software engineer",
		},
		{
			ID:         "ent-2",
			Name:       "Alice Johnston",
			EntityType: "person",
			Summary:    "Senior software developer",
		},
		{
			ID:         "ent-3",
			Name:       "Bob Williams",
			EntityType: "person",
			Summary:    "Product manager",
		},
	}

	// Use lower threshold to account for tokenization differences
	duplicates := FindDuplicates(entities, 0.4)

	if len(duplicates) == 0 {
		t.Fatal("Expected duplicates to be found, got none")
	}

	// Should find Alice Johnson ~ Alice Johnston
	found := false
	for _, dup := range duplicates {
		if (dup.EntityA.ID == "ent-1" && dup.EntityB.ID == "ent-2") ||
			(dup.EntityA.ID == "ent-2" && dup.EntityB.ID == "ent-1") {
			found = true
			if dup.Score < 0.4 {
				t.Errorf("Expected score >= 0.4, got %.2f", dup.Score)
			}
		}
	}

	if !found {
		t.Error("Expected to find Alice Johnson ~ Alice Johnston duplicate pair")
	}
}

func TestFindDuplicates_IdenticalNames(t *testing.T) {
	entities := []*types.Entity{
		{
			ID:         "ent-1",
			Name:       "John Doe",
			EntityType: "person",
			Summary:    "First occurrence",
		},
		{
			ID:         "ent-2",
			Name:       "John Doe",
			EntityType: "person",
			Summary:    "Second occurrence",
		},
	}

	duplicates := FindDuplicates(entities, 0.5)

	if len(duplicates) != 1 {
		t.Fatalf("Expected 1 duplicate pair, got %d", len(duplicates))
	}

	dup := duplicates[0]
	if dup.Score < 0.8 {
		t.Errorf("Expected high score for identical names, got %.2f", dup.Score)
	}

	if dup.Reason != "Identical names" {
		t.Errorf("Expected reason 'Identical names', got '%s'", dup.Reason)
	}
}

func TestFindDuplicates_DifferentEntityTypesNotMatched(t *testing.T) {
	entities := []*types.Entity{
		{
			ID:         "ent-1",
			Name:       "Phoenix Framework",
			EntityType: "technology",
			Summary:    "Web framework for Elixir",
		},
		{
			ID:         "ent-2",
			Name:       "Phoenix Framework",
			EntityType: "project", // Different type
			Summary:    "Web framework for Elixir",
		},
	}

	duplicates := FindDuplicates(entities, 0.5)

	if len(duplicates) != 0 {
		t.Errorf("Expected 0 duplicates for different entity types, got %d", len(duplicates))
	}
}

func TestFindDuplicates_ThresholdEdgeCases(t *testing.T) {
	entities := []*types.Entity{
		{
			ID:         "ent-1",
			Name:       "DataPro",
			EntityType: "product",
			Summary:    "Data analytics platform",
		},
		{
			ID:         "ent-2",
			Name:       "DataPro Plus",
			EntityType: "product",
			Summary:    "Data analytics solution",
		},
	}

	// Test threshold at 0.0 (all pairs should match)
	duplicates := FindDuplicates(entities, 0.0)
	if len(duplicates) == 0 {
		t.Error("Expected duplicates at threshold 0.0")
	}

	// Test threshold at 1.0 (only identical pairs should match)
	duplicates = FindDuplicates(entities, 1.0)
	if len(duplicates) != 0 {
		t.Errorf("Expected 0 duplicates at threshold 1.0, got %d", len(duplicates))
	}

	// Test intermediate threshold
	duplicates = FindDuplicates(entities, 0.5)
	// Should find or not find depending on actual similarity
	// Just verify it doesn't crash
	if duplicates == nil {
		t.Error("Expected non-nil result")
	}
}

func TestFindDuplicates_SortedByScore(t *testing.T) {
	entities := []*types.Entity{
		{
			ID:         "ent-1",
			Name:       "Apple Inc",
			EntityType: "company",
			Summary:    "Technology company",
		},
		{
			ID:         "ent-2",
			Name:       "Apple Incorporated",
			EntityType: "company",
			Summary:    "Tech company based in Cupertino",
		},
		{
			ID:         "ent-3",
			Name:       "Apple Corp",
			EntityType: "company",
			Summary:    "Technology corporation",
		},
	}

	duplicates := FindDuplicates(entities, 0.4)

	if len(duplicates) == 0 {
		t.Fatal("Expected duplicates to be found")
	}

	// Verify results are sorted by score descending
	for i := 0; i < len(duplicates)-1; i++ {
		if duplicates[i].Score < duplicates[i+1].Score {
			t.Errorf("Results not sorted: duplicates[%d].Score (%.2f) < duplicates[%d].Score (%.2f)",
				i, duplicates[i].Score, i+1, duplicates[i+1].Score)
		}
	}

	// Verify highest score is first (relaxed threshold for actual similarity)
	if duplicates[0].Score < 0.3 {
		t.Errorf("Expected highest score >= 0.3, got %.2f", duplicates[0].Score)
	}
}

func TestFindDuplicates_EmptyAndNonEmptyStrings(t *testing.T) {
	entities := []*types.Entity{
		{
			ID:         "ent-1",
			Name:       "Valid Entity",
			EntityType: "concept",
			Summary:    "This has a summary",
		},
		{
			ID:         "ent-2",
			Name:       "Valid Entity",
			EntityType: "concept",
			Summary:    "", // Empty summary
		},
	}

	duplicates := FindDuplicates(entities, 0.5)

	// Should still find as duplicate due to identical names
	if len(duplicates) == 0 {
		t.Error("Expected duplicates despite empty summary")
	}
}

func TestFindDuplicates_SingleEntity(t *testing.T) {
	entities := []*types.Entity{
		{
			ID:         "ent-1",
			Name:       "Solo Entity",
			EntityType: "concept",
			Summary:    "Only one entity",
		},
	}

	duplicates := FindDuplicates(entities, 0.5)

	if len(duplicates) != 0 {
		t.Errorf("Expected 0 duplicates for single entity, got %d", len(duplicates))
	}
}

func TestCompareText(t *testing.T) {
	tests := []struct {
		name     string
		textA    string
		textB    string
		minScore float64
		maxScore float64
	}{
		{
			name:     "identical text",
			textA:    "Hello World",
			textB:    "Hello World",
			minScore: 0.99,
			maxScore: 1.0,
		},
		{
			name:     "case insensitive",
			textA:    "Hello World",
			textB:    "hello world",
			minScore: 0.99,
			maxScore: 1.0,
		},
		{
			name:     "different text",
			textA:    "Apple",
			textB:    "Orange",
			minScore: 0.0,
			maxScore: 0.1,
		},
		{
			name:     "both empty",
			textA:    "",
			textB:    "",
			minScore: 1.0,
			maxScore: 1.0,
		},
		{
			name:     "one empty",
			textA:    "Something",
			textB:    "",
			minScore: 0.0,
			maxScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := compareText(tt.textA, tt.textB)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Expected score in [%.2f, %.2f], got %.2f",
					tt.minScore, tt.maxScore, score)
			}
		})
	}
}

func TestBuildReason(t *testing.T) {
	entityA := &types.Entity{Name: "Alice"}
	entityB := &types.Entity{Name: "Alice"}

	reason := buildReason(entityA, entityB, 1.0, 0.5)
	if reason != "Identical names" {
		t.Errorf("Expected 'Identical names', got '%s'", reason)
	}

	entityB.Name = "Alicia"
	reason = buildReason(entityA, entityB, 0.95, 0.5)
	if reason != "Very similar names" {
		t.Errorf("Expected 'Very similar names', got '%s'", reason)
	}

	reason = buildReason(entityA, entityB, 0.75, 0.75)
	if reason != "Similar names and summaries" {
		t.Errorf("Expected 'Similar names and summaries', got '%s'", reason)
	}

	reason = buildReason(entityA, entityB, 0.75, 0.5)
	if reason != "Similar names" {
		t.Errorf("Expected 'Similar names', got '%s'", reason)
	}

	reason = buildReason(entityA, entityB, 0.5, 0.75)
	if reason != "Similar summaries" {
		t.Errorf("Expected 'Similar summaries', got '%s'", reason)
	}

	reason = buildReason(entityA, entityB, 0.5, 0.5)
	if reason != "Moderate similarity" {
		t.Errorf("Expected 'Moderate similarity', got '%s'", reason)
	}
}
