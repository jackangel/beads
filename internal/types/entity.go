// Package types defines core data structures for the bd issue tracker.
package types

import (
	"time"
)

// Entity represents a knowledge graph entity extracted from episodes.
// Entities are evolving summaries of real-world things (people, products, concepts)
// discovered in issue data, commits, comments, and other sources.
type Entity struct {
	// ===== Core Identification =====
	ID         string `json:"id"`          // Entity identifier (e.g., "bd-a3f8e9")
	EntityType string `json:"entity_type"` // Flexible type (e.g., "person", "product", "concept")
	Name       string `json:"name"`        // Display name

	// ===== Content =====
	Summary string `json:"summary"` // Evolving summary text

	// ===== Custom Metadata =====
	// Metadata holds arbitrary JSON data for flexible extension
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// ===== Timestamps =====
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// ===== Attribution =====
	CreatedBy string `json:"created_by,omitempty"` // Actor who created this entity
	UpdatedBy string `json:"updated_by,omitempty"` // Actor who last updated this entity
}
