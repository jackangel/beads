// Package types defines core data structures for the bd issue tracker.
package types

import (
	"time"
)

// Episode represents an immutable snapshot of ingested data.
// Episodes are the raw material from which entities and relationships are extracted.
// Once created, episodes are never modified (no UpdatedAt or UpdatedBy).
type Episode struct {
	// ===== Core Identification =====
	ID string `json:"id"` // Episode identifier

	// ===== Content =====
	Timestamp time.Time `json:"timestamp"` // When data was ingested
	Source    string    `json:"source"`    // Where data came from (e.g., "github", "jira", "manual")
	RawData   []byte    `json:"raw_data"`  // BLOB of raw input

	// ===== Extraction Results =====
	EntitiesExtracted []string `json:"entities_extracted,omitempty"` // List of entity IDs extracted from this episode

	// ===== Custom Metadata =====
	// Metadata holds arbitrary JSON data for flexible extension
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// ===== Timestamps =====
	CreatedAt time.Time `json:"created_at"`
}
