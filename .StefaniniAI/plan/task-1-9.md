# Task 1-9: Add MCP tools for episode operations

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: LOW
- **Phase**: 1
- **Depends On**: 1-4, 1-6

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\tools.py` (EXISTING)

## Instructions

Add MCP tool functions for episode operations (provenance tracking for entity extraction).

**Outcome:** AI assistants can create, list, and show episodes (immutable records of raw conversation/observation data).

**Tools to add:**

```python
@mcp_server.tool()
async def beads_episode_create(
    source: str,
    file_path: str,
    timestamp: Optional[str] = None,
    entities_extracted: Optional[List[str]] = None,
    extract: bool = False,
    working_dir: Optional[str] = None
) -> Episode:
    """Create a new episode (provenance record for entity extraction).
    
    Args:
        source: Episode source (e.g., "github", "jira", "manual", "conversation")
        file_path: Path to raw data file (text, JSON, etc.)
        timestamp: Episode timestamp (ISO 8601, optional, defaults to now)
        entities_extracted: Pre-linked entity IDs (optional)
        extract: Auto-extract entities after creation (optional, default False)
        working_dir: beads workspace directory (optional)
    
    Returns:
        Episode: Created episode with ID
    """
    client = await _get_client(working_dir)
    params = CreateEpisodeParams(
        source=source,
        file_path=file_path,
        timestamp=timestamp,
        entities_extracted=entities_extracted,
        extract=extract
    )
    return await client.create_episode(params)


@mcp_server.tool()
async def beads_episode_list(
    source: Optional[str] = None,
    limit: int = 50,
    working_dir: Optional[str] = None
) -> List[Episode]:
    """List episodes with optional filtering.
    
    Args:
        source: Filter by source (optional)
        limit: Maximum number of results (default 50)
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[Episode]: Matching episodes
    """
    client = await _get_client(working_dir)
    return await client.list_episodes(source=source, limit=limit)


@mcp_server.tool()
async def beads_episode_show(
    episode_id: str,
    working_dir: Optional[str] = None
) -> Episode:
    """Show detailed information about an episode.
    
    Args:
        episode_id: Episode ID
        working_dir: beads workspace directory (optional)
    
    Returns:
        Episode: Episode details (raw_data not included, only size)
    """
    client = await _get_client(working_dir)
    return await client.show_episode(episode_id)


@mcp_server.tool()
async def beads_episode_extract(
    episode_id: str,
    working_dir: Optional[str] = None
) -> dict[str, Any]:
    """Extract entities and relationships from an episode using LLM.
    
    Args:
        episode_id: Episode ID to process
        working_dir: beads workspace directory (optional)
    
    Returns:
        dict: Extraction results (entities_created, relationships_created, updated_at)
    """
    client = await _get_client(working_dir)
    return await client.extract_episode(episode_id)
```

**Key features:**
- `extract` flag in create: auto-trigger LLM extraction after episode creation (task 3-4)
- `episode_extract` tool: manually trigger extraction for existing episode (task 3-2)
- Episodes are immutable: no update/delete operations (append-only log)

## Architecture Pattern

**Episode as Provenance** (from Architecture.md):
- Episodes link entities to their source (conversation, meeting, observation)
- Raw data stored as BLOB (not returned in JSON, only size)
- Episodes are immutable: once created, never modified
- Extraction is idempotent: can run multiple times (updates `extracted_at`)

**MCP Tool Pattern** (same as tasks 1-7, 1-8):
- Tools wrap client methods
- Use `_get_client(working_dir)` for connection pooling
- Docstrings with Args/Returns sections

## Validation Criteria
- [ ] 4 episode tools added (create, list, show, extract)
- [ ] All tools decorated with `@mcp_server.tool()`
- [ ] All tools have complete docstrings
- [ ] `extract` parameter in create (bool, default False)
- [ ] `episode_extract` tool returns dict (not Episode)
- [ ] No update/delete tools (episodes are immutable)
- [ ] All tools use `await _get_client(working_dir)`
- [ ] Return types match Pydantic models (Episode or dict)
- [ ] No syntax errors

## Impact Analysis
- **Direct impact**: MCP tools.py (adds 4 episode tools)
- **Indirect impact**: AI assistants can record conversation provenance
- **Dependencies**: Task 1-12 (tests) validates these tools, Task 3-1 (extraction pipeline) implements the extraction logic

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Feature 1: Entity Extraction Pipeline" for episode processing)
- Architecture: Episode provenance pattern from Architecture.md "Knowledge Graph Architecture (v8)" section
- Existing pattern: Tasks 1-7, 1-8 (entity/relationship tools)

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
