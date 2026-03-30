package dolt

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/steveyegge/beads/internal/similarity"
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
// When filters.TextQuery is set, performs two-step search:
// 1. SQL filters to get candidate set
// 2. In-memory cosine similarity ranking (name + summary)
// Results are sorted by similarity score descending when TextQuery is used.
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

	// Add ordering (by created_at descending by default) - will be overridden if TextQuery is set
	query += " ORDER BY created_at DESC"

	// Add pagination - will be applied after similarity ranking if TextQuery is set
	applyLimit := filters.Limit
	applyOffset := filters.Offset
	if filters.TextQuery != "" {
		// For text similarity, fetch all candidates (up to reasonable limit)
		// Pagination will be applied after scoring
		applyLimit = 0
		applyOffset = 0
	}

	if applyLimit > 0 {
		query += fmt.Sprintf(" LIMIT %d", applyLimit)
	}
	if applyOffset > 0 {
		query += fmt.Sprintf(" OFFSET %d", applyOffset)
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

	// Step 2: Apply text similarity ranking if TextQuery is set
	if filters.TextQuery != "" {
		entities = s.rankEntitiesBySimilarity(entities, filters)
	}

	return entities, nil
}

// rankEntitiesBySimilarity applies in-memory cosine similarity ranking to entities.
// Scores each entity against the query text, filters by threshold, sorts by score, and applies pagination.
func (s *DoltStore) rankEntitiesBySimilarity(entities []*types.Entity, filters storage.EntityFilters) []*types.Entity {
	if len(entities) == 0 {
		return entities
	}

	// Normalize and tokenize query
	normalizedQuery := similarity.NormalizeText(filters.TextQuery)
	queryTokens := similarity.Tokenize(normalizedQuery)

	// Set default threshold if not specified
	threshold := filters.TextSimilarityThreshold
	if threshold == 0 {
		threshold = 0.1
	}

	// Score each entity
	type scoredEntity struct {
		entity *types.Entity
		score  float64
	}

	scored := []scoredEntity{}
	for _, entity := range entities {
		// Combine name and summary for scoring
		text := entity.Name
		if entity.Summary != "" {
			text += " " + entity.Summary
		}
		normalizedText := similarity.NormalizeText(text)
		entityTokens := similarity.Tokenize(normalizedText)

		// Calculate cosine similarity
		score := similarity.CosineSimilarity(queryTokens, entityTokens)

		// Only include entities above threshold
		if score >= threshold {
			scored = append(scored, scoredEntity{entity: entity, score: score})
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Apply pagination
	start := filters.Offset
	if start > len(scored) {
		return []*types.Entity{}
	}

	end := len(scored)
	if filters.Limit > 0 {
		end = start + filters.Limit
		if end > len(scored) {
			end = len(scored)
		}
	}

	// Extract entities from scored results
	result := make([]*types.Entity, end-start)
	for i := start; i < end; i++ {
		result[i-start] = scored[i].entity
	}

	return result
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

// MergeEntities merges sourceEntityID into targetEntityID.
// This enables entity deduplication by moving all relationships from source to target
// and marking the source as soft-deleted (merged_into = targetEntityID).
// All operations are performed atomically within a single transaction.
func (s *DoltStore) MergeEntities(ctx context.Context, sourceEntityID, targetEntityID, actor string) error {
	if sourceEntityID == "" {
		return fmt.Errorf("source entity ID must be specified")
	}
	if targetEntityID == "" {
		return fmt.Errorf("target entity ID must be specified")
	}
	if sourceEntityID == targetEntityID {
		return fmt.Errorf("cannot merge entity into itself")
	}
	if actor == "" {
		actor = "system"
	}

	commitMsg := fmt.Sprintf("Merged entity %s into %s", sourceEntityID, targetEntityID)
	
	return s.RunInTransaction(ctx, commitMsg, func(tx storage.Transaction) error {
		// Cast to doltTransaction to access internal tx for direct SQL operations
		doltTx, ok := tx.(*doltTransaction)
		if !ok {
			return fmt.Errorf("transaction is not a doltTransaction")
		}
		
		// 1. VALIDATION: Verify both entities exist and are not already merged
		var sourceMergedInto sql.NullString
		err := doltTx.tx.QueryRowContext(ctx, 
			"SELECT merged_into FROM entities WHERE id = ?", 
			sourceEntityID).Scan(&sourceMergedInto)
		if err == sql.ErrNoRows {
			return fmt.Errorf("%w: source entity %s", storage.ErrNotFound, sourceEntityID)
		}
		if err != nil {
			return fmt.Errorf("failed to get source entity: %w", err)
		}
		if sourceMergedInto.Valid && sourceMergedInto.String != "" {
			return fmt.Errorf("source entity %s is already merged into %s", sourceEntityID, sourceMergedInto.String)
		}
		
		var targetMergedInto sql.NullString
		err = doltTx.tx.QueryRowContext(ctx, 
			"SELECT merged_into FROM entities WHERE id = ?", 
			targetEntityID).Scan(&targetMergedInto)
		if err == sql.ErrNoRows {
			return fmt.Errorf("%w: target entity %s", storage.ErrNotFound, targetEntityID)
		}
		if err != nil {
			return fmt.Errorf("failed to get target entity: %w", err)
		}
		if targetMergedInto.Valid && targetMergedInto.String != "" {
			return fmt.Errorf("target entity %s is already merged into %s (chain merges not allowed)", targetEntityID, targetMergedInto.String)
		}
		
		// 2. RELATIONSHIP MIGRATION
		// Handle duplicates: if both source and target have relationships to the same third entity
		// with the same type, keep the one with higher confidence (or newer if confidence equal)
		
		// Get all relationships involving the source entity
		sourceOutgoingRels, err := doltTx.tx.QueryContext(ctx,
			"SELECT id, target_entity_id, relationship_type, created_at FROM relationships WHERE source_entity_id = ?",
			sourceEntityID)
		if err != nil {
			return fmt.Errorf("failed to query source outgoing relationships: %w", err)
		}
		defer sourceOutgoingRels.Close()
		
		// Track relationships to potentially delete (duplicates with lower confidence)
		relsToDelete := []string{}
		
		for sourceOutgoingRels.Next() {
			var relID, targetEntity, relType string
			var createdAt time.Time
			if err := sourceOutgoingRels.Scan(&relID, &targetEntity, &relType, &createdAt); err != nil {
				return fmt.Errorf("failed to scan source outgoing relationship: %w", err)
			}
			
			// Check if target entity already has a relationship to the same third entity with same type
			var existingRelID string
			var existingCreatedAt time.Time
			err := doltTx.tx.QueryRowContext(ctx,
				"SELECT id, created_at FROM relationships WHERE source_entity_id = ? AND target_entity_id = ? AND relationship_type = ?",
				targetEntityID, targetEntity, relType).Scan(&existingRelID, &existingCreatedAt)
			
			if err == nil {
				// Duplicate found - keep the newer one
				if createdAt.After(existingCreatedAt) {
					// Source relationship is newer, delete existing and update source
					relsToDelete = append(relsToDelete, existingRelID)
					if _, err := doltTx.tx.ExecContext(ctx,
						"UPDATE relationships SET source_entity_id = ? WHERE id = ?",
						targetEntityID, relID); err != nil {
						return fmt.Errorf("failed to update source outgoing relationship %s: %w", relID, err)
					}
				} else {
					// Existing relationship is newer or same age, delete source relationship
					relsToDelete = append(relsToDelete, relID)
				}
			} else if err == sql.ErrNoRows {
				// No duplicate, just update the source_entity_id
				if _, err := doltTx.tx.ExecContext(ctx,
					"UPDATE relationships SET source_entity_id = ? WHERE id = ?",
					targetEntityID, relID); err != nil {
					return fmt.Errorf("failed to update source outgoing relationship %s: %w", relID, err)
				}
			} else {
				return fmt.Errorf("failed to check for duplicate outgoing relationship: %w", err)
			}
		}
		if err := sourceOutgoingRels.Err(); err != nil {
			return fmt.Errorf("failed to iterate source outgoing relationships: %w", err)
		}
		
		// Handle incoming relationships (where source is the target)
		sourceIncomingRels, err := doltTx.tx.QueryContext(ctx,
			"SELECT id, source_entity_id, relationship_type, created_at FROM relationships WHERE target_entity_id = ?",
			sourceEntityID)
		if err != nil {
			return fmt.Errorf("failed to query source incoming relationships: %w", err)
		}
		defer sourceIncomingRels.Close()
		
		for sourceIncomingRels.Next() {
			var relID, sourceEntity, relType string
			var createdAt time.Time
			if err := sourceIncomingRels.Scan(&relID, &sourceEntity, &relType, &createdAt); err != nil {
				return fmt.Errorf("failed to scan source incoming relationship: %w", err)
			}
			
			// Check if target entity already has a relationship from the same third entity with same type
			var existingRelID string
			var existingCreatedAt time.Time
			err := doltTx.tx.QueryRowContext(ctx,
				"SELECT id, created_at FROM relationships WHERE source_entity_id = ? AND target_entity_id = ? AND relationship_type = ?",
				sourceEntity, targetEntityID, relType).Scan(&existingRelID, &existingCreatedAt)
			
			if err == nil {
				// Duplicate found - keep the newer one
				if createdAt.After(existingCreatedAt) {
					// Source relationship is newer, delete existing and update source
					relsToDelete = append(relsToDelete, existingRelID)
					if _, err := doltTx.tx.ExecContext(ctx,
						"UPDATE relationships SET target_entity_id = ? WHERE id = ?",
						targetEntityID, relID); err != nil {
						return fmt.Errorf("failed to update source incoming relationship %s: %w", relID, err)
					}
				} else {
					// Existing relationship is newer or same age, delete source relationship
					relsToDelete = append(relsToDelete, relID)
				}
			} else if err == sql.ErrNoRows {
				// No duplicate, just update the target_entity_id
				if _, err := doltTx.tx.ExecContext(ctx,
					"UPDATE relationships SET target_entity_id = ? WHERE id = ?",
					targetEntityID, relID); err != nil {
					return fmt.Errorf("failed to update source incoming relationship %s: %w", relID, err)
				}
			} else {
				return fmt.Errorf("failed to check for duplicate incoming relationship: %w", err)
			}
		}
		if err := sourceIncomingRels.Err(); err != nil {
			return fmt.Errorf("failed to iterate source incoming relationships: %w", err)
		}
		
		// Delete duplicate relationships
		for _, relID := range relsToDelete {
			if _, err := doltTx.tx.ExecContext(ctx,
				"DELETE FROM relationships WHERE id = ?", relID); err != nil {
				return fmt.Errorf("failed to delete duplicate relationship %s: %w", relID, err)
			}
		}
		
		// 3. METADATA MERGE (optional but recommended)
		// Retrieve source and target entities for metadata merge
		var sourceSummary, targetSummary sql.NullString
		var sourceMetadataJSON, targetMetadataJSON sql.NullString
		
		err = doltTx.tx.QueryRowContext(ctx,
			"SELECT summary, metadata FROM entities WHERE id = ?",
			sourceEntityID).Scan(&sourceSummary, &sourceMetadataJSON)
		if err != nil {
			return fmt.Errorf("failed to get source entity data: %w", err)
		}
		
		err = doltTx.tx.QueryRowContext(ctx,
			"SELECT summary, metadata FROM entities WHERE id = ?",
			targetEntityID).Scan(&targetSummary, &targetMetadataJSON)
		if err != nil {
			return fmt.Errorf("failed to get target entity data: %w", err)
		}
		
		// Merge summaries (append source to target if both exist)
		var mergedSummary string
		if targetSummary.Valid && targetSummary.String != "" {
			mergedSummary = targetSummary.String
			if sourceSummary.Valid && sourceSummary.String != "" && sourceSummary.String != targetSummary.String {
				mergedSummary += "\n\n[Merged from " + sourceEntityID + "]: " + sourceSummary.String
			}
		} else if sourceSummary.Valid {
			mergedSummary = sourceSummary.String
		}
		
		// Merge metadata (target wins on conflicts)
		var mergedMetadata map[string]interface{}
		if sourceMetadataJSON.Valid && sourceMetadataJSON.String != "" && sourceMetadataJSON.String != "{}" {
			json.Unmarshal([]byte(sourceMetadataJSON.String), &mergedMetadata)
		}
		if mergedMetadata == nil {
			mergedMetadata = make(map[string]interface{})
		}
		if targetMetadataJSON.Valid && targetMetadataJSON.String != "" && targetMetadataJSON.String != "{}" {
			var targetMetadata map[string]interface{}
			json.Unmarshal([]byte(targetMetadataJSON.String), &targetMetadata)
			// Target wins on conflicts
			for k, v := range targetMetadata {
				mergedMetadata[k] = v
			}
		}
		
		// Update target entity with merged data
		mergedMetadataJSON, err := json.Marshal(mergedMetadata)
		if err != nil {
			return fmt.Errorf("failed to marshal merged metadata: %w", err)
		}
		
		if _, err := doltTx.tx.ExecContext(ctx,
			"UPDATE entities SET summary = ?, metadata = ?, updated_at = ?, updated_by = ? WHERE id = ?",
			mergedSummary, string(mergedMetadataJSON), time.Now().UTC(), actor, targetEntityID); err != nil {
			return fmt.Errorf("failed to update target entity with merged data: %w", err)
		}
		
		// 4. SOFT DELETE SOURCE
		// Mark source entity as merged (do not delete row for provenance)
		if _, err := doltTx.tx.ExecContext(ctx,
			"UPDATE entities SET merged_into = ?, updated_at = ?, updated_by = ? WHERE id = ?",
			targetEntityID, time.Now().UTC(), actor, sourceEntityID); err != nil {
			return fmt.Errorf("failed to mark source entity as merged: %w", err)
		}
		
		// Mark tables as dirty for Dolt commit
		doltTx.markDirty("entities")
		doltTx.markDirty("relationships")
		
		return nil
	})
}
