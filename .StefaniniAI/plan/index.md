# Plan Index

## Overview
- **Goal**: Add 6 intelligence features to beads knowledge graph: entity extraction, semantic search, deduplication, relationship confidence, memory retrieval, and MCP integration
- **Architecture Pattern**: Knowledge Graph Intelligence Layer (v8 extension) - builds on existing entity/relationship/episode storage with LLM-powered extraction, vector search, and graph traversal
- **Overall Risk Level**: MEDIUM
- **Estimated Complexity**: Complex (6 features, 3 phases, ~30 tasks, schema migration, new packages, MCP overhaul)
- **Architecture.md Update Required**: YES - document intelligence layer patterns, LLM extraction pipeline, vector search strategy, temporal context retrieval
- **Update Focus**: Add "Intelligence Layer" section documenting extraction pipeline, semantic search (in-memory cosine fallback), deduplication algorithm (Jaccard/cosine), confidence scoring, memory retrieval query patterns

---

## Phases & Tasks

### Phase 1: MCP Server Foundation & Schema Migration
**Phase Dependencies**: None (entry point)
**Checkpoint After Phase**: YES
**Rollback Trigger**: MCP tests fail, schema migration fails, Python imports break

| Task ID | Description | Agent | Operation | Files | Risk | Depends On | Task File |
|---------|-------------|-------|-----------|-------|------|------------|-----------|
| 1-1 | Add schema columns for v8.1 (confidence, merged_into, extracted_at, embeddings table) | EditMode_Coder | MODIFY | internal/storage/dolt/schema_v8.sql, internal/storage/dolt/migration.go | MEDIUM | None | task-1-1.md |
| 1-2 | Add Confidence field to Relationship type and update storage interface | EditMode_Coder | MODIFY | internal/types/relationship.go, internal/storage/storage.go, internal/storage/dolt/relationships.go | HIGH | 1-1 | task-1-2.md |
| 1-3 | Add MergeEntities method to storage interface | EditMode_Coder | MODIFY | internal/storage/storage.go, internal/storage/dolt/entities.go | MEDIUM | None | task-1-3.md |
| 1-4 | Add Pydantic models for Entity, Relationship, Episode, EntityType, RelationshipType | EditMode_Coder | MODIFY | integrations/beads-mcp/src/beads_mcp/models.py | LOW | None | task-1-4.md |
| 1-5 | Add abstract methods to BdClientBase for entity operations | EditMode_Coder | MODIFY | integrations/beads-mcp/src/beads_mcp/bd_client.py | MEDIUM | 1-4 | task-1-5.md |
| 1-6 | Implement entity/relationship/episode CLI wrappers in CliClient | EditMode_Coder | MODIFY | integrations/beads-mcp/src/beads_mcp/bd_client.py | MEDIUM | 1-4, 1-5 | task-1-6.md |
| 1-7 | Add MCP tools for entity CRUD operations | EditMode_Coder | MODIFY | integrations/beads-mcp/src/beads_mcp/tools.py | MEDIUM | 1-4, 1-6 | task-1-7.md |
| 1-8 | Add MCP tools for relationship operations (with confidence support) | EditMode_Coder | MODIFY | integrations/beads-mcp/src/beads_mcp/tools.py | MEDIUM | 1-4, 1-6, 1-2 | task-1-8.md |
| 1-9 | Add MCP tools for episode operations | EditMode_Coder | MODIFY | integrations/beads-mcp/src/beads_mcp/tools.py | LOW | 1-4, 1-6 | task-1-9.md |
| 1-10 | Add MCP tools for ontology operations | EditMode_Coder | MODIFY | integrations/beads-mcp/src/beads_mcp/tools.py | LOW | 1-4, 1-6 | task-1-10.md |
| 1-11 | Mark old issue-centric MCP tools as deprecated | EditMode_Coder | MODIFY | integrations/beads-mcp/src/beads_mcp/tools.py | LOW | None | task-1-11.md |
| 1-12 | Add Python tests for new MCP entity/relationship tools | EditMode_Coder | CREATE | integrations/beads-mcp/tests/test_entity_tools.py, integrations/beads-mcp/tests/test_relationship_tools.py | LOW | 1-7, 1-8 | task-1-12.md |

---

