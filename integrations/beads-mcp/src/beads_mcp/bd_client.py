"""Client for interacting with bd (beads) CLI and daemon."""

import asyncio
import json
import os
import re
import sys
from abc import ABC, abstractmethod
from typing import Any, List, Optional

from .config import load_config
from .models import (
    AddDependencyParams,
    ClaimIssueParams,
    BlockedIssue,
    BlockedParams,
    CloseIssueParams,
    CreateIssueParams,
    InitParams,
    Issue,
    ListIssuesParams,
    ReadyWorkParams,
    ReopenIssueParams,
    ShowIssueParams,
    Stats,
    UpdateIssueParams,
    Entity,
    Relationship,
    Episode,
    EntityType,
    RelationshipType,
    CreateEntityParams,
    CreateRelationshipParams,
    CreateEpisodeParams,
    EntitySearchParams,
)


def _sanitize_issue_deps(issue: dict) -> dict:
    """Strip raw dependency records that don't match the LinkedIssue schema.

    bd list/ready/blocked --json returns raw dep records (issue_id, depends_on_id,
    type, created_at) but the Pydantic Issue model expects enriched LinkedIssue
    objects (id, title, status, etc.). Replace raw records with empty lists and
    preserve counts so validation succeeds.
    """
    for field, count_field in [
        ("dependencies", "dependency_count"),
        ("dependents", "dependent_count"),
    ]:
        raw = issue.get(field)
        if isinstance(raw, list) and raw:
            # Check if these are raw dep records (have depends_on_id) vs enriched
            if isinstance(raw[0], dict) and "depends_on_id" in raw[0]:
                issue[count_field] = len(raw)
                issue[field] = []
    return issue


class BdError(Exception):
    """Base exception for bd CLI errors."""

    pass


class BdNotFoundError(BdError):
    """Raised when bd command is not found."""

    @staticmethod
    def installation_message(attempted_path: str) -> str:
        """Get helpful installation message.

        Args:
            attempted_path: Path where we tried to find bd

        Returns:
            Formatted error message with installation instructions
        """
        return (
            f"bd CLI not found at: {attempted_path}\n\n"
            "The beads Claude Code plugin requires the bd CLI to be installed separately.\n\n"
            "Install bd CLI:\n"
            "  curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash\n\n"
            "Or visit: https://github.com/steveyegge/beads#installation\n\n"
            "After installation, restart Claude Code to reload the MCP server."
        )


class BdCommandError(BdError):
    """Raised when bd command fails."""

    stderr: str
    returncode: int

    def __init__(self, message: str, stderr: str = "", returncode: int = 1):
        super().__init__(message)
        self.stderr = stderr
        self.returncode = returncode


class BdVersionError(BdError):
    """Raised when bd version is incompatible with MCP server."""

    pass


