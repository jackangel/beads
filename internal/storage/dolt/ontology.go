package dolt

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
)

// RegisterEntityType registers a new entity type schema in the system.
// Uses REPLACE INTO to allow updating existing schemas.
func (s *DoltStore) RegisterEntityType(ctx context.Context, schema *types.EntityTypeSchema) error {
	if schema == nil {
		return fmt.Errorf("schema must not be nil")
	}
	if err := schema.Validate(); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Set timestamps if not already set
	now := time.Now().UTC()
	if schema.CreatedAt.IsZero() {
		schema.CreatedAt = now
	}
	if schema.UpdatedAt.IsZero() {
		schema.UpdatedAt = now
	}

	// REPLACE INTO allows updating existing schemas
	query := `REPLACE INTO entity_types (type_name, schema_json, description, created_at, updated_at, created_by)
	          VALUES (?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, query,
		schema.TypeName, schema.SchemaJSON, schema.Description,
		schema.CreatedAt, schema.UpdatedAt, schema.CreatedBy)
	if err != nil {
		return fmt.Errorf("failed to register entity type: %w", err)
	}

	// Dolt versioning: stage the table and commit
	if _, err := tx.ExecContext(ctx, "CALL DOLT_ADD(?)", "entity_types"); err != nil {
		return fmt.Errorf("dolt add entity_types: %w", err)
	}

	commitMsg := fmt.Sprintf("bd: register entity type %s", schema.TypeName)
	if _, err := tx.ExecContext(ctx, "CALL DOLT_COMMIT('-m', ?, '--author', ?)",
		commitMsg, s.commitAuthorString()); err != nil && !isDoltNothingToCommit(err) {
		return fmt.Errorf("dolt commit: %w", err)
	}

	return tx.Commit()
}

// RegisterRelationshipType registers a new relationship type schema in the system.
// Uses REPLACE INTO to allow updating existing schemas.
func (s *DoltStore) RegisterRelationshipType(ctx context.Context, schema *types.RelationshipTypeSchema) error {
	if schema == nil {
		return fmt.Errorf("schema must not be nil")
	}
	if err := schema.Validate(); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Set timestamps if not already set
	now := time.Now().UTC()
	if schema.CreatedAt.IsZero() {
		schema.CreatedAt = now
	}
	if schema.UpdatedAt.IsZero() {
		schema.UpdatedAt = now
	}

	// REPLACE INTO allows updating existing schemas
	query := `REPLACE INTO relationship_types (type_name, schema_json, description, created_at, updated_at, created_by)
	          VALUES (?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, query,
		schema.TypeName, schema.SchemaJSON, schema.Description,
		schema.CreatedAt, schema.UpdatedAt, schema.CreatedBy)
	if err != nil {
		return fmt.Errorf("failed to register relationship type: %w", err)
	}

	// Dolt versioning: stage the table and commit
	if _, err := tx.ExecContext(ctx, "CALL DOLT_ADD(?)", "relationship_types"); err != nil {
		return fmt.Errorf("dolt add relationship_types: %w", err)
	}

	commitMsg := fmt.Sprintf("bd: register relationship type %s", schema.TypeName)
	if _, err := tx.ExecContext(ctx, "CALL DOLT_COMMIT('-m', ?, '--author', ?)",
		commitMsg, s.commitAuthorString()); err != nil && !isDoltNothingToCommit(err) {
		return fmt.Errorf("dolt commit: %w", err)
	}

	return tx.Commit()
}

// GetEntityTypes retrieves all registered entity type schemas.
func (s *DoltStore) GetEntityTypes(ctx context.Context) ([]*types.EntityTypeSchema, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT type_name, schema_json, description, created_at, updated_at, created_by
	          FROM entity_types
	          ORDER BY type_name`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity types: %w", err)
	}
	defer rows.Close()

	var schemas []*types.EntityTypeSchema
	for rows.Next() {
		var schema types.EntityTypeSchema
		var description sql.NullString
		var createdBy sql.NullString

		err := rows.Scan(&schema.TypeName, &schema.SchemaJSON, &description,
			&schema.CreatedAt, &schema.UpdatedAt, &createdBy)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity type: %w", err)
		}

		if description.Valid {
			schema.Description = description.String
		}
		if createdBy.Valid {
			schema.CreatedBy = createdBy.String
		}

		schemas = append(schemas, &schema)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entity types: %w", err)
	}

	return schemas, nil
}

// GetRelationshipTypes retrieves all registered relationship type schemas.
func (s *DoltStore) GetRelationshipTypes(ctx context.Context) ([]*types.RelationshipTypeSchema, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT type_name, schema_json, description, created_at, updated_at, created_by
	          FROM relationship_types
	          ORDER BY type_name`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query relationship types: %w", err)
	}
	defer rows.Close()

	var schemas []*types.RelationshipTypeSchema
	for rows.Next() {
		var schema types.RelationshipTypeSchema
		var description sql.NullString
		var createdBy sql.NullString

		err := rows.Scan(&schema.TypeName, &schema.SchemaJSON, &description,
			&schema.CreatedAt, &schema.UpdatedAt, &createdBy)
		if err != nil {
			return nil, fmt.Errorf("failed to scan relationship type: %w", err)
		}

		if description.Valid {
			schema.Description = description.String
		}
		if createdBy.Valid {
			schema.CreatedBy = createdBy.String
		}

		schemas = append(schemas, &schema)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating relationship types: %w", err)
	}

	return schemas, nil
}

