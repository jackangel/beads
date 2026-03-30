# System Architecture

*Last Updated: 2026-03-16*
*Confidence Score: 94%*

---

## High-Level Landscape

- **Architecture Type**: Layered CLI Application with Versioned Database Backend (Dolt-powered distributed issue tracker)
- **Primary Tech Stack**: Go 1.25.8, Dolt (MySQL-compatible versioned SQL), Cobra CLI, OpenTelemetry, Python (MCP server)
- **Entry Points**:
  - **CLI**: `bd` binary via `cmd/bd/main.go` (100+ Cobra commands)
  - **Go Library**: `beads.go` (public API: `Open()`, `OpenFromConfig()`, `FindDatabasePath()`)
  - **MCP Server**: `integrations/beads-mcp/` (Python FastMCP for AI assistants)
- **Binary Name**: `bd` (alias: `beads` on Unix)
- **Current Version**: 0.61.0
- **CGO Required**: Yes (Dolt embedded database)

---

## System Topology

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     CLI Layer (cmd/bd/)                          │
│  100+ Cobra commands: create, list, update, close, ready, ...   │
│  All commands support --json for programmatic use               │
│  Groups: issues, views, deps, sync, setup, maint, advanced     │
└──────────────────────────────┬──────────────────────────────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
              v                v                v
┌──────────────────┐ ┌─────────────────┐ ┌──────────────────┐
│  Public Go API   │ │  MCP Server     │ │  Hook System     │
│  (beads.go)      │ │  (Python)       │ │  (.beads/hooks/) │
│  Open, Find, etc │ │  FastMCP 3.1.1  │ │  on_create/      │
└────────┬─────────┘ └───────┬─────────┘ │  on_update/      │
         │                   │           │  on_close         │
         │                   │           └────────┬──────────┘
         │                   │ (subprocess: bd)   │
         └──────────┬────────┴────────────────────┘
                    │
                    v
┌─────────────────────────────────────────────────────────────────┐
│                    Storage Interface                              │
│             internal/storage/storage.go                           │
│                                                                  │
│  Storage ─── Transaction ─── DoltStorage (composed interfaces)   │
│  CRUD, Dependencies, Labels, Comments, Events, Config, Queries  │
└──────────────────────────────┬──────────────────────────────────┘
                               │
              ┌────────────────┼────────────────┐
              │                                 │
              v                                 v
┌──────────────────────────┐   ┌──────────────────────────┐
│   Embedded Mode          │   │   Server Mode            │
│   (in-process Dolt)      │   │   (dolt sql-server)      │
│   Single-writer          │   │   Multi-writer via TCP   │
│   Default for CLI        │   │   Ephemeral port alloc   │
│                          │   │   Shared server option   │
└──────────┬───────────────┘   └──────────┬───────────────┘
           │                              │
           └──────────────┬───────────────┘
                          │
                          v
┌─────────────────────────────────────────────────────────────────┐
│                    Dolt Database (.beads/dolt/)                   │
│                                                                  │
│  14+ SQL Tables: issues, dependencies, labels, comments,         │
│  events, config, metadata, child_counters, issue_snapshots,     │
│  compaction_snapshots, repo_mtimes, routes, issue_counter,      │
│  interactions, federation_peers                                  │
│                                                                  │
│  Schema v7 with 11 idempotent migrations                        │
│  MySQL-compatible, cell-level merge resolution                  │
│  Auto-commit on every write                                      │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                        Dolt push/pull
                    (or federation peer sync)
                               │
                               v
┌─────────────────────────────────────────────────────────────────┐
│              Remote (DoltHub, S3, GCS, filesystem)                │
│              Cell-level merge, protected branch support           │
└─────────────────────────────────────────────────────────────────┘
```

---

## Knowledge Graph Architecture (v8)

### Overview

Beads v8 introduces a **knowledge graph architecture** that replaces the rigid Epic/Task/Sub-task hierarchy with a flexible entity-relationship model. This enables sophisticated memory systems, temporal reasoning, and custom ontologies.

**Key Characteristics:**
- **Entity-Based Model**: Everything is an entity (work items, meetings, decisions, documents)
- **Temporal Relationships**: Relationships have validity windows (valid_from, valid_until)
- **Custom Ontologies**: User-defined entity and relationship types via Pydantic-like schemas
- **Episode Provenance**: All entities link to source episodes (conversations, events, observations)
- **Backward Compatible**: v7 (issues) and v8 (entities) coexist during migration

### Entity-Relationship Model

```
┌──────────────────────────────────────────────────────────────────┐
│                        ENTITIES                                   │
│  (id, entity_type, name, summary, metadata, created_at, ...)    │
└───────────────────────────┬──────────────────────────────────────┘
                            │
                 ┌──────────┴──────────┐
                 │                     │
                 v                     v
     ┌─────────────────────┐ ┌─────────────────────┐
     │  ENTITY_TYPES       │ │  RELATIONSHIPS      │
     │  (name, schema,     │ │  (source_id,        │
     │   description,      │ │   relationship_type,│
     │   created_at)       │ │   target_id,        │
     └─────────────────────┘ │   valid_from,       │
                              │   valid_until,      │
                              │   metadata)         │
                              └──────────┬──────────┘
                                         │
                                         v
                              ┌─────────────────────┐
                              │ RELATIONSHIP_TYPES  │
                              │ (name, schema,      │
                              │  description,       │
                              │  created_at)        │
                              └─────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                         EPISODES                                  │
