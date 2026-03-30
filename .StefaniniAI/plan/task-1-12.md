# Task 1-12: Add Python tests for new MCP entity/relationship tools

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Risk Level**: LOW
- **Phase**: 1
- **Depends On**: 1-7, 1-8

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\tests\test_entity_tools.py` (NEW)
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\tests\test_relationship_tools.py` (NEW)

## Instructions

Create pytest test files for the new entity and relationship MCP tools. Follow the existing test pattern from `test_tools.py`.

**Outcome:** All new MCP tools are validated with unit tests using mocks.

**File 1: `test_entity_tools.py`**

```python
import pytest
from unittest.mock import AsyncMock, patch

from beads_mcp.models import Entity, EntitySearchParams
from beads_mcp.tools import (
    beads_entity_create,
    beads_entity_list,
    beads_entity_show,
    beads_entity_update,
    beads_entity_delete,
)


@pytest.fixture
def sample_entity():
    return Entity(
        id="ent-abc123",
        entity_type="person",
        name="Alice",
        summary="Senior engineer",
        metadata={"team": "backend"},
        created_at="2026-03-30T10:00:00Z",
        created_by="bob",
    )


@pytest.mark.asyncio
async def test_beads_entity_create(sample_entity):
    mock_client = AsyncMock()
    mock_client.create_entity = AsyncMock(return_value=sample_entity)
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_create(
            entity_type="person",
            name="Alice",
            summary="Senior engineer",
            metadata={"team": "backend"},
            created_by="bob"
        )
    
    assert result.id == "ent-abc123"
    assert result.name == "Alice"
    mock_client.create_entity.assert_called_once()


@pytest.mark.asyncio
async def test_beads_entity_list(sample_entity):
    mock_client = AsyncMock()
    mock_client.list_entities = AsyncMock(return_value=[sample_entity])
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_list(entity_type="person", limit=10)
    
    assert len(result) == 1
    assert result[0].name == "Alice"
    mock_client.list_entities.assert_called_once()


@pytest.mark.asyncio
async def test_beads_entity_show(sample_entity):
    mock_client = AsyncMock()
    mock_client.show_entity = AsyncMock(return_value=sample_entity)
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_show("ent-abc123")
    
    assert result.id == "ent-abc123"
    mock_client.show_entity.assert_called_once_with("ent-abc123")


@pytest.mark.asyncio
async def test_beads_entity_update(sample_entity):
    updated_entity = Entity(**sample_entity.model_dump())
    updated_entity.summary = "Principal engineer"
    
    mock_client = AsyncMock()
    mock_client.update_entity = AsyncMock(return_value=updated_entity)
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_update("ent-abc123", summary="Principal engineer")
    
    assert result.summary == "Principal engineer"
    mock_client.update_entity.assert_called_once()


@pytest.mark.asyncio
async def test_beads_entity_delete():
    mock_client = AsyncMock()
    mock_client.delete_entity = AsyncMock(return_value={"message": "Entity deleted"})
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_delete("ent-abc123")
    
    assert result["message"] == "Entity deleted"
    mock_client.delete_entity.assert_called_once_with("ent-abc123")
```

**File 2: `test_relationship_tools.py`**

