"""Integration tests for MCP relationship tools."""

from datetime import datetime, timezone
from unittest.mock import AsyncMock, patch

import pytest

from beads_mcp.models import Relationship
from beads_mcp.tools import (
    beads_relationship_create,
    beads_relationship_delete,
    beads_relationship_list,
    beads_relationship_show,
    beads_relationship_update,
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
def sample_relationship():
    """Create a sample relationship for testing."""
    return Relationship(
        id="rel-xyz789",
        source_entity_id="ent-abc123",
        relationship_type="leads",
        target_entity_id="ent-def456",
        valid_from="2026-03-01T00:00:00Z",
        valid_until=None,
        confidence=0.95,
        metadata={"context": "team restructure", "source": "org_chart"},
        created_at="2026-03-30T10:00:00Z",
        created_by="admin",
    )


@pytest.mark.asyncio
async def test_beads_relationship_create(sample_relationship):
    """Test beads_relationship_create tool."""
    mock_client = AsyncMock()
    mock_client.create_relationship = AsyncMock(return_value=sample_relationship)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_create(
            source_entity_id="ent-abc123",
            relationship_type="leads",
            target_entity_id="ent-def456",
            valid_from="2026-03-01T00:00:00Z",
            confidence=0.95,
            metadata={"context": "team restructure", "source": "org_chart"},
        )

    assert result.id == "rel-xyz789"
    assert result.source_entity_id == "ent-abc123"
    assert result.relationship_type == "leads"
    assert result.target_entity_id == "ent-def456"
    assert result.confidence == 0.95
    mock_client.create_relationship.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_create_minimal():
    """Test beads_relationship_create with minimal required fields."""
    minimal_relationship = Relationship(
        id="rel-min001",
        source_entity_id="ent-src",
        relationship_type="mentions",
        target_entity_id="ent-tgt",
        valid_from="2026-03-30T10:00:00Z",
        confidence=1.0,
        metadata={},
        created_at="2026-03-30T10:00:00Z",
        created_by="system",
    )
    mock_client = AsyncMock()
    mock_client.create_relationship = AsyncMock(return_value=minimal_relationship)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_create(
            source_entity_id="ent-src",
            relationship_type="mentions",
            target_entity_id="ent-tgt",
        )

    assert result.id == "rel-min001"
    assert result.confidence == 1.0
    assert result.valid_until is None
    mock_client.create_relationship.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_create_with_temporal_bounds(sample_relationship):
    """Test beads_relationship_create with valid_until set."""
    temporal_relationship = Relationship(
        id="rel-temp123",
        source_entity_id="ent-abc123",
        relationship_type="employed_at",
        target_entity_id="ent-company",
        valid_from="2024-01-01T00:00:00Z",
        valid_until="2025-12-31T23:59:59Z",
        confidence=1.0,
        metadata={"role": "engineer"},
        created_at="2026-03-30T10:00:00Z",
        created_by="hr_system",
    )
    mock_client = AsyncMock()
    mock_client.create_relationship = AsyncMock(return_value=temporal_relationship)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_create(
            source_entity_id="ent-abc123",
            relationship_type="employed_at",
            target_entity_id="ent-company",
            valid_from="2024-01-01T00:00:00Z",
            valid_until="2025-12-31T23:59:59Z",
            metadata={"role": "engineer"},
        )

    assert result.valid_from == "2024-01-01T00:00:00Z"
    assert result.valid_until == "2025-12-31T23:59:59Z"
    mock_client.create_relationship.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_list(sample_relationship):
    """Test beads_relationship_list tool."""
    mock_client = AsyncMock()
    mock_client.list_relationships = AsyncMock(return_value=[sample_relationship])

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_list(source_entity_id="ent-abc123")

    assert len(result) == 1
    assert result[0].id == "rel-xyz789"
    assert result[0].source_entity_id == "ent-abc123"
    mock_client.list_relationships.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_list_with_filters(sample_relationship):
    """Test beads_relationship_list with multiple filters."""
    mock_client = AsyncMock()
    mock_client.list_relationships = AsyncMock(return_value=[sample_relationship])

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_list(
            source_entity_id="ent-abc123",
            relationship_type="leads",
            min_confidence=0.9,
        )

    assert len(result) == 1
    assert result[0].relationship_type == "leads"
    assert result[0].confidence >= 0.9
    mock_client.list_relationships.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_list_by_target():
    """Test beads_relationship_list filtered by target entity."""
    relationships = [
        Relationship(
            id="rel-001",
            source_entity_id="ent-alice",
            relationship_type="reports_to",
            target_entity_id="ent-manager",
            valid_from="2026-01-01T00:00:00Z",
            confidence=1.0,
            metadata={},
            created_at="2026-03-30T10:00:00Z",
            created_by="system",
        ),
        Relationship(
            id="rel-002",
            source_entity_id="ent-bob",
            relationship_type="reports_to",
            target_entity_id="ent-manager",
            valid_from="2026-01-01T00:00:00Z",
            confidence=1.0,
            metadata={},
            created_at="2026-03-30T10:00:00Z",
            created_by="system",
        ),
    ]
    mock_client = AsyncMock()
    mock_client.list_relationships = AsyncMock(return_value=relationships)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_list(
            target_entity_id="ent-manager", relationship_type="reports_to"
        )

    assert len(result) == 2
    assert all(r.target_entity_id == "ent-manager" for r in result)
    assert all(r.relationship_type == "reports_to" for r in result)
    mock_client.list_relationships.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_list_empty():
    """Test beads_relationship_list with no results."""
    mock_client = AsyncMock()
    mock_client.list_relationships = AsyncMock(return_value=[])

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_list(source_entity_id="ent-notfound")

    assert len(result) == 0
    mock_client.list_relationships.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_show(sample_relationship):
    """Test beads_relationship_show tool."""
    mock_client = AsyncMock()
    mock_client.show_relationship = AsyncMock(return_value=sample_relationship)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_show("rel-xyz789")

    assert result.id == "rel-xyz789"
    assert result.source_entity_id == "ent-abc123"
    assert result.relationship_type == "leads"
    assert result.target_entity_id == "ent-def456"
    mock_client.show_relationship.assert_called_once_with("rel-xyz789")


@pytest.mark.asyncio
async def test_beads_relationship_update(sample_relationship):
    """Test beads_relationship_update tool."""
    updated_relationship = Relationship(
        id="rel-xyz789",
        source_entity_id="ent-abc123",
        relationship_type="leads",
        target_entity_id="ent-def456",
        valid_from="2026-03-01T00:00:00Z",
        valid_until=None,
        confidence=0.85,
        metadata={
            "context": "team restructure",
            "source": "org_chart",
            "updated": "confidence_adjusted",
        },
        created_at="2026-03-30T10:00:00Z",
        created_by="admin",
    )
    mock_client = AsyncMock()
    mock_client.update_relationship = AsyncMock(return_value=updated_relationship)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_update(
            relationship_id="rel-xyz789",
            confidence=0.85,
            metadata={
                "context": "team restructure",
                "source": "org_chart",
                "updated": "confidence_adjusted",
            },
        )

    assert result.id == "rel-xyz789"
    assert result.confidence == 0.85
    assert result.metadata["updated"] == "confidence_adjusted"
    mock_client.update_relationship.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_update_temporal_bounds():
    """Test beads_relationship_update to set end date."""
    updated_relationship = Relationship(
        id="rel-xyz789",
        source_entity_id="ent-abc123",
        relationship_type="leads",
        target_entity_id="ent-def456",
        valid_from="2026-03-01T00:00:00Z",
        valid_until="2026-12-31T23:59:59Z",
        confidence=1.0,
        metadata={},
        created_at="2026-03-30T10:00:00Z",
        created_by="admin",
    )
    mock_client = AsyncMock()
    mock_client.update_relationship = AsyncMock(return_value=updated_relationship)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_update(
            relationship_id="rel-xyz789", valid_until="2026-12-31T23:59:59Z"
        )

    assert result.valid_until == "2026-12-31T23:59:59Z"
    mock_client.update_relationship.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_delete():
    """Test beads_relationship_delete tool."""
    deletion_result = {"deleted": True, "relationship_id": "rel-xyz789"}
    mock_client = AsyncMock()
    mock_client.delete_relationship = AsyncMock(return_value=deletion_result)

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_relationship_delete("rel-xyz789")

    assert result["deleted"] is True
    assert result["relationship_id"] == "rel-xyz789"
    mock_client.delete_relationship.assert_called_once_with("rel-xyz789")


@pytest.mark.asyncio
async def test_beads_relationship_create_error():
    """Test beads_relationship_create with error handling."""
    mock_client = AsyncMock()
    mock_client.create_relationship = AsyncMock(
        side_effect=Exception("Source entity 'ent-invalid' not found")
    )

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        with pytest.raises(Exception) as exc_info:
            await beads_relationship_create(
                source_entity_id="ent-invalid",
                relationship_type="leads",
                target_entity_id="ent-def456",
            )

    assert "Source entity 'ent-invalid' not found" in str(exc_info.value)
    mock_client.create_relationship.assert_called_once()


@pytest.mark.asyncio
async def test_beads_relationship_show_not_found():
    """Test beads_relationship_show with non-existent relationship."""
    mock_client = AsyncMock()
    mock_client.show_relationship = AsyncMock(
        side_effect=Exception("Relationship 'rel-notfound' not found")
    )

    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        with pytest.raises(Exception) as exc_info:
            await beads_relationship_show("rel-notfound")

    assert "Relationship 'rel-notfound' not found" in str(exc_info.value)
    mock_client.show_relationship.assert_called_once_with("rel-notfound")
