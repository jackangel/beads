// Package types defines core data structures for the bd issue tracker.
package types

import (
	"fmt"
	"time"
)

// Relationship represents a typed edge between entities in the knowledge graph.
// Relationships support temporal validity to track how connections change over time.
type Relationship struct {
	// ===== Core Identification =====
	ID               string `json:"id"`                 // Relationship identifier
	SourceEntityID   string `json:"source_entity_id"`   // From entity
	RelationshipType string `json:"relationship_type"`  // Flexible type (e.g., "uses", "implements", "replaces")
	TargetEntityID   string `json:"target_entity_id"`   // To entity

	// ===== Temporal Validity =====
	ValidFrom  time.Time  `json:"valid_from"`            // When this relationship starts
	ValidUntil *time.Time `json:"valid_until,omitempty"` // When this relationship ends (NULL = still valid)

	// ===== Custom Metadata =====
	// Metadata holds arbitrary JSON data for flexible extension
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// ===== Confidence Score =====
	// Confidence represents the certainty of this relationship (0.0-1.0).
	// AI-extracted relationships typically start with lower confidence.
	// Human-curated relationships default to 1.0.
	// NULL (nil) is treated as 1.0 for filtering.
	Confidence *float64 `json:"confidence,omitempty"`

	// ===== Timestamps =====
	CreatedAt time.Time `json:"created_at"`

	// ===== Attribution =====
	CreatedBy string `json:"created_by,omitempty"` // Actor who created this relationship
}

// IsValidAt checks if this relationship is valid at the given time.
// Returns true if the time falls within [ValidFrom, ValidUntil).
// If ValidUntil is nil, the relationship is considered valid indefinitely.
func (r *Relationship) IsValidAt(t time.Time) bool {
	// Check if time is before the relationship starts
	if t.Before(r.ValidFrom) {
		return false
	}

	// If ValidUntil is nil, relationship is valid indefinitely
	if r.ValidUntil == nil {
		return true
	}

	// Check if time is before the relationship ends
	return t.Before(*r.ValidUntil)
}

// ValidateConfidence checks if the confidence score is valid.
// Confidence must be between 0.0 and 1.0 (inclusive) if set.
// Returns nil if confidence is nil or valid, error otherwise.
func (r *Relationship) ValidateConfidence() error {
	if r.Confidence == nil {
		return nil
	}
	if *r.Confidence < 0.0 || *r.Confidence > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0, got %.2f", *r.Confidence)
	}
	return nil
}