### Phase 2: Confidence, Deduplication & Semantic Search
**Phase Dependencies**: Phase 1 (needs schema migration, confidence field, MergeEntities interface)
**Checkpoint After Phase**: YES
**Rollback Trigger**: Performance degrades, deduplication produces false positives, semantic search returns irrelevant results

| Task ID | Description | Agent | Operation | Files | Risk | Depends On | Task File |
|---------|-------------|-------|-----------|-------|------|------------|-----------|
| 2-1 | Add CLI flags for relationship confidence (create/update/list) | EditMode_Coder | MODIFY | cmd/bd/relationship_create.go, cmd/bd/relationship_update.go, cmd/bd/relationship_list.go | LOW | 1-2 | task-2-1.md |
| 2-2 | Extract similarity functions to reusable package | EditMode_Coder | CREATE | internal/similarity/similarity.go, internal/similarity/similarity_test.go | LOW | None | task-2-2.md |
| 2-3 | Create entity deduplication package with Jaccard/cosine algorithms | EditMode_Coder | CREATE | internal/dedup/entity.go, internal/dedup/entity_test.go | MEDIUM | 2-2 | task-2-3.md |
| 2-4 | Add CLI command for finding duplicate entities | EditMode_Coder | CREATE | cmd/bd/entity_find_duplicates.go | MEDIUM | 2-3 | task-2-4.md |
| 2-5 | Add CLI command for merging entities | EditMode_Coder | CREATE | cmd/bd/entity_merge.go | MEDIUM | 1-3 | task-2-5.md |
| 2-6 | Add semantic search to EntityFilters (in-memory cosine fallback) | EditMode_Coder | MODIFY | internal/storage/storage.go, internal/storage/dolt/entities.go | MEDIUM | 2-2 | task-2-6.md |
| 2-7 | Add CLI command for semantic entity search | EditMode_Coder | CREATE | cmd/bd/entity_search.go | LOW | 2-6 | task-2-7.md |
| 2-8 | Add MCP tool for entity_search | EditMode_Coder | MODIFY | integrations/beads-mcp/src/beads_mcp/tools.py, integrations/beads-mcp/src/beads_mcp/bd_client.py | LOW | 2-7 | task-2-8.md |
| 2-9 | Add MCP tools for entity_merge and entity_find_duplicates | EditMode_Coder | MODIFY | integrations/beads-mcp/src/beads_mcp/tools.py, integrations/beads-mcp/src/beads_mcp/bd_client.py | LOW | 2-4, 2-5 | task-2-9.md |

---

### Phase 3: Extraction Pipeline & Memory Retrieval
**Phase Dependencies**: Phase 1 (needs schema migration for extracted_at), Phase 2 (confidence for relationship filtering)
**Checkpoint After Phase**: YES
**Rollback Trigger**: LLM extraction fails, memory retrieval returns irrelevant context, extraction hook causes deadlocks

| Task ID | Description | Agent | Operation | Files | Risk | Depends On | Task File |
|---------|-------------|-------|-----------|-------|------|------------|-----------|
| 3-1 | Create LLM extraction package with Anthropic SDK integration | EditMode_Coder | CREATE | internal/extraction/llm.go, internal/extraction/llm_test.go | MEDIUM | None | task-3-1.md |
| 3-2 | Add CLI command for single episode extraction | EditMode_Coder | CREATE | cmd/bd/episode_extract.go | MEDIUM | 3-1, 1-1 | task-3-2.md |
| 3-3 | Add CLI command for batch episode extraction | EditMode_Coder | CREATE | cmd/bd/episode_extract_all.go | MEDIUM | 3-1, 1-1 | task-3-3.md |
| 3-4 | Add --extract flag to episode create command | EditMode_Coder | MODIFY | cmd/bd/episode_create.go | LOW | 3-1 | task-3-4.md |
| 3-5 | Create memory retrieval package with graph traversal | EditMode_Coder | CREATE | internal/retrieval/context.go, internal/retrieval/types.go, internal/retrieval/retrieval_test.go | HIGH | 2-6, 1-2 | task-3-5.md |
| 3-6 | Add RetrieveMemory method to storage interface | EditMode_Coder | MODIFY | internal/storage/storage.go, internal/storage/dolt/retrieval.go | HIGH | 3-5 | task-3-6.md |
| 3-7 | Add CLI command for memory retrieval | EditMode_Coder | CREATE | cmd/bd/memory_retrieve.go | MEDIUM | 3-6 | task-3-7.md |
| 3-8 | Add MCP tools for episode_extract and memory_retrieve | EditMode_Coder | MODIFY | integrations/beads-mcp/src/beads_mcp/tools.py, integrations/beads-mcp/src/beads_mcp/bd_client.py | MEDIUM | 3-2, 3-7 | task-3-8.md |
| 3-9 | Add Python tests for extraction and retrieval MCP tools | EditMode_Coder | CREATE | integrations/beads-mcp/tests/test_extraction_tools.py, integrations/beads-mcp/tests/test_retrieval_tools.py | LOW | 3-8 | task-3-9.md |

