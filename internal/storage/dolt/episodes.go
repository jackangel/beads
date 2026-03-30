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

// CreateEpisode creates a new episode in the episodes table.
// Episodes are immutable provenance records - once created, they are never modified.
// The episode ID must be unique and all required fields must be specified.
func (s *DoltStore) CreateEpisode(ctx context.Context, episode *types.Episode) error {
	if episode == nil {
		return fmt.Errorf("episode must not be nil")
	}
	if episode.ID == "" {
		return fmt.Errorf("episode ID must be specified")
	}
	if episode.Source == "" {
		return fmt.Errorf("episode source must be specified")
	}
	if episode.Timestamp.IsZero() {
		return fmt.Errorf("episode timestamp must be specified")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Marshal entities_extracted to JSON array
	var entitiesExtractedJSON []byte
	if episode.EntitiesExtracted != nil && len(episode.EntitiesExtracted) > 0 {
		entitiesExtractedJSON, err = json.Marshal(episode.EntitiesExtracted)
		if err != nil {
			return fmt.Errorf("failed to marshal entities_extracted: %w", err)
		}
	} else {
		entitiesExtractedJSON = []byte("[]")
	}

	// Marshal metadata to JSON object
	var metadataJSON []byte
	if episode.Metadata != nil && len(episode.Metadata) > 0 {
		metadataJSON, err = json.Marshal(episode.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	// Set created_at timestamp if not already set
	now := time.Now().UTC()
	if episode.CreatedAt.IsZero() {
		episode.CreatedAt = now
	}

	// Insert episode
	query := `INSERT INTO episodes (id, timestamp, source, raw_data, entities_extracted, metadata, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, query,
		episode.ID, episode.Timestamp, episode.Source, episode.RawData,
		string(entitiesExtractedJSON),
		string(metadataJSON),
		episode.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert episode: %w", err)
	}

	// Dolt versioning: stage the table and commit
	if _, err := tx.ExecContext(ctx, "CALL DOLT_ADD(?)", "episodes"); err != nil {
		return fmt.Errorf("dolt add episodes: %w", err)
	}

	commitMsg := fmt.Sprintf("bd: create episode %s", episode.ID)
	if _, err := tx.ExecContext(ctx, "CALL DOLT_COMMIT('-m', ?, '--author', ?)",
		commitMsg, s.commitAuthorString()); err != nil && !isDoltNothingToCommit(err) {
		return fmt.Errorf("dolt commit: %w", err)
	}

	return tx.Commit()
}

// GetEpisode retrieves an episode by its ID.
// Returns storage.ErrNotFound if the episode does not exist.
func (s *DoltStore) GetEpisode(ctx context.Context, id string) (*types.Episode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, timestamp, source, raw_data, entities_extracted, metadata, created_at
	          FROM episodes WHERE id = ?`

	var episode types.Episode
	var rawData sql.NullString // BLOB is read as string/bytes
	var entitiesExtractedJSON sql.NullString
	var metadataJSON sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&episode.ID, &episode.Timestamp, &episode.Source, &rawData,
		&entitiesExtractedJSON,
		&metadataJSON,
		&episode.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: episode %s", storage.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get episode: %w", err)
	}

	// Map raw_data BLOB field
	if rawData.Valid {
		episode.RawData = []byte(rawData.String)
	}

	// Unmarshal entities_extracted JSON array
	if entitiesExtractedJSON.Valid && entitiesExtractedJSON.String != "" && entitiesExtractedJSON.String != "[]" {
		if err := json.Unmarshal([]byte(entitiesExtractedJSON.String), &episode.EntitiesExtracted); err != nil {
			return nil, fmt.Errorf("failed to unmarshal entities_extracted: %w", err)
		}
	}

	// Unmarshal metadata JSON object
	if metadataJSON.Valid && metadataJSON.String != "" && metadataJSON.String != "{}" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &episode.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &episode, nil
}

// SearchEpisodes finds episodes matching the provided filters.
// Returns an empty slice if no episodes match the criteria.
// Episodes are ordered by timestamp descending (newest first) by default.
func (s *DoltStore) SearchEpisodes(ctx context.Context, filters storage.EpisodeFilters) ([]*types.Episode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build WHERE clause from filters
	whereClauses := []string{}
	args := []interface{}{}

	if filters.Source != "" {
		whereClauses = append(whereClauses, "source = ?")
		args = append(args, filters.Source)
	}

	if filters.TimestampStart != nil {
		whereClauses = append(whereClauses, "timestamp >= ?")
		args = append(args, *filters.TimestampStart)
	}

	if filters.TimestampEnd != nil {
		whereClauses = append(whereClauses, "timestamp <= ?")
		args = append(args, *filters.TimestampEnd)
	}

	// Build base query
	query := "SELECT id, timestamp, source, raw_data, entities_extracted, metadata, created_at FROM episodes"
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Add ordering (by timestamp descending, newest first)
	query += " ORDER BY timestamp DESC"

	// Add pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.Limit)
	}
	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search episodes: %w", err)
	}
	defer rows.Close()

	episodes := []*types.Episode{}
	for rows.Next() {
		var episode types.Episode
		var rawData sql.NullString
		var entitiesExtractedJSON sql.NullString
		var metadataJSON sql.NullString

		err := rows.Scan(
			&episode.ID, &episode.Timestamp, &episode.Source, &rawData,
			&entitiesExtractedJSON,
			&metadataJSON,
			&episode.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan episode: %w", err)
		}

		// Map raw_data BLOB field
		if rawData.Valid {
			episode.RawData = []byte(rawData.String)
		}

		// Unmarshal entities_extracted JSON array
		if entitiesExtractedJSON.Valid && entitiesExtractedJSON.String != "" && entitiesExtractedJSON.String != "[]" {
			if err := json.Unmarshal([]byte(entitiesExtractedJSON.String), &episode.EntitiesExtracted); err != nil {
				return nil, fmt.Errorf("failed to unmarshal entities_extracted: %w", err)
			}
		}

		// Unmarshal metadata JSON object
		if metadataJSON.Valid && metadataJSON.String != "" && metadataJSON.String != "{}" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &episode.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		episodes = append(episodes, &episode)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate episodes: %w", err)
	}

	// Post-filtering for entities_extracted (JSON array queries in Dolt require special handling)
	// Filter in-memory to find episodes containing any of the specified entity IDs
	if len(filters.EntitiesExtracted) > 0 {
		filtered := []*types.Episode{}
		for _, episode := range episodes {
			if containsAnyEntity(episode.EntitiesExtracted, filters.EntitiesExtracted) {
				filtered = append(filtered, episode)
			}
		}
		episodes = filtered
	}

	return episodes, nil
}

// containsAnyEntity checks if an episode's entities_extracted list contains any of the filter entity IDs.
func containsAnyEntity(episodeEntities, filterEntities []string) bool {
	if len(episodeEntities) == 0 {
		return false
	}
	// Build a set of episode entities for efficient lookup
	entitySet := make(map[string]bool, len(episodeEntities))
	for _, entityID := range episodeEntities {
		entitySet[entityID] = true
	}
	// Check if any filter entity exists in the episode's entities
	for _, filterEntity := range filterEntities {
		if entitySet[filterEntity] {
			return true
		}
	}
	return false
}
