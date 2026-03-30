# Implementation Plan: Knowledge Graph Migration

## Overview
- **Goal**: Migrate beads from Epic/Task/Sub-task hierarchy to a general-purpose memory system based on knowledge graphs with entities, relationships, episodes, and custom ontologies
- **Architecture Pattern**: Repository Pattern with Interface-Based Storage (following existing `internal/storage/` design)
- **Overall Risk Level**: **HIGH**
- **Estimated Complexity**: Complex (6-8 weeks for full migration)

**Critical Context:**
- This is a **breaking architectural change** that affects the core data model
- Requires careful phased rollout with backward compatibility
- Must maintain data integrity during migration
- Will touch 30+ files across storage, CLI, types, and tests

---

## Safety Analysis

### High-Risk Areas Identified

1. **Data Loss Risk**: Migrating existing issues to new schema without data loss
2. **Breaking Changes**: CLI commands will change significantly (`bd children` → `bd relationships`)
3. **Schema Compatibility**: Dolt schema v7 → v8 migration with rollback capability
4. **Type System Overhaul**: Replacing enum-based `IssueType` with flexible `EntityType`
5. **Dependency Graph Integrity**: Converting parent-child relationships to temporal graph edges
6. **Test Coverage**: 50+ test files need updating to new data model
7. **External Integrations**: GitHub/GitLab/Jira/Linear syncs assume issue hierarchy
8. **MCP Server**: Python MCP server exposes issue-centric tools that need refactoring

### Mitigation Strategies

- **Dual-Mode Operation**: Support both old and new schemas during transition (v7 + v8)
- **Incremental Migration**: Migrate data in batches with validation checkpoints
- **Feature Flags**: Use config flags to enable/disable knowledge graph features
- **Rollback Plan**: Each phase has explicit rollback instructions
- **Comprehensive Testing**: Add integration tests before touching production schema
- **Deprecation Warnings**: Legacy commands show warnings but continue working

---

## Intent Validation & Implicit Requirements

### Explicit Requirements (from user)
✓ New tables: entities, relationships, episodes, entity_types, relationship_types
✓ Replace IssueType enum with flexible EntityType
✓ Temporal validity windows on relationships
✓ Pydantic-like schema definitions for custom types
✓ Data migration from issues → entities
✓ CLI command migration (bd create → bd entity create)
✓ Backward compatibility with deprecation warnings

### Implicit Requirements (discovered)
⚠ **Query Engine Impact**: `internal/query/` assumes issue-centric filters — needs entity-aware predicates
⚠ **UI Rendering**: `internal/ui/` formats issues — needs entity/relationship formatters
⚠ **Hook System**: Hooks trigger on issue events — needs entity lifecycle hooks
⚠ **Audit Log**: `internal/audit/` logs issue changes — needs entity event schema
⚠ **Telemetry**: OpenTelemetry spans are issue-centric — needs entity spans
⚠ **External Trackers**: GitHub/Jira sync assumes hierarchical issues — needs adapter layer
⚠ **Compaction**: AI compaction works on issues — needs entity compaction
⚠ **Molecules**: Compound issues use parent-child — needs graph-based bonding
⚠ **Federation**: Peer sync assumes issue schema — needs entity-aware sync
⚠ **Routing**: Multi-repo routing uses issue prefixes — needs entity prefixes

### Missing Edge Cases
❌ **Concurrent Writers**: How do temporal windows handle concurrent relationship updates?
❌ **Episode Pruning**: When/how do we expire old episodes? Storage impact?
❌ **Ontology Conflicts**: What happens if two repos define conflicting entity types?
❌ **Migration Failure Recovery**: What if migration fails halfway through?
❌ **Query Performance**: Will in-memory temporal validity checks scale to 10k+ relationships?

---

## Phase 1: Foundation - Type System & Schema Design

**Goal**: Define new data structures without breaking existing system

**Dependencies**: None (can start immediately)

#### Tasks

**Task 1.1**: Define Entity, Relationship, Episode Types
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**: 
  - `internal/types/entity.go` (NEW)
  - `internal/types/relationship.go` (NEW)
  - `internal/types/episode.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Introduces new core types alongside existing Issue type
  - Indirect impact: No dependencies yet (isolated new files)
  - Dependencies: None
- **Architecture Pattern**: Rich Domain Model (following existing `types.Issue` pattern)
- **Validation Criteria**:
  - [ ] Entity struct has id, entity_type, name, summary, metadata fields
  - [ ] Relationship struct has source_entity_id, relationship_type, target_entity_id, valid_from, valid_until
  - [ ] Episode struct has id, timestamp, source, raw_data, entities_extracted
  - [ ] All types compile without errors
  - [ ] JSON marshaling/unmarshaling works for all types
- **Why**: Core types must exist before schema can reference them
- **Risk Level**: LOW
- **Dependencies**: None

**Task 1.2**: Create JSON Schema Definitions for Ontology
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**:
  - `internal/types/ontology.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds ontology registry types (EntityTypeSchema, RelationshipTypeSchema)
  - Indirect impact: None yet
  - Dependencies: Task 1.1 (needs Entity/Relationship types)
- **Architecture Pattern**: Registry Pattern with JSON Schema validation
- **Validation Criteria**:
  - [ ] EntityTypeSchema has type_name and schema_json fields
  - [ ] RelationshipTypeSchema has type_name and schema_json fields
  - [ ] JSON schema validation functions work correctly
  - [ ] Can register and retrieve custom type schemas
- **Why**: Custom ontology requires schema definition and validation
- **Risk Level**: LOW
- **Dependencies**: Task 1.1

**Task 1.3**: Design Dolt SQL Schema (v8)
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**:
  - `internal/storage/dolt/schema_v8.sql` (NEW)
- **Impact Analysis**:
  - Direct impact: Defines new database tables (entities, relationships, episodes, entity_types, relationship_types)
  - Indirect impact: Existing `issues` table remains unchanged (coexistence)
  - Dependencies: Task 1.1, 1.2 (needs type definitions)
