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

// CreateRelationship creates a new relationship in the relationships table.
// The relationship ID must be unique and all required fields must be specified.
// ValidFrom is required; ValidUntil is optional (nil means still valid).
func (s *DoltStore) CreateRelationship(ctx context.Context, rel *types.Relationship) error {
	if rel == nil {
		return fmt.Errorf("relationship must not be nil")
	}
	if rel.ID == "" {
		return fmt.Errorf("relationship ID must be specified")
	}
	if rel.SourceEntityID == "" {
		return fmt.Errorf("source entity ID must be specified")
	}
	if rel.TargetEntityID == "" {
		return fmt.Errorf("target entity ID must be specified")
	}
	if rel.RelationshipType == "" {
		return fmt.Errorf("relationship type must be specified")
	}
	if rel.ValidFrom.IsZero() {
		return fmt.Errorf("valid_from must be specified")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Marshal metadata to JSON
	var metadataJSON []byte
	if rel.Metadata != nil && len(rel.Metadata) > 0 {
		metadataJSON, err = json.Marshal(rel.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	// Set timestamp if not already set
	now := time.Now().UTC()
	if rel.CreatedAt.IsZero() {
		rel.CreatedAt = now
	}

	// Validate confidence if provided
	if err := rel.ValidateConfidence(); err != nil {
		return fmt.Errorf("invalid confidence: %w", err)
	}

	// Insert relationship
	query := `INSERT INTO relationships (id, source_entity_id, relationship_type, target_entity_id, valid_from, valid_until, metadata, created_at, created_by, confidence)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, query,
		rel.ID, rel.SourceEntityID, rel.RelationshipType, rel.TargetEntityID,
		rel.ValidFrom, rel.ValidUntil,
		string(metadataJSON),
		rel.CreatedAt, rel.CreatedBy, rel.Confidence)
	if err != nil {
		return fmt.Errorf("failed to insert relationship: %w", err)
	}

	// Dolt versioning: stage the table and commit
	if _, err := tx.ExecContext(ctx, "CALL DOLT_ADD(?)", "relationships"); err != nil {
		return fmt.Errorf("dolt add relationships: %w", err)
	}

	commitMsg := fmt.Sprintf("bd: create relationship %s", rel.ID)
	if _, err := tx.ExecContext(ctx, "CALL DOLT_COMMIT('-m', ?, '--author', ?)",
		commitMsg, s.commitAuthorString()); err != nil && !isDoltNothingToCommit(err) {
		return fmt.Errorf("dolt commit: %w", err)
	}

	return tx.Commit()
}

// GetRelationship retrieves a relationship by its ID.
// Returns storage.ErrNotFound if the relationship does not exist.
func (s *DoltStore) GetRelationship(ctx context.Context, id string) (*types.Relationship, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, source_entity_id, relationship_type, target_entity_id, valid_from, valid_until, metadata, created_at, created_by, confidence
	          FROM relationships WHERE id = ?`

	var rel types.Relationship
	var metadataJSON sql.NullString
	var validUntil sql.NullTime
	var createdBy sql.NullString
	var confidence sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&rel.ID, &rel.SourceEntityID, &rel.RelationshipType, &rel.TargetEntityID,
		&rel.ValidFrom, &validUntil,
		&metadataJSON,
		&rel.CreatedAt, &createdBy, &confidence)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: relationship %s", storage.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get relationship: %w", err)
	}

	// Map nullable fields
	if validUntil.Valid {
		rel.ValidUntil = &validUntil.Time
	}
	if createdBy.Valid {
		rel.CreatedBy = createdBy.String
	}
	if confidence.Valid {
		rel.Confidence = &confidence.Float64
	}

	// Unmarshal metadata
	if metadataJSON.Valid && metadataJSON.String != "" && metadataJSON.String != "{}" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &rel.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &rel, nil
}

