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
