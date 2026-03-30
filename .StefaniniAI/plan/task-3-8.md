# Task 3-8: Add MCP tools for episode_extract and memory_retrieve

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: MEDIUM
- **Phase**: 3
- **Depends On**: 3-2, 3-7

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\tools.py` (EXISTING)
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\bd_client.py` (EXISTING)

## Instructions
Add MCP tools for episode extraction and memory retrieval.

**Step 1: Add methods to bd_client.py:**
```python
# In BdClientBase:
@abstractmethod
async def extract_episode(self, episode_id: str) -> dict[str, Any]: ...

@abstractmethod
async def retrieve_memory(self, query: str, max_hops: int = 2, top_k: int = 5, min_confidence: float = 0.5) -> dict[str, Any]: ...

# In CliClient:
async def extract_episode(self, episode_id: str) -> dict[str, Any]:
    result = await self._run_bd_command(["bd", "--json", "episode", "extract", episode_id])
    return result

async def retrieve_memory(self, query: str, max_hops: int = 2, top_k: int = 5, min_confidence: float = 0.5) -> dict[str, Any]:
    result = await self._run_bd_command([
        "bd", "--json", "memory", "retrieve",
        "--query", query,
        "--hops", str(max_hops),
        "--top", str(top_k),
        "--min-confidence", str(min_confidence)
    ])
    return result
```

**Step 2: Add tools to tools.py:**
```python
@mcp_server.tool()
async def beads_episode_extract(
    episode_id: str,
    working_dir: Optional[str] = None
) -> dict[str, Any]:
    """Extract entities and relationships from an episode using LLM.
    
    Requires ANTHROPIC_API_KEY environment variable.
    
    Args:
        episode_id: Episode ID to process
        working_dir: beads workspace directory (optional)
    
    Returns:
        dict: Extraction results (entities_created, relationships_created)
    """
    client = await _get_client(working_dir)
    return await client.extract_episode(episode_id)


@mcp_server.tool()
async def beads_memory_retrieve(
    query: str,
    max_hops: int = 2,
    top_k: int = 5,
    min_confidence: float = 0.5,
    working_dir: Optional[str] = None
) -> dict[str, Any]:
    """Retrieve memory context from knowledge graph (semantic search + graph traversal).
    
    Args:
        query: Natural language query
        max_hops: Graph traversal depth (default 2)
        top_k: Max initial entities from semantic search (default 5)
        min_confidence: Minimum relationship confidence (default 0.5)
        working_dir: beads workspace directory (optional)
    
    Returns:
        dict: Memory context (entities, relationships, source_episodes, relevance_scores)
    """
    client = await _get_client(working_dir)
    return await client.retrieve_memory(query, max_hops, top_k, min_confidence)
```

## Validation Criteria
- [ ] `BdClientBase` has abstract methods for extract_episode and retrieve_memory
- [ ] `CliClient` implements both
- [ ] `beads_episode_extract` tool added
- [ ] `beads_memory_retrieve` tool added
- [ ] Both tools have complete docstrings
- [ ] No syntax errors

## Impact Analysis
- **Direct impact**: MCP client + tools (adds extraction and retrieval)
- **Indirect impact**: AI assistants can extract knowledge and query memory
- **Dependencies**: Task 3-2 (extract CLI), Task 3-7 (retrieve CLI)

## Context
- Pattern: Tasks 1-7, 1-8 (entity/relationship tools)

## User Feedback
*(Empty)*