// UpdateRelationship updates an existing relationship's fields.
// Only non-zero/non-nil fields in the relationship parameter are updated.
// For temporal relationships, consider whether to update in place or create a new row with a new validity window.
func (s *DoltStore) UpdateRelationship(ctx context.Context, rel *types.Relationship) error {
	if rel == nil {
		return fmt.Errorf("relationship must not be nil")
	}
	if rel.ID == "" {
		return fmt.Errorf("relationship ID must be specified")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Read inside transaction to check existence
	var exists int
	err = tx.QueryRowContext(ctx, "SELECT 1 FROM relationships WHERE id = ?", rel.ID).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("%w: relationship %s", storage.ErrNotFound, rel.ID)
	}
	if err != nil {
		return fmt.Errorf("failed to check relationship existence: %w", err)
	}

	// Build dynamic UPDATE query based on non-zero fields
	setClauses := []string{}
	args := []interface{}{}

	if rel.SourceEntityID != "" {
		setClauses = append(setClauses, "source_entity_id = ?")
		args = append(args, rel.SourceEntityID)
	}
	if rel.RelationshipType != "" {
		setClauses = append(setClauses, "relationship_type = ?")
		args = append(args, rel.RelationshipType)
	}
	if rel.TargetEntityID != "" {
		setClauses = append(setClauses, "target_entity_id = ?")
		args = append(args, rel.TargetEntityID)
	}
	if !rel.ValidFrom.IsZero() {
		setClauses = append(setClauses, "valid_from = ?")
		args = append(args, rel.ValidFrom)
	}
	// Allow explicit update of ValidUntil (to close temporal window or extend it)
	// Note: This updates ValidUntil even if it's nil (sets to NULL)
	if rel.ValidUntil != nil || len(setClauses) > 0 {
		// Only update valid_until if other fields are being updated or if explicitly provided
		// This check prevents accidental nil updates when only ValidUntil is provided
		if rel.ValidUntil != nil {
			setClauses = append(setClauses, "valid_until = ?")
			args = append(args, rel.ValidUntil)
		}
	}
	if rel.Metadata != nil {
		metadataJSON, err := json.Marshal(rel.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		setClauses = append(setClauses, "metadata = ?")
		args = append(args, string(metadataJSON))
	}
	if rel.Confidence != nil {
		// Validate confidence before updating
		if err := rel.ValidateConfidence(); err != nil {
			return fmt.Errorf("invalid confidence: %w", err)
		}
		setClauses = append(setClauses, "confidence = ?")
		args = append(args, rel.Confidence)
	}

	// If no fields to update, return error
	if len(setClauses) == 0 {
		return fmt.Errorf("no fields to update")
	}

	args = append(args, rel.ID)

	query := fmt.Sprintf("UPDATE relationships SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to update relationship: %w", err)
	}

	// Dolt versioning: stage the table and commit
	if _, err := tx.ExecContext(ctx, "CALL DOLT_ADD(?)", "relationships"); err != nil {
		return fmt.Errorf("dolt add relationships: %w", err)
	}

	commitMsg := fmt.Sprintf("bd: update relationship %s", rel.ID)
	if _, err := tx.ExecContext(ctx, "CALL DOLT_COMMIT('-m', ?, '--author', ?)",
		commitMsg, s.commitAuthorString()); err != nil && !isDoltNothingToCommit(err) {
		return fmt.Errorf("dolt commit: %w", err)
	}

	return tx.Commit()
}

