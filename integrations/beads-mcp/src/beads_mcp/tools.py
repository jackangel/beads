"""MCP tools for beads issue tracker."""

import asyncio
import logging
import os
import subprocess
import sys
from contextvars import ContextVar
from functools import lru_cache
from typing import Annotated, Any, TYPE_CHECKING

from .bd_client import create_bd_client, BdClientBase, BdError

logger = logging.getLogger(__name__)

if TYPE_CHECKING:
    from typing import List
from .models import (
    AddDependencyParams,
    BlockedIssue,
    BlockedParams,
    ClaimIssueParams,
    CloseIssueParams,
    CreateEntityParams,
    CreateEpisodeParams,
    CreateIssueParams,
    CreateRelationshipParams,
    DependencyType,
    Entity,
    EntitySearchParams,
    EntityType,
    Episode,
    InitParams,
    Issue,
    IssueStatus,
    IssueType,
    ListIssuesParams,
    ReadyWorkParams,
    Relationship,
    RelationshipType,
    ReopenIssueParams,
    ShowIssueParams,
    Stats,
    UpdateIssueParams,
)

# ContextVar for request-scoped workspace routing
current_workspace: ContextVar[str | None] = ContextVar('workspace', default=None)

# Connection pool for per-project clients
_connection_pool: dict[str, BdClientBase] = {}
_pool_lock = asyncio.Lock()

# Version checking state (per-pool client)
_version_checked: set[str] = set()

# Default constants
DEFAULT_ISSUE_TYPE: IssueType = "task"
DEFAULT_DEPENDENCY_TYPE: DependencyType = "blocks"


def _register_client_for_cleanup(client: BdClientBase) -> None:
    """Register client with server cleanup system.
    
    This ensures client connections are properly closed on server shutdown.
    Import is deferred to avoid circular dependency.
    """
    try:
        from . import server
        if hasattr(server, '_registered_clients'):
            server._registered_clients.append(client)
    except (ImportError, AttributeError):
        # Server module not available or cleanup not initialized - that's ok
        pass


def _resolve_beads_redirect(beads_dir: str, workspace_root: str) -> str | None:
    """Follow a .beads/redirect file to the actual beads directory.

    Args:
        beads_dir: Path to the .beads directory that may contain a redirect
        workspace_root: The workspace root directory (parent of beads_dir)

    Returns:
        Resolved workspace root if redirect is valid, None otherwise
    """
    import glob

    redirect_path = os.path.join(beads_dir, "redirect")
    if not os.path.isfile(redirect_path):
        return None

    try:
        with open(redirect_path, 'r') as f:
            redirect_target = f.read().strip()

        if not redirect_target:
            return None

        # Resolve relative to workspace_root (the redirect is written from the perspective
        # of being inside workspace_root, not inside workspace_root/.beads)
        # e.g., redirect contains "../../mayor/rig/.beads"
        # from polecats/capable/, this resolves to mayor/rig/.beads
        resolved = os.path.normpath(os.path.join(workspace_root, redirect_target))

        if not os.path.isdir(resolved):
            logger.debug(f"Redirect target {resolved} does not exist")
            return None

        # Verify the redirected location has a valid database
        db_files = glob.glob(os.path.join(resolved, "*.db"))
        valid_dbs = [f for f in db_files if ".backup" not in os.path.basename(f)]

        if not valid_dbs:
            logger.debug(f"Redirect target {resolved} has no valid .db files")
            return None

        # Return the workspace root of the redirected location (parent of .beads)
        return os.path.dirname(resolved)

    except Exception as e:
        logger.debug(f"Failed to follow redirect: {e}")
        return None


def _find_beads_db_in_tree(start_dir: str | None = None) -> str | None:
    """Walk up directory tree looking for .beads/*.db (matches Go CLI behavior).

    Also follows .beads/redirect files to shared beads locations, which is
    essential for polecat/crew directories that share a central database.

    Args:
        start_dir: Starting directory (default: current working directory)

    Returns:
        Absolute path to workspace root containing .beads/*.db, or None if not found
    """
    import glob

    try:
        current = os.path.abspath(start_dir or os.getcwd())

        # Resolve symlinks like Go CLI does
        try:
            current = os.path.realpath(current)
        except Exception:
            pass

        # Walk up directory tree
        while True:
            beads_dir = os.path.join(current, ".beads")
            if os.path.isdir(beads_dir):
                # First, check for redirect file (polecat/crew directories use this)
                redirected = _resolve_beads_redirect(beads_dir, current)
                if redirected:
                    logger.debug(f"Followed redirect from {current} to {redirected}")
                    return redirected

                # No redirect, check for local .db files
                db_files = glob.glob(os.path.join(beads_dir, "*.db"))
                valid_dbs = [f for f in db_files if ".backup" not in os.path.basename(f)]

                if valid_dbs:
                    # Return workspace root (parent of .beads), not the db path
                    return current

            parent = os.path.dirname(current)
            if parent == current:  # Reached filesystem root
                break
            current = parent

        return None

    except Exception as e:
        logger.debug(f"Failed to search for .beads in tree: {e}")
        return None