---

## Rollback Strategy

**If Phase 1 fails:**
1. Revert files: All MCP .py files, schema_v8.sql, relationship.go, storage.go interfaces
2. Restore state: Run `bd migrate rollback-v8.1` (reverses schema changes)
3. Alternative approach: Complete schema migration separately from MCP updates

**If Phase 2 fails:**
1. Revert files: All deduplication/similarity/search files, new CLI commands
2. Restore state: Schema already applied, but no code uses new features (safe)
3. Alternative approach: Use external vector DB (chromem-go, qdrant) instead of in-memory

**If Phase 3 fails:**
1. Revert files: Extraction and retrieval packages, new CLI commands
2. Restore state: Manual extraction via CLI still works, LLM extraction is optional
3. Alternative approach: Batch extraction via external script, simpler retrieval without graph traversal

---

## Architecture Conformance

**Current Patterns Followed**:
- **Storage Interface Abstraction**: All new storage methods added to `storage.Storage` interface, implemented in `dolt/` package
- **CLI Command Pattern**: New commands follow existing Cobra structure with `--json` flag, `init()` registration, routing/redirect support
- **Repository Pattern**: New storage operations use SQL query building, transaction wrappers, error handling from existing code
- **Hook System**: Episode extraction can integrate with existing hook runner (new event type) or use flag-based trigger
- **MCP Subprocess Pattern**: All new MCP tools call `bd --json` subprocess, parse JSON output, return Pydantic models
- **ID Generation**: Entity deduplication uses existing `idgen.GenerateHashID` pattern (hash of name+type+timestamp)
- **Testing**: All new code uses `t.TempDir()`, CGO-aware test setup, `--json` output validation

**Architectural Evolution**:
- **Evolved Patterns**:
  - **Storage Interface Composition**: Adding `RetrieveMemory` continues `DoltStorage` composition pattern (10+ interfaces)
  - **Query Engine Extension**: Semantic search extends filtering beyond SQL to in-memory similarity scoring
  - **CLI Groups**: New `bd entity`, `bd relationship`, `bd episode` groups mirror existing `bd issue` patterns
- **New Patterns Introduced**:
  - **LLM Extraction Pipeline**: New pattern for processing raw data (episodes) via Anthropic SDK, creating entities/relationships
  - **In-Memory Vector Similarity**: Cosine similarity fallback when LLM embeddings unavailable (no external vector DB dependency)
  - **Graph Traversal + Temporal Filtering**: Memory retrieval combines BFS graph walk with validity time windows
  - **Confidence Scoring**: Relationships now carry confidence weights (0.0-1.0) for AI-extracted links

---

## Risk Summary

**High Risk Tasks**: 1-2 (relationship confidence touches many files), 3-5 (memory retrieval graph traversal), 3-6 (storage interface for retrieval)
**Medium Risk Tasks**: 1-1 (schema migration), 1-3 (merge interface), 1-5, 1-6, 1-7, 1-8 (MCP updates), 2-3 (dedup), 2-4, 2-5, 2-6 (semantic search), 3-1, 3-2, 3-3, 3-7, 3-8
**Low Risk Tasks**: 1-4, 1-9, 1-10, 1-11, 1-12, 2-1, 2-2, 2-7, 2-8, 2-9, 3-4, 3-9

**Known Unknowns**:
- Anthropic API rate limits for batch extraction (may need throttling/backoff)
- In-memory cosine similarity performance for large entity sets (>10k entities)
- Dolt SQL performance for complex graph traversal queries (may need pagination)
- MCP tool invocation overhead when calling many `bd` subprocesses (connection pooling already in place)
- Entity ID collision rate with timestamp-based hashing (may need name normalization pre-hashing)