// DeleteRelationship removes a relationship from the system (hard delete).
// Returns storage.ErrNotFound if the relationship does not exist.
// For temporal validity tracking, prefer using UpdateRelationship to set ValidUntil instead of hard deletion.
func (s *DoltStore) DeleteRelationship(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("relationship ID must be specified")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check if relationship exists before deletion
	var exists int
	err = tx.QueryRowContext(ctx, "SELECT 1 FROM relationships WHERE id = ?", id).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("%w: relationship %s", storage.ErrNotFound, id)
	}
	if err != nil {
		return fmt.Errorf("failed to check relationship existence: %w", err)
	}

	// Delete the relationship
	result, err := tx.ExecContext(ctx, "DELETE FROM relationships WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete relationship: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: relationship %s", storage.ErrNotFound, id)
	}

	// Dolt versioning: stage the table and commit
	if _, err := tx.ExecContext(ctx, "CALL DOLT_ADD(?)", "relationships"); err != nil {
		return fmt.Errorf("dolt add relationships: %w", err)
	}

	commitMsg := fmt.Sprintf("bd: delete relationship %s", id)
	if _, err := tx.ExecContext(ctx, "CALL DOLT_COMMIT('-m', ?, '--author', ?)",
		commitMsg, s.commitAuthorString()); err != nil && !isDoltNothingToCommit(err) {
		return fmt.Errorf("dolt commit: %w", err)
	}

	return tx.Commit()
}

// SearchRelationships finds relationships matching the provided filters.
// Returns an empty slice if no relationships match the criteria.
// Supports temporal filtering via ValidAt or ValidAtStart/ValidAtEnd.
func (s *DoltStore) SearchRelationships(ctx context.Context, filters storage.RelationshipFilters) ([]*types.Relationship, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build WHERE clause from filters
	whereClauses := []string{}
	args := []interface{}{}

	if filters.SourceEntityID != "" {
		whereClauses = append(whereClauses, "source_entity_id = ?")
		args = append(args, filters.SourceEntityID)
	}

	if filters.TargetEntityID != "" {
		whereClauses = append(whereClauses, "target_entity_id = ?")
		args = append(args, filters.TargetEntityID)
	}

	if filters.RelationshipType != "" {
		whereClauses = append(whereClauses, "relationship_type = ?")
		args = append(args, filters.RelationshipType)
	}

	// Temporal filtering: ValidAt takes precedence over ValidAtStart/ValidAtEnd
	if filters.ValidAt != nil {
		// Query for relationships valid at a specific point in time
		// Logic: valid_from <= T AND (valid_until IS NULL OR valid_until > T)
		whereClauses = append(whereClauses, "valid_from <= ?")
		args = append(args, *filters.ValidAt)
		whereClauses = append(whereClauses, "(valid_until IS NULL OR valid_until > ?)")
		args = append(args, *filters.ValidAt)
	} else if filters.ValidAtStart != nil && filters.ValidAtEnd != nil {
		// Query for relationships valid during at least part of a time range
		// Logic: valid_from < range_end AND (valid_until IS NULL OR valid_until > range_start)
		whereClauses = append(whereClauses, "valid_from < ?")
		args = append(args, *filters.ValidAtEnd)
		whereClauses = append(whereClauses, "(valid_until IS NULL OR valid_until > ?)")
		args = append(args, *filters.ValidAtStart)
	}

	// Confidence filtering: treat NULL as 1.0
	if filters.MinConfidence != nil {
		whereClauses = append(whereClauses, "COALESCE(confidence, 1.0) >= ?")
		args = append(args, *filters.MinConfidence)
	}
	if filters.MaxConfidence != nil {
		whereClauses = append(whereClauses, "COALESCE(confidence, 1.0) <= ?")
		args = append(args, *filters.MaxConfidence)
	}

	// Build base query
	query := "SELECT id, source_entity_id, relationship_type, target_entity_id, valid_from, valid_until, metadata, created_at, created_by, confidence FROM relationships"
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
		return nil, fmt.Errorf("failed to search relationships: %w", err)
	}
	defer rows.Close()

	relationships := []*types.Relationship{}
	for rows.Next() {
		var rel types.Relationship
		var metadataJSON sql.NullString
		var validUntil sql.NullTime
		var createdBy sql.NullString
		var confidence sql.NullFloat64

		err := rows.Scan(
			&rel.ID, &rel.SourceEntityID, &rel.RelationshipType, &rel.TargetEntityID,
			&rel.ValidFrom, &validUntil,
			&metadataJSON,
			&rel.CreatedAt, &createdBy, &confidence)
		if err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}

		// Map nullable fields
		if validUntil.Valid {
			rel.ValidUntil = &validUntil.Time
		}
		if createdBy.Valid {
			rel.CreatedBy = createdBy.String
		}
		if confidence.Valid {
			rel.Confidence = &confidence.Float64
		}

		// Unmarshal metadata
		if metadataJSON.Valid && metadataJSON.String != "" && metadataJSON.String != "{}" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &rel.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		relationships = append(relationships, &rel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate relationships: %w", err)
	}

	// Post-filtering for metadata (JSON queries in Dolt can be tricky, so filter in-memory)
	if len(filters.Metadata) > 0 {
		filtered := []*types.Relationship{}
		for _, rel := range relationships {
			if matchesMetadata(rel.Metadata, filters.Metadata) {
				filtered = append(filtered, rel)
			}
		}
		return filtered, nil
	}

	return relationships, nil
}

