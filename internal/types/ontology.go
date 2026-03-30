// Package types defines core data structures for the bd issue tracker.
package types

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

// EntityTypeSchema defines a custom entity type with JSON schema validation.
// Entity types define structured data shapes for knowledge graph entities
// (e.g., "person", "product", "concept") with validation rules.
type EntityTypeSchema struct {
	// ===== Core Identification =====
	TypeName string `json:"type_name"` // Entity type name (e.g., "person", "product")

	// ===== Schema Definition =====
	// SchemaJSON is a JSON Schema (draft-07 or later) that validates entity metadata.
	// Example: {"type": "object", "properties": {"email": {"type": "string", "format": "email"}}}
	SchemaJSON string `json:"schema_json"`

	// ===== Documentation =====
	Description string `json:"description"` // Human-readable description of this entity type

	// ===== Timestamps =====
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// ===== Attribution =====
	CreatedBy string `json:"created_by,omitempty"` // Actor who defined this schema
}

// RelationshipTypeSchema defines a custom relationship type with JSON schema validation.
// Relationship types define structured data shapes for knowledge graph edges
// (e.g., "uses", "implements", "replaces") with validation rules.
type RelationshipTypeSchema struct {
	// ===== Core Identification =====
	TypeName string `json:"type_name"` // Relationship type name (e.g., "uses", "replaces")

	// ===== Schema Definition =====
	// SchemaJSON is a JSON Schema (draft-07 or later) that validates relationship metadata.
	// Example: {"type": "object", "properties": {"confidence": {"type": "number", "minimum": 0, "maximum": 1}}}
	SchemaJSON string `json:"schema_json"`

	// ===== Documentation =====
	Description string `json:"description"` // Human-readable description of this relationship type

	// ===== Timestamps =====
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// ===== Attribution =====
	CreatedBy string `json:"created_by,omitempty"` // Actor who defined this schema
}

// ValidateEntityAgainstSchema validates an entity's metadata against an entity type schema.
// Returns an error if:
//   - The schema JSON is malformed
//   - The entity metadata does not conform to the schema
//   - The entity type does not match the schema type
//
// This function uses JSON Schema validation (draft-07) to ensure metadata conforms
// to the defined structure and constraints.
func ValidateEntityAgainstSchema(entity *Entity, schema *EntityTypeSchema) error {
	if entity == nil {
		return fmt.Errorf("entity cannot be nil")
	}
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	// Verify entity type matches schema type
	if entity.EntityType != schema.TypeName {
		return fmt.Errorf("entity type mismatch: entity has type %q, schema defines type %q",
			entity.EntityType, schema.TypeName)
	}

	// Parse the JSON schema
	schemaLoader := gojsonschema.NewStringLoader(schema.SchemaJSON)

	// Convert entity metadata to JSON for validation
	metadataBytes, err := json.Marshal(entity.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal entity metadata: %w", err)
	}
	documentLoader := gojsonschema.NewBytesLoader(metadataBytes)

	// Validate metadata against schema
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	// Check validation result
	if !result.Valid() {
		// Collect all validation errors into a single message
		var errMessages []string
		for _, desc := range result.Errors() {
			errMessages = append(errMessages, desc.String())
		}
		return fmt.Errorf("entity metadata validation failed: %v", errMessages)
	}

	return nil
}

// ValidateRelationshipAgainstSchema validates a relationship's metadata against a relationship type schema.
// Returns an error if:
//   - The schema JSON is malformed
//   - The relationship metadata does not conform to the schema
//   - The relationship type does not match the schema type
//
// This function uses JSON Schema validation (draft-07) to ensure metadata conforms
// to the defined structure and constraints.
func ValidateRelationshipAgainstSchema(rel *Relationship, schema *RelationshipTypeSchema) error {
	if rel == nil {
		return fmt.Errorf("relationship cannot be nil")
	}
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	// Verify relationship type matches schema type
	if rel.RelationshipType != schema.TypeName {
		return fmt.Errorf("relationship type mismatch: relationship has type %q, schema defines type %q",
			rel.RelationshipType, schema.TypeName)
	}

	// Parse the JSON schema
	schemaLoader := gojsonschema.NewStringLoader(schema.SchemaJSON)

	// Convert relationship metadata to JSON for validation
	metadataBytes, err := json.Marshal(rel.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal relationship metadata: %w", err)
	}
	documentLoader := gojsonschema.NewBytesLoader(metadataBytes)

	// Validate metadata against schema
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	// Check validation result
	if !result.Valid() {
		// Collect all validation errors into a single message
		var errMessages []string
		for _, desc := range result.Errors() {
			errMessages = append(errMessages, desc.String())
		}
		return fmt.Errorf("relationship metadata validation failed: %v", errMessages)
	}

	return nil
}

// Validate checks if the EntityTypeSchema has valid field values.
// Returns an error if:
//   - TypeName is empty
//   - SchemaJSON is empty or malformed
func (s *EntityTypeSchema) Validate() error {
	if s.TypeName == "" {
		return fmt.Errorf("type_name is required")
	}
	if s.SchemaJSON == "" {
		return fmt.Errorf("schema_json is required")
	}

	// Verify SchemaJSON is valid JSON and can be parsed as a schema
	schemaLoader := gojsonschema.NewStringLoader(s.SchemaJSON)
	_, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}

	return nil
}

// Validate checks if the RelationshipTypeSchema has valid field values.
// Returns an error if:
//   - TypeName is empty
//   - SchemaJSON is empty or malformed
func (s *RelationshipTypeSchema) Validate() error {
	if s.TypeName == "" {
		return fmt.Errorf("type_name is required")
	}
	if s.SchemaJSON == "" {
		return fmt.Errorf("schema_json is required")
	}

	// Verify SchemaJSON is valid JSON and can be parsed as a schema
	schemaLoader := gojsonschema.NewStringLoader(s.SchemaJSON)
	_, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}

	return nil
}