│  (id, timestamp, source, raw_data, entities_extracted,           │
│   created_at, metadata)                                          │
│                                                                  │
│  Links entities to their origin (conversation, meeting, event)  │
└──────────────────────────────────────────────────────────────────┘
```

### Temporal Validity Windows

Relationships can express time-bounded facts:

```sql
-- "Alice leads Team X from Jan 2026 to Jun 2026"
INSERT INTO relationships (
  source_entity_id,   -- Alice (person entity)
  relationship_type,  -- leads
  target_entity_id,   -- Team X (team entity)
  valid_from,         -- 2026-01-01
  valid_until         -- 2026-06-30
);

-- "Feature Y blocks Feature Z (still active)"
INSERT INTO relationships (
  source_entity_id,   -- Feature Y
  relationship_type,  -- blocks
  target_entity_id,   -- Feature Z
  valid_from,         -- 2026-03-15
  valid_until         -- NULL (ongoing)
);
```

**Query Pattern:**
```sql
-- Find all active relationships for an entity as of today
SELECT * FROM relationships
WHERE source_entity_id = 'entity-abc123'
  AND (valid_from IS NULL OR valid_from <= NOW())
  AND (valid_until IS NULL OR valid_until > NOW());
```

### Custom Ontology System

Users define entity and relationship types via JSON schemas (Pydantic-like):

```json
// Entity Type: "meeting"
{
  "name": "meeting",
  "description": "A scheduled meeting or discussion",
  "schema": {
    "properties": {
      "attendees": {"type": "array", "items": {"type": "string"}},
      "duration_minutes": {"type": "integer"},
      "location": {"type": "string"}
    },
    "required": ["attendees"]
  }
}

// Relationship Type: "mentioned_in"
{
  "name": "mentioned_in",
  "description": "Entity was referenced in another entity",
  "schema": {
    "properties": {
      "context": {"type": "string"},
      "relevance_score": {"type": "number", "min": 0, "max": 1}
    }
  }
}
```

**CLI Usage:**
```bash
bd ontology define entity meeting --schema meeting-schema.json
bd ontology define relationship mentioned_in --schema mentioned-in-schema.json
bd entity create "Q1 Planning" --type meeting --metadata '{"attendees": ["alice", "bob"]}'
bd relationship create entity-123 mentioned_in entity-456
```

### Data Flow (v8)

**Write Path:**
```
User/Agent → bd entity create → Storage.CreateEntity() → Dolt SQL INSERT (entities) → Auto Commit
                                                     ↓
                                          Episode linkage (optional)
```

**Relationship Path:**
```
User/Agent → bd relationship create → Storage.CreateRelationship() → Dolt SQL INSERT (relationships)
                                                                   ↓
                                                    Temporal validity checks
```

**Query Path (with temporal filtering):**
```
User/Agent → bd graph query → Query Engine → Storage.SearchEntities() → Dolt SQL SELECT
                                                                      ↓
                                             In-memory temporal predicate (valid_from/valid_until)
                                                                      ↓
                                                            JSON/Styled Output
