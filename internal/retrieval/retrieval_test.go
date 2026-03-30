package retrieval

import (
	"context"
	"testing"
	"time"

	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
)

// mockStorage provides a test double for storage.Storage interface
type mockStorage struct {
	storage.Storage
	entities      []*types.Entity
	relationships []*types.Relationship
	episodes      []*types.Episode
}

func (m *mockStorage) SearchEntities(ctx context.Context, filters storage.EntityFilters) ([]*types.Entity, error) {
	// Simple mock: return entities matching the TextQuery
	if filters.TextQuery != "" {
		var results []*types.Entity
		for _, e := range m.entities {
			// Simple substring match on name
			if e.Name == filters.TextQuery || filters.TextQuery == "" {
				results = append(results, e)
			}
		}
		return results, nil
	}
	return nil, nil
}

func (m *mockStorage) GetEntity(ctx context.Context, id string) (*types.Entity, error) {
	for _, e := range m.entities {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, nil
}

func (m *mockStorage) SearchRelationships(ctx context.Context, filters storage.RelationshipFilters) ([]*types.Relationship, error) {
	var results []*types.Relationship
	for _, rel := range m.relationships {
		// Filter by source
		if filters.SourceEntityID != "" && rel.SourceEntityID != filters.SourceEntityID {
			continue
		}
		// Filter by target
		if filters.TargetEntityID != "" && rel.TargetEntityID != filters.TargetEntityID {
			continue
		}
		// Filter by confidence
		if filters.MinConfidence != nil {
			if conf, ok := rel.Metadata["confidence"].(float64); ok {
				if conf < *filters.MinConfidence {
					continue
				}
			}
		}
		// Filter by temporal validity
		if filters.ValidAt != nil {
			if !rel.IsValidAt(*filters.ValidAt) {
				continue
			}
		}
		results = append(results, rel)
	}
	return results, nil
}

func TestRetrieveMemory_BasicTraversal(t *testing.T) {
	// Setup mock graph: Alice -> worksOn -> AuthService <- maintainedBy <- Bob
	now := time.Now()
	alice := &types.Entity{ID: "ent-alice", Name: "Alice", EntityType: "person"}
	authService := &types.Entity{ID: "ent-auth", Name: "AuthService", EntityType: "service"}
	bob := &types.Entity{ID: "ent-bob", Name: "Bob", EntityType: "person"}

	relAliceAuth := &types.Relationship{
		ID:               "rel-1",
		SourceEntityID:   "ent-alice",
		RelationshipType: "worksOn",
		TargetEntityID:   "ent-auth",
		ValidFrom:        now.Add(-30 * 24 * time.Hour),
		Metadata:         map[string]interface{}{"confidence": 0.9},
	}

	relBobAuth := &types.Relationship{
		ID:               "rel-2",
		SourceEntityID:   "ent-bob",
		RelationshipType: "maintainedBy",
		TargetEntityID:   "ent-auth",
		ValidFrom:        now.Add(-60 * 24 * time.Hour),
		Metadata:         map[string]interface{}{"confidence": 0.85},
	}

	mock := &mockStorage{
		entities:      []*types.Entity{alice, authService, bob},
		relationships: []*types.Relationship{relAliceAuth, relBobAuth},
	}

	// Execute retrieval starting from Alice
	ctx := context.Background()
	query := MemoryQuery{
		TextQuery:     "Alice",
		MaxHops:       2,
		TopK:          5,
		MinConfidence: 0.5,
	}

	result, err := RetrieveMemory(ctx, mock, query)
	if err != nil {
		t.Fatalf("RetrieveMemory failed: %v", err)
	}

	// Verify results
	if len(result.Entities) < 2 {
		t.Errorf("Expected at least 2 entities (Alice + AuthService), got %d", len(result.Entities))
	}

	if len(result.Relationships) == 0 {
		t.Errorf("Expected relationships, got 0")
	}

	// Check relevance scores
	if score, ok := result.RelevanceScores["ent-alice"]; !ok || score != 1.0 {
		t.Errorf("Expected Alice to have relevance score 1.0, got %v", score)
	}
}

func TestRetrieveMemory_TemporalFiltering(t *testing.T) {
	now := time.Now()
	past := now.Add(-90 * 24 * time.Hour)

	alice := &types.Entity{ID: "ent-alice", Name: "Alice", EntityType: "person"}
	service := &types.Entity{ID: "ent-svc", Name: "Service", EntityType: "service"}

	// Expired relationship
	expiredRel := &types.Relationship{
		ID:               "rel-expired",
		SourceEntityID:   "ent-alice",
		RelationshipType: "workedOn",
		TargetEntityID:   "ent-svc",
		ValidFrom:        past,
		ValidUntil:       &past,
		Metadata:         map[string]interface{}{"confidence": 0.9},
	}

	// Current relationship
	currentRel := &types.Relationship{
		ID:               "rel-current",
		SourceEntityID:   "ent-alice",
		RelationshipType: "worksOn",
		TargetEntityID:   "ent-svc",
		ValidFrom:        now.Add(-30 * 24 * time.Hour),
		Metadata:         map[string]interface{}{"confidence": 0.9},
	}

	mock := &mockStorage{
		entities:      []*types.Entity{alice, service},
		relationships: []*types.Relationship{expiredRel, currentRel},
	}

	ctx := context.Background()
	query := MemoryQuery{
		TextQuery:     "Alice",
		ValidAt:       &now,
		MaxHops:       1,
		TopK:          5,
		MinConfidence: 0.5,
	}

	result, err := RetrieveMemory(ctx, mock, query)
	if err != nil {
		t.Fatalf("RetrieveMemory failed: %v", err)
	}

	// Should only have current relationship, not expired
	foundExpired := false
	foundCurrent := false
	for _, rel := range result.Relationships {
		if rel.ID == "rel-expired" {
			foundExpired = true
		}
		if rel.ID == "rel-current" {
			foundCurrent = true
		}
	}

	if foundExpired {
		t.Errorf("Should not include expired relationship")
	}
	if !foundCurrent {
		t.Errorf("Should include current relationship")
	}
}

func TestRetrieveMemory_ConfidenceFiltering(t *testing.T) {
	now := time.Now()
	alice := &types.Entity{ID: "ent-alice", Name: "Alice", EntityType: "person"}
	service := &types.Entity{ID: "ent-svc", Name: "Service", EntityType: "service"}

	// Low confidence relationship
	lowConfRel := &types.Relationship{
		ID:               "rel-low",
		SourceEntityID:   "ent-alice",
		RelationshipType: "maybeWorksOn",
		TargetEntityID:   "ent-svc",
		ValidFrom:        now,
		Metadata:         map[string]interface{}{"confidence": 0.3},
	}

	// High confidence relationship
	highConfRel := &types.Relationship{
		ID:               "rel-high",
		SourceEntityID:   "ent-alice",
		RelationshipType: "worksOn",
		TargetEntityID:   "ent-svc",
		ValidFrom:        now,
		Metadata:         map[string]interface{}{"confidence": 0.9},
	}

	mock := &mockStorage{
		entities:      []*types.Entity{alice, service},
		relationships: []*types.Relationship{lowConfRel, highConfRel},
	}

	ctx := context.Background()
	query := MemoryQuery{
		TextQuery:     "Alice",
		MaxHops:       1,
		TopK:          5,
		MinConfidence: 0.5, // Filter out low confidence
	}

	result, err := RetrieveMemory(ctx, mock, query)
	if err != nil {
		t.Fatalf("RetrieveMemory failed: %v", err)
	}

	// Should only have high confidence relationship
	foundLow := false
	foundHigh := false
	for _, rel := range result.Relationships {
		if rel.ID == "rel-low" {
			foundLow = true
		}
		if rel.ID == "rel-high" {
			foundHigh = true
		}
	}

	if foundLow {
		t.Errorf("Should filter out low confidence relationship")
	}
	if !foundHigh {
		t.Errorf("Should include high confidence relationship")
	}
}

func TestRetrieveMemory_MaxHopsLimit(t *testing.T) {
	now := time.Now()
	// Chain: A -> B -> C -> D
	entities := []*types.Entity{
		{ID: "ent-a", Name: "A", EntityType: "person"},
		{ID: "ent-b", Name: "B", EntityType: "person"},
		{ID: "ent-c", Name: "C", EntityType: "person"},
		{ID: "ent-d", Name: "D", EntityType: "person"},
	}

	relationships := []*types.Relationship{
		{
			ID:               "rel-ab",
			SourceEntityID:   "ent-a",
			RelationshipType: "knows",
			TargetEntityID:   "ent-b",
			ValidFrom:        now,
			Metadata:         map[string]interface{}{"confidence": 0.9},
		},
		{
			ID:               "rel-bc",
			SourceEntityID:   "ent-b",
			RelationshipType: "knows",
			TargetEntityID:   "ent-c",
			ValidFrom:        now,
			Metadata:         map[string]interface{}{"confidence": 0.9},
		},
		{
			ID:               "rel-cd",
			SourceEntityID:   "ent-c",
			RelationshipType: "knows",
			TargetEntityID:   "ent-d",
			ValidFrom:        now,
			Metadata:         map[string]interface{}{"confidence": 0.9},
		},
	}

	mock := &mockStorage{
		entities:      entities,
		relationships: relationships,
	}

	ctx := context.Background()
	query := MemoryQuery{
		TextQuery:     "A",
		MaxHops:       2, // Should reach B and C, but not D
		TopK:          5,
		MinConfidence: 0.5,
	}

	result, err := RetrieveMemory(ctx, mock, query)
	if err != nil {
		t.Fatalf("RetrieveMemory failed: %v", err)
	}

	// Check that we don't traverse beyond MaxHops
	foundD := false
	for _, e := range result.Entities {
		if e.ID == "ent-d" {
			foundD = true
		}
	}

	if foundD {
		t.Errorf("Should not traverse beyond MaxHops=2 to reach entity D")
	}
}