// GetEntityTypeSchema retrieves the schema for a specific entity type by name.
// Returns storage.ErrNotFound if the type schema does not exist.
func (s *DoltStore) GetEntityTypeSchema(ctx context.Context, typeName string) (*types.EntityTypeSchema, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT type_name, schema_json, description, created_at, updated_at, created_by
	          FROM entity_types WHERE type_name = ?`

	var schema types.EntityTypeSchema
	var description sql.NullString
	var createdBy sql.NullString

	err := s.db.QueryRowContext(ctx, query, typeName).Scan(
		&schema.TypeName, &schema.SchemaJSON, &description,
		&schema.CreatedAt, &schema.UpdatedAt, &createdBy)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: entity type schema %s", storage.ErrNotFound, typeName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entity type schema: %w", err)
	}

	if description.Valid {
		schema.Description = description.String
	}
	if createdBy.Valid {
		schema.CreatedBy = createdBy.String
	}

	return &schema, nil
}

// GetRelationshipTypeSchema retrieves the schema for a specific relationship type by name.
// Returns storage.ErrNotFound if the type schema does not exist.
func (s *DoltStore) GetRelationshipTypeSchema(ctx context.Context, typeName string) (*types.RelationshipTypeSchema, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT type_name, schema_json, description, created_at, updated_at, created_by
	          FROM relationship_types WHERE type_name = ?`

	var schema types.RelationshipTypeSchema
	var description sql.NullString
	var createdBy sql.NullString

	err := s.db.QueryRowContext(ctx, query, typeName).Scan(
		&schema.TypeName, &schema.SchemaJSON, &description,
		&schema.CreatedAt, &schema.UpdatedAt, &createdBy)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: relationship type schema %s", storage.ErrNotFound, typeName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get relationship type schema: %w", err)
	}

	if description.Valid {
		schema.Description = description.String
	}
	if createdBy.Valid {
		schema.CreatedBy = createdBy.String
	}

	return &schema, nil
}

// ValidateEntityAgainstType validates an entity against a registered type schema.
// Returns an error if the entity does not conform to the schema's requirements.
func (s *DoltStore) ValidateEntityAgainstType(ctx context.Context, entity *types.Entity, typeName string) error {
	if entity == nil {
		return fmt.Errorf("entity must not be nil")
	}

	// Retrieve the schema for the specified type
	schema, err := s.GetEntityTypeSchema(ctx, typeName)
	if err != nil {
		return fmt.Errorf("failed to get entity type schema: %w", err)
	}

	// Delegate validation to types package
	if err := types.ValidateEntityAgainstSchema(entity, schema); err != nil {
		return fmt.Errorf("entity validation failed: %w", err)
	}

	return nil
}

// ValidateRelationshipAgainstType validates a relationship against a registered type schema.
// Returns an error if the relationship does not conform to the schema's requirements.
func (s *DoltStore) ValidateRelationshipAgainstType(ctx context.Context, rel *types.Relationship, typeName string) error {
	if rel == nil {
		return fmt.Errorf("relationship must not be nil")
	}

	// Retrieve the schema for the specified type
	schema, err := s.GetRelationshipTypeSchema(ctx, typeName)
	if err != nil {
		return fmt.Errorf("failed to get relationship type schema: %w", err)
	}

	// Delegate validation to types package
	if err := types.ValidateRelationshipAgainstSchema(rel, schema); err != nil {
		return fmt.Errorf("relationship validation failed: %w", err)
	}

	return nil
}