```

### Migration Strategy (v7 → v8)

**Dual-Mode Operation:**
- Both schemas coexist in Dolt database
- Legacy commands (`bd create`, `bd children`) map to v7 tables
- New commands (`bd entity create`, `bd relationship create`) use v8 tables
- Deprecation warnings guide users to migrate

**Data Migration:**
```bash
bd migrate v8 --dry-run           # Preview migration
bd migrate v8 --execute           # Migrate issues → entities
bd migrate v8 --rollback          # Revert to v7 (pre-migration snapshot)
```

**Mapping:**
- `issues` → `entities` (issue_type → entity_type)
- `dependencies` (type=parent-child) → `relationships` (type=parent_of)
- `child_counters` → deprecated (no hierarchical IDs needed)
- Issue metadata → Entity metadata (JSON field preserved)

---

### Components

1. **CLI Layer** (`cmd/bd/`)
   - Location: `cmd/bd/main.go` + ~60 command files
   - Purpose: User-facing CLI with 100+ commands organized into groups
   - Pattern: Cobra command tree with `PersistentPreRun` initialization
   - Key State: `dbPath`, `store`, `jsonOutput`, `hookRunner`, `commandDidWrite` (atomic)
   - Command Groups: `issues`, `views`, `deps`, `sync`, `setup`, `maint`, `advanced`

2. **Public Go API** (`beads.go`)
   - Location: Root `beads.go`
   - Purpose: Library interface for Go extensions
   - Pattern: Type re-export facade (aliases internal types)
   - Key Functions: `Open()`, `OpenFromConfig()`, `FindDatabasePath()`, `FindAllDatabases()`, `GetRedirectInfo()`

3. **Core Domain Types** (`internal/types/`)
   - Location: `internal/types/types.go`
   - Purpose: Domain model definitions
   - Pattern: Rich domain model with 70+ database-mapped fields
   - Key Types: `Issue`, `Dependency`, `Label`, `Comment`, `Event`, `Status`, `IssueType`, `DependencyType`

4. **Storage Interface** (`internal/storage/`)
   - Location: `internal/storage/storage.go`
   - Purpose: Abstract storage contract
   - Pattern: Interface-based dependency inversion
   - Key Interfaces: `Storage` (CRUD), `Transaction`, `DoltStorage` (composed: VersionControl + HistoryViewer + RemoteStore + SyncStore + FederationStore + BulkIssueStore + CompactionStore + AdvancedQueryStore + ...)

5. **Dolt Storage Implementation** (`internal/storage/dolt/`)
   - Location: `internal/storage/dolt/` (15+ files)
   - Purpose: MySQL-compatible Dolt backend
   - Pattern: Repository pattern with SQL query building
   - Key Files: `dolt.go` (DoltStore), `schema.go` (DDL), `issues.go` (CRUD), `dependencies.go`, `queries.go`, `filters.go`, `transactions.go`, `migration.go`, `history.go`, `compact.go`
   - Schema: v7, 14+ tables, 70+ columns on issues table

6. **Workspace & Discovery** (`internal/beads/`)
   - Location: `internal/beads/`
   - Purpose: `.beads/` directory resolution, redirect handling, repo identity
   - Pattern: Singleton RepoContext with one-time resolution
   - Features: Git worktree support, redirect chains (single-level), repo fingerprinting (ComputeRepoID, GetCloneID)

7. **Configuration** (`internal/config/`, `internal/configfile/`)
   - Location: `internal/config/config.go`, `internal/configfile/configfile.go`
   - Purpose: Viper YAML config + metadata.json for Dolt connection settings
   - Pattern: Layered config with 4-level precedence (env var > project > user > legacy)
   - Key Config: `issue_prefix`, `dolt.auto-commit`, `sync.mode`, `dolt.shared-server`

8. **Query Engine** (`internal/query/`)
   - Location: `internal/query/`
   - Purpose: Expression parser for issue filtering
   - Pattern: Two-tier evaluation (SQL compilation for AND-chains, in-memory predicates for complex OR expressions)
   - Syntax: `status=open AND priority<2 AND updated>7d`
   - Operators: `=`, `!=`, `<`, `<=`, `>`, `>=`, AND, OR, NOT

9. **Hook System** (`internal/hooks/`)
   - Location: `internal/hooks/hooks.go`
   - Purpose: Event-driven extensibility
   - Pattern: File-based hooks (`.beads/hooks/on_create`, `on_update`, `on_close`)
   - Execution: Async (fire-and-forget) with 10s timeout, JSON input via stdin

10. **Audit & Telemetry** (`internal/audit/`, `internal/telemetry/`)
    - Location: `internal/audit/audit.go`, `internal/telemetry/telemetry.go`
    - Purpose: Observability and compliance
    - Patterns:
      - Audit: Append-only JSONL log (`.beads/interactions.jsonl`) + SQL `interactions` table
      - Telemetry: OpenTelemetry (opt-in via `BD_OTEL_METRICS_URL` / `BD_OTEL_STDOUT`)
    - Instrumentation: Root span per command, child spans for SQL/AI calls

11. **Routing** (`internal/routing/`)
    - Location: `internal/routing/`
    - Purpose: Multi-repo issue routing and context switching
    - Pattern: Prefix-to-path route table (SQL `routes` table)
    - Features: Per-repo prefixes, contributor/maintainer roles, redirect support

12. **External Integrations** (`internal/github/`, `internal/gitlab/`, `internal/jira/`, `internal/linear/`)
    - Location: `internal/{github,gitlab,jira,linear}/`
    - Purpose: Bidirectional sync with external issue trackers
    - Pattern: Plugin adapter architecture via `internal/tracker/` hooks (Pull/Push phases)

13. **Tracker Sync Framework** (`internal/tracker/`)
    - Location: `internal/tracker/`
    - Purpose: Bidirectional issue synchronization engine
    - Pattern: Hook-based plugin architecture (GenerateID, TransformIssue, ShouldImport, ShouldPush)
    - Conflict Resolution: Timestamp (newer wins), Local (keep beads), External (keep tracker)
    - Incremental: Respects `last_sync` timestamp

14. **MCP Server** (`integrations/beads-mcp/`)
    - Location: `integrations/beads-mcp/src/beads_mcp/`
    - Purpose: AI assistant integration (Claude, etc.)
    - Pattern: Python FastMCP 3.1.1 wrapping `bd` subprocess calls
    - Key Files: `server.py`, `tools.py` (20+ tools), `models.py` (Pydantic), `bd_client.py`
    - Context Engineering: Lazy tool loading, compaction (>20 issues → preview), mini models (~80% context reduction)

15. **Dolt Server Manager** (`internal/doltserver/`)
    - Location: `internal/doltserver/`
    - Purpose: Lifecycle management for `dolt sql-server` processes
    - Pattern: Ephemeral port allocation with PID/port files, platform-aware process discovery
    - Modes: Per-project server, shared server (`~/.beads/shared-server/`)

16. **Compaction Engine** (`internal/compact/`)
    - Location: `internal/compact/`
    - Purpose: Semantic memory decay for old issues
    - Pattern: AI-powered summarization via Claude Haiku API
    - Tiers: Level-based compaction (combines description/design/notes/acceptance)
    - Safety: Size validation (compacted < original), retry with exponential backoff

17. **Molecules & Recipes** (`internal/molecules/`, `internal/recipes/`)
    - Location: `internal/molecules/molecules.go`, `internal/recipes/recipes.go`
    - Purpose: Compound workflows and parameterized templates
    - Molecules: Bonded issues for lineage tracking (swarm/patrol/work types)
    - Recipes: 12 built-in integrations (Cursor, Claude, Aider, Junie, etc.)

18. **ID Generation** (`internal/idgen/`)
    - Location: `internal/idgen/`
    - Purpose: Distributed-safe issue ID generation
    - Pattern: SHA256 hash of (title + description + creator + timestamp + nonce), base36 encoded
    - Format: `prefix-hash` (e.g., `bd-a3f8e9`), 3-8 char hashes, progressive scaling

19. **UI Layer** (`internal/ui/`)
    - Location: `internal/ui/`
    - Purpose: Terminal output formatting
    - Pattern: Semantic styling (Tufte-inspired), markdown rendering (Charm/Glamour), paging
    - Design System: Small Unicode symbols (`○ ◐ ● ✓ ❄`), semantic colors, never emoji

20. **Git Integration** (`internal/git/`)
    - Location: `internal/git/gitdir.go`
    - Purpose: Repository detection and VCS operations
    - Features: Git + Jujutsu support, security hardening (SEC-001/002: disable hooks/templates)

21. **Entity Storage** (`internal/storage/dolt/entities.go`) [v8]
    - Location: `internal/storage/dolt/entities.go`
    - Purpose: CRUD operations for entities
    - Pattern: Repository pattern mirroring issues.go structure
    - Features: Create, read, update, delete entities with custom entity_types

22. **Relationship Storage** (`internal/storage/dolt/relationships.go`) [v8]
    - Location: `internal/storage/dolt/relationships.go`
    - Purpose: CRUD operations for relationships with temporal validity
    - Pattern: Repository pattern with temporal window queries
    - Features: Temporal filtering (valid_from/valid_until), bidirectional navigation

23. **Episode Storage** (`internal/storage/dolt/episodes.go`) [v8]
    - Location: `internal/storage/dolt/episodes.go`
    - Purpose: Provenance tracking for entity extraction
    - Pattern: Append-only event log
    - Features: Links entities to source conversations, meetings, observations

24. **Ontology Storage** (`internal/storage/dolt/ontology.go`) [v8]
    - Location: `internal/storage/dolt/ontology.go`
    - Purpose: Custom entity and relationship type definitions
    - Pattern: Schema registry with JSON validation
    - Features: Define, list, validate custom types via Pydantic-like schemas

25. **Knowledge Graph CLI Commands** (`cmd/bd/entity.go`, `cmd/bd/relationship.go`, etc.) [v8]
    - Location: `cmd/bd/{entity,relationship,episode,ontology,graph}.go`
    - Purpose: User-facing CLI for knowledge graph operations
    - Pattern: Cobra command groups mirroring issue commands
    - Commands: `bd entity create/list/update`, `bd relationship create/query`, `bd episode create/show`, `bd ontology define/list`, `bd graph query/visualize`

### Data Flow

**Write Path:**
```
User/Agent → CLI Command → Storage.CreateIssue() → Dolt SQL INSERT → Auto Dolt Commit → [bd dolt push]
```

**Read Path:**
```
User/Agent → CLI Command → Query Engine (parse DSL) → Storage.SearchIssues() → Dolt SQL SELECT → JSON/Styled Output
```

**Sync Path:**
```
Local Dolt DB ←─ bd dolt push/pull ─→ Remote (DoltHub/S3/GCS)
                    or
