# Task 1-2: Add Confidence field to Relationship type and update storage interface

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: HIGH
- **Phase**: 1
- **Depends On**: 1-1

## Files
- `e:\Projects\BeadsMemory\beads\internal\types\relationship.go` (EXISTING)
- `e:\Projects\BeadsMemory\beads\internal\storage\storage.go` (EXISTING)
- `e:\Projects\BeadsMemory\beads\internal\storage\dolt\relationships.go` (EXISTING)

## Instructions

Add `Confidence` field to the `Relationship` struct and update all storage operations to persist/query confidence values.

**Outcome:** Relationships can carry a confidence score (0.0-1.0) indicating certainty of the link. AI-extracted relationships start with lower confidence, human-curated relationships default to 1.0.

**Changes required:**

1. **In `internal/types/relationship.go`:**
   - Add `Confidence *float64 \`json:"confidence,omitempty"\`` field to `Relationship` struct
   - Make it a pointer so it's optional (omitempty in JSON, nullable in SQL)
   - Add validation helper: `func (r *Relationship) ValidateConfidence() error` (must be 0.0-1.0 if set)

2. **In `internal/storage/storage.go`:**
   - Add `MinConfidence *float64` to `RelationshipFilters` struct
   - Add `MaxConfidence *float64` to `RelationshipFilters` struct (for completeness)
   - Update interface comments to document confidence filtering

3. **In `internal/storage/dolt/relationships.go`:**
   - Update `CreateRelationship`: Add `confidence` to INSERT column list and args
   - Update `UpdateRelationship`: Add `confidence` to SET clause (if non-nil)
   - Update `SearchRelationships`: Add WHERE clauses for `MinConfidence` and `MaxConfidence` filters
   - Update `GetRelationshipsWithTemporalFilter`: Apply confidence filter if present
   - Update all SELECT statements to include `confidence` column in scan targets
   - Handle NULL confidence in SQL scans (map to nil pointer in Go)

**Key patterns to follow:**
- Use `COALESCE(confidence, 1.0)` in WHERE clauses for min/max confidence filtering (treat NULL as 1.0)
- Update SQL scan order: after `created_by`, before any new columns
- Use pointer semantics: `Confidence *float64` matches existing nullable fields like `ValidUntil *time.Time`

## Architecture Pattern

**Storage Interface Pattern** (from Architecture.md):
- All data access goes through `storage.Storage` interface
- `DoltStore` implements interface with SQL query building
- Filters use struct embedding: `RelationshipFilters` already has `Metadata map[string]interface{}`; confidence is now a first-class field (not metadata)
- Partial updates: `UpdateRelationship` only sets non-zero fields (check `if entity.Confidence != nil`)

**Type Safety**:
- Confidence is `*float64` (same pattern as `ValidUntil *time.Time`)
- JSON omitempty removes field when nil
- SQL NULL maps to nil pointer

## Validation Criteria
- [ ] `Relationship` struct has `Confidence *float64` field
- [ ] `ValidateConfidence()` method rejects values <0.0 or >1.0
- [ ] `RelationshipFilters` has `MinConfidence` and `MaxConfidence` fields
- [ ] `CreateRelationship` persists confidence to SQL
- [ ] `UpdateRelationship` handles confidence updates (nil = no change, value = update)
- [ ] `SearchRelationships` filters by min/max confidence
- [ ] All SELECT queries include confidence column
- [ ] NULL confidence in DB maps to nil in Go struct
- [ ] Existing relationships (NULL confidence) treated as 1.0 in filters
- [ ] No compilation errors after change (100+ commands depend on types package)

## Impact Analysis
- **Direct impact**: Core relationship type used everywhere (relationship_create.go, relationship_update.go, relationship_list.go, graph commands, MCP server)
- **Indirect impact**: All code that reads relationships now sees confidence field (but can ignore it)
- **Dependencies**: Task 1-8 (MCP relationship tools), Task 2-1 (CLI confidence flags), Task 3-5 (memory retrieval filtering) depend on this

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Feature 4: Relationship Confidence/Weight" section for full rationale and current struct definition)
- Architecture: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Architecture.md` (storage interface composition pattern)
- Existing filters: `internal/storage/storage.go` `RelationshipFilters` struct (follow same pattern as `ValidAt`, `Metadata` fields)

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
