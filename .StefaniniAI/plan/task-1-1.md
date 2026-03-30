# Task 1-1: Add schema columns for v8.1 (confidence, merged_into, extracted_at, embeddings table)

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: MEDIUM
- **Phase**: 1
- **Depends On**: None

## Files
- `e:\Projects\BeadsMemory\beads\internal\storage\dolt\schema_v8.sql` (EXISTING)
- `e:\Projects\BeadsMemory\beads\internal\storage\dolt\migration.go` (EXISTING)

## Instructions

Add v8.1 schema extensions to support intelligence layer features. This is an **additive migration** — no breaking changes, only new columns and one new table.

**Outcome:** The v8 schema is extended with:
1. `relationships.confidence` column (FLOAT, nullable, default 1.0) for AI-extracted relationship scoring
2. `entities.merged_into` column (VARCHAR(255), nullable) for soft-delete tracking after entity merge
3. `episodes.extracted_at` column (DATETIME, nullable) to track which episodes have been processed by LLM extraction
4. New `entity_embeddings` table (optional, for future vector search) with columns: `entity_id VARCHAR(255) PRIMARY KEY`, `embedding BLOB`, `model VARCHAR(255)`, `created_at DATETIME`

**Implementation approach:**
- Add `ALTER TABLE` statements to `schema_v8.sql` (these will run on `bd migrate` or on startup if schema is out of date)
- Use `ADD COLUMN IF NOT EXISTS` for idempotent migrations (Dolt supports this syntax)
- Add migration version tracking to `migration.go` (increment from v8 to v8.1)
- Ensure existing v8 databases can upgrade without data loss

**Key constraints:**
- All new columns must be nullable or have defaults (backward compatibility)
- `confidence` default to 1.0 (existing relationships are assumed certain)
- `entity_embeddings.embedding` is BLOB (supports float32 arrays serialized as bytes)
- Add index on `entities.merged_into` (for filtering out soft-deleted entities)
- Add index on `episodes.extracted_at` (for finding unprocessed episodes)

## Architecture Pattern

**Schema Migration Pattern** (from Architecture.md):
- Use idempotent DDL (`IF NOT EXISTS`, `ADD COLUMN IF NOT EXISTS`)
- Migration files are versioned and tracked in `migration.go`
- Dolt auto-commits all schema changes
- Follow existing pattern from v7 → v8 migration in `migrate_to_v8.go`

**Additive Schema Evolution**:
- New columns are nullable or have defaults
- No column drops, no type changes (avoid breaking existing queries)
- Index additions are safe (improve performance without schema lock)

## Validation Criteria
- [ ] `relationships` table has `confidence FLOAT NULL DEFAULT 1.0` column
- [ ] `entities` table has `merged_into VARCHAR(255) NULL` column with index
- [ ] `episodes` table has `extracted_at DATETIME NULL` column with index
- [ ] `entity_embeddings` table exists with correct schema
- [ ] Migration runs successfully on existing v8 database (test with temp DB)
- [ ] No breaking changes to existing queries (all columns nullable/defaulted)
- [ ] `bd migrate status` shows v8.1 applied
- [ ] Schema matches pattern from existing v7→v8 migration

## Impact Analysis
- **Direct impact**: Dolt schema for entities, relationships, episodes tables
- **Indirect impact**: All storage layer queries inherit new columns (but don't use them yet)
- **Dependencies**: All future tasks rely on this schema (Phase 1-2 uses confidence, Phase 2-3 uses merged_into, Phase 3 uses extracted_at)

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Feature 4: Relationship Confidence" section for confidence rationale, "Feature 3: Entity Deduplication" for merged_into, "Feature 1: Entity Extraction" for extracted_at)
- Architecture: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Architecture.md` (see "Knowledge Graph Architecture (v8)" section)
- Existing migration: `e:\Projects\BeadsMemory\beads\internal\storage\dolt\migrate_to_v8.go` (reference for migration patterns)

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