def _resolve_workspace_root(path: str) -> str:
    """Resolve workspace root to git repo root if inside a git repo.
    
    Args:
        path: Directory path to resolve
        
    Returns:
        Git repo root if inside git repo, otherwise the original path
    """
    try:
        result = subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            cwd=path,
            capture_output=True,
            text=True,
            check=False,
            shell=sys.platform == "win32",
            stdin=subprocess.DEVNULL,  # Prevent inheriting MCP's stdin
        )
        if result.returncode == 0:
            return result.stdout.strip()
    except Exception as e:
        logger.debug(f"Git detection failed for {path}: {e}")
        pass
    
    return os.path.abspath(path)


@lru_cache(maxsize=128)
def _canonicalize_path(path: str) -> str:
    """Canonicalize workspace path to handle symlinks and git repos.
    
    This ensures that different paths pointing to the same project
    (e.g., via symlinks) use the same client connection.
    
    Args:
        path: Workspace directory path
        
    Returns:
        Canonical path (handles symlinks and submodules correctly)
    """
    # 1. Resolve symlinks
    real = os.path.realpath(path)
    
    # 2. Check for local .beads directory (submodule edge case)
    # Submodules should use their own .beads, not the parent repo's
    if os.path.exists(os.path.join(real, ".beads")):
        return real
    
    # 3. Try to find git toplevel
    # This ensures we connect to the right client for the git repo
    return _resolve_workspace_root(real)


async def _health_check_client(client: BdClientBase) -> bool:
    """Check if a client is healthy and responsive.
    
    Args:
        client: Client to health check
        
    Returns:
        True if client is healthy, False otherwise
    """
    # Only health check clients that support ping
    if not hasattr(client, 'ping'):
        return True
    
    try:
        await client.ping()
        return True
    except Exception:
        # Any exception means the client is stale/unhealthy
        return False


async def _reconnect_client(canonical: str, max_retries: int = 3) -> BdClientBase:
    """Attempt to reconnect with exponential backoff.

    Args:
        canonical: Canonical workspace path
        max_retries: Maximum number of retry attempts (default: 3)

    Returns:
        New client instance

    Raises:
        BdError: If all reconnection attempts fail
    """
    for attempt in range(max_retries):
        try:
            client = create_bd_client(
                working_dir=canonical
            )
            
            # Verify new client works
            if await _health_check_client(client):
                _register_client_for_cleanup(client)
                return client
                
        except Exception:
            if attempt < max_retries - 1:
                # Exponential backoff: 0.1s, 0.2s, 0.4s
                backoff = 0.1 * (2 ** attempt)
                await asyncio.sleep(backoff)
            continue
    
    raise BdError(
        f"Failed to connect after {max_retries} attempts. "
        "The bd client may be misconfigured or unresponsive."
    )


async def _get_client(working_dir: str | None = None) -> BdClientBase:
    """Get a BdClient instance for the current workspace.
    
    Uses connection pool to manage per-project clients.
    Workspace is auto-detected using the same logic as CLI:
    1. working_dir parameter (explicit override)
    2. current_workspace ContextVar (from workspace_root parameter)
    3. BEADS_WORKING_DIR environment variable
    4. Walk up from CWD looking for .beads/*.db
    
    Performs health check before returning cached client.
    On failure, drops from pool and attempts reconnection with exponential backoff.
    
    Performs version check on first connection to each workspace.
    Uses CLI client for all operations.

    Args:
        working_dir: Optional workspace directory override

    Returns:
        Configured BdClientBase instance for the current workspace

    Raises:
        BdError: If no workspace found, or bd is not installed, or version is incompatible
    """
    # Determine workspace using standard search order (matches Go CLI)
    workspace = working_dir or current_workspace.get() or os.environ.get("BEADS_WORKING_DIR")
    
    # Auto-detect from CWD if not explicitly set (NEW!)
    if not workspace:
        workspace = _find_beads_db_in_tree()
        if workspace:
            logger.debug(f"Auto-detected workspace from CWD: {workspace}")
    
    if not workspace:
        raise BdError(
            "No beads workspace found. Either:\n"
            "  1. Call context(workspace_root=\"/path/to/project\"), OR\n"
            "  2. Run from a directory containing .beads/, OR\n"
            "  3. Set BEADS_WORKING_DIR environment variable"
        )
    
    # Canonicalize path to handle symlinks and deduplicate connections
    canonical = _canonicalize_path(workspace)
    
    # Thread-safe connection pool access
    async with _pool_lock:
        if canonical in _connection_pool:
            # Health check cached client before returning
            client = _connection_pool[canonical]
            if not await _health_check_client(client):
                # Stale connection - remove from pool and reconnect
                del _connection_pool[canonical]
                if canonical in _version_checked:
                    _version_checked.remove(canonical)
                
                # Attempt reconnection with backoff
                client = await _reconnect_client(canonical)
                _connection_pool[canonical] = client
        else:
            # Create new client for this workspace
            client = create_bd_client(
                working_dir=canonical
            )
            
            # Register for cleanup
            _register_client_for_cleanup(client)
            
            # Add to pool
            _connection_pool[canonical] = client
    
    # Check version once per workspace (only for CLI client)
    if canonical not in _version_checked:
        if hasattr(client, '_check_version'):
            await client._check_version()
        _version_checked.add(canonical)

    return client