class BdClientBase(ABC):
    """Abstract base class for bd clients (CLI or daemon)."""

    @abstractmethod
    async def ready(self, params: Optional[ReadyWorkParams] = None) -> List[Issue]:
        """Get ready work (issues with no blockers)."""
        pass

    @abstractmethod
    async def list_issues(self, params: Optional[ListIssuesParams] = None) -> List[Issue]:
        """List issues with optional filters."""
        pass

    @abstractmethod
    async def show(self, params: ShowIssueParams) -> Issue:
        """Show detailed issue information."""
        pass

    @abstractmethod
    async def create(self, params: CreateIssueParams) -> Issue:
        """Create a new issue."""
        pass

    @abstractmethod
    async def update(self, params: UpdateIssueParams) -> Issue:
        """Update an existing issue."""
        pass

    @abstractmethod
    async def claim(self, params: ClaimIssueParams) -> Issue:
        """Atomically claim an issue for work."""
        pass

    @abstractmethod
    async def close(self, params: CloseIssueParams) -> List[Issue]:
        """Close one or more issues."""
        pass

    @abstractmethod
    async def reopen(self, params: ReopenIssueParams) -> List[Issue]:
        """Reopen one or more closed issues."""
        pass

    @abstractmethod
    async def add_dependency(self, params: AddDependencyParams) -> None:
        """Add a dependency between issues."""
        pass

    @abstractmethod
    async def quickstart(self) -> str:
        """Get quickstart guide."""
        pass

    @abstractmethod
    async def stats(self) -> Stats:
        """Get repository statistics."""
        pass

    @abstractmethod
    async def blocked(self, params: Optional[BlockedParams] = None) -> List[BlockedIssue]:
        """Get blocked issues."""
        pass

    @abstractmethod
    async def init(self, params: Optional[InitParams] = None) -> str:
        """Initialize a new beads database."""
        pass

    @abstractmethod
    async def inspect_migration(self) -> dict[str, Any]:
        """Get migration plan and database state for agent analysis."""
        pass

    @abstractmethod
    async def get_schema_info(self) -> dict[str, Any]:
        """Get current database schema for inspection."""
        pass

    @abstractmethod
    async def repair_deps(self, fix: bool = False) -> dict[str, Any]:
        """Find and optionally fix orphaned dependency references.
        
        Args:
            fix: If True, automatically remove orphaned dependencies
            
        Returns:
            Dict with orphans_found, orphans list, and fixed count if fix=True
        """
        pass

    @abstractmethod
    async def detect_pollution(self, clean: bool = False) -> dict[str, Any]:
        """Detect test issues that leaked into production database.
        
        Args:
            clean: If True, delete detected test issues
            
        Returns:
            Dict with detected test issues and deleted count if clean=True
        """
        pass

    @abstractmethod
    async def validate(self, checks: str | None = None, fix_all: bool = False) -> dict[str, Any]:
        """Run database validation checks.
        
        Args:
            checks: Comma-separated list of checks (orphans,duplicates,pollution,conflicts)
            fix_all: If True, auto-fix all fixable issues
            
        Returns:
            Dict with validation results for each check
        """
        pass

    # Entity operations
    @abstractmethod
    async def create_entity(self, params: CreateEntityParams) -> Entity:
        """Create a new entity."""
        pass

    @abstractmethod
    async def list_entities(self, params: Optional[EntitySearchParams] = None) -> List[Entity]:
        """List entities with optional filters."""
        pass

    @abstractmethod
    async def show_entity(self, entity_id: str) -> Entity:
        """Show detailed entity information."""
        pass

    @abstractmethod
    async def update_entity(self, entity_id: str, **kwargs) -> Entity:
        """Update an existing entity."""
        pass

    @abstractmethod
    async def delete_entity(self, entity_id: str) -> dict[str, Any]:
        """Delete an entity."""
        pass

    @abstractmethod
    async def search_entities(self, query: str, entity_type: Optional[str] = None, limit: int = 10, threshold: Optional[float] = None) -> List[Entity]:
        """Search entities by query."""
        pass

    @abstractmethod
    async def merge_entities(self, source_id: str, target_id: str) -> dict[str, Any]:
        """Merge two entities."""
        pass

    @abstractmethod
    async def find_duplicate_entities(self, entity_type: Optional[str] = None, threshold: float = 0.8) -> List[dict[str, Any]]:
        """Find duplicate entities."""
        pass

    # Relationship operations
    @abstractmethod
    async def create_relationship(self, params: CreateRelationshipParams) -> Relationship:
        """Create a new relationship."""
        pass

    @abstractmethod
    async def list_relationships(self, source_id: Optional[str] = None, target_id: Optional[str] = None, relationship_type: Optional[str] = None, min_confidence: Optional[float] = None) -> List[Relationship]:
        """List relationships with optional filters."""
        pass

    @abstractmethod
    async def show_relationship(self, relationship_id: str) -> Relationship:
        """Show detailed relationship information."""
        pass

    @abstractmethod
    async def update_relationship(self, relationship_id: str, **kwargs) -> Relationship:
        """Update an existing relationship."""
        pass

    @abstractmethod
    async def delete_relationship(self, relationship_id: str) -> dict[str, Any]:
        """Delete a relationship."""
        pass

    # Episode operations
    @abstractmethod
    async def create_episode(self, params: CreateEpisodeParams) -> Episode:
        """Create a new episode."""
        pass

    @abstractmethod
    async def list_episodes(self, source: Optional[str] = None, limit: int = 50) -> List[Episode]:
        """List episodes with optional filters."""
        pass

    @abstractmethod
    async def show_episode(self, episode_id: str) -> Episode:
        """Show detailed episode information."""
        pass

    @abstractmethod
    async def extract_episode(self, episode_id: str) -> dict[str, Any]:
        """Extract entities and relationships from an episode."""
        pass

    # Ontology operations
    @abstractmethod
    async def register_entity_type(self, name: str, schema: dict[str, Any], description: str) -> EntityType:
        """Register a new entity type."""
        pass

    @abstractmethod
    async def register_relationship_type(self, name: str, schema: dict[str, Any], description: str) -> RelationshipType:
        """Register a new relationship type."""
        pass

    @abstractmethod
    async def list_entity_types(self) -> List[EntityType]:
        """List all registered entity types."""
        pass

    @abstractmethod
    async def list_relationship_types(self) -> List[RelationshipType]:
        """List all registered relationship types."""
        pass

    # Memory retrieval
    @abstractmethod
    async def retrieve_memory(self, query: str, max_hops: int = 2, top_k: int = 5, min_confidence: float = 0.5) -> dict[str, Any]:
        """Retrieve memory using graph traversal."""
        pass