```python
import pytest
from unittest.mock import AsyncMock, patch

from beads_mcp.models import Relationship
from beads_mcp.tools import (
    beads_relationship_create,
    beads_relationship_list,
    beads_relationship_show,
    beads_relationship_update,
    beads_relationship_delete,
)


@pytest.fixture
def sample_relationship():
    return Relationship(
        id="rel-xyz789",
        source_entity_id="ent-abc123",
        relationship_type="leads",
        target_entity_id="ent-def456",
        valid_from="2026-01-01T00:00:00Z",
        valid_until=None,
        confidence=0.95,
        metadata={"context": "Q1 planning"},
        created_at="2026-03-30T10:00:00Z",
        created_by="alice",
    )


@pytest.mark.asyncio
async def test_beads_relationship_create(sample_relationship):
    mock_client = AsyncMock()
    mock_client.create_relationship = AsyncMock(return_value=sample_relationship)
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_create(
            source_entity_id="ent-abc123",
            relationship_type="leads",
            target_entity_id="ent-def456",
            confidence=0.95,
            metadata={"context": "Q1 planning"}
        )
    
    assert result.id == "rel-xyz789"
    assert result.confidence == 0.95
    mock_client.create_relationship.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_list(sample_relationship):
    mock_client = AsyncMock()
    mock_client.list_relationships = AsyncMock(return_value=[sample_relationship])
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_list(
            source_entity_id="ent-abc123",
            min_confidence=0.8
        )
    
    assert len(result) == 1
    assert result[0].confidence == 0.95
    mock_client.list_relationships.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_show(sample_relationship):
    mock_client = AsyncMock()
    mock_client.show_relationship = AsyncMock(return_value=sample_relationship)
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_show("rel-xyz789")
    
    assert result.id == "rel-xyz789"
    mock_client.show_relationship.assert_called_once_with("rel-xyz789")


@pytest.mark.asyncio
async def test_beads_relationship_update(sample_relationship):
    updated = Relationship(**sample_relationship.model_dump())
    updated.confidence = 1.0
    
    mock_client = AsyncMock()
    mock_client.update_relationship = AsyncMock(return_value=updated)
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_update("rel-xyz789", confidence=1.0)
    
    assert result.confidence == 1.0
    mock_client.update_relationship.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_delete():
    mock_client = AsyncMock()
    mock_client.delete_relationship = AsyncMock(return_value={"message": "Relationship deleted"})
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_delete("rel-xyz789")
    
    assert result["message"] == "Relationship deleted"
    mock_client.delete_relationship.assert_called_once_with("rel-xyz789")
```

**Pattern to follow:**
- Use `@pytest.mark.asyncio` for all async tests
- Mock `_get_client` to return `AsyncMock` client
- Mock client methods to return sample fixtures
- Assert return values match expected Pydantic models
- Assert client methods called with expected arguments
- Use `@pytest.fixture` for sample data (follow `conftest.py` patterns)

**Note:** Episode and ontology tools (tasks 1-9, 1-10) can be tested in future tasks (Phase 3), or added to these files. Priority is entity/relationship coverage.

## Architecture Pattern

**Test Pattern** (from existing test_tools.py):
- Mock `_get_client` with `patch("beads_mcp.tools._get_client", return_value=mock_client)`
- Mock client methods with `AsyncMock`: `mock_client.method_name = AsyncMock(return_value=...)`
- Use `assert_called_once()` or `assert_called_once_with(args)` to verify calls
- Fixtures provide sample data (reusable across tests)
- Use `model_dump()` from Pydantic to clone fixtures for updates

**Connection Pool Reset** (from conftest.py):
- Tests automatically reset connection pool via `autouse=True` fixture
- No need to manually clear pool in each test

## Validation Criteria
- [ ] `test_entity_tools.py` created with 5 tests (create, list, show, update, delete)
- [ ] `test_relationship_tools.py` created with 5 tests (create, list, show, update, delete)
- [ ] All tests use `@pytest.mark.asyncio` decorator
- [ ] All tests mock `_get_client` and client methods
- [ ] All tests assert return values and method calls
- [ ] Fixtures provide sample Entity and Relationship data
- [ ] Tests follow existing pattern from `test_tools.py`
- [ ] All tests pass (`pytest tests/test_entity_tools.py tests/test_relationship_tools.py`)

## Impact Analysis
- **Direct impact**: Test coverage for new MCP tools
- **Indirect impact**: Validates tasks 1-7 and 1-8 implementations
- **Dependencies**: None (end of Phase 1 task chain)

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Python MCP Tests" section for test patterns)
- Existing tests: `integrations/beads-mcp/tests/test_tools.py` (copy pattern from `test_beads_ready_work`, `test_beads_create_issue`, etc.)
- Test infrastructure: `integrations/beads-mcp/tests/conftest.py` (connection pool reset fixture)

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