async def beads_ready_work(
    limit: Annotated[int, "Maximum number of issues to return (1-100)"] = 10,
    priority: Annotated[int | None, "Filter by priority (0-4, 0=highest)"] = None,
    assignee: Annotated[str | None, "Filter by assignee"] = None,
    labels: Annotated[list[str] | None, "Filter by labels (AND: must have ALL)"] = None,
    labels_any: Annotated[list[str] | None, "Filter by labels (OR: must have at least one)"] = None,
    unassigned: Annotated[bool, "Filter to only unassigned issues"] = False,
    sort_policy: Annotated[str | None, "Sort policy: hybrid (default), priority, oldest"] = None,
    parent: Annotated[str | None, "Filter to descendants of this bead/epic"] = None,
) -> list[Issue]:
    """Find issues with no blocking dependencies that are ready to work on.

    Ready work = status is 'open' AND no blocking dependencies.
    Perfect for agents to claim next work!

    Use 'parent' to filter to all descendants of an epic/bead.

    DEPRECATED: This tool uses the legacy v7 issue-tracking API.
    Consider using the v8 knowledge graph API (entity/relationship/episode tools) for new work.
    Legacy tools will be maintained for backward compatibility.
    """
    client = await _get_client()
    params = ReadyWorkParams(
        limit=limit,
        priority=priority,
        assignee=assignee,
        labels=labels,
        labels_any=labels_any,
        unassigned=unassigned,
        sort_policy=sort_policy,
        parent_id=parent,
    )
    return await client.ready(params)


async def beads_list_issues(
    status: Annotated[IssueStatus | None, "Filter by status (open, in_progress, blocked, deferred, closed, or custom)"] = None,
    priority: Annotated[int | None, "Filter by priority (0-4, 0=highest)"] = None,
    issue_type: Annotated[IssueType | None, "Filter by type (bug, feature, task, epic, chore, decision, or custom)"] = None,
    assignee: Annotated[str | None, "Filter by assignee"] = None,
    labels: Annotated[list[str] | None, "Filter by labels (AND: must have ALL)"] = None,
    labels_any: Annotated[list[str] | None, "Filter by labels (OR: must have at least one)"] = None,
    query: Annotated[str | None, "Search in title (case-insensitive substring)"] = None,
    unassigned: Annotated[bool, "Filter to only unassigned issues"] = False,
    limit: Annotated[int, "Maximum number of issues to return (1-100)"] = 20,
) -> list[Issue]:
    """List all issues with optional filters.

    DEPRECATED: This tool uses the legacy v7 issue-tracking API.
    Consider using the v8 knowledge graph API (entity/relationship/episode tools) for new work.
    Legacy tools will be maintained for backward compatibility.
    """
    client = await _get_client()

    params = ListIssuesParams(
        status=status,
        priority=priority,
        issue_type=issue_type,
        assignee=assignee,
        labels=labels,
        labels_any=labels_any,
        query=query,
        unassigned=unassigned,
        limit=limit,
    )
    return await client.list_issues(params)


async def beads_show_issue(
    issue_id: Annotated[str, "Issue ID (e.g., bd-1)"],
) -> Issue:
    """Show detailed information about a specific issue.

    Includes full description, dependencies, and dependents.

    DEPRECATED: This tool uses the legacy v7 issue-tracking API.
    Consider using the v8 knowledge graph API (entity/relationship/episode tools) for new work.
    Legacy tools will be maintained for backward compatibility.
    """
    client = await _get_client()
    params = ShowIssueParams(issue_id=issue_id)
    return await client.show(params)


async def beads_create_issue(
    title: Annotated[str, "Issue title"],
    description: Annotated[str, "Issue description"] = "",
    design: Annotated[str | None, "Design notes"] = None,
    acceptance: Annotated[str | None, "Acceptance criteria"] = None,
    external_ref: Annotated[str | None, "External reference (e.g., gh-9, jira-ABC)"] = None,
    priority: Annotated[int, "Priority (0-4, 0=highest)"] = 2,
    issue_type: Annotated[IssueType, "Type: bug, feature, task, epic, chore, decision, or custom"] = DEFAULT_ISSUE_TYPE,
    assignee: Annotated[str | None, "Assignee username"] = None,
    labels: Annotated[list[str] | None, "List of labels"] = None,
    id: Annotated[str | None, "Explicit issue ID (e.g., bd-42)"] = None,
    deps: Annotated[list[str] | None, "Dependencies (e.g., ['bd-20', 'blocks:bd-15'])"] = None,
) -> Issue:
    """Create a new issue.

    IMPORTANT: Always provide a meaningful description with context about:
    - Why this issue exists (problem statement or need)
    - What needs to be done (scope and approach)
    - How you discovered it (if applicable)

    Issues without descriptions lack context for future work and make prioritization difficult.

    Use this when you discover new work during your session.
    Link it back with beads_add_dependency using 'discovered-from' type.

    DEPRECATED: This tool uses the legacy v7 issue-tracking API.
    Consider using the v8 knowledge graph API (entity/relationship/episode tools) for new work.
    Legacy tools will be maintained for backward compatibility.
    """
    client = await _get_client()
    params = CreateIssueParams(
        title=title,
        description=description,
        design=design,
        acceptance=acceptance,
        external_ref=external_ref,
        priority=priority,
        issue_type=issue_type,
        assignee=assignee,
        labels=labels or [],
        id=id,
        deps=deps or [],
    )
    return await client.create(params)


