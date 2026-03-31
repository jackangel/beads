package migrations

import (
	"database/sql"
	"fmt"
)

// MigrateV81Extensions adds v8.1 schema extensions for intelligence layer features:
// - relationships.confidence: AI-extracted relationship scoring (FLOAT, nullable, default 1.0)
// - entities.merged_into: soft-delete tracking for entity merge (VARCHAR(255), nullable) with index
// - episodes.extracted_at: LLM extraction timestamp (DATETIME, nullable) with index
// - entity_embeddings: vector search support table
//
// All changes are additive (no breaking changes to existing queries).
// Idempotent: checks for existence before adding columns/tables.
// Skips gracefully if v8 tables don't exist yet (user hasn't run `bd v8 migrate`).
func MigrateV81Extensions(db *sql.DB) error {
	// Check if v8 tables exist - if not, skip this migration gracefully
	// (v8 tables are created by manual migration, not automatic init)
	v8TablesExist := true
	for _, table := range []string{"entities", "relationships", "episodes"} {
		exists, err := tableExists(db, table)
		if err != nil {
			return fmt.Errorf("failed to check if table %s exists: %w", table, err)
		}
		if !exists {
			v8TablesExist = false
			break
		}
	}
	
	if !v8TablesExist {
		// v8 tables don't exist yet - skip this migration
		// It will run automatically after `bd v8 migrate` creates the tables
		return nil
	}

	// 1. Add confidence column to relationships table
	exists, err := columnExists(db, "relationships", "confidence")
	if err != nil {
		return fmt.Errorf("failed to check confidence column: %w", err)
	}
	if !exists {
		_, err = db.Exec(`
			ALTER TABLE relationships 
			ADD COLUMN confidence FLOAT NULL DEFAULT 1.0 
			COMMENT 'AI-extracted confidence score (0.0-1.0), default 1.0 for manually created relationships'
		`)
		if err != nil {
			return fmt.Errorf("failed to add confidence column to relationships: %w", err)
		}
	}

	// 2. Add merged_into column to entities table
	exists, err = columnExists(db, "entities", "merged_into")
	if err != nil {
		return fmt.Errorf("failed to check merged_into column: %w", err)
	}
	if !exists {
		_, err = db.Exec(`
			ALTER TABLE entities 
			ADD COLUMN merged_into VARCHAR(255) NULL 
			COMMENT 'Target entity ID if this entity was merged (soft delete)'
		`)
		if err != nil {
			return fmt.Errorf("failed to add merged_into column to entities: %w", err)
		}

		// Add index for filtering out merged entities
		if !indexExists(db, "entities", "idx_entities_merged_into") {
			_, err = db.Exec(`
				CREATE INDEX idx_entities_merged_into ON entities(merged_into)
			`)
			if err != nil {
				return fmt.Errorf("failed to create merged_into index: %w", err)
			}
		}
	}

	// 3. Add extracted_at column to episodes table
	exists, err = columnExists(db, "episodes", "extracted_at")
	if err != nil {
		return fmt.Errorf("failed to check extracted_at column: %w", err)
	}
	if !exists {
		_, err = db.Exec(`
			ALTER TABLE episodes 
			ADD COLUMN extracted_at DATETIME NULL 
			COMMENT 'Timestamp when LLM extraction was completed for this episode'
		`)
		if err != nil {
			return fmt.Errorf("failed to add extracted_at column to episodes: %w", err)
		}

		// Add index for finding unprocessed episodes
		if !indexExists(db, "episodes", "idx_episodes_extracted_at") {
			_, err = db.Exec(`
				CREATE INDEX idx_episodes_extracted_at ON episodes(extracted_at)
			`)
			if err != nil {
				return fmt.Errorf("failed to create extracted_at index: %w", err)
			}
		}
	}

	// 4. Create entity_embeddings table (optional, for future vector search)
	tableExists, err := tableExists(db, "entity_embeddings")
	if err != nil {
		return fmt.Errorf("failed to check entity_embeddings table: %w", err)
	}
	if !tableExists {
		_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS entity_embeddings (
    entity_id VARCHAR(255) PRIMARY KEY,
    embedding BLOB,
    model VARCHAR(255),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_entity_embeddings_entity FOREIGN KEY (entity_id) 
        REFERENCES entities(id) ON DELETE CASCADE
)
		`)
		if err != nil {
			return fmt.Errorf("failed to create entity_embeddings table: %w", err)
		}
	}

	return nil
}
