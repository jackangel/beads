# Task 1-6: Implement entity/relationship/episode CLI wrappers in CliClient

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: MEDIUM
- **Phase**: 1
- **Depends On**: 1-4, 1-5

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\bd_client.py` (EXISTING)

## Instructions

Implement all 25 abstract methods from task 1-5 in the concrete `CliClient` class. Each method calls the `bd` CLI subprocess with appropriate arguments and parses JSON output into Pydantic models.

**Outcome:** The MCP server can invoke all knowledge graph operations via subprocess calls to `bd --json`.

**Implementation pattern (follow existing `create`, `list_issues`, `show` methods):**

```python
async def create_entity(self, params: CreateEntityParams) -> Entity:
    args = ["bd", "--json", "entity", "create",
            "--entity-type", params.entity_type,
            "--name", params.name,
            "--summary", params.summary]
    
    if params.metadata:
        args.extend(["--metadata", json.dumps(params.metadata)])
    if params.created_by:
        args.extend(["--created-by", params.created_by])
    if params.id:
        args.extend(["--id", params.id])
    
    result = await self._run_bd_command(args)
    return Entity(**result)

async def list_entities(self, params: Optional[EntitySearchParams] = None) -> List[Entity]:
    args = ["bd", "--json", "entity", "list"]
    
    if params:
        if params.entity_type:
            args.extend(["--entity-type", params.entity_type])
        if params.name:
            args.extend(["--name", params.name])
        if params.created_by:
            args.extend(["--created-by", params.created_by])
        if params.limit:
            args.extend(["--limit", str(params.limit)])
        if params.offset:
            args.extend(["--offset", str(params.offset)])
    
    result = await self._run_bd_command(args)
    # Handle array response
    if isinstance(result, list):
        return [Entity(**e) for e in result]
    return []

async def show_entity(self, entity_id: str) -> Entity:
    result = await self._run_bd_command(["bd", "--json", "entity", "show", entity_id])
    return Entity(**result)

# Similar pattern for:
# - update_entity, delete_entity, search_entities, merge_entities, find_duplicate_entities
# - create_relationship, list_relationships, show_relationship, update_relationship, delete_relationship
# - create_episode, list_episodes, show_episode, extract_episode
# - register_entity_type, register_relationship_type, list_entity_types, list_relationship_types
# - retrieve_memory
```

**Key patterns:**
- Use `self._run_bd_command(args)` for all subprocess calls (existing helper)
- Always include `--json` flag after `bd`
- Parse result into Pydantic models: `Entity(**result)`, `Relationship(**result)`, etc.
- Handle list responses: `[Entity(**e) for e in result]`
- Handle dict responses for operations like merge/delete: `return result` (no model)
- Convert Python types to CLI args: `str(limit)`, `json.dumps(metadata)`

**Special cases:**
- `create_episode`: `params.file_path` must be read and piped via stdin if needed, or use `--file` flag
- `extract_episode`: Returns dict with stats (not Episode model)
- `find_duplicate_entities`: Returns list of dicts with `{entity_a, entity_b, similarity, reason}`
- `retrieve_memory`: Returns complex dict with `{entities: [...], relationships: [...], episodes: [...], relevance_scores: {...}}`

## Architecture Pattern

**Subprocess CLI Wrapper Pattern** (from existing CliClient):
- Use `_run_bd_command` for all `bd` invocations
- `_run_bd_command` handles: cwd, stdin=DEVNULL, shell on Windows, JSON parsing, error handling
- Construct args list carefully: `["bd", "--json", "command", "subcommand", "--flag", "value"]`
- Never use f-strings in args (keep args separate for proper escaping)

**Error Handling**:
- `_run_bd_command` raises exception on non-zero exit code
- Let exceptions propagate (MCP framework handles them)
- No need for manual try/except in most methods

## Validation Criteria
- [ ] All 25 methods from task 1-5 implemented in `CliClient`
- [ ] Each method calls `bd --json [command] [subcommand]`
- [ ] All methods return correct Pydantic model type
- [ ] List methods handle array responses correctly
- [ ] Optional parameters converted to CLI flags correctly
- [ ] Metadata/schema dicts serialized with `json.dumps`
- [ ] No syntax errors (async/await correct, type hints match abstract methods)
- [ ] Methods follow existing pattern from `create`, `list_issues`, `show`

## Impact Analysis
- **Direct impact**: CliClient implementation (enables all MCP tools)
- **Indirect impact**: All MCP tools in task 1-7, 1-8, 1-9, 1-10 call these methods
- **Dependencies**: Tasks 1-7 through 1-10 depend on these implementations

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Feature 6: MCP Server Update" section for subprocess pattern)
- Existing pattern: `integrations/beads-mcp/src/beads_mcp/bd_client.py` `CliClient` class (copy patterns from `create`, `list_issues`, `show`, `update`, `claim`, `close`)
- CLI commands: These map to CLI commands created in future tasks, but we can implement client methods now (they'll fail gracefully if CLI commands don't exist yet)

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