class BdCliClient(BdClientBase):
    """Client for calling bd CLI commands and parsing JSON output."""

    bd_path: str
    beads_dir: str | None
    beads_db: str | None
    actor: str | None
    no_auto_flush: bool
    no_auto_import: bool
    working_dir: str | None

    def __init__(
        self,
        bd_path: str | None = None,
        beads_dir: str | None = None,
        beads_db: str | None = None,
        actor: str | None = None,
        no_auto_flush: bool | None = None,
        no_auto_import: bool | None = None,
        working_dir: str | None = None,
    ):
        """Initialize bd client.

        Args:
            bd_path: Path to bd executable (optional, loads from config if not provided)
            beads_dir: Path to .beads directory (optional, loads from config if not provided)
            beads_db: Path to beads database file (deprecated, optional, loads from config if not provided)
            actor: Actor name for audit trail (optional, loads from config if not provided)
            no_auto_flush: Disable automatic JSONL sync (optional, loads from config if not provided)
            no_auto_import: Disable automatic JSONL import (optional, loads from config if not provided)
            working_dir: Working directory for bd commands (optional, loads from config/env if not provided)
        """
        config = load_config()
        self.bd_path = bd_path if bd_path is not None else config.beads_path
        self.beads_dir = beads_dir if beads_dir is not None else config.beads_dir
        self.beads_db = beads_db if beads_db is not None else config.beads_db
        self.actor = actor if actor is not None else config.beads_actor
        self.no_auto_flush = no_auto_flush if no_auto_flush is not None else config.beads_no_auto_flush
        self.no_auto_import = no_auto_import if no_auto_import is not None else config.beads_no_auto_import
        self.working_dir = working_dir if working_dir is not None else config.beads_working_dir

    def _get_working_dir(self) -> str:
        """Get working directory for bd commands.

        Returns:
            Working directory path, falls back to current directory if not configured
        """
        if self.working_dir:
            return self.working_dir
        # Use process working directory (set by MCP client at spawn time)
        return os.getcwd()

    def _global_flags(self) -> list[str]:
        """Build list of global flags for bd commands.

        Returns:
            List of global flag arguments
        """
        flags = []
        # NOTE: --db flag removed in v0.20.1, bd now auto-discovers database via cwd
        # We pass cwd via _run_command instead
        if self.actor:
            flags.extend(["--actor", self.actor])
        if self.no_auto_flush:
            flags.append("--no-auto-flush")
        if self.no_auto_import:
            flags.append("--no-auto-import")
        return flags

    async def _run_command(self, *args: str, cwd: str | None = None) -> Any:
        """Run bd command and parse JSON output.

        Args:
            *args: Command arguments to pass to bd
            cwd: Optional working directory override for this command

        Returns:
            Parsed JSON output (dict or list)

        Raises:
            BdNotFoundError: If bd command not found
            BdCommandError: If bd command fails
        """
        cmd = [self.bd_path, *args, *self._global_flags(), "--json"]
        working_dir = cwd if cwd is not None else self._get_working_dir()

        # Set up environment with database configuration
        env = os.environ.copy()
        if self.beads_dir:
            env["BEADS_DIR"] = self.beads_dir
        elif self.beads_db:
            env["BEADS_DB"] = self.beads_db

        # Log database routing for debugging
        import sys
        if self.beads_dir:
            db_info = f"BEADS_DIR={self.beads_dir}"
        elif self.beads_db:
            db_info = f"BEADS_DB={self.beads_db} (deprecated)"
        else:
            db_info = "auto-discover"
        print(f"[beads-mcp] Running bd command: {' '.join(args)}", file=sys.stderr)
        print(f"[beads-mcp]   Database: {db_info}", file=sys.stderr)
        print(f"[beads-mcp]   Working dir: {working_dir}", file=sys.stderr)

        try:
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdin=asyncio.subprocess.DEVNULL,  # Prevent inheriting MCP's stdin
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=working_dir,
                env=env,
            )
            stdout, stderr = await process.communicate()
        except FileNotFoundError as e:
            raise BdNotFoundError(BdNotFoundError.installation_message(self.bd_path)) from e

        if process.returncode != 0:
            raise BdCommandError(
                f"bd command failed: {stderr.decode()}",
                stderr=stderr.decode(),
                returncode=process.returncode or 1,
            )

        stdout_str = stdout.decode().strip()
        if not stdout_str:
            return {}

        try:
            result: object = json.loads(stdout_str)
            return result
        except json.JSONDecodeError as e:
            raise BdCommandError(
                f"Failed to parse bd JSON output: {e}",
                stderr=stdout_str,
            ) from e

    async def _check_version(self) -> None:
        """Check that bd CLI version meets minimum requirements.

        Raises:
            BdVersionError: If bd version is incompatible
            BdNotFoundError: If bd command not found
        """
        # Minimum required version
        min_version = (0, 9, 0)

        try:
            process = await asyncio.create_subprocess_exec(
                self.bd_path,
                "version",
                stdin=asyncio.subprocess.DEVNULL,  # Prevent inheriting MCP's stdin
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=self._get_working_dir(),
            )
            stdout, stderr = await process.communicate()
        except FileNotFoundError as e:
            raise BdNotFoundError(BdNotFoundError.installation_message(self.bd_path)) from e

        if process.returncode != 0:
            raise BdCommandError(
                f"bd version failed: {stderr.decode()}",
                stderr=stderr.decode(),
                returncode=process.returncode or 1,
            )

        # Parse version from output like "bd version 0.9.2"
        version_output = stdout.decode().strip()
        match = re.search(r"(\d+)\.(\d+)\.(\d+)", version_output)
        if not match:
            raise BdVersionError(f"Could not parse bd version from: {version_output}")

        version = tuple(int(x) for x in match.groups())

        if version < min_version:
            min_ver_str = ".".join(str(x) for x in min_version)
            cur_ver_str = ".".join(str(x) for x in version)
            install_cmd = "curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash"
            raise BdVersionError(
                f"bd version {cur_ver_str} is too old. "
                f"This MCP server requires bd >= {min_ver_str}. "
                f"Update with: {install_cmd}"
            )

    async def ready(self, params: ReadyWorkParams | None = None) -> list[Issue]:
        """Get ready work (issues with no blocking dependencies).

        Args:
            params: Query parameters

        Returns:
            List of ready issues
        """
        params = params or ReadyWorkParams()
        args = ["ready", "--limit", str(params.limit)]

        if params.priority is not None:
            args.extend(["--priority", str(params.priority)])
        if params.assignee:
            args.extend(["--assignee", params.assignee])
        if params.labels:
            for label in params.labels:
                args.extend(["--label", label])
        if params.labels_any:
            for label in params.labels_any:
                args.extend(["--label-any", label])
        if params.unassigned:
            args.append("--unassigned")
        if params.sort_policy:
            args.extend(["--sort", params.sort_policy])
        if params.parent_id:
            args.extend(["--parent", params.parent_id])

        data = await self._run_command(*args)
        if not isinstance(data, list):
            return []

        return [Issue.model_validate(_sanitize_issue_deps(issue)) for issue in data]

    async def list_issues(self, params: ListIssuesParams | None = None) -> list[Issue]:
        """List issues with optional filters.

        Args:
            params: Query parameters

        Returns:
            List of issues
        """
        params = params or ListIssuesParams()
        args = ["list"]

        if params.status:
            args.extend(["--status", params.status])
        if params.priority is not None:
            args.extend(["--priority", str(params.priority)])
        if params.issue_type:
            args.extend(["--type", params.issue_type])
        if params.assignee:
            args.extend(["--assignee", params.assignee])
        if params.labels:
            for label in params.labels:
                args.extend(["--label", label])
        if params.labels_any:
            for label in params.labels_any:
                args.extend(["--label-any", label])
        if params.query:
            args.extend(["--title", params.query])
        if params.unassigned:
            args.append("--no-assignee")
        if params.limit:
            args.extend(["--limit", str(params.limit)])

        data = await self._run_command(*args)
        if not isinstance(data, list):
            return []

        return [Issue.model_validate(_sanitize_issue_deps(issue)) for issue in data]

    async def show(self, params: ShowIssueParams) -> Issue:
        """Show issue details.

        Args:
            params: Issue ID to show

        Returns:
            Issue details

        Raises:
            BdCommandError: If issue not found
        """
        data = await self._run_command("show", params.issue_id)
        # bd show returns an array, extract first element
        if isinstance(data, list):
            if not data:
                raise BdCommandError(f"Issue not found: {params.issue_id}")
            data = data[0]
        
        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for show {params.issue_id}")

        return Issue.model_validate(data)

    async def create(self, params: CreateIssueParams) -> Issue:
        """Create a new issue.

        Args:
            params: Issue creation parameters

        Returns:
            Created issue
        """
        args = ["create", params.title, "-p", str(params.priority), "-t", params.issue_type]

        if params.description:
            args.extend(["-d", params.description])
        if params.design:
            args.extend(["--design", params.design])
        if params.acceptance:
            args.extend(["--acceptance", params.acceptance])
        if params.external_ref:
            args.extend(["--external-ref", params.external_ref])
        if params.assignee:
            args.extend(["--assignee", params.assignee])
        if params.id:
            args.extend(["--id", params.id])
        for label in params.labels:
            args.extend(["-l", label])
        if params.deps:
            args.extend(["--deps", ",".join(params.deps)])

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for create")

        return Issue.model_validate(data)

    async def update(self, params: UpdateIssueParams) -> Issue:
        """Update an issue.

        Args:
            params: Issue update parameters

        Returns:
            Updated issue
        """
        args = ["update", params.issue_id]

        if params.status:
            args.extend(["--status", params.status])
        if params.priority is not None:
            args.extend(["--priority", str(params.priority)])
        if params.assignee:
            args.extend(["--assignee", params.assignee])
        if params.title:
            args.extend(["--title", params.title])
        if params.description:
            args.extend(["--description", params.description])
        if params.design:
            args.extend(["--design", params.design])
        if params.acceptance_criteria:
            args.extend(["--acceptance", params.acceptance_criteria])
        if params.notes:
            args.extend(["--notes", params.notes])
        if params.external_ref:
            args.extend(["--external-ref", params.external_ref])

        data = await self._run_command(*args)
        # bd update returns an array, extract first element
        if isinstance(data, list):
            if not data:
                raise BdCommandError(f"Issue not found: {params.issue_id}")
            data = data[0]
        
        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for update {params.issue_id}")

        return Issue.model_validate(data)

    async def claim(self, params: ClaimIssueParams) -> Issue:
        """Atomically claim an issue via bd update --claim.

        Args:
            params: Claim parameters

        Returns:
            Claimed issue
        """
        data = await self._run_command("update", params.issue_id, "--claim")
        # bd update returns an array, extract first element
        if isinstance(data, list):
            if not data:
                raise BdCommandError(f"Issue not found: {params.issue_id}")
            data = data[0]

        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for claim {params.issue_id}")

        return Issue.model_validate(data)

    async def close(self, params: CloseIssueParams) -> list[Issue]:
        """Close an issue.

        Args:
            params: Close parameters

        Returns:
            List containing closed issue
        """
        args = ["close", params.issue_id, "--reason", params.reason]

        data = await self._run_command(*args)
        if not isinstance(data, list):
            raise BdCommandError(f"Invalid response for close {params.issue_id}")

        return [Issue.model_validate(issue) for issue in data]

    async def reopen(self, params: ReopenIssueParams) -> list[Issue]:
        """Reopen one or more closed issues.

        Args:
            params: Reopen parameters

        Returns:
            List of reopened issues
        """
        args = ["reopen", *params.issue_ids]

        if params.reason:
            args.extend(["--reason", params.reason])

        data = await self._run_command(*args)
        if not isinstance(data, list):
            raise BdCommandError(f"Invalid response for reopen {params.issue_ids}")

        return [Issue.model_validate(issue) for issue in data]

    async def add_dependency(self, params: AddDependencyParams) -> None:
        """Add a dependency between issues.

        Args:
            params: Dependency parameters
        """
        # bd dep add doesn't return JSON, just prints confirmation
        cmd = [
            self.bd_path,
            "dep",
            "add",
            params.issue_id,
            params.depends_on_id,
            "--type",
            params.dep_type,
            *self._global_flags(),
        ]

        # Set up environment with database configuration
        env = os.environ.copy()
        if self.beads_dir:
            env["BEADS_DIR"] = self.beads_dir
        elif self.beads_db:
            env["BEADS_DB"] = self.beads_db

        try:
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdin=asyncio.subprocess.DEVNULL,  # Prevent inheriting MCP's stdin
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=self._get_working_dir(),
                env=env,
            )
            _stdout, stderr = await process.communicate()
        except FileNotFoundError as e:
            raise BdNotFoundError(BdNotFoundError.installation_message(self.bd_path)) from e

        if process.returncode != 0:
            raise BdCommandError(
                f"bd dep add failed: {stderr.decode()}",
                stderr=stderr.decode(),
                returncode=process.returncode or 1,
            )

    async def quickstart(self) -> str:
        """Get bd quickstart guide.

        Returns:
            Quickstart guide text
        """
        cmd = [self.bd_path, "quickstart"]

        try:
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdin=asyncio.subprocess.DEVNULL,  # Prevent inheriting MCP's stdin
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=self._get_working_dir(),
            )
            stdout, stderr = await process.communicate()
        except FileNotFoundError as e:
            raise BdNotFoundError(BdNotFoundError.installation_message(self.bd_path)) from e

        if process.returncode != 0:
            raise BdCommandError(
                f"bd quickstart failed: {stderr.decode()}",
                stderr=stderr.decode(),
                returncode=process.returncode or 1,
            )

        return stdout.decode()

    async def stats(self) -> Stats:
        """Get statistics about issues.

        Returns:
            Statistics object
        """
        data = await self._run_command("stats")
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for stats")

        return Stats.model_validate(data)

    async def blocked(self, params: BlockedParams | None = None) -> list[BlockedIssue]:
        """Get blocked issues.

        Args:
            params: Query parameters

        Returns:
            List of blocked issues with blocking information
        """
        params = params or BlockedParams()
        args = ["blocked"]
        if params.parent_id:
            args.extend(["--parent", params.parent_id])

        data = await self._run_command(*args)
        if not isinstance(data, list):
            return []

        return [BlockedIssue.model_validate(_sanitize_issue_deps(issue)) for issue in data]

    async def inspect_migration(self) -> dict[str, Any]:
        """Get migration plan and database state for agent analysis.

        Returns:
            Migration plan dict with registered_migrations, warnings, etc.
        """
        data = await self._run_command("migrate", "--inspect")
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for inspect_migration")
        return data

    async def get_schema_info(self) -> dict[str, Any]:
        """Get current database schema for inspection.

        Returns:
            Schema info dict with tables, version, config, sample IDs, etc.
        """
        data = await self._run_command("info", "--schema")
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for get_schema_info")
        return data

    async def repair_deps(self, fix: bool = False) -> dict[str, Any]:
        """Find and optionally fix orphaned dependency references.

        Args:
            fix: If True, automatically remove orphaned dependencies

        Returns:
            Dict with orphans_found, orphans list, and fixed count if fix=True
        """
        args = ["repair-deps"]
        if fix:
            args.append("--fix")

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for repair-deps")
        return data

    async def detect_pollution(self, clean: bool = False) -> dict[str, Any]:
        """Detect test issues that leaked into production database.

        Args:
            clean: If True, delete detected test issues

        Returns:
            Dict with detected test issues and deleted count if clean=True
        """
        args = ["detect-pollution"]
        if clean:
            args.extend(["--clean", "--yes"])

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for detect-pollution")
        return data

    async def validate(self, checks: str | None = None, fix_all: bool = False) -> dict[str, Any]:
        """Run database validation checks.

        Args:
            checks: Comma-separated list of checks (orphans,duplicates,pollution,conflicts)
            fix_all: If True, auto-fix all fixable issues

        Returns:
            Dict with validation results for each check
        """
        args = ["validate"]
        if checks:
            args.extend(["--checks", checks])
        if fix_all:
            args.append("--fix-all")

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for validate")
        return data

    async def init(self, params: InitParams | None = None) -> str:
        """Initialize bd in current directory.

        Args:
            params: Initialization parameters

        Returns:
            Initialization output message
        """
        params = params or InitParams()
        cmd = [self.bd_path, "init"]

        if params.prefix:
            cmd.extend(["--prefix", params.prefix])

        # NOTE: Do NOT add --db flag for init!
        # init creates a NEW database in the current directory.
        # Only add actor-related flags.
        if self.actor:
            cmd.extend(["--actor", self.actor])

        try:
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdin=asyncio.subprocess.DEVNULL,  # Prevent inheriting MCP's stdin
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=self._get_working_dir(),
            )
            stdout, stderr = await process.communicate()
        except FileNotFoundError as e:
            raise BdNotFoundError(BdNotFoundError.installation_message(self.bd_path)) from e

        if process.returncode != 0:
            raise BdCommandError(
                f"bd init failed: {stderr.decode()}",
                stderr=stderr.decode(),
                returncode=process.returncode or 1,
            )

        return stdout.decode()

    # =========================================================================
    # ENTITY OPERATIONS
    # =========================================================================

    async def create_entity(self, params: CreateEntityParams) -> Entity:
        """Create a new entity.

        Args:
            params: Entity creation parameters

        Returns:
            Created entity
        """
        args = ["entity", "create", "--entity-type", params.entity_type, "--name", params.name, "--summary", params.summary]

        if params.metadata:
            args.extend(["--metadata", json.dumps(params.metadata)])
        if params.created_by:
            args.extend(["--created-by", params.created_by])
        if params.id:
            args.extend(["--id", params.id])

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for create_entity")

        return Entity.model_validate(data)

    async def list_entities(self, params: Optional[EntitySearchParams] = None) -> List[Entity]:
        """List entities with optional filters.

        Args:
            params: Search parameters

        Returns:
            List of entities
        """
        params = params or EntitySearchParams()
        args = ["entity", "list"]

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

        data = await self._run_command(*args)
        if not isinstance(data, list):
            return []

        return [Entity.model_validate(e) for e in data]

    async def show_entity(self, entity_id: str) -> Entity:
        """Show detailed entity information.

        Args:
            entity_id: Entity ID to show

        Returns:
            Entity details

        Raises:
            BdCommandError: If entity not found
        """
        data = await self._run_command("entity", "show", entity_id)
        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for show_entity {entity_id}")

        return Entity.model_validate(data)

    async def update_entity(self, entity_id: str, **kwargs) -> Entity:
        """Update an existing entity.

        Args:
            entity_id: Entity ID to update
            **kwargs: Fields to update (name, summary, metadata)

        Returns:
            Updated entity
        """
        args = ["entity", "update", entity_id]

        if "name" in kwargs:
            args.extend(["--name", kwargs["name"]])
        if "summary" in kwargs:
            args.extend(["--summary", kwargs["summary"]])
        if "metadata" in kwargs:
            args.extend(["--metadata", json.dumps(kwargs["metadata"])])

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for update_entity {entity_id}")

        return Entity.model_validate(data)

    async def delete_entity(self, entity_id: str) -> dict[str, Any]:
        """Delete an entity.

        Args:
            entity_id: Entity ID to delete

        Returns:
            Deletion confirmation dict
        """
        data = await self._run_command("entity", "delete", entity_id)
        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for delete_entity {entity_id}")

        return data

    async def search_entities(self, query: str, entity_type: Optional[str] = None, limit: int = 10, threshold: Optional[float] = None) -> List[Entity]:
        """Search entities by query.

        Args:
            query: Search query string
            entity_type: Filter by entity type (optional)
            limit: Maximum number of results
            threshold: Similarity threshold 0.0-1.0 (optional)

        Returns:
            List of matching entities
        """
        args = ["entity", "search", "--query", query, "--top", str(limit)]
        
        if entity_type:
            args.extend(["--entity-type", entity_type])
        if threshold is not None:
            args.extend(["--threshold", str(threshold)])

        data = await self._run_command(*args)
        if not isinstance(data, list):
            return []

        return [Entity.model_validate(e) for e in data]

    async def merge_entities(self, source_id: str, target_id: str) -> dict[str, Any]:
        """Merge two entities.

        Args:
            source_id: Source entity ID (will be merged into target)
            target_id: Target entity ID (will receive merge)

        Returns:
            Merge result dict
        """
        data = await self._run_command("entity", "merge", source_id, target_id)
        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for merge_entities {source_id} -> {target_id}")

        return data

    async def find_duplicate_entities(self, entity_type: Optional[str] = None, threshold: float = 0.8) -> List[dict[str, Any]]:
        """Find duplicate entities.

        Args:
            entity_type: Optional entity type filter
            threshold: Similarity threshold (0.0-1.0)

        Returns:
            List of duplicate pairs with similarity scores
        """
        args = ["entity", "find-duplicates", "--threshold", str(threshold)]

        if entity_type:
            args.extend(["--entity-type", entity_type])

        data = await self._run_command(*args)
        if not isinstance(data, list):
            return []

        return data

    # =========================================================================
    # RELATIONSHIP OPERATIONS
    # =========================================================================

    async def create_relationship(self, params: CreateRelationshipParams) -> Relationship:
        """Create a new relationship.

        Args:
            params: Relationship creation parameters

        Returns:
            Created relationship
        """
        args = [
            "relationship", "create",
            "--from", params.source_entity_id,
            "--type", params.relationship_type,
            "--to", params.target_entity_id,
        ]

        if params.valid_from:
            args.extend(["--valid-from", params.valid_from])
        if params.valid_until:
            args.extend(["--valid-until", params.valid_until])
        if params.confidence is not None:
            args.extend(["--confidence", str(params.confidence)])
        if params.metadata:
            args.extend(["--metadata", json.dumps(params.metadata)])

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for create_relationship")

        return Relationship.model_validate(data)

    async def list_relationships(
        self,
        source_id: Optional[str] = None,
        target_id: Optional[str] = None,
        relationship_type: Optional[str] = None,
        min_confidence: Optional[float] = None,
    ) -> List[Relationship]:
        """List relationships with optional filters.

        Args:
            source_id: Filter by source entity ID
            target_id: Filter by target entity ID
            relationship_type: Filter by relationship type
            min_confidence: Minimum confidence threshold

        Returns:
            List of relationships
        """
        args = ["relationship", "list"]

        if source_id:
            args.extend(["--from", source_id])
        if target_id:
            args.extend(["--to", target_id])
        if relationship_type:
            args.extend(["--type", relationship_type])
        if min_confidence is not None:
            args.extend(["--min-confidence", str(min_confidence)])

        data = await self._run_command(*args)
        if not isinstance(data, list):
            return []

        return [Relationship.model_validate(r) for r in data]

    async def show_relationship(self, relationship_id: str) -> Relationship:
        """Show detailed relationship information.

        Args:
            relationship_id: Relationship ID to show

        Returns:
            Relationship details

        Raises:
            BdCommandError: If relationship not found
        """
        data = await self._run_command("relationship", "show", relationship_id)
        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for show_relationship {relationship_id}")

        return Relationship.model_validate(data)

    async def update_relationship(self, relationship_id: str, **kwargs) -> Relationship:
        """Update an existing relationship.

        Args:
            relationship_id: Relationship ID to update
            **kwargs: Fields to update (valid_until, confidence, metadata)

        Returns:
            Updated relationship
        """
        args = ["relationship", "update", relationship_id]

        if "valid_until" in kwargs:
            args.extend(["--valid-until", kwargs["valid_until"]])
        if "confidence" in kwargs:
            args.extend(["--confidence", str(kwargs["confidence"])])
        if "metadata" in kwargs:
            args.extend(["--metadata", json.dumps(kwargs["metadata"])])

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for update_relationship {relationship_id}")

        return Relationship.model_validate(data)

    async def delete_relationship(self, relationship_id: str) -> dict[str, Any]:
        """Delete a relationship.

        Args:
            relationship_id: Relationship ID to delete

        Returns:
            Deletion confirmation dict
        """
        data = await self._run_command("relationship", "delete", relationship_id)
        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for delete_relationship {relationship_id}")

        return data

    # =========================================================================
    # EPISODE OPERATIONS
    # =========================================================================

    async def create_episode(self, params: CreateEpisodeParams) -> Episode:
        """Create a new episode.

        Args:
            params: Episode creation parameters

        Returns:
            Created episode
        """
        args = ["episode", "create", "--source", params.source, "--file", params.file_path]

        if params.timestamp:
            args.extend(["--timestamp", params.timestamp])
        if params.entities_extracted:
            args.extend(["--entities", ",".join(params.entities_extracted)])
        if params.extract:
            args.append("--extract")

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for create_episode")

        return Episode.model_validate(data)

    async def list_episodes(self, source: Optional[str] = None, limit: int = 50) -> List[Episode]:
        """List episodes with optional filters.

        Args:
            source: Filter by source type
            limit: Maximum number of results

        Returns:
            List of episodes
        """
        args = ["episode", "list", "--limit", str(limit)]

        if source:
            args.extend(["--source", source])

        data = await self._run_command(*args)
        if not isinstance(data, list):
            return []

        return [Episode.model_validate(e) for e in data]

    async def show_episode(self, episode_id: str) -> Episode:
        """Show detailed episode information.

        Args:
            episode_id: Episode ID to show

        Returns:
            Episode details

        Raises:
            BdCommandError: If episode not found
        """
        data = await self._run_command("episode", "show", episode_id)
        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for show_episode {episode_id}")

        return Episode.model_validate(data)

    async def extract_episode(self, episode_id: str) -> dict[str, Any]:
        """Extract entities and relationships from an episode.

        Args:
            episode_id: Episode ID to extract from

        Returns:
            Extraction result dict with stats
        """
        data = await self._run_command("episode", "extract", episode_id)
        if not isinstance(data, dict):
            raise BdCommandError(f"Invalid response for extract_episode {episode_id}")

        return data

    # =========================================================================
    # ONTOLOGY OPERATIONS
    # =========================================================================

    async def register_entity_type(self, name: str, schema: dict[str, Any], description: str) -> EntityType:
        """Register a new entity type.

        Args:
            name: Entity type name
            schema: JSON schema for validation
            description: Entity type description

        Returns:
            Registered entity type
        """
        args = [
            "ontology", "register-entity-type",
            "--name", name,
            "--schema", json.dumps(schema),
            "--description", description,
        ]

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for register_entity_type")

        return EntityType.model_validate(data)

    async def register_relationship_type(self, name: str, schema: dict[str, Any], description: str) -> RelationshipType:
        """Register a new relationship type.

        Args:
            name: Relationship type name
            schema: JSON schema for validation
            description: Relationship type description

        Returns:
            Registered relationship type
        """
        args = [
            "ontology", "register-relationship-type",
            "--name", name,
            "--schema", json.dumps(schema),
            "--description", description,
        ]

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for register_relationship_type")

        return RelationshipType.model_validate(data)

    async def list_entity_types(self) -> List[EntityType]:
        """List all registered entity types.

        Returns:
            List of entity types
        """
        data = await self._run_command("ontology", "list", "--entity-types")
        if not isinstance(data, list):
            return []

        return [EntityType.model_validate(et) for et in data]

    async def list_relationship_types(self) -> List[RelationshipType]:
        """List all registered relationship types.

        Returns:
            List of relationship types
        """
        data = await self._run_command("ontology", "list", "--relationship-types")
        if not isinstance(data, list):
            return []

        return [RelationshipType.model_validate(rt) for rt in data]

    # =========================================================================
    # MEMORY RETRIEVAL
    # =========================================================================

    async def retrieve_memory(
        self,
        query: str,
        max_hops: int = 2,
        top_k: int = 5,
        min_confidence: float = 0.5,
    ) -> dict[str, Any]:
        """Retrieve memory using graph traversal.

        Args:
            query: Search query
            max_hops: Maximum graph traversal depth
            top_k: Maximum number of results
            min_confidence: Minimum confidence threshold

        Returns:
            Memory context dict with entities, relationships, episodes, and relevance scores
        """
        args = [
            "memory", "retrieve",
            "--query", query,
            "--hops", str(max_hops),
            "--top", str(top_k),
            "--min-confidence", str(min_confidence),
        ]

        data = await self._run_command(*args)
        if not isinstance(data, dict):
            raise BdCommandError("Invalid response for retrieve_memory")

        return data


# Backwards compatibility alias
BdClient = BdCliClient


def create_bd_client(
    prefer_daemon: bool = False,
    bd_path: Optional[str] = None,
    beads_dir: Optional[str] = None,
    beads_db: Optional[str] = None,
    actor: Optional[str] = None,
    no_auto_flush: Optional[bool] = None,
    no_auto_import: Optional[bool] = None,
    working_dir: Optional[str] = None,
) -> BdClientBase:
    """Create a bd CLI client.

    Args:
        prefer_daemon: Deprecated, ignored. Kept for API compatibility.
        bd_path: Path to bd executable
        beads_dir: Path to .beads directory
        beads_db: Path to beads database (deprecated)
        actor: Actor name for audit trail
        no_auto_flush: Disable auto-flush
        no_auto_import: Disable auto-import
        working_dir: Working directory for database discovery

    Returns:
        BdClientBase implementation (CLI)
    """
    return BdCliClient(
        bd_path=bd_path,
        beads_dir=beads_dir,
        beads_db=beads_db,
        actor=actor,
        no_auto_flush=no_auto_flush,
        no_auto_import=no_auto_import,
        working_dir=working_dir,
    )
