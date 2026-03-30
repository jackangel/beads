# Task 2-9: Add MCP tools for entity_merge and entity_find_duplicates

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: LOW
- **Phase**: 2
- **Depends On**: 2-4, 2-5

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\tools.py` (EXISTING)
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\bd_client.py` (EXISTING - add methods to BdClientBase and CliClient)

## Instructions
Add MCP tools for entity merge and duplicate detection.

**Step 1: Add methods to bd_client.py (BdClientBase abstract + CliClient concrete):**
```python
# In BdClientBase:
@abstractmethod
async def merge_entities(self, source_id: str, target_id: str) -> dict[str, Any]: ...

@abstractmethod
async def find_duplicate_entities(self, entity_type: Optional[str] = None, threshold: float = 0.8) -> List[dict[str, Any]]: ...

# In CliClient:
async def merge_entities(self, source_id: str, target_id: str) -> dict[str, Any]:
    result = await self._run_bd_command(["bd", "--json", "entity", "merge", source_id, target_id])
    return result

async def find_duplicate_entities(self, entity_type: Optional[str] = None, threshold: float = 0.8) -> List[dict[str, Any]]:
    args = ["bd", "--json", "entity", "find-duplicates", "--threshold", str(threshold)]
    if entity_type:
        args.extend(["--entity-type", entity_type])
    result = await self._run_bd_command(args)
    return result.get("duplicates", [])
```

**Step 2: Add tools to tools.py:**
```python
@mcp_server.tool()
async def beads_entity_merge(
    source_id: str,
    target_id: str,
    working_dir: Optional[str] = None
) -> dict[str, Any]:
    """Merge source entity into target entity (deduplication).
    
    Args:
        source_id: Source entity ID (will be soft-deleted)
        target_id: Target entity ID (receives all relationships)
        working_dir: beads workspace directory (optional)
    
    Returns:
        dict: Merge confirmation
    """
    client = await _get_client(working_dir)
    return await client.merge_entities(source_id, target_id)


@mcp_server.tool()
async def beads_entity_find_duplicates(
    entity_type: Optional[str] = None,
    threshold: float = 0.8,
    working_dir: Optional[str] = None
) -> List[dict[str, Any]]:
    """Find potential duplicate entities using similarity scoring.
    
    Args:
        entity_type: Filter by entity type (optional)
        threshold: Similarity threshold 0.0-1.0 (default 0.8)
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[dict]: Duplicate pairs with similarity scores
    """
    client = await _get_client(working_dir)
    return await client.find_duplicate_entities(entity_type, threshold)
```

## Validation Criteria
- [ ] `BdClientBase` has abstract methods for merge and find_duplicates
- [ ] `CliClient` implements both methods
- [ ] `beads_entity_merge` tool added
- [ ] `beads_entity_find_duplicates` tool added
- [ ] Both tools have complete docstrings
- [ ] No syntax errors

## Impact Analysis
- **Direct impact**: MCP client + tools (adds merge and dedup)
- **Indirect impact**: AI assistants can detect and merge duplicates
- **Dependencies**: Task 2-4 (find-duplicates CLI), Task 2-5 (merge CLI)

## Context
- Pattern: Task 1-7 (entity tools)
- CLI: Task 2-4, 2-5

## User Feedback
*(Empty)*
