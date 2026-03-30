# Task 1-5: Add abstract methods to BdClientBase for entity operations

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: MEDIUM
- **Phase**: 1
- **Depends On**: 1-4

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\bd_client.py` (EXISTING)

## Instructions

Extend the `BdClientBase` abstract class with abstract methods for all entity, relationship, episode, and ontology operations. These methods define the contract that `CliClient` (subprocess implementation) must fulfill.

**Outcome:** The MCP server has a well-defined interface for knowledge graph operations. Future implementations (e.g., gRPC client, REST client) can implement the same interface.

**Methods to add to `BdClientBase` (after existing issue methods):**

```python
# Entity operations
@abstractmethod
async def create_entity(self, params: CreateEntityParams) -> Entity: ...

@abstractmethod
async def list_entities(self, params: Optional[EntitySearchParams] = None) -> List[Entity]: ...

@abstractmethod
async def show_entity(self, entity_id: str) -> Entity: ...

@abstractmethod
async def update_entity(self, entity_id: str, **kwargs) -> Entity: ...

@abstractmethod
async def delete_entity(self, entity_id: str) -> dict[str, Any]: ...

@abstractmethod
async def search_entities(self, query: str, limit: int = 10) -> List[Entity]: ...

@abstractmethod
async def merge_entities(self, source_id: str, target_id: str) -> dict[str, Any]: ...

@abstractmethod
async def find_duplicate_entities(self, entity_type: Optional[str] = None, threshold: float = 0.8) -> List[dict[str, Any]]: ...

# Relationship operations
@abstractmethod
async def create_relationship(self, params: CreateRelationshipParams) -> Relationship: ...

@abstractmethod
async def list_relationships(self, source_id: Optional[str] = None, target_id: Optional[str] = None, relationship_type: Optional[str] = None, min_confidence: Optional[float] = None) -> List[Relationship]: ...

@abstractmethod
async def show_relationship(self, relationship_id: str) -> Relationship: ...

@abstractmethod
async def update_relationship(self, relationship_id: str, **kwargs) -> Relationship: ...

@abstractmethod
async def delete_relationship(self, relationship_id: str) -> dict[str, Any]: ...

# Episode operations
@abstractmethod
async def create_episode(self, params: CreateEpisodeParams) -> Episode: ...

@abstractmethod
async def list_episodes(self, source: Optional[str] = None, limit: int = 50) -> List[Episode]: ...

@abstractmethod
async def show_episode(self, episode_id: str) -> Episode: ...

@abstractmethod
async def extract_episode(self, episode_id: str) -> dict[str, Any]: ...

# Ontology operations
@abstractmethod
async def register_entity_type(self, name: str, schema: dict[str, Any], description: str) -> EntityType: ...

@abstractmethod
async def register_relationship_type(self, name: str, schema: dict[str, Any], description: str) -> RelationshipType: ...

@abstractmethod
async def list_entity_types(self) -> List[EntityType]: ...

@abstractmethod
async def list_relationship_types(self) -> List[RelationshipType]: ...

# Graph operations (already exist, no changes needed)
# - explore_graph
# - traverse_graph
# - export_graph

# Memory retrieval
@abstractmethod
async def retrieve_memory(self, query: str, max_hops: int = 2, top_k: int = 5, min_confidence: float = 0.5) -> dict[str, Any]: ...
```

**Imports to add at top of file:**
```python
from .models import (
    Entity, Relationship, Episode, EntityType, RelationshipType,
    CreateEntityParams, CreateRelationshipParams, CreateEpisodeParams, EntitySearchParams
)
```

**Placement:** Add these methods after existing issue methods (after `validate`), before the concrete `CliClient` implementation begins.

## Architecture Pattern

**Abstract Base Class Pattern** (from existing bd_client.py):
- All methods are @abstractmethod (must be implemented by concrete class)
- Use async/await for all methods (consistent with existing issue methods)
- Return typed objects (Pydantic models) not raw dicts
- Optional parameters use `Optional[]` with defaults
- Methods match CLI command structure: `bd entity create` → `create_entity`

**Interface Segregation**:
- Each domain has CRUD methods: create, list, show, update, delete
- Entity domain adds: search, merge, find_duplicates
- Relationship domain adds: confidence filtering
- Episode domain adds: extract
- Ontology domain: register + list only (no CRUD on types)

## Validation Criteria
- [ ] All 25 abstract methods added to `BdClientBase`
- [ ] Methods grouped by domain (entity, relationship, episode, ontology, memory)
- [ ] All methods use @abstractmethod decorator
- [ ] All methods are async (async def)
- [ ] Type hints use Pydantic models from task 1-4
- [ ] Imports added at top of file
- [ ] No syntax errors (ABC pattern correct)
- [ ] Methods placed after existing issue methods

## Impact Analysis
- **Direct impact**: Abstract base class (contract for all client implementations)
- **Indirect impact**: Forces CliClient to implement all methods (task 1-6)
- **Dependencies**: Task 1-6 (CliClient implementation) must implement these methods

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Feature 6: MCP Server Update" section for existing abstract methods)
- Existing pattern: `integrations/beads-mcp/src/beads_mcp/bd_client.py` `BdClientBase` class
- Pydantic models: Task 1-4 creates all model types used in method signatures

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
