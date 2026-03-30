# Task 1-10: Add MCP tools for ontology operations

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: LOW
- **Phase**: 1
- **Depends On**: 1-4, 1-6

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\tools.py` (EXISTING)

## Instructions

Add MCP tool functions for ontology operations (custom entity and relationship type definitions).

**Outcome:** AI assistants can register custom entity/relationship types and list existing types.

**Tools to add:**

```python
@mcp_server.tool()
async def beads_ontology_register_entity_type(
    name: str,
    schema: dict[str, Any],
    description: str,
    working_dir: Optional[str] = None
) -> EntityType:
    """Register a custom entity type with JSON schema validation.
    
    Args:
        name: Entity type name (e.g., "meeting", "document", "person")
        schema: JSON schema for entity metadata validation (Pydantic-like)
        description: Human-readable description of the type
        working_dir: beads workspace directory (optional)
    
    Returns:
        EntityType: Registered entity type
    
    Example:
        schema = {
            "properties": {
                "attendees": {"type": "array", "items": {"type": "string"}},
                "duration_minutes": {"type": "integer"}
            },
            "required": ["attendees"]
        }
        beads_ontology_register_entity_type("meeting", schema, "A scheduled meeting")
    """
    client = await _get_client(working_dir)
    return await client.register_entity_type(name, schema, description)


@mcp_server.tool()
async def beads_ontology_register_relationship_type(
    name: str,
    schema: dict[str, Any],
    description: str,
    working_dir: Optional[str] = None
) -> RelationshipType:
    """Register a custom relationship type with JSON schema validation.
    
    Args:
        name: Relationship type name (e.g., "mentioned_in", "leads", "blocks")
        schema: JSON schema for relationship metadata validation
        description: Human-readable description of the type
        working_dir: beads workspace directory (optional)
    
    Returns:
        RelationshipType: Registered relationship type
    
    Example:
        schema = {
            "properties": {
                "context": {"type": "string"},
                "relevance_score": {"type": "number", "minimum": 0, "maximum": 1}
            }
        }
        beads_ontology_register_relationship_type("mentioned_in", schema, "Entity referenced in another")
    """
    client = await _get_client(working_dir)
    return await client.register_relationship_type(name, schema, description)


@mcp_server.tool()
async def beads_ontology_list_entity_types(
    working_dir: Optional[str] = None
) -> List[EntityType]:
    """List all registered entity types.
    
    Args:
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[EntityType]: All entity types
    """
    client = await _get_client(working_dir)
    return await client.list_entity_types()


@mcp_server.tool()
async def beads_ontology_list_relationship_types(
    working_dir: Optional[str] = None
) -> List[RelationshipType]:
    """List all registered relationship types.
    
    Args:
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[RelationshipType]: All relationship types
    """
    client = await _get_client(working_dir)
    return await client.list_relationship_types()
```

**Key features:**
- Ontology types are validated by JSON schema (same pattern as Pydantic)
- Types registered once, referenced many times (e.g., `--entity-type meeting`)
- No update/delete operations (types are immutable once registered)
- Example schemas in docstrings help users understand the feature

## Architecture Pattern

**Ontology as Type System** (from Architecture.md):
- Entity types define metadata schema: what fields are valid for a "meeting" entity?
- Relationship types define link metadata: what context is stored for "mentioned_in" relationship?
- JSON schema validation ensures metadata conforms to type definition
- Types registered via CLI, validated on entity/relationship creation

**MCP Tool Pattern** (same as tasks 1-7, 1-8, 1-9):
- Tools wrap client methods
- Use `_get_client(working_dir)` for connection pooling
- Docstrings with Args/Returns sections
- Include example usage in docstrings (helps AI assistants understand feature)

## Validation Criteria
- [ ] 4 ontology tools added (register entity type, register relationship type, list entity types, list relationship types)
- [ ] All tools decorated with `@mcp_server.tool()`
- [ ] All tools have complete docstrings with examples
- [ ] Example schemas in register_entity_type and register_relationship_type docstrings
- [ ] No update/delete tools (types are immutable)
- [ ] All tools use `await _get_client(working_dir)`
- [ ] Return types match Pydantic models (EntityType, RelationshipType)
- [ ] No syntax errors

## Impact Analysis
- **Direct impact**: MCP tools.py (adds 4 ontology tools)
- **Indirect impact**: AI assistants can define custom type systems
- **Dependencies**: Task 1-12 (tests) validates these tools

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see v8 ontology system description)
- Architecture: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Architecture.md` "Custom Ontology System" section
- Existing pattern: Tasks 1-7, 1-8, 1-9 (entity/relationship/episode tools)

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