// GetRelationshipsWithTemporalFilter retrieves relationships for an entity with temporal and directional filtering.
// The validAt parameter filters to relationships valid at the specified time.
// The direction parameter controls whether to return outgoing (source), incoming (target), or both types of relationships.
func (s *DoltStore) GetRelationshipsWithTemporalFilter(ctx context.Context, entityID string, validAt time.Time, direction storage.RelationshipDirection) ([]*types.Relationship, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entityID == "" {
		return nil, fmt.Errorf("entity ID must be specified")
	}

	// Build WHERE clause based on direction
	whereClauses := []string{}
	args := []interface{}{}

	switch direction {
	case storage.RelationshipDirectionOutgoing:
		whereClauses = append(whereClauses, "source_entity_id = ?")
		args = append(args, entityID)
	case storage.RelationshipDirectionIncoming:
		whereClauses = append(whereClauses, "target_entity_id = ?")
		args = append(args, entityID)
	case storage.RelationshipDirectionBoth:
		whereClauses = append(whereClauses, "(source_entity_id = ? OR target_entity_id = ?)")
		args = append(args, entityID, entityID)
	default:
		return nil, fmt.Errorf("invalid relationship direction: %d", direction)
	}

	// Add temporal filtering
	// Logic: valid_from <= T AND (valid_until IS NULL OR valid_until > T)
	whereClauses = append(whereClauses, "valid_from <= ?")
	args = append(args, validAt)
	whereClauses = append(whereClauses, "(valid_until IS NULL OR valid_until > ?)")
	args = append(args, validAt)

	// Build query
	query := `SELECT id, source_entity_id, relationship_type, target_entity_id, valid_from, valid_until, metadata, created_at, created_by, confidence 
	          FROM relationships 
	          WHERE ` + strings.Join(whereClauses, " AND ") + `
	          ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query relationships: %w", err)
	}
	defer rows.Close()

	relationships := []*types.Relationship{}
	for rows.Next() {
		var rel types.Relationship
		var metadataJSON sql.NullString
		var validUntil sql.NullTime
		var createdBy sql.NullString
		var confidence sql.NullFloat64

		err := rows.Scan(
			&rel.ID, &rel.SourceEntityID, &rel.RelationshipType, &rel.TargetEntityID,
			&rel.ValidFrom, &validUntil,
			&metadataJSON,
			&rel.CreatedAt, &createdBy, &confidence)
		if err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}

		// Map nullable fields
		if validUntil.Valid {
			rel.ValidUntil = &validUntil.Time
		}
		if createdBy.Valid {
			rel.CreatedBy = createdBy.String
		}
		if confidence.Valid {
			rel.Confidence = &confidence.Float64
		}

		// Unmarshal metadata
		if metadataJSON.Valid && metadataJSON.String != "" && metadataJSON.String != "{}" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &rel.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		relationships = append(relationships, &rel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate relationships: %w", err)
	}

	return relationships, nil
}