async def beads_update_issue(
    issue_id: Annotated[str, "Issue ID (e.g., bd-1)"],
    status: Annotated[IssueStatus | None, "New status (open, in_progress, blocked, deferred, closed, or custom)"] = None,
    priority: Annotated[int | None, "New priority (0-4)"] = None,
    assignee: Annotated[str | None, "New assignee"] = None,
    title: Annotated[str | None, "New title"] = None,
    description: Annotated[str | None, "Issue description"] = None,
    design: Annotated[str | None, "Design notes"] = None,
    acceptance_criteria: Annotated[str | None, "Acceptance criteria"] = None,
    notes: Annotated[str | None, "Additional notes"] = None,
    external_ref: Annotated[str | None, "External reference (e.g., gh-9, jira-ABC)"] = None,
) -> Issue | list[Issue]:
    """Update an existing issue.

    Note: Setting status to 'closed' or 'open' will automatically route to
    beads_close_issue() or beads_reopen_issue() respectively to ensure
    proper approval workflows are followed.

    DEPRECATED: This tool uses the legacy v7 issue-tracking API.
    Consider using the v8 knowledge graph API (entity/relationship/episode tools) for new work.
    Legacy tools will be maintained for backward compatibility.
    """
    # Smart routing: intercept lifecycle status changes and route to dedicated tools
    if status == "closed":
        # Route to close tool to respect approval workflows
        reason = notes if notes else "Completed"
        return await beads_close_issue(issue_id=issue_id, reason=reason)
    
    if status == "open":
        # Route to reopen tool to respect approval workflows
        reason = notes if notes else "Reopened"
        return await beads_reopen_issue(issue_ids=[issue_id], reason=reason)
    
    # Normal attribute updates proceed as usual
    client = await _get_client()
    params = UpdateIssueParams(
        issue_id=issue_id,
        status=status,
        priority=priority,
        assignee=assignee,
        title=title,
        description=description,
        design=design,
        acceptance_criteria=acceptance_criteria,
        notes=notes,
        external_ref=external_ref,
    )
    return await client.update(params)


async def beads_claim_issue(
    issue_id: Annotated[str, "Issue ID (e.g., bd-1)"],
) -> Issue:
    """Atomically claim an issue for work.

    Uses `bd update <id> --claim` semantics: sets assignee + in_progress in one
    compare-and-swap operation and fails if already claimed.

    DEPRECATED: This tool uses the legacy v7 issue-tracking API.
    Consider using the v8 knowledge graph API (entity/relationship/episode tools) for new work.
    Legacy tools will be maintained for backward compatibility.
    """
    client = await _get_client()
    params = ClaimIssueParams(issue_id=issue_id)
    return await client.claim(params)


async def beads_close_issue(
    issue_id: Annotated[str, "Issue ID (e.g., bd-1)"],
    reason: Annotated[str, "Reason for closing"] = "Completed",
) -> list[Issue]:
    """Close (complete) an issue.

    Mark work as done when you've finished implementing/fixing it.

    DEPRECATED: This tool uses the legacy v7 issue-tracking API.
    Consider using the v8 knowledge graph API (entity/relationship/episode tools) for new work.
    Legacy tools will be maintained for backward compatibility.
    """
    client = await _get_client()
    params = CloseIssueParams(issue_id=issue_id, reason=reason)
    return await client.close(params)


async def beads_reopen_issue(
    issue_ids: Annotated[list[str], "Issue IDs to reopen (e.g., ['bd-1', 'bd-2'])"],
    reason: Annotated[str | None, "Reason for reopening"] = None,
) -> list[Issue]:
    """Reopen one or more closed issues.

    Sets status to 'open' and clears the closed_at timestamp.
    More explicit than 'update --status open'.

    DEPRECATED: This tool uses the legacy v7 issue-tracking API.
    Consider using the v8 knowledge graph API (entity/relationship/episode tools) for new work.
    Legacy tools will be maintained for backward compatibility.
    """
    client = await _get_client()
    params = ReopenIssueParams(issue_ids=issue_ids, reason=reason)
    return await client.reopen(params)


