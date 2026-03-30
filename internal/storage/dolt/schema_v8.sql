-- Schema Version 8: Knowledge Graph Tables
--
-- This schema adds support for the memory/knowledge graph system alongside
-- the existing issue tracker schema (v7). The v7 tables (issues, dependencies,
-- labels, comments, events, etc.) remain intact for backward compatibility.
--
-- New tables:
--   - entities: Knowledge graph nodes (people, products, concepts)
--   - relationships: Typed edges between entities with temporal validity
--   - episodes: Immutable provenance log of ingested data
--   - entity_types: Custom ontology definitions for entity types
--   - relationship_types: Custom ontology definitions for relationship types
--
-- Design principles:
--   - Episodes are immutable (raw material, never updated)
--   - Entities evolve over time (summary gets refined)
--   - Relationships have temporal validity (can expire)
--   - Custom ontology support via JSON Schema validation
--
-- Compatible with: MySQL 5.7+, Dolt

-- ============================================================================
-- ENTITIES TABLE
-- ============================================================================
-- Represents knowledge graph entities extracted from episodes.
-- Entities are evolving summaries of real-world things (people, products,
-- concepts) discovered in issue data, commits, comments, and other sources.
--
-- v8.1 Extensions:
-- - merged_into: VARCHAR(255), nullable. Target entity ID if this entity was
--   merged into another entity (soft delete for deduplication).
--
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
    
    -- v8.1 Extensions (added via ALTER TABLE migration)
    -- merged_into VARCHAR(255) NULL,
    
    -- Indexes for query performance
    INDEX idx_entities_type (entity_type),
    INDEX idx_entities_name (name(255)),
    INDEX idx_entities_created_at (created_at),
    INDEX idx_entities_updated_at (updated_at)
    -- v8.1: INDEX idx_entities_merged_into (merged_into)
);

-- ============================================================================
-- RELATIONSHIPS TABLE
-- ============================================================================
-- Represents typed edges between entities in the knowledge graph.
-- Relationships support temporal validity to track how connections change
-- over time (e.g., "Person X worked at Company Y from 2020 to 2023").
--
-- v8.1 Extensions:
-- - confidence: FLOAT, nullable, default 1.0. AI-extracted confidence score
--   (0.0-1.0) for machine-extracted relationships. Manually created 
--   relationships default to 1.0 (fully confident).
--
CREATE TABLE IF NOT EXISTS relationships (
    -- Core Identification
    id VARCHAR(255) PRIMARY KEY,
    source_entity_id VARCHAR(255) NOT NULL,
    relationship_type VARCHAR(255) NOT NULL,
    target_entity_id VARCHAR(255) NOT NULL,
    
    -- Temporal Validity
    valid_from DATETIME NOT NULL,
    valid_until DATETIME NULL,  -- NULL means still valid
    
    -- Custom Metadata (flexible JSON for extension)
    metadata JSON DEFAULT (JSON_OBJECT()),
    
    -- Timestamps
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Attribution
    created_by VARCHAR(255) NOT NULL,
    
    -- v8.1 Extensions (added via ALTER TABLE migration)
    -- confidence FLOAT NULL DEFAULT 1.0,
    
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
);

-- ============================================================================
-- EPISODES TABLE
-- ============================================================================
-- Represents immutable snapshots of ingested data.
-- Episodes are the raw material from which entities and relationships are
-- extracted. Once created, episodes are never modified (no updated_at or
-- updated_by fields). This preserves complete provenance.
--
-- v8.1 Extensions:
-- - extracted_at: DATETIME, nullable. Timestamp when LLM extraction was
--   completed for this episode. NULL means not yet processed.
--
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
    
    -- v8.1 Extensions (added via ALTER TABLE migration)
    -- extracted_at DATETIME NULL,
    
    -- Indexes for query performance
    INDEX idx_episodes_timestamp (timestamp),
    INDEX idx_episodes_source (source),
    INDEX idx_episodes_created_at (created_at)
    -- v8.1: INDEX idx_episodes_extracted_at (extracted_at)
);

-- ============================================================================
-- ENTITY_TYPES TABLE
-- ============================================================================
-- Defines custom entity types with JSON Schema validation.
-- Entity types define structured data shapes for knowledge graph entities
-- (e.g., "person", "product", "concept") with validation rules.
--
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
);

-- ============================================================================
-- RELATIONSHIP_TYPES TABLE
-- ============================================================================
-- Defines custom relationship types with JSON Schema validation.
-- Relationship types define structured data shapes for knowledge graph edges
-- (e.g., "uses", "implements", "replaces") with validation rules.
--
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
);

-- ============================================================================
-- ENTITY_EMBEDDINGS TABLE (v8.1)
-- ============================================================================
-- Stores vector embeddings for entities to support semantic/vector search.
-- Embeddings are generated by external models (e.g., text-embedding-3-small,
-- all-MiniLM-L6-v2) and stored as BLOB (serialized float32 arrays).
--
-- Added in v8.1 via migration. Optional - future feature for semantic search.
--
CREATE TABLE IF NOT EXISTS entity_embeddings (
    -- Core Identification
    entity_id VARCHAR(255) PRIMARY KEY,
    
    -- Embedding Data
    embedding BLOB,  -- Serialized float32 array (model-specific dimensionality)
    model VARCHAR(255),  -- Model name (e.g., "text-embedding-3-small")
    
    -- Timestamps
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign Key
    CONSTRAINT fk_entity_embeddings_entity FOREIGN KEY (entity_id) 
        REFERENCES entities(id) ON DELETE CASCADE
);

