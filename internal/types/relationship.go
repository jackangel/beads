// Package types defines core data structures for the bd issue tracker.
package types

import (
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
