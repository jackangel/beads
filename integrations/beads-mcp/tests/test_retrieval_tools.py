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
