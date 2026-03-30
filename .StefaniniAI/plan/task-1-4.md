# Task 1-4: Add Pydantic models for Entity, Relationship, Episode, EntityType, RelationshipType

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: LOW
- **Phase**: 1
- **Depends On**: None

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\models.py` (EXISTING)

## Instructions

Add Pydantic models for all v8 knowledge graph types to enable MCP tools to return strongly-typed data structures.

**Outcome:** MCP tools can parse `bd --json` output for entity/relationship/episode commands and return typed Python objects to AI assistants.

**Models to add:**

```python
class Entity(BaseModel):
    id: str
    entity_type: str
    name: str
    summary: str
    metadata: dict[str, Any] = Field(default_factory=dict)
    created_at: str  # ISO 8601 datetime string
    updated_at: Optional[str] = None
    created_by: str
    updated_by: Optional[str] = None
    merged_into: Optional[str] = None  # v8.1 soft-delete tracking

class Relationship(BaseModel):
    id: str
    source_entity_id: str
    relationship_type: str
    target_entity_id: str
    valid_from: str  # ISO 8601 datetime string
    valid_until: Optional[str] = None
    confidence: Optional[float] = None  # v8.1 confidence scoring (0.0-1.0)
    metadata: dict[str, Any] = Field(default_factory=dict)
    created_at: str
    created_by: str

class Episode(BaseModel):
    id: str
    timestamp: str  # ISO 8601 datetime string
    source: str  # e.g., "github", "jira", "manual", "conversation"
    raw_data_size: int  # Size in bytes (raw_data is BLOB, not returned in JSON)
    entities_extracted: list[str] = Field(default_factory=list)  # entity IDs
    metadata: dict[str, Any] = Field(default_factory=dict)
    created_at: str
    extracted_at: Optional[str] = None  # v8.1 extraction timestamp

class EntityType(BaseModel):
    name: str
    schema: dict[str, Any]  # JSON schema for metadata validation
    description: str
    created_at: str

class RelationshipType(BaseModel):
    name: str
    schema: dict[str, Any]  # JSON schema for relationship metadata
    description: str
    created_at: str
```

**Additional param models (for MCP tool inputs):**

```python
class CreateEntityParams(BaseModel):
    entity_type: str
    name: str
    summary: str
    metadata: Optional[dict[str, Any]] = None
    created_by: Optional[str] = None
    id: Optional[str] = None  # Allow custom ID

class EntitySearchParams(BaseModel):
    entity_type: Optional[str] = None
    name: Optional[str] = None
    created_by: Optional[str] = None
    limit: int = 50
    offset: int = 0

class CreateRelationshipParams(BaseModel):
    source_entity_id: str
    relationship_type: str
    target_entity_id: str
    valid_from: Optional[str] = None  # ISO 8601, defaults to now
    valid_until: Optional[str] = None
    confidence: Optional[float] = None  # v8.1
    metadata: Optional[dict[str, Any]] = None

class CreateEpisodeParams(BaseModel):
    source: str
    file_path: str  # Path to raw data file
    timestamp: Optional[str] = None  # ISO 8601, defaults to now
    entities_extracted: Optional[list[str]] = None
    extract: bool = False  # Auto-extract after creation?
```

**Placement:** Add these models after existing `Issue`, `Dependency`, `Stats` models in the file, before the test fixtures at the end.

## Architecture Pattern

**Pydantic Model Pattern** (from existing models.py):
- Use `BaseModel` from pydantic
- All required fields first, optional fields with `Optional[]` and default
- Use `Field(default_factory=...)` for mutable defaults (dicts, lists)
- Datetime fields are strings (ISO 8601 format from `bd --json`)
- JSON-serializable: no custom types, keep it simple

**Naming Convention**:
- Models match Go struct names (Entity, Relationship, Episode)
- Param models use `Create*Params`, `*SearchParams` pattern
- Field names match JSON keys from `bd --json` output (snake_case)

## Validation Criteria
- [ ] All 5 core models added (Entity, Relationship, Episode, EntityType, RelationshipType)
- [ ] All 4 param models added (CreateEntityParams, EntitySearchParams, CreateRelationshipParams, CreateEpisodeParams)
- [ ] Models match JSON output from `bd entity create --json`, `bd relationship create --json`, etc.
- [ ] Confidence field is Optional[float] (v8.1)
- [ ] merged_into field is Optional[str] (v8.1)
- [ ] extracted_at field is Optional[str] (v8.1)
- [ ] No syntax errors (pydantic imports correct, Field used correctly)
- [ ] Models placed in correct section of models.py (after Issue models)

## Impact Analysis
- **Direct impact**: MCP models.py (foundation for all entity/relationship MCP tools)
- **Indirect impact**: All future MCP tools import these models
- **Dependencies**: Tasks 1-5, 1-6, 1-7, 1-8, 1-9, 1-10 use these models

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Feature 6: MCP Server Update" section for existing model patterns)
- Existing models: `integrations/beads-mcp/src/beads_mcp/models.py` (follow same structure as Issue, IssueBase, etc.)

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
