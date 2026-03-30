# Task 1-3: Add MergeEntities method to storage interface

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: MEDIUM
- **Phase**: 1
- **Depends On**: None (schema migration 1-1 provides merged_into column, but interface can be defined independently)

## Files
- `e:\Projects\BeadsMemory\beads\internal\storage\storage.go` (EXISTING)
- `e:\Projects\BeadsMemory\beads\internal\storage\dolt\entities.go` (EXISTING)

## Instructions

Add `MergeEntities` method to the `EntityStore` interface and implement it in `DoltStore`. This enables entity deduplication by merging a source entity into a target entity.

**Outcome:** When two entities are discovered to be duplicates, `MergeEntities(ctx, sourceID, targetID)` moves all relationships from source to target, marks source as soft-deleted (`merged_into = targetID`), and records provenance.

**Interface signature (in `internal/storage/storage.go`):**
```go
// MergeEntities merges sourceEntityID into targetEntityID.
// - Moves all relationships (source and target) from source entity to target entity
// - Sets source entity's merged_into field to targetEntityID
// - Preserves metadata and summary by appending to target (optional)
// - Returns error if source or target not found, or if source already merged
MergeEntities(ctx context.Context, sourceEntityID, targetEntityID, actor string) error
```

**Implementation (in `internal/storage/dolt/entities.go`):**
1. **Validation:**
   - Verify both source and target entities exist
   - Check source is not already merged (`merged_into IS NULL`)
   - Check target is not already merged (prevent chain merges)
   
2. **Relationship migration:**
   - Update all relationships where `source_entity_id = sourceEntityID` to point to `targetEntityID`
   - Update all relationships where `target_entity_id = sourceEntityID` to point to `targetEntityID`
   - Handle duplicate relationship prevention: if source→X and target→X both exist, keep the one with higher confidence (or newer created_at if confidence equal)

3. **Soft delete source:**
   - Update source entity: `SET merged_into = ?, updated_at = ?, updated_by = ?`
   - Do NOT delete entity row (preserve provenance)

4. **Metadata merge (optional, recommended):**
   - Append source summary to target summary (separated by newline)
   - Merge metadata JSON objects (target wins on conflicts)

5. **Transaction:**
   - Wrap all operations in a single transaction (`RunInTransaction`)
   - Commit message: `"Merged entity {sourceID} into {targetID}"`

**Edge cases:**
- If source and target have relationship to same third entity with same type: keep higher confidence one, delete lower
- If both relationships have same confidence: keep newer one (by created_at)
- If source is referenced in episodes (EntitiesExtracted): update episode records to replace sourceID with targetID

## Architecture Pattern

**Repository Pattern with Transactions** (from Architecture.md):
- All multi-step writes use `RunInTransaction(ctx, commitMsg, fn)`
- Error handling: return errors, caller wraps with context
- SQL query building: use parameterized queries (`?` placeholders)

**Soft Delete Pattern**:
- Never hard-delete entities (preserve audit trail)
- Use `merged_into` column to mark entity as merged
- Queries should filter `WHERE merged_into IS NULL` to exclude merged entities (future task)

**Relationship Cascade**:
- When entity is merged, all its relationships transfer to target
- This mirrors FK cascade behavior but with merge semantics

## Validation Criteria
- [ ] `EntityStore` interface has `MergeEntities` method signature
- [ ] `DoltStore` implements `MergeEntities` in entities.go
- [ ] Method validates source and target exist
- [ ] Method rejects if source already merged
- [ ] All relationships from source transferred to target
- [ ] Duplicate relationships handled (keep higher confidence)
- [ ] Source entity's `merged_into` field set to targetID
- [ ] All operations wrapped in transaction
- [ ] Transaction commit message includes entity IDs
- [ ] Returns error if source/target not found
- [ ] Unit test covers merge operation (test file TBD in Phase 6)

## Impact Analysis
- **Direct impact**: Storage interface (used by all CLI commands)
- **Indirect impact**: Entity queries in future tasks should filter out merged entities
- **Dependencies**: Task 2-5 (entity_merge CLI command) calls this method

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Feature 3: Entity Deduplication/Resolution" section for rationale and design notes on merge behavior)
- Architecture: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Architecture.md` (transaction pattern, repository pattern)
- Existing transaction wrapper: `internal/storage/dolt/transactions.go` `RunInTransaction` method

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
