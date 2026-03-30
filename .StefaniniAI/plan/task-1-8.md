# Task 1-8: Add MCP tools for relationship operations (with confidence support)

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: MEDIUM
- **Phase**: 1
- **Depends On**: 1-4, 1-6, 1-2

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\tools.py` (EXISTING)

## Instructions

Add MCP tool functions for relationship CRUD operations with confidence scoring support (task 1-2).

**Outcome:** AI assistants can create, list, show, update, and delete relationships, including setting/filtering by confidence scores.

**Tools to add:**

```python
@mcp_server.tool()
async def beads_relationship_create(
    source_entity_id: str,
    relationship_type: str,
    target_entity_id: str,
    valid_from: Optional[str] = None,
    valid_until: Optional[str] = None,
    confidence: Optional[float] = None,
    metadata: Optional[dict[str, Any]] = None,
    working_dir: Optional[str] = None
) -> Relationship:
    """Create a new relationship between two entities.
    
    Args:
        source_entity_id: Source entity ID
        relationship_type: Relationship type (e.g., "leads", "blocks", "mentioned_in")
        target_entity_id: Target entity ID
        valid_from: Relationship start timestamp (ISO 8601, optional, defaults to now)
        valid_until: Relationship end timestamp (ISO 8601, optional, null = ongoing)
        confidence: Confidence score 0.0-1.0 (optional, defaults to 1.0)
        metadata: Additional structured data (optional)
        working_dir: beads workspace directory (optional)
    
    Returns:
        Relationship: Created relationship with ID
    """
    client = await _get_client(working_dir)
    params = CreateRelationshipParams(
        source_entity_id=source_entity_id,
        relationship_type=relationship_type,
        target_entity_id=target_entity_id,
        valid_from=valid_from,
        valid_until=valid_until,
        confidence=confidence,
        metadata=metadata
    )
    return await client.create_relationship(params)


@mcp_server.tool()
async def beads_relationship_list(
    source_entity_id: Optional[str] = None,
    target_entity_id: Optional[str] = None,
    relationship_type: Optional[str] = None,
    min_confidence: Optional[float] = None,
    working_dir: Optional[str] = None
) -> List[Relationship]:
    """List relationships with optional filtering.
    
    Args:
        source_entity_id: Filter by source entity (optional)
        target_entity_id: Filter by target entity (optional)
        relationship_type: Filter by type (optional)
        min_confidence: Minimum confidence threshold (optional, 0.0-1.0)
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[Relationship]: Matching relationships
    """
    client = await _get_client(working_dir)
    return await client.list_relationships(
        source_id=source_entity_id,
        target_id=target_entity_id,
        relationship_type=relationship_type,
        min_confidence=min_confidence
    )


@mcp_server.tool()
async def beads_relationship_show(
    relationship_id: str,
    working_dir: Optional[str] = None
) -> Relationship:
    """Show detailed information about a relationship.
    
    Args:
        relationship_id: Relationship ID
        working_dir: beads workspace directory (optional)
    
    Returns:
        Relationship: Relationship details
    """
    client = await _get_client(working_dir)
    return await client.show_relationship(relationship_id)


@mcp_server.tool()
async def beads_relationship_update(
    relationship_id: str,
    valid_until: Optional[str] = None,
    confidence: Optional[float] = None,
    metadata: Optional[dict[str, Any]] = None,
    working_dir: Optional[str] = None
) -> Relationship:
    """Update a relationship's fields.
    
    Args:
        relationship_id: Relationship ID
        valid_until: New end timestamp (ISO 8601, optional)
        confidence: New confidence score 0.0-1.0 (optional)
        metadata: New metadata (optional, replaces existing)
        working_dir: beads workspace directory (optional)
    
    Returns:
        Relationship: Updated relationship
    """
    client = await _get_client(working_dir)
    return await client.update_relationship(
        relationship_id,
        valid_until=valid_until,
        confidence=confidence,
        metadata=metadata
    )


@mcp_server.tool()
async def beads_relationship_delete(
    relationship_id: str,
    working_dir: Optional[str] = None
) -> dict[str, Any]:
    """Delete a relationship.
    
    Args:
        relationship_id: Relationship ID
        working_dir: beads workspace directory (optional)
    
    Returns:
        dict: Deletion confirmation
    """
    client = await _get_client(working_dir)
    return await client.delete_relationship(relationship_id)
```

**Key features:**
- `confidence` parameter in create/update (maps to task 1-2 Confidence field)
- `min_confidence` filter in list (maps to task 1-2 RelationshipFilters.MinConfidence)
- Temporal validity: `valid_from`, `valid_until` (existing v8 feature)

## Architecture Pattern

**MCP Tool Pattern** (same as task 1-7):
- Tools are thin wrappers around client methods
- Use `_get_client(working_dir)` for connection pooling
- Docstrings with Args/Returns sections

**Confidence Scoring Pattern**:
- Confidence is optional: if not provided, defaults to 1.0 (certain)
- AI-extracted relationships should set confidence based on extraction quality
- Human-curated relationships omit confidence (defaults to 1.0)

## Validation Criteria
- [ ] 5 relationship tools added (create, list, show, update, delete)
- [ ] All tools decorated with `@mcp_server.tool()`
- [ ] All tools have complete docstrings
- [ ] `confidence` parameter in create/update (float, optional)
- [ ] `min_confidence` filter in list (float, optional)
- [ ] All tools use `await _get_client(working_dir)`
- [ ] Return types match Pydantic models (Relationship or dict)
- [ ] No syntax errors

## Impact Analysis
- **Direct impact**: MCP tools.py (adds 5 relationship tools)
- **Indirect impact**: AI assistants can manage relationships with confidence scoring
- **Dependencies**: Task 1-12 (tests) validates these tools

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Feature 4: Relationship Confidence" for rationale)
- Confidence field: Task 1-2 adds Confidence to Relationship type
- Existing pattern: Task 1-7 entity tools (follow same structure)

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