async def beads_add_dependency(
    issue_id: Annotated[str, "Issue that has the dependency (e.g., bd-2)"],
    depends_on_id: Annotated[str, "Issue that issue_id depends on (e.g., bd-1)"],
    dep_type: Annotated[
        DependencyType,
        "Dependency type: blocks, related, parent-child, or discovered-from",
    ] = DEFAULT_DEPENDENCY_TYPE,
) -> str:
    """Add a dependency relationship between two issues.

    Types:
    - blocks: depends_on_id must complete before issue_id can start
    - related: Soft connection, doesn't block progress
    - parent-child: Epic/subtask hierarchical relationship
    - discovered-from: Track that issue_id was discovered while working on depends_on_id

    Use 'discovered-from' when you find new work during your session.

    DEPRECATED: This tool uses the legacy v7 issue-tracking API.
    Consider using the v8 knowledge graph API (entity/relationship/episode tools) for new work.
    Legacy tools will be maintained for backward compatibility.
    """
    client = await _get_client()
    params = AddDependencyParams(
        issue_id=issue_id,
        depends_on_id=depends_on_id,
        dep_type=dep_type,
    )
    try:
        await client.add_dependency(params)
        return f"Added dependency: {issue_id} depends on {depends_on_id} ({dep_type})"
    except BdError as e:
        return f"Error: {str(e)}"


async def beads_quickstart() -> str:
    """Get bd quickstart guide.

    Read this first to understand how to use beads (bd) commands.
    """
    client = await _get_client()
    return await client.quickstart()


async def beads_stats() -> Stats:
    """Get statistics about issues.

    Returns total issues, open, in_progress, closed, blocked, ready issues,
    and average lead time in hours.

    DEPRECATED: This tool uses the legacy v7 issue-tracking API.
    Consider using the v8 knowledge graph API (entity/relationship/episode tools) for new work.
    Legacy tools will be maintained for backward compatibility.
    """
    client = await _get_client()
    return await client.stats()


async def beads_blocked(
    parent: Annotated[str | None, "Filter to descendants of this bead/epic"] = None,
) -> list[BlockedIssue]:
    """Get blocked issues.

    Returns issues that have blocking dependencies, showing what blocks them.

    Use 'parent' to filter to all descendants of an epic/bead.

    DEPRECATED: This tool uses the legacy v7 issue-tracking API.
    Consider using the v8 knowledge graph API (entity/relationship/episode tools) for new work.
    Legacy tools will be maintained for backward compatibility.
    """
    client = await _get_client()
    params = BlockedParams(parent_id=parent)
    return await client.blocked(params)


async def beads_inspect_migration() -> dict[str, Any]:
    """Get migration plan and database state for agent analysis.
    
    AI agents should:
    1. Review registered_migrations to understand what will run
    2. Check warnings array for issues (missing config, version mismatch)
    3. Verify missing_config is empty before migrating
    4. Check invariants_to_check to understand safety guarantees
    
    Returns migration plan, current db state, warnings, and invariants.
    """
    client = await _get_client()
    return await client.inspect_migration()


async def beads_get_schema_info() -> dict[str, Any]:
    """Get current database schema for inspection.
    
    Returns tables, schema version, config, sample issue IDs, and detected prefix.
    Useful for verifying database state before migrations.
    """
    client = await _get_client()
    return await client.get_schema_info()


async def beads_repair_deps(
    fix: Annotated[bool, "If True, automatically remove orphaned dependencies"] = False,
) -> dict[str, Any]:
    """Find and optionally fix orphaned dependency references.
    
    Scans all issues for dependencies pointing to non-existent issues.
    Returns orphaned dependencies and optionally removes them with fix=True.
    
    Returns dict with:
    - orphans_found: number of orphaned dependencies
    - orphans: list of orphaned dependency details
    - fixed: number of orphans fixed (if fix=True)
    """
    client = await _get_client()
    return await client.repair_deps(fix=fix)


async def beads_detect_pollution(
    clean: Annotated[bool, "If True, delete detected test issues"] = False,
) -> dict[str, Any]:
    """Detect test issues that leaked into production database.
    
    Detects test issues using pattern matching:
    - Titles starting with 'test', 'benchmark', 'sample', 'tmp', 'temp'
    - Sequential numbering (test-1, test-2, ...)
    - Generic descriptions or no description
    - Created in rapid succession
    
    Returns dict with detected test issues and deleted count if clean=True.
    """
    client = await _get_client()
    return await client.detect_pollution(clean=clean)


async def beads_validate(
    checks: Annotated[str | None, "Comma-separated list of checks (orphans,duplicates,pollution,conflicts)"] = None,
    fix_all: Annotated[bool, "If True, auto-fix all fixable issues"] = False,
) -> dict[str, Any]:
    """Run comprehensive database health checks.
    
    Available checks:
    - orphans: Orphaned dependencies (references to deleted issues)
    - duplicates: Duplicate issues (identical content)
    - pollution: Test pollution (leaked test issues)
    - conflicts: Git merge conflicts in JSONL
    
    If checks is None, runs all checks.
    
    Returns dict with validation results for each check.
    """
    client = await _get_client()
    return await client.validate(checks=checks, fix_all=fix_all)


