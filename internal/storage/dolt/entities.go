package dolt

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
)

// CreateEntity creates a new entity in the entities table.
// The entity ID must be unique and the entity_type must be specified.
func (s *DoltStore) CreateEntity(ctx context.Context, entity *types.Entity) error {
	if entity == nil {
		return fmt.Errorf("entity must not be nil")
	}
	if entity.ID == "" {
		return fmt.Errorf("entity ID must be specified")
	}
	if entity.EntityType == "" {
		return fmt.Errorf("entity type must be specified")
	}
	if entity.Name == "" {
		return fmt.Errorf("entity name must be specified")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Marshal metadata to JSON
	var metadataJSON []byte
	if entity.Metadata != nil && len(entity.Metadata) > 0 {
		metadataJSON, err = json.Marshal(entity.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	// Set timestamps if not already set
	now := time.Now().UTC()
	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = now
	}
	if entity.UpdatedAt.IsZero() {
		entity.UpdatedAt = now
	}

	// Insert entity
	query := `INSERT INTO entities (id, entity_type, name, summary, metadata, created_at, updated_at, created_by, updated_by)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, query,
		entity.ID, entity.EntityType, entity.Name, entity.Summary,
		string(metadataJSON),
		entity.CreatedAt, entity.UpdatedAt,
		entity.CreatedBy, entity.UpdatedBy)
	if err != nil {
		return fmt.Errorf("failed to insert entity: %w", err)
	}

	// Dolt versioning: stage the table and commit
	if _, err := tx.ExecContext(ctx, "CALL DOLT_ADD(?)", "entities"); err != nil {
		return fmt.Errorf("dolt add entities: %w", err)
	}

	commitMsg := fmt.Sprintf("bd: create entity %s", entity.ID)
	if _, err := tx.ExecContext(ctx, "CALL DOLT_COMMIT('-m', ?, '--author', ?)",
		commitMsg, s.commitAuthorString()); err != nil && !isDoltNothingToCommit(err) {
		return fmt.Errorf("dolt commit: %w", err)
	}

	return tx.Commit()
}

// GetEntity retrieves an entity by its ID.
// Returns storage.ErrNotFound if the entity does not exist.
func (s *DoltStore) GetEntity(ctx context.Context, id string) (*types.Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, entity_type, name, summary, metadata, created_at, updated_at, created_by, updated_by
	          FROM entities WHERE id = ?`

	var entity types.Entity
	var metadataJSON sql.NullString
	var summary sql.NullString
	var createdBy sql.NullString
	var updatedBy sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&entity.ID, &entity.EntityType, &entity.Name, &summary,
		&metadataJSON,
		&entity.CreatedAt, &entity.UpdatedAt,
		&createdBy, &updatedBy)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: entity %s", storage.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Map nullable fields
	if summary.Valid {
		entity.Summary = summary.String
	}
	if createdBy.Valid {
		entity.CreatedBy = createdBy.String
	}
	if updatedBy.Valid {
		entity.UpdatedBy = updatedBy.String
	}

	// Unmarshal metadata
	if metadataJSON.Valid && metadataJSON.String != "" && metadataJSON.String != "{}" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &entity.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &entity, nil
}

// UpdateEntity updates an existing entity's fields.
// Only non-zero fields in the entity parameter are updated.
func (s *DoltStore) UpdateEntity(ctx context.Context, entity *types.Entity) error {
	if entity == nil {
		return fmt.Errorf("entity must not be nil")
	}
	if entity.ID == "" {
		return fmt.Errorf("entity ID must be specified")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Read inside transaction to check existence
	var exists int
	err = tx.QueryRowContext(ctx, "SELECT 1 FROM entities WHERE id = ?", entity.ID).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("%w: entity %s", storage.ErrNotFound, entity.ID)
	}
	if err != nil {
		return fmt.Errorf("failed to check entity existence: %w", err)
	}

	// Build dynamic UPDATE query based on non-zero fields
	setClauses := []string{"updated_at = ?"}
	args := []interface{}{time.Now().UTC()}

	if entity.EntityType != "" {
		setClauses = append(setClauses, "entity_type = ?")
		args = append(args, entity.EntityType)
	}
	if entity.Name != "" {
		setClauses = append(setClauses, "name = ?")
		args = append(args, entity.Name)
	}
	if entity.Summary != "" {
		setClauses = append(setClauses, "summary = ?")
		args = append(args, entity.Summary)
	}
	if entity.Metadata != nil {
		metadataJSON, err := json.Marshal(entity.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		setClauses = append(setClauses, "metadata = ?")
		args = append(args, string(metadataJSON))
	}
	if entity.UpdatedBy != "" {
		setClauses = append(setClauses, "updated_by = ?")
		args = append(args, entity.UpdatedBy)
	}

	// If only updated_at would be set, there's nothing to update
	if len(setClauses) == 1 {
		return fmt.Errorf("no fields to update")
	}

	args = append(args, entity.ID)

	query := fmt.Sprintf("UPDATE entities SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	// Dolt versioning: stage the table and commit
	if _, err := tx.ExecContext(ctx, "CALL DOLT_ADD(?)", "entities"); err != nil {
		return fmt.Errorf("dolt add entities: %w", err)
	}

	commitMsg := fmt.Sprintf("bd: update entity %s", entity.ID)
	if _, err := tx.ExecContext(ctx, "CALL DOLT_COMMIT('-m', ?, '--author', ?)",
		commitMsg, s.commitAuthorString()); err != nil && !isDoltNothingToCommit(err) {
		return fmt.Errorf("dolt commit: %w", err)
	}

	return tx.Commit()
}

// DeleteEntity removes an entity from the system (hard delete).
// Returns storage.ErrNotFound if the entity does not exist.
func (s *DoltStore) DeleteEntity(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("entity ID must be specified")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check if entity exists before deletion
	var exists int
	err = tx.QueryRowContext(ctx, "SELECT 1 FROM entities WHERE id = ?", id).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("%w: entity %s", storage.ErrNotFound, id)
	}
	if err != nil {
		return fmt.Errorf("failed to check entity existence: %w", err)
	}

	// Delete the entity
	result, err := tx.ExecContext(ctx, "DELETE FROM entities WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: entity %s", storage.ErrNotFound, id)
	}

	// Dolt versioning: stage the table and commit
	if _, err := tx.ExecContext(ctx, "CALL DOLT_ADD(?)", "entities"); err != nil {
		return fmt.Errorf("dolt add entities: %w", err)
	}

	commitMsg := fmt.Sprintf("bd: delete entity %s", id)
	if _, err := tx.ExecContext(ctx, "CALL DOLT_COMMIT('-m', ?, '--author', ?)",
		commitMsg, s.commitAuthorString()); err != nil && !isDoltNothingToCommit(err) {
		return fmt.Errorf("dolt commit: %w", err)
	}

	return tx.Commit()
}

// SearchEntities finds entities matching the provided filters.
// Returns an empty slice if no entities match the criteria.
func (s *DoltStore) SearchEntities(ctx context.Context, filters storage.EntityFilters) ([]*types.Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build WHERE clause from filters
	whereClauses := []string{}
	args := []interface{}{}

	if filters.EntityType != "" {
		whereClauses = append(whereClauses, "entity_type = ?")
		args = append(args, filters.EntityType)
	}

	if filters.Name != "" {
		whereClauses = append(whereClauses, "name LIKE ?")
		args = append(args, "%"+filters.Name+"%")
	}

	if filters.CreatedBy != "" {
		whereClauses = append(whereClauses, "created_by = ?")
		args = append(args, filters.CreatedBy)
	}

	// Build base query
	query := "SELECT id, entity_type, name, summary, metadata, created_at, updated_at, created_by, updated_by FROM entities"
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Add ordering (by created_at descending by default)
	query += " ORDER BY created_at DESC"

	// Add pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.Limit)
	}
	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search entities: %w", err)
	}
	defer rows.Close()

	entities := []*types.Entity{}
	for rows.Next() {
		var entity types.Entity
		var metadataJSON sql.NullString
		var summary sql.NullString
		var createdBy sql.NullString
		var updatedBy sql.NullString

		err := rows.Scan(
			&entity.ID, &entity.EntityType, &entity.Name, &summary,
			&metadataJSON,
			&entity.CreatedAt, &entity.UpdatedAt,
			&createdBy, &updatedBy)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}

		// Map nullable fields
		if summary.Valid {
			entity.Summary = summary.String
		}
		if createdBy.Valid {
			entity.CreatedBy = createdBy.String
		}
		if updatedBy.Valid {
			entity.UpdatedBy = updatedBy.String
		}

		// Unmarshal metadata
		if metadataJSON.Valid && metadataJSON.String != "" && metadataJSON.String != "{}" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &entity.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		entities = append(entities, &entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate entities: %w", err)
	}

	// Post-filtering for metadata (JSON queries in Dolt can be tricky, so filter in-memory)
	if len(filters.Metadata) > 0 {
		filtered := []*types.Entity{}
		for _, entity := range entities {
			if matchesMetadata(entity.Metadata, filters.Metadata) {
				filtered = append(filtered, entity)
			}
		}
		entities = filtered
	}

	return entities, nil
}

// matchesMetadata checks if an entity's metadata contains all required key-value pairs.
func matchesMetadata(entityMetadata, filterMetadata map[string]interface{}) bool {
	if entityMetadata == nil {
		return len(filterMetadata) == 0
	}
	for key, filterValue := range filterMetadata {
		entityValue, exists := entityMetadata[key]
		if !exists {
			return false
		}
		// Simple equality check (could be enhanced for deep comparison if needed)
		if fmt.Sprintf("%v", entityValue) != fmt.Sprintf("%v", filterValue) {
			return false
		}
	}
	return true
}
