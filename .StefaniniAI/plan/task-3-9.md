# Task 3-9: Add Python tests for extraction and retrieval MCP tools

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Risk Level**: LOW
- **Phase**: 3
- **Depends On**: 3-8

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\tests\test_extraction_tools.py` (NEW)
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\tests\test_retrieval_tools.py` (NEW)

## Instructions
Create pytest tests for extraction and retrieval MCP tools.

**File: `test_extraction_tools.py`**
```python
import pytest
from unittest.mock import AsyncMock, patch
from beads_mcp.tools import beads_episode_extract


@pytest.mark.asyncio
async def test_beads_episode_extract():
    mock_client = AsyncMock()
    mock_client.extract_episode = AsyncMock(return_value={
        "episode_id": "ep-abc123",
        "entities_created": 3,
        "relationships_created": 2
    })
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_episode_extract("ep-abc123")
    
    assert result["entities_created"] == 3
    assert result["relationships_created"] == 2
    mock_client.extract_episode.assert_called_once_with("ep-abc123")
```

**File: `test_retrieval_tools.py`**
```python
import pytest
from unittest.mock import AsyncMock, patch
from beads_mcp.tools import beads_memory_retrieve


@pytest.mark.asyncio
async def test_beads_memory_retrieve():
    mock_context = {
        "entities": [{"id": "ent-1", "name": "Alice"}],
        "relationships": [],
        "relevance_scores": {"ent-1": 1.0}
    }
    
    mock_client = AsyncMock()
    mock_client.retrieve_memory = AsyncMock(return_value=mock_context)
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_memory_retrieve("Alice's role", max_hops=2, top_k=5)
    
    assert len(result["entities"]) == 1
    assert result["entities"][0]["name"] == "Alice"
    mock_client.retrieve_memory.assert_called_once_with("Alice's role", 2, 5, 0.5)
```

## Validation Criteria
- [ ] `test_extraction_tools.py` created with test for beads_episode_extract
- [ ] `test_retrieval_tools.py` created with test for beads_memory_retrieve
- [ ] All tests use `@pytest.mark.asyncio`
- [ ] All tests mock `_get_client` and client methods
- [ ] Tests assert return values and method calls
- [ ] All tests pass

## Impact Analysis
- **Direct impact**: Test coverage for extraction and retrieval tools
- **Indirect impact**: Validates Phase 3 implementations
- **Dependencies**: Task 3-8 (MCP tools)

## Context
- Pattern: Task 1-12 (entity/relationship test pattern)

## User Feedback
*(Empty)*
