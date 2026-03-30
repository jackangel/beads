package dolt

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// MigrateToV8 executes the migration from schema v7 to v8.
// Creates the 5 new knowledge graph tables (entities, relationships, episodes,
// entity_types, relationship_types) as defined in schema_v8.sql.
//
// This migration is transactional - all tables are created atomically or none.
// Does NOT migrate data from v7 tables (that's handled separately).
// Updates schema_version in config table to "8" on success.
func MigrateToV8(ctx context.Context, db *sql.DB) error {
	log.Println("Starting migration to schema v8...")

	// Check current schema version
	currentVersion, err := GetSchemaVersion(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	if currentVersion == "8" {
		log.Println("Schema is already at version 8, skipping migration")
		return nil
	}

	if currentVersion != "7" {
		return fmt.Errorf("migration to v8 requires schema v7, but found v%s", currentVersion)
	}

	// Start transaction for atomicity
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Create entities table
	log.Println("Creating entities table...")
	_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS entities (
    -- Core Identification
    id VARCHAR(255) PRIMARY KEY,
    entity_type VARCHAR(255) NOT NULL,
    name VARCHAR(500) NOT NULL,
    
    -- Content
    summary TEXT,
    
    -- Custom Metadata (flexible JSON for extension)
    metadata JSON DEFAULT (JSON_OBJECT()),
    
    -- Timestamps
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Attribution
    created_by VARCHAR(255) NOT NULL,
    updated_by VARCHAR(255) DEFAULT '',
    
    -- Indexes for query performance
    INDEX idx_entities_type (entity_type),
    INDEX idx_entities_name (name(255)),
    INDEX idx_entities_created_at (created_at),
    INDEX idx_entities_updated_at (updated_at)
)`)
	if err != nil {
		return fmt.Errorf("failed to create entities table: %w", err)
	}

	// Create relationships table
	log.Println("Creating relationships table...")
	_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS relationships (
    -- Core Identification
    id VARCHAR(255) PRIMARY KEY,
    source_entity_id VARCHAR(255) NOT NULL,
    relationship_type VARCHAR(255) NOT NULL,
    target_entity_id VARCHAR(255) NOT NULL,
    
    -- Temporal Validity
    valid_from DATETIME NOT NULL,
    valid_until DATETIME NULL,
    
    -- Custom Metadata (flexible JSON for extension)
    metadata JSON DEFAULT (JSON_OBJECT()),
    
    -- Timestamps
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Attribution
    created_by VARCHAR(255) NOT NULL,
    
    -- Indexes for query performance
    INDEX idx_relationships_source (source_entity_id),
    INDEX idx_relationships_target (target_entity_id),
    INDEX idx_relationships_type (relationship_type),
    INDEX idx_relationships_valid_from (valid_from),
    INDEX idx_relationships_valid_until (valid_until),
    INDEX idx_relationships_temporal (source_entity_id, valid_from, valid_until),
    INDEX idx_relationships_created_at (created_at),
    
    -- Foreign Keys
    CONSTRAINT fk_relationships_source FOREIGN KEY (source_entity_id) 
        REFERENCES entities(id) ON DELETE CASCADE,
    CONSTRAINT fk_relationships_target FOREIGN KEY (target_entity_id) 
        REFERENCES entities(id) ON DELETE CASCADE
)`)
	if err != nil {
		return fmt.Errorf("failed to create relationships table: %w", err)
	}

	// Create episodes table
	log.Println("Creating episodes table...")
	_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS episodes (
    -- Core Identification
    id VARCHAR(255) PRIMARY KEY,
    
    -- Content
    timestamp DATETIME NOT NULL,
    source VARCHAR(255) NOT NULL,
    raw_data BLOB,
    
    -- Extraction Results (JSON array of entity IDs)
    entities_extracted JSON DEFAULT (JSON_ARRAY()),
    
    -- Custom Metadata (flexible JSON for extension)
    metadata JSON DEFAULT (JSON_OBJECT()),
    
    -- Timestamps (created_at only - episodes are immutable)
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Indexes for query performance
    INDEX idx_episodes_timestamp (timestamp),
    INDEX idx_episodes_source (source),
    INDEX idx_episodes_created_at (created_at)
)`)
	if err != nil {
		return fmt.Errorf("failed to create episodes table: %w", err)
	}

	// Create entity_types table
	log.Println("Creating entity_types table...")
	_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS entity_types (
    -- Core Identification
    type_name VARCHAR(255) PRIMARY KEY,
    
    -- Schema Definition (JSON Schema for validation)
    schema_json TEXT NOT NULL,
    
    -- Documentation
    description TEXT,
    
    -- Timestamps
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Attribution
    created_by VARCHAR(255) NOT NULL,
    
    -- Indexes for query performance
    INDEX idx_entity_types_created_at (created_at)
)`)
	if err != nil {
		return fmt.Errorf("failed to create entity_types table: %w", err)
	}

	// Create relationship_types table
	log.Println("Creating relationship_types table...")
	_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS relationship_types (
    -- Core Identification
    type_name VARCHAR(255) PRIMARY KEY,
    
    -- Schema Definition (JSON Schema for validation)
    schema_json TEXT NOT NULL,
    
    -- Documentation
    description TEXT,
    
    -- Timestamps
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Attribution
    created_by VARCHAR(255) NOT NULL,
    
    -- Indexes for query performance
    INDEX idx_relationship_types_created_at (created_at)
)`)
	if err != nil {
		return fmt.Errorf("failed to create relationship_types table: %w", err)
	}

	// ============================================================================
	// DATA MIGRATION: Migrate data from v7 tables to v8 tables
	// ============================================================================

	log.Println("Starting data migration from v7 to v8...")

	// STEP 1: Migrate issues → entities
	log.Println("Migrating issues to entities...")
	
	// Get total count of issues
	var issueCount int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM issues").Scan(&issueCount)
	if err != nil {
		return fmt.Errorf("failed to count issues: %w", err)
	}
	log.Printf("Found %d issues to migrate", issueCount)

	// Batch migrate issues to entities (1000 rows at a time)
	batchSize := 1000
	for offset := 0; offset < issueCount; offset += batchSize {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO entities (id, entity_type, name, summary, metadata, created_at, updated_at, created_by, updated_by)
			SELECT 
				id,
				CASE 
					WHEN issue_type = 'epic' THEN 'epic'
					WHEN issue_type = 'task' THEN 'task'
					WHEN issue_type = 'sub-task' THEN 'subtask'
					ELSE 'issue'
				END as entity_type,
				title as name,
				description as summary,
				metadata,
				created_at,
				updated_at,
				created_by,
				updated_by
			FROM issues
			LIMIT ? OFFSET ?
		`, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to migrate issues batch at offset %d: %w", offset, err)
		}
		log.Printf("Migrated %d/%d issues...", min(offset+batchSize, issueCount), issueCount)
	}

	// STEP 2: Migrate dependencies → relationships
	log.Println("Migrating dependencies to relationships...")
	
	// Get total count of dependencies
	var depCount int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM dependencies").Scan(&depCount)
	if err != nil {
		return fmt.Errorf("failed to count dependencies: %w", err)
	}
	log.Printf("Found %d dependencies to migrate", depCount)

	// Batch migrate dependencies to relationships (1000 rows at a time)
	for offset := 0; offset < depCount; offset += batchSize {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO relationships (id, source_entity_id, relationship_type, target_entity_id, valid_from, valid_until, metadata, created_at, created_by)
			SELECT 
				CONCAT('rel-', issue_id, '-', depends_on_id, '-', type) as id,
				issue_id as source_entity_id,
				type as relationship_type,
				depends_on_id as target_entity_id,
				created_at as valid_from,
				NULL as valid_until,
				metadata,
				created_at,
				created_by
			FROM dependencies
			LIMIT ? OFFSET ?
		`, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to migrate dependencies batch at offset %d: %w", offset, err)
		}
		log.Printf("Migrated %d/%d dependencies...", min(offset+batchSize, depCount), depCount)
	}

	// STEP 3: Migrate events → episodes
	log.Println("Migrating events to episodes...")
	
	// Get total count of events
	var eventCount int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&eventCount)
	if err != nil {
		return fmt.Errorf("failed to count events: %w", err)
	}
	log.Printf("Found %d events to migrate", eventCount)

	// Batch migrate events to episodes (1000 rows at a time)
	for offset := 0; offset < eventCount; offset += batchSize {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO episodes (id, timestamp, source, raw_data, entities_extracted, metadata, created_at)
			SELECT 
				id,
				created_at as timestamp,
				'legacy_events' as source,
				CAST(comment AS BINARY) as raw_data,
				JSON_ARRAY(issue_id) as entities_extracted,
				JSON_OBJECT('event_type', event_type, 'actor', actor, 'old_value', old_value, 'new_value', new_value) as metadata,
				created_at
			FROM events
			LIMIT ? OFFSET ?
		`, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to migrate events batch at offset %d: %w", offset, err)
		}
		log.Printf("Migrated %d/%d events...", min(offset+batchSize, eventCount), eventCount)
	}

	// STEP 4: Validate row counts after migration
	log.Println("Validating data migration...")
	
	// Validate entities count
	var entityCount int
	err = tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM entities WHERE entity_type IN ('epic', 'task', 'subtask', 'issue')").Scan(&entityCount)
	if err != nil {
		return fmt.Errorf("failed to count migrated entities: %w", err)
	}
	if entityCount != issueCount {
		return fmt.Errorf("entity count mismatch: expected %d issues, got %d entities", issueCount, entityCount)
	}
	log.Printf("✓ Entity count validated: %d entities", entityCount)

	// Validate relationships count
	var relCount int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM relationships").Scan(&relCount)
	if err != nil {
		return fmt.Errorf("failed to count migrated relationships: %w", err)
	}
	if relCount != depCount {
		return fmt.Errorf("relationship count mismatch: expected %d dependencies, got %d relationships", depCount, relCount)
	}
	log.Printf("✓ Relationship count validated: %d relationships", relCount)

	// Validate episodes count
	var episodeCount int
	err = tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM episodes WHERE source = 'legacy_events'").Scan(&episodeCount)
	if err != nil {
		return fmt.Errorf("failed to count migrated episodes: %w", err)
	}
	if episodeCount != eventCount {
		return fmt.Errorf("episode count mismatch: expected %d events, got %d episodes", eventCount, episodeCount)
	}
	log.Printf("✓ Episode count validated: %d episodes", episodeCount)

	log.Println("Data migration completed successfully")

	// Update schema version to 8
	log.Println("Updating schema version to 8...")
	_, err = tx.ExecContext(ctx,
		"INSERT INTO config (`key`, `value`) VALUES ('schema_version', '8') "+
			"ON DUPLICATE KEY UPDATE `value` = '8'")
	if err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Validate the migration succeeded
	if err := ValidateV8Schema(ctx, db); err != nil {
		return fmt.Errorf("migration validation failed: %w", err)
	}

	log.Println("Migration to schema v8 completed successfully")
	return nil
}