- **Architecture Pattern**: Versioned Schema Evolution (v7 → v8)
- **Validation Criteria**:
  - [ ] SQL syntax is valid MySQL DDL
  - [ ] Tables have appropriate indexes (entity_type, source_entity_id, target_entity_id, valid_from, valid_until)
  - [ ] Foreign key constraints are defined where appropriate
  - [ ] schema_version table updated to v8
  - [ ] No conflicts with existing v7 tables
- **Why**: Schema must be designed before migration logic can be written
- **Risk Level**: MEDIUM (schema design errors ripple through system)
- **Dependencies**: Task 1.1, 1.2

**Task 1.4**: Create Migration Utility Framework
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**:
  - `internal/storage/dolt/migration_v8.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds migration framework for v7 → v8
  - Indirect impact: Extends existing migration pattern in `internal/storage/dolt/migration.go`
  - Dependencies: Task 1.3 (needs schema)
- **Architecture Pattern**: Migration Framework (following existing `migration.go` pattern)
- **Validation Criteria**:
  - [ ] MigrateToV8() function exists
  - [ ] RollbackFromV8() function exists
  - [ ] Migration runs in a transaction
  - [ ] Validation checks run before and after migration
  - [ ] Progress logging for batch operations
- **Why**: Framework must exist before data migration can occur
- **Risk Level**: HIGH (migration bugs cause data loss)
- **Dependencies**: Task 1.3

#### Phase 1 Validation
- **Checkpoint**: Yes (before Phase 2)
- **Validation Required**:
  - All new types compile and pass unit tests
  - SQL schema validates against Dolt
  - Migration framework has rollback capability
  - No breaking changes to existing v7 schema
- **Rollback Trigger**:
  - If schema validation fails
  - If migration framework tests fail
  - If any compilation errors

---

## Phase 2: Storage Interface Extension

**Goal**: Extend storage interface without breaking existing implementations

**Dependencies**: Phase 1 complete

#### Tasks

**Task 2.1**: Extend Storage Interface with Entity Operations
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Files**:
  - `internal/storage/storage.go` (EXISTING - will modify)
- **Impact Analysis**:
  - Direct impact: Adds new methods to Storage interface
  - Indirect impact: All storage implementations must implement new methods (DoltStore)
  - Dependencies: Task 1.1 (needs Entity type)
- **Architecture Pattern**: Interface Composition (add EntityStore sub-interface)
- **Validation Criteria**:
  - [ ] EntityStore interface defined with CreateEntity, GetEntity, UpdateEntity, DeleteEntity, SearchEntities
  - [ ] Storage interface composes EntityStore
  - [ ] No breaking changes to existing Storage interface methods
  - [ ] Interface compiles without errors
- **Why**: Interface contract must be defined before implementation
- **Risk Level**: MEDIUM (interface changes affect all implementers)
- **Dependencies**: Task 1.1

**Task 2.2**: Extend Storage Interface with Relationship Operations
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Files**:
  - `internal/storage/storage.go` (EXISTING - will modify)
- **Impact Analysis**:
  - Direct impact: Adds RelationshipStore sub-interface
  - Indirect impact: All storage implementations must implement new methods
  - Dependencies: Task 1.2 (needs Relationship type)
- **Architecture Pattern**: Interface Composition (add RelationshipStore sub-interface)
- **Validation Criteria**:
  - [ ] RelationshipStore interface defined with CreateRelationship, GetRelationship, UpdateRelationship, DeleteRelationship, SearchRelationships, GetRelationshipsWithTemporalFilter
  - [ ] Storage interface composes RelationshipStore
  - [ ] Methods support temporal validity window queries
  - [ ] Interface compiles without errors
- **Why**: Relationship operations need their own interface contract
- **Risk Level**: MEDIUM
- **Dependencies**: Task 1.2

**Task 2.3**: Extend Storage Interface with Episode Operations
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Files**:
  - `internal/storage/storage.go` (EXISTING - will modify)
- **Impact Analysis**:
  - Direct impact: Adds EpisodeStore sub-interface
  - Indirect impact: Storage implementations must implement episode methods
  - Dependencies: Task 1.1 (needs Episode type)
- **Architecture Pattern**: Interface Composition (add EpisodeStore sub-interface)
- **Validation Criteria**:
  - [ ] EpisodeStore interface defined with CreateEpisode, GetEpisode, SearchEpisodes
  - [ ] Storage interface composes EpisodeStore
  - [ ] Methods support filtering by source, timestamp range, extracted entities
  - [ ] Interface compiles without errors
- **Why**: Episodes are the provenance layer and need their own operations
- **Risk Level**: LOW (episodes are append-only)
- **Dependencies**: Task 1.1

**Task 2.4**: Extend Storage Interface with Ontology Operations
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Files**:
  - `internal/storage/storage.go` (EXISTING - will modify)
- **Impact Analysis**:
  - Direct impact: Adds OntologyStore sub-interface
  - Indirect impact: Storage implementations must implement ontology methods
  - Dependencies: Task 1.2 (needs ontology types)
- **Architecture Pattern**: Interface Composition (add OntologyStore sub-interface)
- **Validation Criteria**:
  - [ ] OntologyStore interface defined with RegisterEntityType, RegisterRelationshipType, GetEntityTypes, GetRelationshipTypes, ValidateEntityAgainstType, ValidateRelationshipAgainstType
  - [ ] Storage interface composes OntologyStore
  - [ ] Schema validation methods exist
  - [ ] Interface compiles without errors
- **Why**: Custom type registration needs interface contract
- **Risk Level**: MEDIUM (validation logic is complex)
- **Dependencies**: Task 1.2

#### Phase 2 Validation
- **Checkpoint**: Yes (before Phase 3)
- **Validation Required**:
  - All interface extensions compile
  - No breaking changes to existing Storage interface methods
  - Documentation comments are complete
  - Interface design reviewed and approved
- **Rollback Trigger**:
  - If interface design is flawed (discovered during implementation)
  - If breaking changes are introduced

---

## Phase 3: Dolt Storage Implementation

**Goal**: Implement new interfaces in DoltStore without breaking existing functionality

**Dependencies**: Phase 2 complete

#### Tasks

**Task 3.1**: Implement EntityStore in DoltStore
- **Agent**: EditMode_Coder
- **Operation**: CREATE + MODIFY
- **Files**:
  - `internal/storage/dolt/entities.go` (NEW)
  - `internal/storage/dolt/dolt.go` (EXISTING - will modify to embed EntityStore methods)
- **Impact Analysis**:
  - Direct impact: Adds entity CRUD operations to DoltStore
  - Indirect impact: None (new functionality, no changes to existing code paths)
  - Dependencies: Task 2.1 (needs EntityStore interface), Task 1.3 (needs schema v8)
- **Architecture Pattern**: Repository Pattern with SQL query building
- **Validation Criteria**:
  - [ ] CreateEntity inserts into entities table
  - [ ] GetEntity retrieves by ID with JSON metadata parsing
  - [ ] UpdateEntity updates fields and metadata
  - [ ] DeleteEntity marks as deleted (soft delete with tombstone)
  - [ ] SearchEntities filters by entity_type, name, metadata fields
  - [ ] All operations use transactions
  - [ ] Unit tests pass
- **Why**: Entity operations are the core of the new system
- **Risk Level**: HIGH (SQL query bugs cause data corruption)
- **Dependencies**: Task 2.1, 1.3

**Task 3.2**: Implement RelationshipStore in DoltStore
- **Agent**: EditMode_Coder
- **Operation**: CREATE + MODIFY
- **Files**:
  - `internal/storage/dolt/relationships.go` (NEW)
  - `internal/storage/dolt/dolt.go` (EXISTING - will modify)
- **Impact Analysis**:
  - Direct impact: Adds relationship CRUD operations with temporal validity
  - Indirect impact: None (new functionality)
  - Dependencies: Task 2.2 (needs RelationshipStore interface), Task 1.3 (needs schema v8)
- **Architecture Pattern**: Repository Pattern with temporal query support
- **Validation Criteria**:
  - [ ] CreateRelationship inserts with valid_from, valid_until
  - [ ] GetRelationship retrieves with temporal validation
  - [ ] UpdateRelationship creates new temporal window (preserves history)
  - [ ] DeleteRelationship sets valid_until to now (soft delete)
  - [ ] SearchRelationships supports temporal filtering (active at time T)
  - [ ] GetRelationshipsWithTemporalFilter returns only valid relationships
  - [ ] Unit tests cover temporal edge cases (overlapping windows, point-in-time queries)
- **Why**: Temporal relationships are critical for knowledge graph evolution
- **Risk Level**: HIGH (temporal logic bugs cause incorrect graph traversal)
- **Dependencies**: Task 2.2, 1.3

**Task 3.3**: Implement EpisodeStore in DoltStore
- **Agent**: EditMode_Coder
- **Operation**: CREATE + MODIFY
- **Files**:
  - `internal/storage/dolt/episodes.go` (NEW)
  - `internal/storage/dolt/dolt.go` (EXISTING - will modify)
- **Impact Analysis**:
  - Direct impact: Adds episode CRUD operations (append-only provenance log)
  - Indirect impact: None (new functionality)
  - Dependencies: Task 2.3 (needs EpisodeStore interface), Task 1.3 (needs schema v8)
- **Architecture Pattern**: Append-Only Log with provenance tracking
- **Validation Criteria**:
  - [ ] CreateEpisode inserts with timestamp, source, raw_data BLOB
  - [ ] GetEpisode retrieves by ID
  - [ ] SearchEpisodes filters by source, timestamp range, extracted entities
  - [ ] Episodes are immutable (no update or delete)
  - [ ] Unit tests pass
- **Why**: Episodes are the ground truth provenance layer
- **Risk Level**: LOW (append-only, simple operations)
- **Dependencies**: Task 2.3, 1.3

**Task 3.4**: Implement OntologyStore in DoltStore
- **Agent**: EditMode_Coder
- **Operation**: CREATE + MODIFY
- **Files**:
  - `internal/storage/dolt/ontology.go` (NEW)
  - `internal/storage/dolt/dolt.go` (EXISTING - will modify)
- **Impact Analysis**:
  - Direct impact: Adds custom type registration and validation
  - Indirect impact: Entity/Relationship operations must validate against schemas
  - Dependencies: Task 2.4 (needs OntologyStore interface), Task 1.3 (needs schema v8)
- **Architecture Pattern**: Registry Pattern with JSON Schema validation
- **Validation Criteria**:
  - [ ] RegisterEntityType inserts into entity_types table with schema_json
  - [ ] RegisterRelationshipType inserts into relationship_types table
  - [ ] GetEntityTypes retrieves all registered entity types
  - [ ] GetRelationshipTypes retrieves all registered relationship types
  - [ ] ValidateEntityAgainstType validates JSON against schema
  - [ ] ValidateRelationshipAgainstType validates JSON against schema
  - [ ] Unit tests cover schema validation edge cases
- **Why**: Custom ontology enables flexible domain modeling
- **Risk Level**: MEDIUM (validation logic is complex)
- **Dependencies**: Task 2.4, 1.3

**Task 3.5**: Implement Schema v8 Migration Logic
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Files**:
  - `internal/storage/dolt/migration_v8.go` (EXISTING - will modify from Task 1.4)
- **Impact Analysis**:
  - Direct impact: Adds logic to migrate issues → entities, dependencies → relationships
  - Indirect impact: Creates episodes from events table
  - Dependencies: Task 1.4 (needs migration framework), Task 3.1-3.4 (needs implementations)
- **Architecture Pattern**: Batch Migration with Checkpointing
- **Validation Criteria**:
  - [ ] MigrateToV8() converts all issues to entities (preserving all data)
  - [ ] Dependencies convert to relationships with temporal windows (valid_from = created_at, valid_until = NULL)
  - [ ] Events convert to episodes (with issue_id as extracted entity)
  - [ ] Parent-child dependencies create temporal relationships
  - [ ] Migration runs in batches (1000 rows at a time)
  - [ ] Progress logging every batch
  - [ ] Validation checks after migration (row counts match, no data loss)
  - [ ] RollbackFromV8() restores v7 schema
  - [ ] Integration tests pass
- **Why**: Data migration is the critical path for cutover
- **Risk Level**: **CRITICAL** (data loss risk if migration fails)
- **Dependencies**: Task 1.4, 3.1, 3.2, 3.3, 3.4

#### Phase 3 Validation
- **Checkpoint**: **YES** (critical checkpoint - validate before Phase 4)
- **Validation Required**:
  - All storage implementations pass unit tests
  - Integration tests with real Dolt database pass
  - Migration script runs successfully on test database
  - No data loss in migration (checksum validation)
  - Rollback restores v7 schema correctly
  - Performance benchmarks meet targets (same or better than v7)
- **Rollback Trigger**:
  - If any storage operation fails tests
  - If migration causes data loss
  - If performance degrades significantly
  - If rollback fails

---

## Phase 4: CLI Command Migration

**Goal**: Add new CLI commands and deprecate old ones gracefully

**Dependencies**: Phase 3 complete

#### Tasks

**Task 4.1**: Create bd entity Commands
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**:
  - `cmd/bd/entity.go` (NEW)
  - `cmd/bd/entity_create.go` (NEW)
  - `cmd/bd/entity_list.go` (NEW)
  - `cmd/bd/entity_show.go` (NEW)
  - `cmd/bd/entity_update.go` (NEW)
  - `cmd/bd/entity_delete.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds new entity management commands
  - Indirect impact: None (new commands don't affect existing commands)
  - Dependencies: Task 3.1 (needs EntityStore implementation)
- **Architecture Pattern**: Cobra Command Pattern (following existing cmd structure)
- **Validation Criteria**:
  - [ ] bd entity create --entity-type <type> --name <name> --description <desc> --json
  - [ ] bd entity list --entity-type <type> --json
  - [ ] bd entity show <id> --json
  - [ ] bd entity update <id> --name <name> --summary <summary> --json
  - [ ] bd entity delete <id> --json
  - [ ] All commands support --json flag
  - [ ] Commands validate input (entity_type registered, schema validation)
  - [ ] Unit tests pass
- **Why**: Entity commands are the new primary interface
- **Risk Level**: LOW (new commands, no breaking changes)
- **Dependencies**: Task 3.1

**Task 4.2**: Create bd relationship Commands
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**:
  - `cmd/bd/relationship.go` (NEW)
  - `cmd/bd/relationship_create.go` (NEW)
  - `cmd/bd/relationship_list.go` (NEW)
  - `cmd/bd/relationship_show.go` (NEW)
  - `cmd/bd/relationship_update.go` (NEW)
  - `cmd/bd/relationship_delete.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds relationship management commands
  - Indirect impact: None
  - Dependencies: Task 3.2 (needs RelationshipStore implementation)
- **Architecture Pattern**: Cobra Command Pattern
- **Validation Criteria**:
  - [ ] bd relationship create --from <entity-id> --type <rel-type> --to <entity-id> --valid-from <time> --valid-until <time> --json
  - [ ] bd relationship list --from <entity-id> --json (shows outgoing relationships)
  - [ ] bd relationship list --to <entity-id> --json (shows incoming relationships)
  - [ ] bd relationship show <id> --json
  - [ ] bd relationship update <id> --valid-until <time> --json (closes temporal window)
  - [ ] bd relationship delete <id> --json (sets valid_until to now)
  - [ ] All commands support --json flag
  - [ ] Temporal filtering works correctly
  - [ ] Unit tests pass
- **Why**: Relationship commands enable graph exploration
- **Risk Level**: MEDIUM (temporal query logic is complex)
- **Dependencies**: Task 3.2

**Task 4.3**: Create bd episode Commands
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**:
  - `cmd/bd/episode.go` (NEW)
  - `cmd/bd/episode_create.go` (NEW)
  - `cmd/bd/episode_list.go` (NEW)
  - `cmd/bd/episode_show.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds episode management commands
  - Indirect impact: None
  - Dependencies: Task 3.3 (needs EpisodeStore implementation)
- **Architecture Pattern**: Cobra Command Pattern
- **Validation Criteria**:
  - [ ] bd episode create --source <source> --file <raw-data-file> --json
  - [ ] bd episode list --source <source> --since <time> --json
  - [ ] bd episode show <id> --json
  - [ ] Commands support --json flag
  - [ ] Unit tests pass
- **Why**: Episodes are the provenance layer
- **Risk Level**: LOW (simple append-only operations)
- **Dependencies**: Task 3.3

**Task 4.4**: Create bd ontology Commands
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**:
  - `cmd/bd/ontology.go` (NEW)
  - `cmd/bd/ontology_register_entity_type.go` (NEW)
  - `cmd/bd/ontology_register_relationship_type.go` (NEW)
  - `cmd/bd/ontology_list.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds custom type registration commands
  - Indirect impact: None
  - Dependencies: Task 3.4 (needs OntologyStore implementation)
- **Architecture Pattern**: Cobra Command Pattern
- **Validation Criteria**:
  - [ ] bd ontology register-entity-type --name <type> --schema <json-schema-file> --json
  - [ ] bd ontology register-relationship-type --name <type> --schema <json-schema-file> --json
  - [ ] bd ontology list --json (shows all entity and relationship types)
  - [ ] Commands validate JSON schema syntax
  - [ ] Unit tests pass
- **Why**: Custom ontology enables domain-specific modeling
- **Risk Level**: LOW
- **Dependencies**: Task 3.4

**Task 4.5**: Add Deprecation Warnings to Legacy Commands
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Files**:
  - `cmd/bd/create.go` (EXISTING - will modify)
  - `cmd/bd/children.go` (EXISTING - will modify)
  - `cmd/bd/update.go` (EXISTING - will modify)
- **Impact Analysis**:
  - Direct impact: Adds deprecation warnings to stdout (not stderr, to preserve --json output)
  - Indirect impact: None (commands still work)
  - Dependencies: Task 4.1, 4.2 (so we can point users to new commands)
- **Architecture Pattern**: Graceful Deprecation
- **Validation Criteria**:
  - [ ] bd create shows warning: "This command is deprecated. Use 'bd entity create' instead."
  - [ ] bd children shows warning: "This command is deprecated. Use 'bd relationship list' instead."
  - [ ] Warnings appear only in human-readable mode (not with --json flag)
  - [ ] Commands still function correctly
  - [ ] Unit tests pass
- **Why**: Users need migration guidance
- **Risk Level**: LOW
- **Dependencies**: Task 4.1, 4.2

**Task 4.6**: Create bd graph Commands (Graph Exploration)
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**:
  - `cmd/bd/graph.go` (NEW)
  - `cmd/bd/graph_explore.go` (NEW)
  - `cmd/bd/graph_traverse.go` (NEW)
  - `cmd/bd/graph_visualize.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds graph exploration commands
  - Indirect impact: None
  - Dependencies: Task 3.2 (needs RelationshipStore), Task 3.1 (needs EntityStore)
- **Architecture Pattern**: Graph Traversal Algorithms
- **Validation Criteria**:
  - [ ] bd graph explore <entity-id> --depth <n> --json (BFS/DFS traversal)
  - [ ] bd graph traverse <from-id> <to-id> --json (shortest path)
  - [ ] bd graph visualize <entity-id> --format dot (Graphviz output)
  - [ ] Commands respect temporal validity windows
  - [ ] Unit tests pass
- **Why**: Graph exploration is the killer feature
- **Risk Level**: MEDIUM (graph algorithms are complex)
- **Dependencies**: Task 3.1, 3.2

#### Phase 4 Validation
- **Checkpoint**: Yes (before Phase 5)
- **Validation Required**:
  - All new commands compile and pass unit tests
  - --json output conforms to spec
  - Deprecation warnings display correctly
  - Graph traversal algorithms are correct
  - No breaking changes to existing commands
- **Rollback Trigger**:
  - If commands break existing workflows
  - If graph algorithms are incorrect

---

## Phase 5: Data Migration Tools & Cutover

**Goal**: Provide tools for safe data migration and cutover to v8

**Dependencies**: Phase 4 complete

#### Tasks

**Task 5.1**: Create bd migrate Command
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**:
  - `cmd/bd/migrate.go` (NEW)
  - `cmd/bd/migrate_to_v8.go` (NEW)
  - `cmd/bd/migrate_rollback.go` (NEW)
  - `cmd/bd/migrate_status.go` (NEW)
  - `cmd/bd/migrate_validate.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds migration orchestration commands
  - Indirect impact: Uses Task 3.5 migration logic
  - Dependencies: Task 3.5 (needs migration SQL)
- **Architecture Pattern**: Migration Orchestration with Safety Checks
- **Validation Criteria**:
  - [ ] bd migrate to-v8 --dry-run (preview changes without applying)
  - [ ] bd migrate to-v8 (runs migration with progress bar)
  - [ ] bd migrate rollback (reverts to v7)
  - [ ] bd migrate status (shows current schema version)
  - [ ] bd migrate validate (checksums data integrity)
  - [ ] Commands use transactions
  - [ ] Progress logging every 1000 rows
  - [ ] Unit tests pass
- **Why**: Users need a safe migration path
- **Risk Level**: **CRITICAL** (orchestrates data migration)
- **Dependencies**: Task 3.5

**Task 5.2**: Create bd compat Command (Dual-Mode Toggle)
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**:
  - `cmd/bd/compat.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds config flag to toggle v7/v8 mode
  - Indirect impact: All commands must respect compatibility mode
  - Dependencies: Task 3.5 (migration must be complete before toggle)
- **Architecture Pattern**: Feature Flag Configuration
- **Validation Criteria**:
  - [ ] bd compat set v7 (uses old schema)
  - [ ] bd compat set v8 (uses new schema)
  - [ ] bd compat status (shows current mode)
  - [ ] Config persists to metadata.json
  - [ ] All CLI commands respect compat mode
  - [ ] Unit tests pass
- **Why**: Enables gradual cutover with rollback capability
- **Risk Level**: HIGH (affects all operations)
- **Dependencies**: Task 3.5

**Task 5.3**: Update Data Migration Script Documentation
- **Agent**: EditMode_Designer
- **Operation**: CREATE
- **Files**:
  - `docs/MIGRATION_V8.md` (NEW)
- **Impact Analysis**:
  - Direct impact: Documents migration process
  - Indirect impact: None
  - Dependencies: Task 5.1, 5.2
- **Architecture Pattern**: Migration Runbook
- **Validation Criteria**:
  - [ ] Step-by-step migration instructions
  - [ ] Rollback procedures documented
  - [ ] Common errors and solutions
  - [ ] Checklist for pre-migration validation
  - [ ] Post-migration verification steps
- **Why**: Users and operators need clear guidance
- **Risk Level**: LOW (documentation only)
- **Dependencies**: Task 5.1, 5.2

**Task 5.4**: Create Migration Integration Tests
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**:
  - `internal/storage/dolt/migration_v8_test.go` (NEW)
  - `tests/migration_integration_test.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds comprehensive migration tests
  - Indirect impact: None
  - Dependencies: Task 3.5, 5.1
- **Architecture Pattern**: Integration Testing with Real Dolt
- **Validation Criteria**:
  - [ ] Tests migrate sample database with 100+ issues
  - [ ] Validates no data loss (checksums match)
  - [ ] Tests rollback restores original state
  - [ ] Tests temporal relationship validity
  - [ ] Tests custom ontology registration
  - [ ] All tests pass
- **Why**: Migration must be thoroughly tested before production use
- **Risk Level**: **CRITICAL** (catches migration bugs)
- **Dependencies**: Task 3.5, 5.1

#### Phase 5 Validation
- **Checkpoint**: **YES** (critical checkpoint before production cutover)
- **Validation Required**:
  - Migration tests pass on 1000+ row database
  - Rollback restores original state perfectly
  - Checksums validate no data loss
  - Documentation is complete and accurate
  - Dry-run mode works correctly
- **Rollback Trigger**:
  - If migration tests fail
  - If data loss detected
  - If rollback fails

---

## Phase 6: Tests, Validation & Documentation

**Goal**: Update all tests, validate system integrity, update documentation

**Dependencies**: Phase 5 complete

#### Tasks

**Task 6.1**: Update Storage Layer Tests
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Files**:
  - `internal/storage/dolt/dolt_test.go` (EXISTING - will modify)
  - `internal/storage/dolt/entities_test.go` (NEW)
  - `internal/storage/dolt/relationships_test.go` (NEW)
  - `internal/storage/dolt/episodes_test.go` (NEW)
  - `internal/storage/dolt/ontology_test.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds tests for new storage operations
  - Indirect impact: Existing issue tests continue to work (v7 compat)
  - Dependencies: Phase 3 (needs storage implementations)
- **Architecture Pattern**: Table-Driven Tests (following existing pattern)
- **Validation Criteria**:
  - [ ] Entity CRUD operations covered
  - [ ] Relationship temporal queries covered
  - [ ] Episode append-only operations covered
  - [ ] Ontology registration and validation covered
  - [ ] All tests use t.TempDir() for isolation
  - [ ] Test coverage >80% for new code
  - [ ] All tests pass
- **Why**: Storage layer is the foundation - must be rock solid
- **Risk Level**: LOW (test code)
- **Dependencies**: Phase 3

**Task 6.2**: Update CLI Command Tests
- **Agent**: EditMode_Coder
- **Operation**: MODIFY + CREATE
- **Files**:
  - `cmd/bd/entity_test.go` (NEW)
  - `cmd/bd/relationship_test.go` (NEW)
  - `cmd/bd/episode_test.go` (NEW)
  - `cmd/bd/ontology_test.go` (NEW)
  - `cmd/bd/graph_test.go` (NEW)
  - `cmd/bd/migrate_test.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds tests for new CLI commands
  - Indirect impact: None
  - Dependencies: Phase 4, 5 (needs CLI commands)
- **Architecture Pattern**: CLI Integration Tests
- **Validation Criteria**:
  - [ ] All entity commands tested
  - [ ] All relationship commands tested (including temporal queries)
  - [ ] All episode commands tested
  - [ ] All ontology commands tested
  - [ ] Graph exploration commands tested
  - [ ] Migration commands tested
  - [ ] All tests use --json output for validation
  - [ ] All tests pass
- **Why**: CLI is the user interface - must work correctly
- **Risk Level**: LOW (test code)
- **Dependencies**: Phase 4, 5

**Task 6.3**: Update Type System Tests
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Files**:
  - `internal/types/entity_test.go` (NEW)
  - `internal/types/relationship_test.go` (NEW)
  - `internal/types/episode_test.go` (NEW)
  - `internal/types/ontology_test.go` (NEW)
- **Impact Analysis**:
  - Direct impact: Adds tests for new types
  - Indirect impact: None
  - Dependencies: Phase 1 (needs type definitions)
- **Architecture Pattern**: Unit Tests
- **Validation Criteria**:
  - [ ] Entity JSON marshaling/unmarshaling tested
  - [ ] Relationship temporal validity tested
  - [ ] Episode immutability tested
  - [ ] Ontology schema validation tested
  - [ ] All tests pass
- **Why**: Types are the contract between layers
- **Risk Level**: LOW (test code)
- **Dependencies**: Phase 1

**Task 6.4**: Update Architecture.md
- **Agent**: EditMode_Designer
- **Operation**: MODIFY
- **Files**:
  - `.StefaniniAI/Architecture.md` (EXISTING - will modify)
- **Impact Analysis**:
  - Direct impact: Documents new knowledge graph architecture
  - Indirect impact: None
  - Dependencies: All phases (architecture is now finalized)
- **Architecture Pattern**: Architecture Documentation
- **Validation Criteria**:
  - [ ] Knowledge graph architecture section added
  - [ ] Entity/Relationship/Episode data model documented
  - [ ] Temporal validity windows explained
  - [ ] Custom ontology system documented
  - [ ] Migration strategy documented
  - [ ] Updated architecture diagram
  - [ ] Architectural Evolution section added (documents pattern shift from hierarchical to graph)
- **Why**: Architecture.md is the source of truth for AI agents
- **Risk Level**: LOW (documentation only)
- **Dependencies**: All phases

**Task 6.5**: Update CLI_REFERENCE.md
- **Agent**: EditMode_Designer
- **Operation**: MODIFY
- **Files**:
  - `docs/CLI_REFERENCE.md` (EXISTING - will modify)
- **Impact Analysis**:
  - Direct impact: Documents new CLI commands
  - Indirect impact: Marks old commands as deprecated
  - Dependencies: Phase 4 (needs CLI commands)
- **Architecture Pattern**: Reference Documentation
- **Validation Criteria**:
  - [ ] All entity commands documented
  - [ ] All relationship commands documented
  - [ ] All episode commands documented
  - [ ] All ontology commands documented
  - [ ] All graph commands documented
  - [ ] Migration commands documented
  - [ ] Deprecation notices on old commands
  - [ ] Examples provided for all commands
- **Why**: Users need up-to-date command reference
- **Risk Level**: LOW (documentation only)
- **Dependencies**: Phase 4

**Task 6.6**: Update AGENTS.md / AGENT_INSTRUCTIONS.md
- **Agent**: EditMode_Designer
- **Operation**: MODIFY
- **Files**:
  - `AGENTS.md` (EXISTING - will modify)
  - `AGENT_INSTRUCTIONS.md` (EXISTING - will modify)
- **Impact Analysis**:
  - Direct impact: Updates AI agent instructions for knowledge graph
  - Indirect impact: None
  - Dependencies: Phase 4 (needs CLI commands)
- **Architecture Pattern**: Agent Documentation
- **Validation Criteria**:
  - [ ] Knowledge graph workflow documented
  - [ ] Entity/relationship command examples
  - [ ] Migration guidance for agents
  - [ ] Updated "use bd for ALL tracking" section
  - [ ] Examples use new commands
- **Why**: AI agents need guidance on knowledge graph usage
- **Risk Level**: LOW (documentation only)
- **Dependencies**: Phase 4

**Task 6.7**: Update MCP Server for Knowledge Graph
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Files**:
  - `integrations/beads-mcp/src/beads_mcp/tools.py` (EXISTING - will modify)
  - `integrations/beads-mcp/src/beads_mcp/models.py` (EXISTING - will modify)
- **Impact Analysis**:
  - Direct impact: Adds entity/relationship tools to MCP server
  - Indirect impact: Deprecates issue-centric tools
  - Dependencies: Phase 4 (needs CLI commands)
- **Architecture Pattern**: MCP Tool Wrappers
- **Validation Criteria**:
  - [ ] entity_create, entity_list, entity_show, entity_update tools added
  - [ ] relationship_create, relationship_list, relationship_show tools added
  - [ ] episode_create, episode_list tools added
  - [ ] graph_explore, graph_traverse tools added
  - [ ] Old tools marked as deprecated
  - [ ] Pydantic models updated
  - [ ] Unit tests pass
- **Why**: AI assistants use MCP server for beads access
- **Risk Level**: MEDIUM (breaks AI agent workflows temporarily)
- **Dependencies**: Phase 4

#### Phase 6 Validation
- **Checkpoint**: Yes (final validation before production)
- **Validation Required**:
  - All tests pass (unit + integration + CLI)
  - Test coverage >80% for new code
  - Documentation is complete and accurate
  - MCP server works with new commands
  - No breaking changes to existing workflows (without deprecation warnings)
- **Rollback Trigger**:
  - If test coverage is insufficient
  - If documentation is incomplete

---

## Safety Checkpoints

1. **After Phase 1**: Type system and schema design validated
   - Validation: All types compile, schema validates against Dolt
   - Reason: Foundation must be correct before building on it

2. **After Phase 3**: Storage implementation and migration tested
   - Validation: Migration tests pass on 1000+ row database, no data loss
   - Reason: This is the most critical phase - data migration can cause irreversible damage

3. **After Phase 5**: Migration tools validated in production-like environment
   - Validation: Dry-run migration on production clone succeeds, rollback works
   - Reason: Must validate migration works on real data before production cutover

---

## Rollback Strategy

**If Phase 1 fails:**
1. Revert files: Delete all new files in `internal/types/` (entity.go, relationship.go, episode.go, ontology.go)
2. Restore state: No state to restore (no database changes)
3. Re-run validation from: Beginning
4. Alternative approach: Simplify ontology system (remove custom type schemas)

**If Phase 2 fails:**
1. Revert files: Revert `internal/storage/storage.go` to v7 interface
2. Restore state: No database changes yet
3. Re-run validation from: Phase 2 design review
4. Alternative approach: Split interfaces into separate files instead of sub-interfaces

**If Phase 3 fails:**
1. Revert files: Delete `internal/storage/dolt/entities.go`, `relationships.go`, `episodes.go`, `ontology.go`, `migration_v8.go`
2. Restore state: Run RollbackFromV8() to revert schema to v7
3. Re-run validation from: Phase 3 storage implementation
4. Alternative approach: Implement read-only operations first, writes later

**If Phase 4 fails:**
1. Revert files: Delete all `cmd/bd/entity*.go`, `cmd/bd/relationship*.go`, etc.
2. Restore state: No database changes (commands don't run)
3. Re-run validation from: Phase 4 CLI commands
4. Alternative approach: Simplify CLI (fewer commands, more flags on existing commands)

**If Phase 5 fails:**
1. Revert files: Delete `cmd/bd/migrate*.go`, `cmd/bd/compat.go`
2. Restore state: Run `bd migrate rollback` to revert to v7
3. Re-run validation from: Phase 5 migration tools
4. Alternative approach: Manual migration scripts instead of CLI orchestration

**If Phase 6 fails:**
1. Revert files: No code changes (tests and docs)
2. Restore state: No state changes
3. Re-run validation from: Failed tests
4. Alternative approach: Ship with lower test coverage, backfill tests later (NOT RECOMMENDED)

---

## Architecture Conformance

**Current Patterns Followed**:
- **Repository Pattern**: New storage implementations follow existing `internal/storage/dolt/` pattern
- **Interface Composition**: Storage interface uses sub-interfaces (EntityStore, RelationshipStore, etc.)
- **Versioned Schema Evolution**: Schema v7 → v8 with migration framework
- **Cobra Command Pattern**: New CLI commands follow existing `cmd/bd/` structure
- **Rich Domain Model**: New types (Entity, Relationship, Episode) follow existing `types.Issue` pattern
- **Transaction Pattern**: All writes use `RunInTransaction()` for atomicity
- **Graceful Deprecation**: Old commands continue working with warnings

**Architectural Evolution**:
- **Evolved Patterns**: 
  - **Hierarchical Issue Model → Knowledge Graph**: Replacing parent-child hierarchy with temporal graph relationships
  - **Enum-Based Type System → Flexible Ontology**: Replacing `IssueType` enum with custom entity/relationship types
  - **Simple Dependencies → Temporal Relationships**: Adding valid_from/valid_until to relationships
  - **Event Log → Episode Provenance**: Replacing events table with episode-based provenance
- **New Patterns Introduced**:
  - **Temporal Validity Windows**: Relationships have time-bounded validity
  - **Custom Ontology Registry**: Developer-defined entity and relationship types
  - **Episode Provenance**: Immutable append-only log of raw data
  - **Graph Traversal**: BFS/DFS exploration with temporal constraints
  - **JSON Schema Validation**: Pydantic-like schemas for custom types
- **Architecture.md Update Required**: **YES**
- **Update Focus**:
  - Core data model (Entity, Relationship, Episode)
  - Temporal validity and knowledge graph evolution
  - Custom ontology system
  - Migration strategy and compatibility modes
  - Updated architecture diagrams showing graph topology
  - New command groups (entity, relationship, episode, ontology, graph)

**Boundaries Respected**:
- **Storage Abstraction**: All database access goes through `storage.Storage` interface
- **CLI Independence**: CLI commands don't directly access Dolt (use storage layer)
- **Type Safety**: New types use Go structs with validation
- **Transaction Boundaries**: All multi-step writes use transactions
- **Backward Compatibility**: v7 schema remains operational during transition

---

## Risk Summary

**High Risk Tasks**: 1.4, 3.1, 3.2, 3.5, 5.1, 5.2 (9 total)
**Medium Risk Tasks**: 1.3, 2.1, 2.2, 2.4, 3.4, 4.2, 4.6, 6.7 (8 total)
**Low Risk Tasks**: 1.1, 1.2, 2.3, 3.3, 4.1, 4.3, 4.4, 4.5, 5.3, 6.1, 6.2, 6.3, 6.4, 6.5, 6.6 (15 total)

**Mitigation**:
- **Data Loss Prevention**: All high-risk tasks have rollback procedures
- **Phased Rollout**: Each phase has validation checkpoint before proceeding
- **Dual-Mode Operation**: v7 and v8 schemas coexist during transition
- **Testing Strategy**: 1000+ row migration tests before production
- **Dry-Run Mode**: Migration preview without applying changes
- **Progress Logging**: Batch operations log progress every 1000 rows
- **Checksum Validation**: Data integrity validation after migration

**Known Unknowns**:
- **Concurrent Writers**: Behavior of temporal windows under concurrent updates not fully specified
  - Investigation needed: Implement optimistic locking or last-write-wins?
- **Episode Pruning**: No strategy defined for expiring old episodes
  - Investigation needed: Time-based TTL? Size-based compaction?
- **Ontology Conflicts**: How to handle conflicting entity type definitions across repos
  - Investigation needed: Namespace isolation? Merge strategies?
- **Query Performance**: In-memory temporal filtering may not scale to 100k+ relationships
  - Investigation needed: Add SQL temporal indexes? Materialize active relationships?
- **External Tracker Sync**: GitHub/Jira assume hierarchical issues
  - Investigation needed: Map entity graph to issue hierarchy? Disable sync during migration?
- **Federation**: Peer sync assumes v7 schema
  - Investigation needed: Version negotiation protocol? Dual-schema federation?
- **Compaction**: AI compaction works on issues
  - Investigation needed: Extend to entities? Separate compaction logic?

---

## Estimated Timeline

- **Phase 1**: 1 week (foundation, low risk)
- **Phase 2**: 3 days (interface design)
- **Phase 3**: 2-3 weeks (storage implementation, migration logic - HIGH RISK)
- **Phase 4**: 1-2 weeks (CLI commands)
- **Phase 5**: 1 week (migration tools, validation - CRITICAL)
- **Phase 6**: 1 week (tests, documentation)

**Total**: 6-8 weeks for full migration

---

## Success Criteria

- [ ] All phases complete with validation checkpoints passed
- [ ] Migration tests pass on 1000+ row database with 0% data loss
- [ ] Rollback restores v7 schema perfectly
- [ ] All new CLI commands work with --json flag
- [ ] Deprecation warnings guide users to new commands
- [ ] Test coverage >80% for new code
- [ ] Documentation updated (Architecture.md, CLI_REFERENCE.md, AGENTS.md, MIGRATION_V8.md)
- [ ] MCP server updated with entity/relationship tools
- [ ] Production migration dry-run succeeds on clone of production database
- [ ] Performance benchmarks meet or exceed v7 (same query latency, same storage size)

---

## Final Notes

This is a **major architectural transformation** that fundamentally changes how beads models data. The migration from a hierarchical issue tracker to a knowledge graph with temporal relationships and custom ontologies is complex and high-risk.

**Critical Success Factors:**
1. **Thorough Testing**: Migration MUST be tested on large databases (1000+ rows) before production
2. **Rollback Capability**: Every phase must have a tested rollback procedure
3. **Incremental Cutover**: Use dual-mode operation (v7 + v8) to allow gradual migration
4. **Data Integrity**: Checksum validation after every migration step
5. **User Communication**: Clear documentation and deprecation warnings

**Recommended Approach:**
- Start with Phase 1-2 (low risk, no database changes)
- Get design review and approval before Phase 3
- Implement Phase 3 with extensive testing (this is the critical path)
- Validate rollback works before proceeding to Phase 4
- Use feature flags for gradual rollout to users

**Post-Migration Work (Not Included in This Plan):**
- External tracker adapters (GitHub, Jira, etc.) need entity mapping logic
- Federation protocol needs versioning for v7/v8 peers
- Compaction system needs entity-aware summarization
- Query engine needs optimization for temporal filtering at scale
- UI enhancements for graph visualization
