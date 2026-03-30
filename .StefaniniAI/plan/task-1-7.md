# Task 1-7: Add MCP tools for entity CRUD operations

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: MEDIUM
- **Phase**: 1
- **Depends On**: 1-4, 1-6

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\tools.py` (EXISTING)

## Instructions

Add MCP tool functions for entity CRUD operations. These tools wrap the `CliClient` methods from task 1-6 and expose them to AI assistants via the FastMCP framework.

**Outcome:** AI assistants can create, list, show, update, and delete entities through MCP tool calls.

**Tools to add (after existing issue tools, before test fixtures):**

```python
@mcp_server.tool()
async def beads_entity_create(
    entity_type: str,
    name: str,
    summary: str,
    metadata: Optional[dict[str, Any]] = None,
    created_by: Optional[str] = None,
    id: Optional[str] = None,
    working_dir: Optional[str] = None
) -> Entity:
    """Create a new entity in the knowledge graph.
    
    Args:
        entity_type: Type of entity (e.g., "person", "project", "meeting")
        name: Entity name (unique within type)
        summary: Brief description of the entity
        metadata: Additional structured data (optional)
        created_by: Creator identifier (optional)
        id: Custom entity ID (optional, auto-generated if not provided)
        working_dir: beads workspace directory (optional)
    
    Returns:
        Entity: Created entity with ID
    """
    client = await _get_client(working_dir)
    params = CreateEntityParams(
        entity_type=entity_type,
        name=name,
        summary=summary,
        metadata=metadata,
        created_by=created_by,
        id=id
    )
    return await client.create_entity(params)


@mcp_server.tool()
async def beads_entity_list(
    entity_type: Optional[str] = None,
    name: Optional[str] = None,
    created_by: Optional[str] = None,
    limit: int = 50,
    offset: int = 0,
    working_dir: Optional[str] = None
) -> List[Entity]:
    """List entities with optional filtering.
    
    Args:
        entity_type: Filter by entity type (optional)
        name: Filter by name substring (optional)
        created_by: Filter by creator (optional)
        limit: Maximum number of results (default 50)
        offset: Pagination offset (default 0)
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[Entity]: Matching entities
    """
    client = await _get_client(working_dir)
    params = EntitySearchParams(
        entity_type=entity_type,
        name=name,
        created_by=created_by,
        limit=limit,
        offset=offset
    )
    return await client.list_entities(params)


@mcp_server.tool()
async def beads_entity_show(
    entity_id: str,
    working_dir: Optional[str] = None
) -> Entity:
    """Show detailed information about an entity.
    
    Args:
        entity_id: Entity ID
        working_dir: beads workspace directory (optional)
    
    Returns:
        Entity: Entity details
    """
    client = await _get_client(working_dir)
    return await client.show_entity(entity_id)


@mcp_server.tool()
async def beads_entity_update(
    entity_id: str,
    name: Optional[str] = None,
    summary: Optional[str] = None,
    metadata: Optional[dict[str, Any]] = None,
    working_dir: Optional[str] = None
) -> Entity:
    """Update an entity's fields.
    
    Args:
        entity_id: Entity ID
        name: New name (optional)
        summary: New summary (optional)
        metadata: New metadata (optional, replaces existing)
        working_dir: beads workspace directory (optional)
    
    Returns:
        Entity: Updated entity
    """
    client = await _get_client(working_dir)
    return await client.update_entity(
        entity_id,
        name=name,
        summary=summary,
        metadata=metadata
    )


@mcp_server.tool()
async def beads_entity_delete(
    entity_id: str,
    working_dir: Optional[str] = None
) -> dict[str, Any]:
    """Delete an entity (WARNING: also deletes all associated relationships).
    
    Args:
        entity_id: Entity ID
        working_dir: beads workspace directory (optional)
    
    Returns:
        dict: Deletion confirmation
    """
    client = await _get_client(working_dir)
    return await client.delete_entity(entity_id)
```

**Pattern to follow:**
- All tools decorated with `@mcp_server.tool()`
- All tools have docstring with Args/Returns (FastMCP uses this for tool descriptions)
- All tools accept optional `working_dir` parameter (for multi-workspace routing)
- All tools call `_get_client(working_dir)` to get client instance (connection pooling)
- All tools are async (await client methods)
- Return types match Pydantic models from task 1-4

**Naming convention:**
- Tools prefixed with `beads_` (namespace)
- Tool names match CLI commands: `bd entity create` → `beads_entity_create`

## Architecture Pattern

**MCP Tool Pattern** (from existing tools.py):
- Tools are thin wrappers around client methods
- Use `_get_client(working_dir)` for connection pooling
- Leverage Python type hints for FastMCP auto-documentation
- Docstrings must follow Google style (Args, Returns sections)
- Optional parameters use `Optional[]` with None defaults

**Connection Pooling**:
- `_get_client` maintains per-workspace client instances
- Connection pool cleared between tests (see conftest.py)

## Validation Criteria
- [ ] 5 entity tools added (create, list, show, update, delete)
- [ ] All tools decorated with `@mcp_server.tool()`
- [ ] All tools have complete docstrings (Args, Returns)
- [ ] All tools accept `working_dir` parameter
- [ ] All tools use `await _get_client(working_dir)`
- [ ] Return types match Pydantic models (Entity or dict)
- [ ] No syntax errors (async/await correct, imports added)
- [ ] Tools placed after existing issue tools, before test fixtures

## Impact Analysis
- **Direct impact**: MCP tools.py (adds 5 entity tools)
- **Indirect impact**: AI assistants can now manage entities via MCP
- **Dependencies**: Task 1-12 (tests) validates these tools

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Feature 6: MCP Server Update" section for tool patterns)
- Existing pattern: `integrations/beads-mcp/src/beads_mcp/tools.py` (copy pattern from `beads_create_issue`, `beads_list_issues`, etc.)
- Pydantic models: Task 1-4 defines Entity, CreateEntityParams, EntitySearchParams

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