// RollbackFromV8 reverts the schema from v8 back to v7.
// Drops the 5 knowledge graph tables and updates schema_version to "7".
//
// This operation is transactional - all tables are dropped atomically or none.
// WARNING: This will delete all data in the v8 tables.
func RollbackFromV8(ctx context.Context, db *sql.DB) error {
	log.Println("Starting rollback from schema v8 to v7...")

	// Check current schema version
	currentVersion, err := GetSchemaVersion(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	if currentVersion == "7" {
		log.Println("Schema is already at version 7, skipping rollback")
		return nil
	}

	if currentVersion != "8" {
		return fmt.Errorf("rollback from v8 requires schema v8, but found v%s", currentVersion)
	}

	// Start transaction for atomicity
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Drop tables in reverse order (relationships first due to foreign keys)
	tables := []string{
		"relationship_types",
		"entity_types",
		"episodes",
		"relationships", // Must be before entities due to FKs
		"entities",
	}

	for _, table := range tables {
		log.Printf("Dropping table %s...", table)
		_, err = tx.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	// Update schema version back to 7
	log.Println("Updating schema version to 7...")
	_, err = tx.ExecContext(ctx,
		"INSERT INTO config (`key`, `value`) VALUES ('schema_version', '7') "+
			"ON DUPLICATE KEY UPDATE `value` = '7'")
	if err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Println("Rollback to schema v7 completed successfully")
	return nil
}

// ValidateV8Schema validates that schema v8 is correctly installed.
// Checks that all 5 new tables exist and all required indexes are present.
//
// Returns an error with details if any validation check fails.
func ValidateV8Schema(ctx context.Context, db *sql.DB) error {
	log.Println("Validating schema v8...")

	// Check schema version
	version, err := GetSchemaVersion(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}
	if version != "8" {
		return fmt.Errorf("schema version is %s, expected 8", version)
	}

	// Check all required tables exist
	requiredTables := []string{
		"entities",
		"relationships",
		"episodes",
		"entity_types",
		"relationship_types",
	}

	for _, table := range requiredTables {
		var count int
		err := db.QueryRowContext(ctx, "SHOW TABLES LIKE ?", table).Scan(&count)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("table %s does not exist", table)
			}
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}
	}

	// Check key indexes exist
	type indexCheck struct {
		table string
		index string
	}

	requiredIndexes := []indexCheck{
		{"entities", "idx_entities_type"},
		{"entities", "idx_entities_name"},
		{"relationships", "idx_relationships_source"},
		{"relationships", "idx_relationships_target"},
		{"relationships", "idx_relationships_type"},
		{"relationships", "idx_relationships_temporal"},
		{"episodes", "idx_episodes_timestamp"},
		{"episodes", "idx_episodes_source"},
		{"entity_types", "idx_entity_types_created_at"},
		{"relationship_types", "idx_relationship_types_created_at"},
	}

	for _, check := range requiredIndexes {
		rows, err := db.QueryContext(ctx,
			"SHOW INDEX FROM "+check.table+" WHERE Key_name = ?", check.index)
		if err != nil {
			return fmt.Errorf("failed to check index %s on table %s: %w",
				check.index, check.table, err)
		}
		hasRows := rows.Next()
		rows.Close()
		if !hasRows {
			return fmt.Errorf("index %s does not exist on table %s",
				check.index, check.table)
		}
	}

	// Check foreign keys on relationships table
	rows, err := db.QueryContext(ctx, `
		SELECT CONSTRAINT_NAME 
		FROM information_schema.TABLE_CONSTRAINTS 
		WHERE TABLE_SCHEMA = DATABASE() 
		  AND TABLE_NAME = 'relationships' 
		  AND CONSTRAINT_TYPE = 'FOREIGN KEY'`)
	if err != nil {
		return fmt.Errorf("failed to check foreign keys on relationships table: %w", err)
	}
	defer rows.Close()

	fkCount := 0
	for rows.Next() {
		fkCount++
	}
	if fkCount < 2 {
		return fmt.Errorf("relationships table should have 2 foreign keys, found %d", fkCount)
	}

	log.Println("Schema v8 validation passed")
	return nil
}

// GetSchemaVersion returns the current schema version from the config table.
// Returns "7" for v7, "8" for v8, or an error if the version cannot be determined.
//
// If the schema_version key doesn't exist in config, returns "0" to indicate
// an unversioned or very old database.
func GetSchemaVersion(ctx context.Context, db *sql.DB) (string, error) {
	var version string
	err := db.QueryRowContext(ctx,
		"SELECT `value` FROM config WHERE `key` = 'schema_version'").Scan(&version)
	if err != nil {
		if err == sql.ErrNoRows {
			// Schema version not set - this is an old or unversioned database
			return "0", nil
		}
		return "", fmt.Errorf("failed to query schema version: %w", err)
	}

	// Trim whitespace and validate
	version = strings.TrimSpace(version)
	if version == "" {
		return "0", nil
	}

	return version, nil
}
