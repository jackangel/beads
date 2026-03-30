# Task 2-8: Add MCP tool for entity_search

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: LOW
- **Phase**: 2
- **Depends On**: 2-7

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\tools.py` (EXISTING)
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\bd_client.py` (EXISTING - search_entities method from task 1-6)

## Instructions
Add MCP tool for semantic entity search (wraps `bd entity search` from task 2-7).

**Implementation in tools.py:**
```python
@mcp_server.tool()
async def beads_entity_search(
    query: str,
    entity_type: Optional[str] = None,
    top: int = 10,
    working_dir: Optional[str] = None
) -> List[Entity]:
    """Search entities using natural language query (semantic search).
    
    Args:
        query: Natural language search query
        entity_type: Filter by entity type (optional)
        top: Maximum number of results (default 10)
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[Entity]: Most relevant entities sorted by similarity
    
    Example:
        beads_entity_search("Alice's role in auth service")
    """
    client = await _get_client(working_dir)
    return await client.search_entities(query, limit=top)
```

Note: The `search_entities` method was already added to `BdClientBase` and `CliClient` in task 1-6. This tool just exposes it.

## Validation Criteria
- [ ] `beads_entity_search` tool added
- [ ] Tool accepts query, entity_type, top, working_dir
- [ ] Calls `client.search_entities`
- [ ] Docstring complete with example
- [ ] No syntax errors

## Impact Analysis
- **Direct impact**: MCP tools.py (adds semantic search tool)
- **Indirect impact**: AI assistants can semantically search entities
- **Dependencies**: Task 2-7 (CLI command), Task 1-6 (client method)

## Context
- Pattern: Task 1-7 (entity tools)
- CLI: Task 2-7

## User Feedback
*(Empty)*