Local Dolt DB ←─ Tracker Sync ─→ GitHub/GitLab/Jira/Linear
```

**MCP Path:**
```
AI Assistant → MCP Server (Python) → subprocess bd --json → CLI → Storage → Dolt
```

---

## Patterns & Standards

### Communication Patterns

- **Type**: SQL (Dolt embedded or TCP), subprocess (MCP → bd), file-based hooks
- **Internal**: Direct Go function calls through interfaces
- **External**: REST APIs for GitHub/GitLab/Jira/Linear integrations
- **AI Integration**: MCP protocol (FastMCP), Claude API for compaction
- **Sync**: Dolt-native push/pull (cell-level merge), JSONL for backup/export

### Logic Patterns

- **Repository Pattern**: `internal/storage/` defines abstract `Storage` interface; `internal/storage/dolt/` provides Dolt implementation
- **Interface Composition**: `DoltStorage` composes 10+ sub-interfaces (VersionControl, HistoryViewer, RemoteStore, etc.)
- **Transaction Pattern**: `RunInTransaction(ctx, commitMsg, fn)` for atomic multi-operation writes
- **Command Pattern**: Each CLI command is a separate file in `cmd/bd/` with `init()` registration
- **Two-Tier Query**: Simple AND-chains → SQL WHERE clauses; complex expressions → in-memory predicates with pre-filtering
- **Plugin/Hook Pattern**: Tracker adapters inject custom logic via Pull/Push hooks (not inheritance)
- **Singleton Pattern**: `RepoContext` for workspace path resolution (one-time init)
- **Content Hashing**: SHA256 content hash per issue for deduplication and change detection

### Cross-Cutting Concerns

- **Authentication**: Git-based actor identity (from git config or `--actor` flag); federation peers use encrypted credentials
- **Logging**: Append-only JSONL audit log (`.beads/interactions.jsonl`) + SQL `interactions` table; OpenTelemetry traces (opt-in)
- **Error Handling**: Go idiom (error returns); graceful shutdown with signal handling (SIGTERM, SIGHUP); batch commit flush on exit
- **Configuration**: Viper YAML with 4-level precedence; `metadata.json` for Dolt connection; environment variables for telemetry
- **Validation**: `internal/validation/` for priority (0-4, P0-P4), issue types (with aliases), ID prefixes (multi-hyphen support, allowlisting)
- **Security**: Git hook/template disabling (SEC-001/002), federation peer password encryption, readonly mode for sandboxed workers

---

## Design System (CLI Output)

### Status Icons (Unicode, NOT emoji)
- `○` Open
- `◐` In Progress
- `●` Blocked
- `✓` Closed
- `❄` Deferred

### Priority Indicators
- `● P0` Critical (colored)
- `● P1` High
- `● P2` Medium
- `● P3` Low
- `● P4` Backlog

### Typography & Styling
- Tufte-inspired semantic coloring
- Markdown rendering via Charm/Glamour
- JSON output (`--json`) for all commands (programmatic use)
- Interactive pagination for large outputs

### Anti-Patterns
- **NEVER** use emoji-style icons (🔴🟠🟡🔵⚪) in CLI output
- **ALWAYS** use small Unicode symbols with semantic colors

---

## Boundaries & Constraints

### NEVER (Anti-Patterns)
- ❌ Never bypass the Storage interface to access Dolt directly from CLI commands
- ❌ Never use emoji in CLI output (use Unicode symbols: `○ ◐ ● ✓ ❄`)
- ❌ Never create test issues in the production database (use `t.TempDir()`)
- ❌ Never manually modify `.beads/dolt/` directory
- ❌ Never use `bd edit` from AI agents (interactive editor)
- ❌ Never chain `.beads/redirect` files (single-level only)
- ❌ Never use shell heredocs in GitHub Actions YAML `run: |` blocks
- ❌ Never reuse a failed CI tag (bump to next patch version)
- ❌ Never poll GitHub API in loops during releases (rate limit: 5000 req/hr shared)
- ❌ Never import from storage layer in domain types layer
- ❌ Never run raw `CGO_ENABLED=1 go test` on macOS without ICU flags (use `scripts/test-cgo.sh`)

### ALWAYS (Required Patterns)
- ✅ Always use `--json` flag for programmatic CLI output
- ✅ Always include `--description` when creating issues
- ✅ Always use the Storage interface for data access
- ✅ Always add `--json` flag to new commands
- ✅ Always run `go test -short ./...` before committing
- ✅ Always run `golangci-lint run ./...` before committing
- ✅ Always use `t.TempDir()` in Go tests
- ✅ Always use non-interactive flags for file operations (`cp -f`, `mv -f`, `rm -f`)
- ✅ Always push to remote after completing work (`git push`)
- ✅ Always use `bd` for issue tracking (not markdown TODOs)
- ✅ Always link discovered work with `discovered-from` dependencies
- ✅ Always use `scripts/update-versions.sh` for version bumps (version.go is source of truth)

---

## File Organization

### Directory Structure
```
beads/
├── cmd/bd/                   → CLI entry point + 60+ command files (Cobra)
├── beads.go                  → Public Go library API
├── beads_test.go             → Root integration tests
├── internal/
│   ├── types/                → Core domain models (Issue, Dependency, etc.)
│   ├── storage/              → Storage interface definition
│   │   ├── dolt/             → Dolt implementation (schema, CRUD, queries, migrations)
│   │   ├── doltutil/         → Dolt utility functions
│   │   └── embeddeddolt/     → Embedded mode helpers
│   ├── beads/                → Workspace discovery, redirect, repo identity
│   ├── config/               → Viper YAML configuration
│   ├── configfile/           → metadata.json parser
│   ├── query/                → Query DSL parser & evaluator
│   ├── routing/              → Multi-repo routing (prefix → path)
│   ├── hooks/                → Event hooks (on_create, on_update, on_close)
│   ├── audit/                → Append-only JSONL audit log
│   ├── telemetry/            → OpenTelemetry instrumentation (opt-in)
│   ├── ui/                   → Terminal styling, markdown rendering, paging
│   ├── validation/           → Input validation (priority, types, IDs)
│   ├── idgen/                → Hash-based ID generation (SHA256, base36)
│   ├── compact/              → AI-powered semantic compaction (Claude Haiku)
│   ├── molecules/            → Compound issues (bonded work items)
│   ├── recipes/              → Formula/template system (12 integrations)
│   ├── tracker/              → Bidirectional sync framework (hook-based plugins)
│   ├── doltserver/           → Dolt server lifecycle management
│   ├── lockfile/             → Process-level mutual exclusion
│   ├── git/                  → Git/Jujutsu VCS integration
│   ├── github/               → GitHub API client & sync
│   ├── gitlab/               → GitLab API client & sync
│   ├── jira/                 → Jira API client & sync
│   ├── linear/               → Linear API client & sync
│   ├── timeparsing/          → Natural language date parsing
│   ├── debug/                → Debug logging utilities
│   ├── testutil/             → Test infrastructure (Dolt containers)
│   ├── utils/                → Shared helper utilities
│   ├── formula/              → Formula execution engine
│   └── templates/            → Template definitions
├── integrations/
│   ├── beads-mcp/            → Python MCP server (FastMCP 3.1.1 for AI assistants)
│   ├── claude-code/          → Claude Code integration configs
│   └── junie/                → JetBrains Junie integration
├── examples/                 → Usage examples (bash-agent, python-agent, team-workflow, etc.)
├── docs/                     → Documentation (30+ markdown files)
├── scripts/                  → Build, test, release, and utility scripts
├── tests/                    → Integration & regression tests
├── .github/workflows/        → CI/CD (ci, release, nightly, regression, deploy-docs)
├── website/                  → Documentation website
├── npm-package/              → NPM distribution
├── winget/                   → Windows Package Manager manifest
└── .beads/dolt/              → Local Dolt database (gitignored, source of truth)
```

### Naming Conventions
- **Commands**: One file per command in `cmd/bd/` (e.g., `create.go`, `list.go`, `ready.go`)
- **Internal Packages**: Purpose-named directories under `internal/` (e.g., `storage`, `query`, `hooks`)
- **Tests**: Go convention `*_test.go` alongside source files; table-driven tests preferred
- **Scripts**: Shell scripts in `scripts/` (e.g., `test.sh`, `release.sh`, `bump-version.sh`)
- **Config**: YAML for user config, JSON for metadata, TOML for user recipes

---

## Database Schema (Dolt, Schema v7 + v8)

### v7 Tables (Legacy, Deprecated but Maintained)

| Table | Purpose | Key Fields |
|-------|---------|------------|
| `issues` | Core work items | 70+ columns: id, title, description, status, priority, issue_type, assignee, timestamps, metadata JSON, agent fields, gate fields, molecule fields |
| `dependencies` | Issue relationships | issue_id, depends_on_id, type (blocks/related/parent-child/discovered-from), metadata JSON, thread_id |
| `labels` | Issue tags | issue_id, label (composite PK) |
| `comments` | Discussion threads | id (UUID), issue_id, author, text, created_at |
| `events` | Audit trail | id (UUID), issue_id, event_type, actor, old_value, new_value, comment |
| `child_counters` | Hierarchical ID tracking | parent_id, last_child |

### v8 Tables (Knowledge Graph)

| Table | Purpose | Key Fields |
|-------|---------|------------|
| `entities` | Generic entities (replaces issues) | id, entity_type, name, summary, metadata JSON, created_at, updated_at, created_by, status |
| `relationships` | Temporal relationships | id, source_entity_id, relationship_type, target_entity_id, valid_from, valid_until, metadata JSON, created_at |
| `episodes` | Provenance tracking | id, timestamp, source, raw_data TEXT, entities_extracted TEXT, created_at, metadata JSON |
| `entity_types` | Custom entity type registry | name (PK), schema JSON, description, created_at |
| `relationship_types` | Custom relationship type registry | name (PK), schema JSON, description, created_at |

### Shared Tables (Both v7 and v8)

| Table | Purpose | Key Fields |
|-------|---------|------------|
| `config` | Key-value settings | key, value |
| `metadata` | System metadata | key, value |
| `issue_snapshots` | Pre-compaction snapshots | issue_id, compaction_level, original/compressed sizes, original_content |
| `compaction_snapshots` | Compaction data | issue_id, compaction_level, snapshot_json (BLOB) |
| `repo_mtimes` | Multi-repo change detection | repo_path, jsonl_path, mtime_ns |
| `routes` | Prefix-to-path routing | prefix, path |
| `issue_counter` | Sequential ID counter | prefix, last_id |
| `interactions` | Agent audit log | id, kind, actor, issue_id, model, prompt, response, tool_name |
| `federation_peers` | Peer sync credentials | name, remote_url, username, password_encrypted, sovereignty |

### Migration Path

**Coexistence Strategy:**
- Schema v7 and v8 tables live side-by-side in the same Dolt database
- Legacy commands continue using v7 tables with deprecation warnings
- New commands use v8 tables exclusively
- Migration command (`bd migrate v8`) moves data from v7 → v8
- Rollback capability via Dolt branch/reset

**Migration Mapping:**
```
issues.id               → entities.id
issues.title            → entities.name
issues.description      → entities.summary
issues.issue_type       → entities.entity_type
issues.metadata         → entities.metadata (preserved)
dependencies (parent)   → relationships (type=parent_of)
child_counters          → deprecated (no longer needed)
```

### SQL Views
- `ready_issues` — Unblocked work (open issues with no open blockers)
- `blocked_issues` — Issues with open blocker counts

### Indexes
- Status, priority, type, assignee, created_at, spec_id, external_ref on `issues`
- Composite indexes on dependencies (issue_id, depends_on_id, type)
- Thread index on dependencies for conversation threading
- Created_at indexes on comments and events for chronological queries

---

## Build & Deployment

### Build
- **Tool**: Make + Go 1.25.8 (CGO required)
- **Binary**: `bd` (or `bd.exe` on Windows)
- **Install**: `~/.local/bin/bd` with `beads` alias on Unix
- **Cross-compilation**: GoReleaser with Zig CC wrappers (`--parallelism 1`)
- **Targets**: macOS (arm64, x86_64), Linux (arm64, x86_64), Windows (amd64), FreeBSD (amd64)

### Testing
- `make test` — Standard tests (respects `.test-skip`)
- `make test-full-cgo` — Full CGO-enabled suite
- `make test-regression` — Baseline comparison tests
- `make bench` — Dolt storage benchmarks
- Integration tests use `testcontainers-go` for Dolt server

### CI/CD
- **Workflows**: ci.yml, release.yml, nightly.yml, regression.yml, deploy-docs.yml, test-pypi.yml
- **Platforms**: macOS, Linux, Windows, FreeBSD
- **Coverage**: Codecov with component-level reporting
- **Releases**: Tag-triggered via GoReleaser, signed macOS binaries
- **Version Management**: `scripts/update-versions.sh` (version.go is source of truth)

### Distribution
- Homebrew formula (auto-generated by GoReleaser)
- NPM package (`npm-package/`)
- Windows Package Manager (`winget/`)
- Nix flake (`flake.nix`)
- pip install (`integrations/beads-mcp/`)

---

## Key Architectural Decisions

### 1. Dolt as Primary Storage
**Decision**: Use Dolt (version-controlled MySQL-compatible database) instead of SQLite, PostgreSQL, or flat files.
**Rationale**: Native version control, cell-level merge, distributed push/pull without custom sync protocols. Issues travel with code.

### 2. Embedded + Server Dual Mode
**Decision**: Support both in-process (embedded) and TCP (server) database access.
**Rationale**: Embedded for single-user CLI (zero setup), server for multi-writer concurrent access. Transparent switching.

### 3. Hash-Based IDs
**Decision**: SHA256-derived IDs instead of sequential integers.
**Rationale**: Enables distributed concurrent creation without central coordinator. No collision risk across branches/clones.

### 4. Interface-Based Storage
**Decision**: Abstract `Storage` interface with concrete `DoltStore` implementation.
**Rationale**: Testability, potential future backends, clean separation of concerns.

### 5. Two-Tier Query Evaluation
**Decision**: Compile simple AND-chains to SQL, evaluate complex ORs in-memory.
**Rationale**: Performance optimization — most queries are simple filters. Complex queries pre-filter via SQL then refine in-memory.

### 6. Wisps (Ephemeral Local-Only Issues)
**Decision**: Molecule execution creates local-only wisps that are hard-deleted (no tombstones).
**Rationale**: Fast local iteration without sync overhead. Only the digest (outcome) enters shared history.

### 7. AI-Powered Compaction
**Decision**: Use Claude Haiku API for semantic summarization of old issues.
**Rationale**: Reduces database bloat while preserving meaning. Size validation ensures compacted < original.

### 8. Hook-Based Tracker Integration
**Decision**: External trackers integrate via Pull/Push hooks rather than inheritance.
**Rationale**: Flexible adapter pattern allows each tracker to inject custom logic without modifying core sync engine.

### 9. Knowledge Graph Architecture (v8 Migration)
**Decision**: Migrate from rigid Epic/Task/Sub-task hierarchy to flexible entity-relationship graph.
**Rationale**: Enables sophisticated memory systems, custom ontologies, temporal reasoning, and provenance tracking. Supports AI agent workflows beyond issue tracking.

### 10. Temporal Relationship Validity
**Decision**: Add valid_from/valid_until timestamps to relationships.
**Rationale**: Enables time-bounded facts ("Alice led Team X from Jan-Jun 2026"), historical queries, and relationship lifecycle tracking.

### 11. Custom Ontology System
**Decision**: Replace hardcoded issue_type enum with user-defined entity/relationship types.
**Rationale**: Flexibility for domain-specific modeling (meetings, decisions, documents, goals) without code changes. Pydantic-like JSON schema validation.

### 12. Episode Provenance
**Decision**: Link entities to source episodes (conversations, meetings, observations).
**Rationale**: Maintains audit trail of entity extraction, enables traceability, supports AI supervised workflows.

---

## Architectural Evolution

### Pattern Shift (v7 → v8)

| Aspect | v7 (Hierarchical) | v8 (Knowledge Graph) |
|--------|-------------------|----------------------|
| **Data Model** | Epic → Task → Sub-task (3 levels) | Entities + Relationships (unlimited depth) |
| **Type System** | Enum-based (`IssueType`) | Custom ontology (`entity_types` table) |
| **Relationships** | `dependencies` table (blocks/related/parent-child) | `relationships` table with temporal validity |
| **Identity** | Hierarchical IDs (`bd-123`, `bd-123-1`) | Hash-based IDs (same as v7) |
| **Provenance** | Event log (`events` table) | Episode linkage (`episodes` table) |
| **Query Model** | Issue-centric filters (status, priority, type) | Graph traversal with temporal predicates |
| **CLI Commands** | `bd create`, `bd children`, `bd update` | `bd entity`, `bd relationship`, `bd episode`, `bd ontology`, `bd graph` |
| **Storage Layer** | `issues.go`, `dependencies.go` | `entities.go`, `relationships.go`, `episodes.go`, `ontology.go` |
| **Schema Version** | v7 (14 tables, issue-centric) | v8 (19 tables, graph-centric + v7 for backward compat) |

### Migration Timeline

**Phase 1: Foundation (Weeks 1-2)**
- New types: `Entity`, `Relationship`, `Episode`
- Schema v8 tables created
- Storage interface extended

**Phase 2: Storage Layer (Weeks 3-4)**
- Implement `entities.go`, `relationships.go`, `episodes.go`, `ontology.go`
- Temporal validity query logic
- Migration command (`bd migrate v8`)

**Phase 3: CLI Commands (Weeks 5-6)**
- `bd entity create/list/update/delete`
- `bd relationship create/query/update/delete`
- `bd episode create/show/list`
- `bd ontology define/list/validate`
- `bd graph query/visualize/export`

**Phase 4: Deprecation (Weeks 7-8)**
- Deprecation warnings on legacy commands
- Migration guide documentation
- Backward compatibility testing

### Breaking Changes

**User-Facing:**
- `bd children` → `bd relationship query --source <id> --type parent_of`
- `--type epic/task/sub-task` → `--type <custom-entity-type>`
- `--parent <id>` → `bd relationship create <id> parent_of <parent-id>`

**API-Facing:**
- `Storage.CreateIssue()` → `Storage.CreateEntity()`
- `types.Issue` → `types.Entity` (different field names)
- `types.IssueType` → custom type strings

**MCP Server:**
- `mcp__beads__create()` → `mcp__beads__entity_create()`
- New tools: `mcp__beads__relationship_create()`, `mcp__beads__graph_query()`

### Backward Compatibility

**Maintained:**
- All v7 tables remain functional
- Legacy CLI commands continue working (with warnings)
- Existing data untouched until explicit migration
- Dolt push/pull works for both v7 and v8 tables

**Deprecated but Functional:**
- `bd create --type epic/task/sub-task` (maps to v7 tables)
- `bd children` (queries v7 dependencies)
- `--parent` flag (creates v7 parent-child dependency)

**Removed in v9:**
- v7 tables (after 1-year deprecation period)
- Legacy issue-centric CLI commands
- `child_counters` table

---

## Change History

### 2026-03-16 (Update 2) - Knowledge Graph Architecture Documentation
- Added "Knowledge Graph Architecture (v8)" section with entity-relationship model
- Documented temporal validity windows and custom ontology system
- Updated database schema to include v8 tables (entities, relationships, episodes, entity_types, relationship_types)
- Added new components for entity/relationship/episode/ontology storage and CLI commands
- Added "Architectural Evolution" section documenting pattern shift from hierarchical to graph
- Updated architecture diagrams with v8 data flow
- Documented migration strategy and backward compatibility approach
- Confidence: 94% (+2% for comprehensive v8 knowledge graph architecture)

### 2026-03-16 (Update 1) - Initial Architecture Extraction
- Full codebase analysis: 30+ internal packages, 100+ CLI commands, 14+ database tables
- Mapped all entry points, data flows, and integration patterns
- Documented schema v7, dual connection modes, and distributed ID generation
- Confidence: 92%

---

## Confidence Report

- **Score**: 94%
- **Breakdown**:
  - ✅ (+20%) Successfully mapped all entry points (CLI, Go API, MCP server, hooks)
  - ✅ (+20%) Identified data persistence patterns (Dolt SQL, 19+ tables, schema v7+v8, dual mode)
  - ✅ (+20%) Validated internal dependency graph (30+ packages, interface-based storage, tracker plugins)
  - ✅ (+20%) Discovered infrastructure/deployment (CI/CD, GoReleaser, 5 distribution channels, cross-compilation)
  - ✅ (+12%) Bonus: Mapped advanced features (molecules, compaction, federation, gates, HOP)
  - ✅ (+2%) Documented knowledge graph architecture (v8 migration, entity-relationship model, temporal validity, custom ontologies)
- **Known Unknowns**:
  - RPC protocol details (`internal/rpc/protocol.go`) — referenced in docs but not deeply inspected
  - Full federation peer sync protocol internals
  - Exact Claude plugin (`claude-plugin/`) structure and agent definitions
  - Complete OTel metric names and dashboard configurations
  - Website build pipeline details
  - v8 query performance characteristics at scale (10k+ entities/relationships)
