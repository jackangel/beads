"""Integration tests for MCP entity tools."""

from datetime import datetime, timezone
from unittest.mock import AsyncMock, patch

import pytest

from beads_mcp.models import Entity, EntitySearchParams
from beads_mcp.tools import (
    beads_entity_create,
    beads_entity_delete,
    beads_entity_list,
    beads_entity_show,
    beads_entity_update,
)


@pytest.fixture(autouse=True)
def reset_connection_pool():
    """Reset connection pool before and after each test."""
    from beads_mcp import tools

    # Reset connection pool before each test
    tools._connection_pool.clear()
    yield
    # Reset connection pool after each test
    tools._connection_pool.clear()


@pytest.fixture
def sample_entity():
    """Create a sample entity for testing."""
    return Entity(
        id="ent-abc123",
        entity_type="person",
        name="Alice",
        summary="Senior engineer on backend team",
        metadata={"team": "backend", "location": "remote"},
        created_at="2026-03-30T10:00:00Z",
        created_by="bob",
    )


@pytest.mark.asyncio
async def test_beads_entity_create(sample_entity):
    """Test beads_entity_create tool."""
    mock_client = AsyncMock()
    mock_client.create_entity = AsyncMock(return_value=sample_entity)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_create(
            entity_type="person",
            name="Alice",
            summary="Senior engineer on backend team",
            metadata={"team": "backend", "location": "remote"},
            created_by="bob",
        )

    assert result.id == "ent-abc123"
    assert result.name == "Alice"
    assert result.entity_type == "person"
    assert result.summary == "Senior engineer on backend team"
    assert result.metadata == {"team": "backend", "location": "remote"}
    mock_client.create_entity.assert_called_once()


@pytest.mark.asyncio
async def test_beads_entity_create_with_custom_id(sample_entity):
    """Test beads_entity_create with custom ID."""
    custom_entity = Entity(
        id="ent-custom-001",
        entity_type="project",
        name="Beads v2",
        summary="Knowledge graph enhancement project",
        metadata={},
        created_at="2026-03-30T10:00:00Z",
        created_by="admin",
    )
    mock_client = AsyncMock()
    mock_client.create_entity = AsyncMock(return_value=custom_entity)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_create(
            entity_type="project",
            name="Beads v2",
            summary="Knowledge graph enhancement project",
            id="ent-custom-001",
            created_by="admin",
        )

    assert result.id == "ent-custom-001"
    assert result.name == "Beads v2"
    mock_client.create_entity.assert_called_once()


@pytest.mark.asyncio
async def test_beads_entity_list(sample_entity):
    """Test beads_entity_list tool."""
    mock_client = AsyncMock()
    mock_client.list_entities = AsyncMock(return_value=[sample_entity])

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_list(entity_type="person", limit=10)

    assert len(result) == 1
    assert result[0].id == "ent-abc123"
    assert result[0].name == "Alice"
    mock_client.list_entities.assert_called_once()


@pytest.mark.asyncio
async def test_beads_entity_list_with_filters(sample_entity):
    """Test beads_entity_list with multiple filters."""
    mock_client = AsyncMock()
    mock_client.list_entities = AsyncMock(return_value=[sample_entity])

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_list(
            entity_type="person", name="Alice", created_by="bob", limit=50, offset=0
        )

    assert len(result) == 1
    assert result[0].name == "Alice"
    assert result[0].created_by == "bob"
    mock_client.list_entities.assert_called_once()


@pytest.mark.asyncio
async def test_beads_entity_list_empty():
    """Test beads_entity_list with no results."""
    mock_client = AsyncMock()
    mock_client.list_entities = AsyncMock(return_value=[])

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_list(entity_type="project")

    assert len(result) == 0
    mock_client.list_entities.assert_called_once()


@pytest.mark.asyncio
async def test_beads_entity_show(sample_entity):
    """Test beads_entity_show tool."""
    mock_client = AsyncMock()
    mock_client.show_entity = AsyncMock(return_value=sample_entity)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_show("ent-abc123")

    assert result.id == "ent-abc123"
    assert result.name == "Alice"
    assert result.summary == "Senior engineer on backend team"
    mock_client.show_entity.assert_called_once_with("ent-abc123")


@pytest.mark.asyncio
async def test_beads_entity_update(sample_entity):
    """Test beads_entity_update tool."""
    updated_entity = Entity(
        id="ent-abc123",
        entity_type="person",
        name="Alice",
        summary="Principal engineer on backend team",
        metadata={"team": "backend", "location": "remote", "level": "principal"},
        created_at="2026-03-30T10:00:00Z",
        updated_at="2026-03-30T11:00:00Z",
        created_by="bob",
        updated_by="admin",
    )
    mock_client = AsyncMock()
    mock_client.update_entity = AsyncMock(return_value=updated_entity)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_update(
            entity_id="ent-abc123",
            summary="Principal engineer on backend team",
            metadata={"team": "backend", "location": "remote", "level": "principal"},
        )

    assert result.id == "ent-abc123"
    assert result.summary == "Principal engineer on backend team"
    assert result.updated_at == "2026-03-30T11:00:00Z"
    assert result.metadata["level"] == "principal"
    mock_client.update_entity.assert_called_once()


@pytest.mark.asyncio
async def test_beads_entity_update_name_only(sample_entity):
    """Test beads_entity_update with name change only."""
    updated_entity = Entity(
        id="ent-abc123",
        entity_type="person",
        name="Alice Smith",
        summary="Senior engineer on backend team",
        metadata={"team": "backend", "location": "remote"},
        created_at="2026-03-30T10:00:00Z",
        updated_at="2026-03-30T11:00:00Z",
        created_by="bob",
        updated_by="admin",
    )
    mock_client = AsyncMock()
    mock_client.update_entity = AsyncMock(return_value=updated_entity)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_update(entity_id="ent-abc123", name="Alice Smith")

    assert result.name == "Alice Smith"
    assert result.updated_at is not None
    mock_client.update_entity.assert_called_once()


@pytest.mark.asyncio
async def test_beads_entity_delete():
    """Test beads_entity_delete tool."""
    deletion_result = {
        "deleted": True,
        "entity_id": "ent-abc123",
        "relationships_deleted": 3,
    }
    mock_client = AsyncMock()
    mock_client.delete_entity = AsyncMock(return_value=deletion_result)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_entity_delete("ent-abc123")

    assert result["deleted"] is True
    assert result["entity_id"] == "ent-abc123"
    assert result["relationships_deleted"] == 3
    mock_client.delete_entity.assert_called_once_with("ent-abc123")


@pytest.mark.asyncio
async def test_beads_entity_create_error():
    """Test beads_entity_create with error handling."""
    mock_client = AsyncMock()
    mock_client.create_entity = AsyncMock(
        side_effect=Exception("Entity type 'invalid' not found")
    )

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        with pytest.raises(Exception) as exc_info:
            await beads_entity_create(
                entity_type="invalid",
                name="Test",
                summary="Test entity",
            )

    assert "Entity type 'invalid' not found" in str(exc_info.value)
    mock_client.create_entity.assert_called_once()


@pytest.mark.asyncio
async def test_beads_entity_show_not_found():
    """Test beads_entity_show with non-existent entity."""
    mock_client = AsyncMock()
    mock_client.show_entity = AsyncMock(
        side_effect=Exception("Entity 'ent-notfound' not found")
    )

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        with pytest.raises(Exception) as exc_info:
            await beads_entity_show("ent-notfound")

    assert "Entity 'ent-notfound' not found" in str(exc_info.value)
    mock_client.show_entity.assert_called_once_with("ent-notfound")