async def beads_init(
    prefix: Annotated[str | None, "Issue prefix (e.g., 'myproject' for myproject-1, myproject-2)"] = None,
) -> str:
    """Initialize bd in current directory.

    Creates .beads/ directory and database file with optional custom prefix.
    """
    client = await _get_client()
    params = InitParams(prefix=prefix)
    return await client.init(params)


# =============================================================================
# V8 Knowledge Graph API (Entity/Relationship/Episode/Ontology)
# =============================================================================


async def beads_entity_create(
    entity_type: Annotated[str, "Type of entity (e.g., 'person', 'project', 'meeting')"],
    name: Annotated[str, "Entity name (unique within type)"],
    summary: Annotated[str, "Brief description of the entity"],
    metadata: Annotated[dict[str, Any] | None, "Additional structured data (optional)"] = None,
    created_by: Annotated[str | None, "Creator identifier (optional)"] = None,
    id: Annotated[str | None, "Custom entity ID (optional, auto-generated if not provided)"] = None,
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> Entity:
    """Create a new entity in the knowledge graph.
    
    Args:
        entity_type: Type of entity (e.g., "person", "project", "meeting")
        name: Entity name (unique within type)
        summary: Brief description of the entity
        metadata: Additional structured data (optional)
        created_by: Creator identifier (optional)
        id: Custom entity ID (optional, auto-generated if not provided)
        working_dir: beads workspace directory (optional)
    
    Returns:
        Entity: Created entity with ID
    """
    client = await _get_client(working_dir)
    params = CreateEntityParams(
        entity_type=entity_type,
        name=name,
        summary=summary,
        metadata=metadata,
        created_by=created_by,
        id=id
    )
    return await client.create_entity(params)


async def beads_entity_list(
    entity_type: Annotated[str | None, "Filter by entity type (optional)"] = None,
    name: Annotated[str | None, "Filter by name substring (optional)"] = None,
    created_by: Annotated[str | None, "Filter by creator (optional)"] = None,
    limit: Annotated[int, "Maximum number of results (default 50)"] = 50,
    offset: Annotated[int, "Pagination offset (default 0)"] = 0,
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> list[Entity]:
    """List entities with optional filtering.
    
    Args:
        entity_type: Filter by entity type (optional)
        name: Filter by name substring (optional)
        created_by: Filter by creator (optional)
        limit: Maximum number of results (default 50)
        offset: Pagination offset (default 0)
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[Entity]: Matching entities
    """
    client = await _get_client(working_dir)
    params = EntitySearchParams(
        entity_type=entity_type,
        name=name,
        created_by=created_by,
        limit=limit,
        offset=offset
    )
    return await client.list_entities(params)


async def beads_entity_show(
    entity_id: Annotated[str, "Entity ID"],
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> Entity:
    """Show detailed information about an entity.
    
    Args:
        entity_id: Entity ID
        working_dir: beads workspace directory (optional)
    
    Returns:
        Entity: Full entity details
    """
    client = await _get_client(working_dir)
    return await client.show_entity(entity_id)


async def beads_entity_update(
    entity_id: Annotated[str, "Entity ID to update"],
    name: Annotated[str | None, "New name (optional)"] = None,
    summary: Annotated[str | None, "New summary (optional)"] = None,
    metadata: Annotated[dict[str, Any] | None, "New metadata (optional)"] = None,
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> Entity:
    """Update an existing entity.
    
    Args:
        entity_id: Entity ID to update
        name: New name (optional)
        summary: New summary (optional)
        metadata: New metadata (optional)
        working_dir: beads workspace directory (optional)
    
    Returns:
        Entity: Updated entity
    """
    client = await _get_client(working_dir)
    kwargs = {}
    if name is not None:
        kwargs['name'] = name
    if summary is not None:
        kwargs['summary'] = summary
    if metadata is not None:
        kwargs['metadata'] = metadata
    return await client.update_entity(entity_id, **kwargs)


async def beads_entity_delete(
    entity_id: Annotated[str, "Entity ID to delete"],
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> dict[str, Any]:
    """Delete an entity (cascade deletes relationships).
    
    Args:
        entity_id: Entity ID to delete
        working_dir: beads workspace directory (optional)
    
    Returns:
        dict: Deletion confirmation with deleted entity count
    """
    client = await _get_client(working_dir)
    return await client.delete_entity(entity_id)


async def beads_relationship_create(
    source_entity_id: Annotated[str, "Source entity ID"],
    relationship_type: Annotated[str, "Relationship type (e.g., 'leads', 'blocks', 'mentioned_in')"],
    target_entity_id: Annotated[str, "Target entity ID"],
    valid_from: Annotated[str | None, "Relationship start timestamp (ISO 8601, optional, defaults to now)"] = None,
    valid_until: Annotated[str | None, "Relationship end timestamp (ISO 8601, optional, null = ongoing)"] = None,
    confidence: Annotated[float | None, "Confidence score 0.0-1.0 (optional, defaults to 1.0)"] = None,
    metadata: Annotated[dict[str, Any] | None, "Additional structured data (optional)"] = None,
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> Relationship:
    """Create a new relationship between two entities.
    
    Args:
        source_entity_id: Source entity ID
        relationship_type: Relationship type (e.g., "leads", "blocks", "mentioned_in")
        target_entity_id: Target entity ID
        valid_from: Relationship start timestamp (ISO 8601, optional, defaults to now)
        valid_until: Relationship end timestamp (ISO 8601, optional, null = ongoing)
        confidence: Confidence score 0.0-1.0 (optional, defaults to 1.0)
        metadata: Additional structured data (optional)
        working_dir: beads workspace directory (optional)
    
    Returns:
        Relationship: Created relationship with ID
    """
    client = await _get_client(working_dir)
    params = CreateRelationshipParams(
        source_entity_id=source_entity_id,
        relationship_type=relationship_type,
        target_entity_id=target_entity_id,
        valid_from=valid_from,
        valid_until=valid_until,
        confidence=confidence,
        metadata=metadata
    )
    return await client.create_relationship(params)


async def beads_relationship_list(
    source_entity_id: Annotated[str | None, "Filter by source entity (optional)"] = None,
    target_entity_id: Annotated[str | None, "Filter by target entity (optional)"] = None,
    relationship_type: Annotated[str | None, "Filter by type (optional)"] = None,
    min_confidence: Annotated[float | None, "Minimum confidence threshold (optional, 0.0-1.0)"] = None,
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> list[Relationship]:
    """List relationships with optional filtering.
    
    Args:
        source_entity_id: Filter by source entity (optional)
        target_entity_id: Filter by target entity (optional)
        relationship_type: Filter by type (optional)
        min_confidence: Minimum confidence threshold (optional, 0.0-1.0)
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[Relationship]: Matching relationships
    """
    client = await _get_client(working_dir)
    return await client.list_relationships(
        source_id=source_entity_id,
        target_id=target_entity_id,
        relationship_type=relationship_type,
        min_confidence=min_confidence
    )


async def beads_relationship_show(
    relationship_id: Annotated[str, "Relationship ID"],
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> Relationship:
    """Show detailed information about a relationship.
    
    Args:
        relationship_id: Relationship ID
        working_dir: beads workspace directory (optional)
    
    Returns:
        Relationship: Full relationship details
    """
    client = await _get_client(working_dir)
    return await client.show_relationship(relationship_id)


async def beads_relationship_update(
    relationship_id: Annotated[str, "Relationship ID to update"],
    valid_from: Annotated[str | None, "New start timestamp (ISO 8601, optional)"] = None,
    valid_until: Annotated[str | None, "New end timestamp (ISO 8601, optional)"] = None,
    confidence: Annotated[float | None, "New confidence score 0.0-1.0 (optional)"] = None,
    metadata: Annotated[dict[str, Any] | None, "New metadata (optional)"] = None,
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> Relationship:
    """Update an existing relationship.
    
    Args:
        relationship_id: Relationship ID to update
        valid_from: New start timestamp (ISO 8601, optional)
        valid_until: New end timestamp (ISO 8601, optional)
        confidence: New confidence score 0.0-1.0 (optional)
        metadata: New metadata (optional)
        working_dir: beads workspace directory (optional)
    
    Returns:
        Relationship: Updated relationship
    """
    client = await _get_client(working_dir)
    kwargs = {}
    if valid_from is not None:
        kwargs['valid_from'] = valid_from
    if valid_until is not None:
        kwargs['valid_until'] = valid_until
    if confidence is not None:
        kwargs['confidence'] = confidence
    if metadata is not None:
        kwargs['metadata'] = metadata
    return await client.update_relationship(relationship_id, **kwargs)


async def beads_relationship_delete(
    relationship_id: Annotated[str, "Relationship ID to delete"],
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> dict[str, Any]:
    """Delete a relationship.
    
    Args:
        relationship_id: Relationship ID to delete
        working_dir: beads workspace directory (optional)
    
    Returns:
        dict: Deletion confirmation
    """
    client = await _get_client(working_dir)
    return await client.delete_relationship(relationship_id)


async def beads_episode_create(
    source: Annotated[str, "Episode source (e.g., 'github', 'jira', 'manual', 'conversation')"],
    file_path: Annotated[str, "Path to raw data file (text, JSON, etc.)"],
    timestamp: Annotated[str | None, "Episode timestamp (ISO 8601, optional, defaults to now)"] = None,
    entities_extracted: Annotated[list[str] | None, "Pre-linked entity IDs (optional)"] = None,
    extract: Annotated[bool, "Auto-extract entities after creation (optional, default False)"] = False,
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> Episode:
    """Create a new episode (provenance record for entity extraction).
    
    Args:
        source: Episode source (e.g., "github", "jira", "manual", "conversation")
        file_path: Path to raw data file (text, JSON, etc.)
        timestamp: Episode timestamp (ISO 8601, optional, defaults to now)
        entities_extracted: Pre-linked entity IDs (optional)
        extract: Auto-extract entities after creation (optional, default False)
        working_dir: beads workspace directory (optional)
    
    Returns:
        Episode: Created episode with ID
    """
    client = await _get_client(working_dir)
    params = CreateEpisodeParams(
        source=source,
        file_path=file_path,
        timestamp=timestamp,
        entities_extracted=entities_extracted,
        extract=extract
    )
    return await client.create_episode(params)


async def beads_episode_list(
    source: Annotated[str | None, "Filter by source (optional)"] = None,
    limit: Annotated[int, "Maximum number of results (default 50)"] = 50,
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> list[Episode]:
    """List episodes with optional filtering.
    
    Args:
        source: Filter by source (optional)
        limit: Maximum number of results (default 50)
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[Episode]: Matching episodes
    """
    client = await _get_client(working_dir)
    return await client.list_episodes(source=source, limit=limit)


async def beads_episode_show(
    episode_id: Annotated[str, "Episode ID"],
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> Episode:
    """Show detailed information about an episode.
    
    Args:
        episode_id: Episode ID
        working_dir: beads workspace directory (optional)
    
    Returns:
        Episode: Episode details (raw_data not included, only size)
    """
    client = await _get_client(working_dir)
    return await client.show_episode(episode_id)


async def beads_episode_extract(
    episode_id: Annotated[str, "Episode ID to process"],
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
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


async def beads_ontology_register_entity_type(
    name: Annotated[str, "Entity type name (e.g., 'meeting', 'document', 'person')"],
    schema: Annotated[dict[str, Any], "JSON schema for entity metadata validation (Pydantic-like)"],
    description: Annotated[str, "Human-readable description of the type"],
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
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


async def beads_ontology_register_relationship_type(
    name: Annotated[str, "Relationship type name (e.g., 'mentioned_in', 'leads', 'blocks')"],
    schema: Annotated[dict[str, Any], "JSON schema for relationship metadata validation"],
    description: Annotated[str, "Human-readable description of the type"],
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
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


async def beads_ontology_list_entity_types(
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> list[EntityType]:
    """List all registered entity types.
    
    Args:
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[EntityType]: All entity types
    """
    client = await _get_client(working_dir)
    return await client.list_entity_types()


async def beads_entity_search(
    query: Annotated[str, "Natural language search query"],
    entity_type: Annotated[str | None, "Filter by entity type (optional)"] = None,
    top: Annotated[int, "Maximum number of results (default 10)"] = 10,
    threshold: Annotated[float | None, "Similarity threshold 0.0-1.0 (optional)"] = None,
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> list[Entity]:
    """Search entities using natural language query (semantic search).
    
    Args:
        query: Natural language search query
        entity_type: Filter by entity type (optional)
        top: Maximum number of results (default 10)
        threshold: Similarity threshold 0.0-1.0 (optional)
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[Entity]: Most relevant entities sorted by similarity
    
    Example:
        beads_entity_search("Alice's role in auth service")
    """
    client = await _get_client(working_dir)
    return await client.search_entities(query, entity_type=entity_type, limit=top, threshold=threshold)


async def beads_entity_merge(
    source_id: Annotated[str, "Source entity ID (will be soft-deleted)"],
    target_id: Annotated[str, "Target entity ID (receives all relationships)"],
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> dict[str, Any]:
    """Merge source entity into target entity (deduplication).
    
    Args:
        source_id: Source entity ID (will be soft-deleted)
        target_id: Target entity ID (receives all relationships)
        working_dir: beads workspace directory (optional)
    
    Returns:
        dict: Merge confirmation
    """
    client = await _get_client(working_dir)
    return await client.merge_entities(source_id, target_id)


async def beads_entity_find_duplicates(
    entity_type: Annotated[str | None, "Filter by entity type (optional)"] = None,
    threshold: Annotated[float, "Similarity threshold 0.0-1.0 (default 0.8)"] = 0.8,
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
) -> list[dict[str, Any]]:
    """Find potential duplicate entities using similarity scoring.
    
    Args:
        entity_type: Filter by entity type (optional)
        threshold: Similarity threshold 0.0-1.0 (default 0.8)
        working_dir: beads workspace directory (optional)
    
    Returns:
        List[dict]: Duplicate pairs with similarity scores
    """
    client = await _get_client(working_dir)
    return await client.find_duplicate_entities(entity_type, threshold)


async def beads_memory_retrieve(
    query: Annotated[str, "Natural language query"],
    max_hops: Annotated[int, "Graph traversal depth (default 2)"] = 2,
    top_k: Annotated[int, "Max initial entities from semantic search (default 5)"] = 5,
    min_confidence: Annotated[float, "Minimum relationship confidence (default 0.5)"] = 0.5,
    working_dir: Annotated[str | None, "beads workspace directory (optional)"] = None,
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
