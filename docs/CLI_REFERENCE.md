# CLI Command Reference

**For:** AI agents and developers using bd command-line interface  
**Version:** 0.21.0+

## Quick Navigation

- [Basic Operations](#basic-operations)
- [Issue Management](#issue-management)
- [Memory System (v8)](#memory-system-v8)
  - [Entity Management](#entity-management)
  - [Relationship Management](#relationship-management)
  - [Episode Management](#episode-management)
  - [Ontology Management](#ontology-management)
  - [Graph Operations](#graph-operations)
- [Dependencies & Labels](#dependencies--labels)
- [Filtering & Search](#filtering--search)
- [Advanced Operations](#advanced-operations)
- [Molecular Chemistry](#molecular-chemistry)
- [Database Management](#database-management)
- [Editor Integration](#editor-integration)

## Basic Operations

### Check Status

```bash
# Check database path and server status
bd info --json

# Example output:
# {
#   "database_path": "/path/to/.beads/beads.db",
#   "issue_prefix": "bd",
#   "agent_mail_enabled": false
# }
```

### Find Work

```bash
# Find ready work (no blockers, not already claimed)
bd ready --json

# Atomically claim an issue from the ready queue
bd update <id> --claim --json               # Fails if already claimed

# Find stale issues (not updated recently)
bd stale --days 30 --json                    # Default: 30 days
bd stale --days 90 --status in_progress --json  # Find abandoned claims
bd stale --limit 20 --json                   # Limit results
```

## Issue Management

### Create Issues

> **Deprecation Notice (v8.0+)**: `bd create` is deprecated in favor of `bd entity create` for the new memory system. See [Memory System (v8)](#memory-system-v8) for details. In v7 compatibility mode, `bd create` continues to work.

```bash
# Basic creation
# IMPORTANT: Always quote titles and descriptions with double quotes
bd create "Issue title" -t bug|feature|task -p 0-4 -d "Description" --json

# Create with explicit ID (for parallel workers)
bd create "Issue title" --id worker1-100 -p 1 --json

# Create with labels (--labels or --label work)
bd create "Issue title" -t bug -p 1 -l bug,critical --json
bd create "Issue title" -t bug -p 1 --label bug,critical --json

# Examples with special characters (all require quoting):
bd create "Fix: auth doesn't validate tokens" -t bug -p 1 --json
bd create "Add support for OAuth 2.0" -d "Implement RFC 6749 (OAuth 2.0 spec)" --json
bd create "Implement auth" --spec-id "docs/specs/auth.md" --json

# Create multiple issues from markdown file
bd create -f feature-plan.md --json

# Create with description from file (avoids shell escaping issues)
bd create "Issue title" --body-file=description.md --json
bd create "Issue title" --body-file description.md -p 1 --json

# Read description from stdin
echo "Description text" | bd create "Issue title" --stdin --json
cat description.md | bd create "Issue title" --stdin -p 1 --json
# --body-file=- also works:
echo "Description text" | bd create "Issue title" --body-file=- --json

# Create epic with hierarchical child tasks
bd create "Auth System" -t epic -p 1 --json                     # Returns: bd-a3f8e9
bd create "Login UI" -p 1 --parent bd-a3f8e9 --json             # Auto-assigned: bd-a3f8e9.1
bd create "Backend validation" -p 1 --parent bd-a3f8e9 --json   # Auto-assigned: bd-a3f8e9.2
bd create "Tests" -p 1 --parent bd-a3f8e9 --json                # Auto-assigned: bd-a3f8e9.3

# Create and link discovered work (one command)
bd create "Found bug" -t bug -p 1 --deps discovered-from:<parent-id> --json

# Create with external reference (v0.9.2+)
bd create "Fix login" -t bug -p 1 --external-ref "gh-123" --json  # Short form
bd create "Fix login" -t bug -p 1 --external-ref "https://github.com/org/repo/issues/123" --json  # Full URL
bd create "Jira task" -t task -p 1 --external-ref "jira-PROJ-456" --json  # Custom prefix
```

### Update Issues

```bash
# Update one or more issues
bd update <id> [<id>...] --claim --json
bd update <id> [<id>...] --priority 1 --json
bd update <id> [<id>...] --spec-id "docs/specs/auth.md" --json

# Update external reference (v0.9.2+)
bd update <id> --external-ref "gh-456" --json           # Short form
bd update <id> --external-ref "jira-PROJ-789" --json    # Custom prefix

# Atomically claim an issue for work (prevents race conditions)
# Sets assignee to you and status to in_progress in one atomic operation
# Fails if already claimed (assignee is not empty)
bd update <id> --claim --json

# Edit issue fields in $EDITOR (HUMANS ONLY - not for agents)
# NOTE: This command is intentionally NOT exposed via the MCP server
# Agents should use 'bd update' with field-specific parameters instead
bd edit <id>                    # Edit description
bd edit <id> --title            # Edit title
bd edit <id> --design           # Edit design notes
bd edit <id> --notes            # Edit notes
bd edit <id> --acceptance       # Edit acceptance criteria
```

### Close/Reopen Issues

```bash
# Complete work (supports multiple IDs)
bd close <id> [<id>...] --reason "Done" --json

# Reopen closed issues (supports multiple IDs)
bd reopen <id> [<id>...] --reason "Reopening" --json
```

### View Issues

```bash
# Show dependency tree
bd dep tree <id>

# Get issue details (supports multiple IDs)
bd show <id> [<id>...] --json

# Show the currently active issue (in-progress, hooked, or last touched)
bd show --current
```

## Memory System (v8)

The v8 memory system replaces the rigid issue hierarchy with a flexible entity-relationship graph. This enables modeling arbitrary concepts, temporal relationships, and provenance tracking.

**Key Benefits:**
- Define custom entity types beyond bugs/features/tasks
- Model relationships with temporal validity
- Track knowledge provenance with episodes
- Graph traversal and exploration
- Full backward compatibility with v7 via compatibility mode

### Entity Management

Entities are the core building blocks of the v8 memory system.

```bash
# Create an entity
bd entity create --entity-type <type> --name <name> --description <desc> --json

# Examples:
bd entity create --entity-type person --name "Alice" --description "Senior engineer" --json
bd entity create --entity-type component --name "auth-service" --description "Authentication microservice" --json
bd entity create --entity-type concept --name "zero-trust" --description "Security architecture principle" --json

# List entities by type
bd entity list --entity-type person --json
bd entity list --entity-type component --json
bd entity list --json  # All entities

# Show entity details
bd entity show <id> --json
bd entity show bd-ent-a3f8e9 --json

# Update entity
bd entity update <id> --name <name> --summary <summary> --json
bd entity update bd-ent-a3f8e9 --name "Alice Smith" --json
bd entity update bd-ent-a3f8e9 --summary "Updated expertise areas" --json

# Delete entity
bd entity delete <id> --json
bd entity delete bd-ent-a3f8e9 --json
```

**Entity Types:**
- Defined via ontology registration (see [Ontology Management](#ontology-management))
- Can represent any domain concept: people, components, decisions, patterns, etc.
- Backward compatible: `issue`, `bug`, `feature`, `task` are built-in entity types in v8

### Relationship Management

Relationships connect entities with optional temporal validity and metadata.

```bash
# Create a relationship
bd relationship create --from <id> --type <type> --to <id> --json

# Examples:
bd relationship create --from bd-ent-alice --type authored --to bd-ent-design-doc --json
bd relationship create --from bd-ent-auth --type depends-on --to bd-ent-db --json
bd relationship create --from bd-ent-api --type implements --to bd-ent-spec-42 --json

# Create with temporal validity
bd relationship create --from <id> --type <type> --to <id> --valid-until "2026-12-31" --json
bd relationship create --from bd-ent-alice --type assigned-to --to bd-ent-proj-x --valid-until "2026-06-30" --json

# List relationships
bd relationship list --from <id> --json  # All relationships from entity
bd relationship list --to <id> --json    # All relationships to entity
bd relationship list --from bd-ent-alice --json
bd relationship list --to bd-ent-auth --json

# Show relationship details
bd relationship show <id> --json
bd relationship show bd-rel-a3f8e9 --json

# Update relationship (e.g., extend validity)
bd relationship update <id> --valid-until <time> --json
bd relationship update bd-rel-a3f8e9 --valid-until "2027-01-15" --json

# Delete relationship
bd relationship delete <id> --json
bd relationship delete bd-rel-a3f8e9 --json
```

**Relationship Types:**
- `authored` - Created by
- `depends-on` - Technical dependency
- `implements` - Implementation of spec/design
- `contains` - Hierarchical containment (replaces parent-child)
- `blocks` - Blocks progress
- `related` - Soft connection
- `assigned-to` - Assignment relationship (with temporal validity)
- Custom types via ontology registration

**Temporal Validity:**
- Relationships can have `valid-from` and `valid-until` timestamps
- Useful for modeling assignments, memberships, temporary states
- Query graph at specific points in time

### Episode Management

Episodes record the provenance of knowledge - how entities and relationships were discovered or created.

```bash
# Create an episode (provenance log)
bd episode create --source <source> --file <file> --json

# Examples:
bd episode create --source "claude-session-42" --file "session-log.json" --json
bd episode create --source "interview:alice" --file "notes-2026-03-16.md" --json
bd episode create --source "code-review:pr-123" --file "review-comments.json" --json

# List episodes
bd episode list --json
bd episode list --source "claude-session-42" --json
bd episode list --since "2026-03-01" --json

# Show episode details
bd episode show <id> --json
bd episode show bd-ep-a3f8e9 --json
```

**Episode Sources:**
- AI session identifiers: `claude-session-42`, `gpt4-conv-xyz`
- Human interactions: `interview:alice`, `meeting:standup-2026-03-16`
- Automated discovery: `code-review:pr-123`, `test-run:build-456`
- Custom sources for domain-specific provenance

**Use Cases:**
- Track which AI session discovered a bug
- Record interview as source of requirement
- Link code review to design decisions
- Audit trail for compliance

### Ontology Management

Define custom entity types and relationship types with JSON schemas.

```bash
# Register a custom entity type
bd ontology register-entity-type --name <type> --schema <file> --json

# Example entity type schemas:
# person-schema.json:
# {
#   "type": "object",
#   "properties": {
#     "name": {"type": "string"},
#     "role": {"type": "string"},
#     "expertise": {"type": "array", "items": {"type": "string"}}
#   },
#   "required": ["name"]
# }

bd ontology register-entity-type --name person --schema person-schema.json --json
bd ontology register-entity-type --name component --schema component-schema.json --json

# Register a custom relationship type
bd ontology register-relationship-type --name <type> --schema <file> --json

# Example relationship type schema:
# assignment-schema.json:
# {
#   "type": "object",
#   "properties": {
#     "from": {"type": "string", "description": "person entity"},
#     "to": {"type": "string", "description": "task or project entity"},
#     "allocation": {"type": "number", "minimum": 0, "maximum": 100}
#   },
#   "temporal": true
# }

bd ontology register-relationship-type --name assigned-to --schema assignment-schema.json --json

# List registered types
bd ontology list --json
bd ontology list --entity-types --json
bd ontology list --relationship-types --json
```

**Schema Validation:**
- JSON Schema format for entity and relationship properties
- Enforced at creation/update time
- Enables rich domain modeling
- Built-in types: `issue`, `bug`, `feature`, `task`, `blocks`, `contains`, etc.

### Graph Operations

Explore and traverse the entity-relationship graph.

```bash
# Explore from an entity (breadth-first)
bd graph explore <id> --depth <n> --json

# Examples:
bd graph explore bd-ent-alice --depth 2 --json  # 2 hops from Alice
bd graph explore bd-ent-auth --depth 3 --json   # 3 hops from auth service

# Traverse shortest path between two entities
bd graph traverse <from-id> <to-id> --json
bd graph traverse bd-ent-alice bd-ent-bug-42 --json
bd graph traverse bd-ent-api bd-ent-db --json

# Visualize graph (DOT format for Graphviz)
bd graph visualize <id> --format dot --json
bd graph visualize bd-ent-proj-x --format dot > project.dot
dot -Tpng project.dot -o project.png  # Render with Graphviz

# Visualize with depth limit
bd graph visualize <id> --depth 2 --format dot --json
bd graph visualize bd-ent-auth --depth 3 --format dot --json
```

**Graph Traversal Examples:**

```bash
# Find all dependencies of a component
bd graph explore bd-ent-auth --depth 3 --json | \
  jq '.nodes[] | select(.relationship_type == "depends-on")'

# Find who worked on a feature
bd graph traverse bd-ent-alice bd-ent-feature-oauth --json

# Export project graph for visualization
bd graph visualize bd-ent-proj-x --format dot > project.dot
```

**Visualization Formats:**
- `dot` - Graphviz DOT format (pipe to `dot` command)
- `json` - Structured graph data for custom visualization

## Dependencies & Labels

### Dependencies

> **Deprecation Notice (v8.0+)**: `bd children` is deprecated in favor of `bd relationship list --type contains`. See [Memory System (v8)](#memory-system-v8) for the new relationship-based approach.

```bash
# List hierarchical children (DEPRECATED - use bd relationship list)
bd children <id>
bd children <id> --json

# Link discovered work (old way - two commands)
bd dep add <discovered-id> <parent-id> --type discovered-from

# Create and link in one command (new way - preferred)
bd create "Issue title" -t bug -p 1 --deps discovered-from:<parent-id> --json
```

### Labels

```bash
# Label management (supports multiple IDs)
bd label add <id> [<id>...] <label> --json
bd label remove <id> [<id>...] <label> --json
bd label list <id> --json
bd label list-all --json
```

### State (Labels as Cache)

For operational state tracking on role beads. Uses `<dimension>:<value>` label convention.
See [LABELS.md](LABELS.md#operational-state-pattern-labels-as-cache) for full pattern documentation.

```bash
# Query current state value
bd state <id> <dimension>                    # Output: value
bd state witness-abc patrol                  # Output: active
bd state --json witness-abc patrol           # {"issue_id": "...", "dimension": "patrol", "value": "active"}

# List all state dimensions on an issue
bd state list <id> --json
bd state list witness-abc                    # patrol: active, mode: normal, health: healthy

# Set state (creates event + updates label atomically)
bd set-state <id> <dimension>=<value> --reason "explanation" --json
bd set-state witness-abc patrol=muted --reason "Investigating stuck polecat"
bd set-state witness-abc mode=degraded --reason "High error rate"
```

**Common dimensions:**
- `patrol`: active, muted, suspended
- `mode`: normal, degraded, maintenance
- `health`: healthy, warning, failing
- `status`: idle, working, blocked

**What `set-state` does:**
1. Creates event bead with reason (source of truth)
2. Removes old `<dimension>:*` label if exists
3. Adds new `<dimension>:<value>` label (cache)

## Filtering & Search

### Basic Filters

```bash
# Filter by status, priority, type
bd list --status open --priority 1 --json               # Status and priority
bd list --assignee alice --json                         # By assignee
bd list --type bug --json                               # By issue type
bd list --id bd-123,bd-456 --json                       # Specific IDs
bd list --spec "docs/specs/" --json                     # Spec prefix
```

### Label Filters

```bash
# Labels (AND: must have ALL)
bd list --label bug,critical --json

# Labels (OR: has ANY)
bd list --label-any frontend,backend --json
```

### Text Search

```bash
# Title search (substring)
bd list --title "auth" --json

# Pattern matching (case-insensitive substring)
bd list --title-contains "auth" --json                  # Search in title
bd list --desc-contains "implement" --json              # Search in description
bd list --notes-contains "TODO" --json                  # Search in notes

# Find beads issue by external reference
bd list --json | jq -r '.[] | select(.external_ref == "gh-123") | .id'
```

### Date Range Filters

```bash
# Date range filters (YYYY-MM-DD or RFC3339)
bd list --created-after 2024-01-01 --json               # Created after date
bd list --created-before 2024-12-31 --json              # Created before date
bd list --updated-after 2024-06-01 --json               # Updated after date
bd list --updated-before 2024-12-31 --json              # Updated before date
bd list --closed-after 2024-01-01 --json                # Closed after date
bd list --closed-before 2024-12-31 --json               # Closed before date
```

### Empty/Null Checks

```bash
# Empty/null checks
bd list --empty-description --json                      # Issues with no description
bd list --no-assignee --json                            # Unassigned issues
bd list --no-labels --json                              # Issues with no labels
```

### Priority Ranges

```bash
# Priority ranges
bd list --priority-min 0 --priority-max 1 --json        # P0 and P1 only
bd list --priority-min 2 --json                         # P2 and below
```

### Combine Filters

```bash
# Combine multiple filters
bd list --status open --priority 1 --label-any urgent,critical --no-assignee --json
```

## Global Flags

Global flags work with any bd command and must appear **before** the subcommand.

### Sandbox Mode

**Auto-detection (v0.21.1+):** bd automatically detects sandboxed environments and enables sandbox mode.

When detected, you'll see: `ℹ️  Sandbox detected, using direct mode`

**Manual override:**

```bash
# Explicitly enable sandbox mode
bd --sandbox <command>
```

**What it does:**
- Uses embedded database mode (no server needed)
- Disables auto-sync operations

**When to use:** Sandboxed environments where the Dolt server can't be controlled (permission restrictions), or when auto-detection doesn't trigger.

### Other Global Flags

```bash
# JSON output for programmatic use
bd --json <command>

# Disable auto-sync
bd --no-auto-flush <command>    # Disable auto-flush
bd --no-auto-import <command>   # Disable auto-import

# Custom database path
bd --db /path/to/.beads/beads.db <command>

# Custom actor for audit trail
bd --actor alice <command>
```

**See also:**
- [TROUBLESHOOTING.md - Sandboxed environments](TROUBLESHOOTING.md#sandboxed-environments-codex-claude-code-etc) for detailed sandbox troubleshooting

## Advanced Operations

### Cleanup

```bash
# Clean up closed issues (bulk deletion)
bd admin cleanup --force --json                                   # Delete ALL closed issues
bd admin cleanup --older-than 30 --force --json                   # Delete closed >30 days ago
bd admin cleanup --dry-run --json                                 # Preview what would be deleted
bd admin cleanup --older-than 90 --cascade --force --json         # Delete old + dependents
```

### Orphan Detection

Find issues referenced in git commits that were never closed:

```bash
# Basic usage - scan current repo
bd orphans

# Cross-repo: scan CODE repo's commits against external BEADS database
cd ~/my-code-repo
bd orphans --db ~/my-beads-repo/.beads/beads.db

# JSON output
bd orphans --json
```

**Use case**: When your beads database lives in a separate repository from your code, run `bd orphans` from the code repo and point `--db` to the external database. This scans commits in your current directory while checking issue status from the specified database.

### Duplicate Detection & Merging

```bash
# Find and merge duplicate issues
bd duplicates                                          # Show all duplicates
bd duplicates --auto-merge                             # Automatically merge all
bd duplicates --dry-run                                # Preview merge operations

# Merge specific duplicate issues
bd merge <source-id...> --into <target-id> --json      # Consolidate duplicates
bd merge bd-42 bd-43 --into bd-41 --dry-run            # Preview merge
```

### Compaction (Memory Decay)

```bash
# Agent-driven compaction
bd admin compact --analyze --json                           # Get candidates for review
bd admin compact --analyze --tier 1 --limit 10 --json       # Limited batch
bd admin compact --apply --id bd-42 --summary summary.txt   # Apply compaction
bd admin compact --apply --id bd-42 --summary - < summary.txt  # From stdin
bd admin compact --stats --json                             # Show statistics

# Legacy AI-powered compaction (requires ANTHROPIC_API_KEY)
bd admin compact --auto --dry-run --all                     # Preview
bd admin compact --auto --all --tier 1                      # Auto-compact tier 1

# Restore compacted issue from git history
bd restore <id>  # View full history at time of compaction
```

### Rename Prefix

```bash
# Rename issue prefix (e.g., from 'knowledge-work-' to 'kw-')
bd rename-prefix kw- --dry-run  # Preview changes
bd rename-prefix kw- --json     # Apply rename
```

### Reset

Remove all local beads data and return to uninitialized state.

```bash
# Preview what would be removed (dry-run)
bd admin reset

# Actually perform the reset
bd admin reset --force
```

**What gets removed:**
- `.beads/` directory (database, config)
- Git hooks installed by bd
- Merge driver configuration
- Sync branch worktrees (`.git/beads-worktrees/`)

**What does NOT get removed:**
- Remote sync branch (if configured)
- Remote Dolt repository data
- Historical git commits

**Important:** If you want a complete clean slate (including remote data), see [Troubleshooting: Old data returns after reset](TROUBLESHOOTING.md#old-data-returns-after-reset).

**Note:** The `--hard` and `--skip-init` flags mentioned in some discussions were never implemented. Use `--force` to perform the reset.

## Molecular Chemistry

Beads uses a chemistry metaphor for template-based workflows. See [MOLECULES.md](MOLECULES.md) for full documentation.

### Phase Transitions

| Phase | State | Storage | Command |
|-------|-------|---------|---------|
| Solid | Proto | `.beads/` | `bd formula list` |
| Liquid | Mol | `.beads/` | `bd mol pour` |
| Vapor | Wisp | `.beads/` (Ephemeral=true, not exported) | `bd mol wisp` |

### Proto/Template Commands

```bash
# List available formulas (templates)
bd formula list --json

# Show proto structure and variables
bd mol show <proto-id> --json

# Extract proto from ad-hoc epic
bd mol distill <epic-id> --json
```

### Pour (Proto to Mol)

```bash
# Instantiate proto as persistent mol (solid → liquid)
bd mol pour <proto-id> --var key=value --json

# Preview what would be created
bd mol pour <proto-id> --var key=value --dry-run

# Assign root issue
bd mol pour <proto-id> --var key=value --assignee alice --json

# Attach additional protos during pour
bd mol pour <proto-id> --attach <other-proto> --json
```

### Wisp Commands

```bash
# Instantiate proto as ephemeral wisp (solid → vapor)
bd mol wisp <proto-id> --var key=value --json

# List all wisps
bd mol wisp list --json
bd mol wisp list --all --json    # Include closed

# Garbage collect orphaned wisps
bd mol wisp gc --json
bd mol wisp gc --age 24h --json  # Custom age threshold
bd mol wisp gc --dry-run         # Preview what would be cleaned

# Purge all closed wisps (bulk cleanup)
bd mol wisp gc --closed              # Preview closed wisp deletion
bd mol wisp gc --closed --force      # Delete all closed wisps
bd mol wisp gc --closed --dry-run    # Detailed dry-run preview
```

### Bonding (Combining Work)

```bash
# Polymorphic combine - handles proto+proto, proto+mol, mol+mol
bd mol bond <A> <B> --json

# Bond types
bd mol bond <A> <B> --type sequential --json   # B runs after A (default)
bd mol bond <A> <B> --type parallel --json     # B runs alongside A
bd mol bond <A> <B> --type conditional --json  # B runs only if A fails

# Phase control
bd mol bond <proto> <mol> --pour --json   # Force persistent spawn
bd mol bond <proto> <mol> --wisp --json   # Force ephemeral spawn

# Dynamic bonding (custom child IDs)
bd mol bond <proto> <mol> --ref arm-{{name}} --var name=ace --json

# Preview bonding
bd mol bond <A> <B> --dry-run
```

### Squash (Wisp to Digest)

```bash
# Compress wisp to permanent digest
bd mol squash <ephemeral-id> --json

# With agent-provided summary
bd mol squash <ephemeral-id> --summary "Work completed" --json

# Preview
bd mol squash <ephemeral-id> --dry-run

# Keep wisp children after squash
bd mol squash <ephemeral-id> --keep-children --json
```

### Burn (Discard Wisp)

```bash
# Delete wisp without digest (destructive)
bd mol burn <ephemeral-id> --json

# Preview
bd mol burn <ephemeral-id> --dry-run

# Skip confirmation
bd mol burn <ephemeral-id> --force --json
```

**Note:** Mol commands use the standard Dolt database access path.

## Gates

Gates are async wait conditions that block dependent work until external conditions are met.
See [DEPENDENCIES.md](DEPENDENCIES.md) for full documentation.

```bash
# List open gates
bd gate list
bd gate list --all                       # Including closed

# Show gate details
bd gate show <gate-id>

# Evaluate gates and close resolved ones
bd gate check                            # All gates
bd gate check --type=gh:pr               # Only PR merge gates
bd gate check --type=gh:run              # Only CI run gates
bd gate check --type=timer               # Only timer gates
bd gate check --type=bead               # Only cross-rig bead gates
bd gate check --dry-run                  # Preview without changes
bd gate check --escalate                 # Escalate failed gates

# Manually resolve a gate
bd gate resolve <gate-id> --reason "Approved"

# Auto-discover CI run IDs for gh:run gates
bd gate discover
bd gate discover --dry-run --branch main

# Add a waiter to a gate
bd gate add-waiter <gate-id> <waiter>
```

## Database Management

### Export / Backup / Bootstrap

```bash
# Export issues to issue JSONL
bd export -o issues.jsonl

# Write or restore the supported JSONL backup snapshot
bd backup
bd backup restore
bd backup export-git
bd backup fetch-git

# Bootstrap a new database from an issue export
bd init --from-jsonl                            # Reads .beads/issues.jsonl

# Configure orphan handling for pulls and bootstrapping
bd config set import.orphan_handling "resurrect"
bd dolt pull  # Respects import.orphan_handling setting
```

**Orphan handling modes** (apply to `bd dolt pull` and `bd init --from-jsonl`):

- **`allow` (default)** - Import orphaned children without parent validation. Most permissive, ensures no data loss even if hierarchy is temporarily broken.
- **`resurrect`** - Search for deleted parents and recreate them as tombstones (Status=Closed, Priority=4). Preserves hierarchy with minimal data.
- **`skip`** - Skip orphaned children with warning. Partial import succeeds but some issues are excluded.
- **`strict`** - Fail immediately if a child's parent is missing. Use when database integrity is critical.

See [CONFIG.md](CONFIG.md#example-import-orphan-handling) and [TROUBLESHOOTING.md](TROUBLESHOOTING.md#import-fails-with-missing-parent-errors) for more details.

### Migration

```bash
# Migrate databases after version upgrade (v7 and earlier)
bd migrate                                             # Detect and migrate old databases
bd migrate --dry-run                                   # Preview migration
bd migrate --cleanup --yes                             # Migrate and remove old files

# AI-supervised migration (check before running bd migrate)
bd migrate --inspect --json                            # Show migration plan for AI agents
bd info --schema --json                                # Get schema, tables, config, sample IDs
```

**Migration workflow for AI agents:**

1. Run `--inspect` to see pending migrations and warnings
2. Check for `missing_config` (like issue_prefix)
3. Review `invariants_to_check` for safety guarantees
4. If warnings exist, fix config issues first
5. Then run `bd migrate` safely

**Migration safety invariants:**

- **required_config_present**: Ensures issue_prefix and schema_version are set
- **foreign_keys_valid**: No orphaned dependencies or labels
- **issue_count_stable**: Issue count doesn't decrease unexpectedly

These invariants prevent data loss and would have caught issues like GH #201 (missing issue_prefix after migration).

### v8 Migration (Memory System)

Migrate from the v7 issue system to the v8 entity-relationship memory system.

```bash
# Check migration status
bd migrate status --json

# Preview v8 migration (shows what will be transformed)
bd migrate to-v8 --dry-run --json

# Perform v8 migration
bd migrate to-v8 --json

# Validate v8 migration integrity
bd migrate validate --json

# Rollback to v7 (if migration issues occur)
bd migrate rollback --json
```

**v8 Migration Process:**

1. **Backup**: Automatically backs up v7 database before migration
2. **Transform**: Converts issues → entities, dependencies → relationships
3. **Validate**: Checks data integrity and schema consistency
4. **Commit**: Finalizes migration if validation passes

**What gets migrated:**

- Issues → Entities (with `issue` entity type)
- Bug/Feature/Task types → Entity types with corresponding schemas
- Dependencies → Relationships (blocks, contains, discovered-from, etc.)
- Parent-child hierarchy → `contains` relationships
- Labels → Entity metadata
- Comments → Associated with entities

**Compatibility:**

- Once migrated to v8, use [Compatibility Mode](#compatibility-mode) to run v7 commands
- Rollback available if issues are detected
- Migration is atomic - either fully succeeds or fully rolls back

**Examples:**

```bash
# Check if migration needed
bd migrate status --json
# Output: {"current_version": "v7", "latest_version": "v8", "migration_needed": true}

# Preview migration
bd migrate to-v8 --dry-run
# Shows: 42 issues → 42 entities, 15 dependencies → 15 relationships

# Perform migration
bd migrate to-v8 --json
# Output: {"status": "success", "entities_created": 42, "relationships_created": 15}

# Validate
bd migrate validate --json
# Output: {"valid": true, "checks": ["foreign_keys", "entity_integrity", "relationship_integrity"]}
```



### Migrate to Sync Branch

Set up a dedicated sync branch for beads data, keeping your working branches clean.

```bash
# Basic setup (creates orphan branch by default)
bd migrate sync beads-sync                             # Create orphan sync branch
bd migrate sync beads-sync --dry-run                   # Preview without changes

# Force reconfigure if already set up
bd migrate sync beads-sync --force                     # Reconfigure sync branch

# Migrate existing non-orphan branch to orphan
bd migrate sync beads-sync --orphan                    # Delete and recreate as orphan
```

**Behavior:**

| Scenario | Result |
|----------|--------|
| Branch doesn't exist | Creates orphan branch (no shared history) |
| Branch exists locally | Uses existing branch as-is |
| Branch exists + `--orphan` | Migrates: deletes and recreates as orphan |
| Remote only | Fetches from remote |
| Remote only + `--orphan` | Creates local orphan (ignores remote) |

**Why orphan branches?**

- Clean "data sync channel" mental model
- No accidental merge risk (git warns loudly)
- Smaller repository footprint (no stale source code)
- Sync branch contains only `.beads/` directory

**After setup:**

- `bd dolt push` commits beads changes to the sync branch via worktree
- Your working branch stays clean of beads commits
- Essential for multi-clone setups where clones work independently

**Safety features for `--orphan` migration:**

- **Unpushed commit check**: If the branch has unpushed commits, migration fails with a helpful error. Use `--force` to override.
- **Existing worktree**: If a worktree exists for the branch, it's automatically removed before migration.
- **Non-destructive to remote**: The remote branch is not modified; use `git push --force` to update it after migration.

### Compatibility Mode

After migrating to v8, you can use compatibility mode to temporarily run v7 commands against the v8 database.

```bash
# Check current compatibility mode
bd compat status
bd compat status --json

# Set compatibility mode
bd compat set v7    # Enable v7 compatibility (translate v7 commands to v8)
bd compat set v8    # Disable compatibility mode (native v8 commands only)
```

**How it works:**

**v7 mode (compatibility enabled):**
- `bd create` → Translates to `bd entity create --entity-type issue`
- `bd children <id>` → Translates to `bd relationship list --from <id> --type contains`
- `bd dep add` → Translates to `bd relationship create`
- All v7 commands work transparently with v8 data

**v8 mode (native, default after migration):**
- Only v8 commands available (`bd entity`, `bd relationship`, etc.)
- v7 commands show deprecation warnings
- Recommended for new workflows

**Examples:**

```bash
# After migration, enable v7 compatibility for existing scripts
bd compat set v7

# Now v7 commands work
bd create "Bug fix" -t bug -p 1 --json
# Internally: bd entity create --entity-type issue --name "Bug fix" --metadata '{"type":"bug","priority":1}'

bd children bd-42
# Internally: bd relationship list --from bd-42 --type contains

# Switch back to v8 native mode
bd compat set v8

# Now only v8 commands work
bd entity create --entity-type person --name "Alice" --json
```

**When to use v7 compatibility:**
- Gradual migration of existing scripts and workflows
- Temporary use of legacy tools that expect v7 commands
- Testing and validation during migration period

**When to use v8 native mode:**
- New workflows leveraging the full memory system
- Graph traversal and custom entity types
- After completing migration and updating all scripts

### Sync Operations

```bash
# Manual sync (push changes to remote)
bd dolt push

# Pull changes from remote
bd dolt pull

# What these do:
# bd dolt push - Commit pending changes to Dolt and push to remote
# bd dolt pull - Pull from remote and merge any updates
```

### Key-Value Store

Store user-defined key-value pairs that persist across sessions. Useful for feature flags, environment config, or agent memory.

```bash
# Set a value
bd kv set <key> <value>
bd kv set feature_flag true
bd kv set api_endpoint https://api.example.com

# Get a value
bd kv get <key>
bd kv get feature_flag                 # Prints: true
bd kv get missing_key                  # Prints: missing_key (not set), exits 1

# Delete a key
bd kv clear <key>
bd kv clear feature_flag

# List all key-value pairs
bd kv list
bd kv list --json                      # Machine-readable output
```

**Storage notes:**
- KV data is stored in the local database with a `kv.` prefix
- KV data syncs via Dolt remotes

**Use cases:**
- Feature flags: `bd set debug_mode true`
- Environment config: `bd set staging_url https://staging.example.com`
- Agent memory: `bd set last_migration 20240115_add_users.sql`
- Session state: `bd set current_sprint 42`

## Issue Types

- `bug` - Something broken that needs fixing
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature composed of multiple issues (supports hierarchical children)
- `chore` - Maintenance work (dependencies, tooling)

**Hierarchical children:** Epics can have child issues with dotted IDs (e.g., `bd-a3f8e9.1`, `bd-a3f8e9.2`). Children are auto-numbered sequentially. Up to 3 levels of nesting supported.

## Issue Statuses

- `open` - Ready to be worked on
- `in_progress` - Currently being worked on
- `blocked` - Cannot proceed (waiting on dependencies)
- `deferred` - Deliberately put on ice for later
- `closed` - Work completed
- `tombstone` - Deleted issue (suppresses resurrections)
- `pinned` - Stays open indefinitely (used for hooks, anchors)

**Note:** The `pinned` status is used by orchestrators for hook management and persistent work items that should never be auto-closed or cleaned up.

## Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (nice-to-have features, minor bugs)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

## Dependency Types

- `blocks` - Hard dependency (issue X blocks issue Y)
- `related` - Soft relationship (issues are connected)
- `parent-child` - Epic/subtask relationship
- `discovered-from` - Track issues discovered during work

Only `blocks` dependencies affect the ready work queue.

**Note:** When creating an issue with a `discovered-from` dependency, the new issue automatically inherits the parent's `source_repo` field.

## External References

The `--external-ref` flag (v0.9.2+) links beads issues to external trackers:

- Supports short form (`gh-123`) or full URL (`https://github.com/...`)
- Portable via Dolt - survives sync across machines
- Custom prefixes work for any tracker (`jira-PROJ-456`, `linear-789`)

## Output Formats

### JSON Output (Recommended for Agents)

Always use `--json` flag for programmatic use:

```bash
# Single issue
bd show bd-42 --json

# List of issues
bd ready --json

# Operation result
bd create "Issue" -p 1 --json
```

### Human-Readable Output

Default output without `--json`:

```bash
bd ready
# ○ bd-42 [P1] [bug] - Fix authentication bug
# ○ bd-43 [P2] [feature] - Add user settings page
```

**Dependency visibility:** When issues have blocking dependencies, they appear inline:

```bash
bd list --parent epic-123
# ○ bd-123.1 [P1] [task] - Design API (blocks: bd-123.2, bd-123.3)
# ○ bd-123.2 [P1] [task] - Implement endpoints (blocked by: bd-123.1, blocks: bd-123.3)
# ○ bd-123.3 [P1] [task] - Add tests (blocked by: bd-123.1, bd-123.2)
```

This makes blocking relationships visible without running `bd show` on each issue.

## Common Patterns for AI Agents

### Claim and Complete Work

```bash
# 1. Find available work
bd ready --json

# 2. Claim issue
bd update bd-42 --claim --json

# 3. Work on it...

# 4. Close when done
bd close bd-42 --reason "Implemented and tested" --json
```

### Discover and Link Work

```bash
# While working on bd-100, discover a bug

# Old way (two commands):
bd create "Found auth bug" -t bug -p 1 --json  # Returns bd-101
bd dep add bd-101 bd-100 --type discovered-from

# New way (one command):
bd create "Found auth bug" -t bug -p 1 --deps discovered-from:bd-100 --json
```

### Batch Operations

```bash
# Update multiple issues at once
bd update bd-41 bd-42 bd-43 --priority 0 --json

# Close multiple issues
bd close bd-41 bd-42 bd-43 --reason "Batch completion" --json

# Add label to multiple issues
bd label add bd-41 bd-42 bd-43 urgent --json
```

### Session Workflow

```bash
# Start of session
bd ready --json  # Find work

# During session
bd create "..." -p 1 --json
bd update bd-42 --claim --json
# ... work ...

# End of session (IMPORTANT!)
bd dolt push  # Force immediate sync, bypass debounce
```

**ALWAYS run `bd dolt push` at end of agent sessions** to ensure changes are committed/pushed immediately.

## Editor Integration

### Setup Commands

```bash
# Setup editor integration (choose based on your editor)
bd setup factory  # Factory.ai Droid - creates/updates AGENTS.md (universal standard)
bd setup codex    # Codex CLI - creates/updates AGENTS.md
bd setup mux      # Mux - creates/updates AGENTS.md
bd setup claude   # Claude Code - installs hooks + manages CLAUDE.md (minimal profile)
bd setup gemini   # Gemini CLI - installs hooks + manages GEMINI.md (minimal profile)
bd setup cursor   # Cursor IDE - creates .cursor/rules/beads.mdc
bd setup aider    # Aider - creates .aider.conf.yml

# Check if integration is installed
bd setup factory --check
bd setup codex --check
bd setup mux --check
bd setup claude --check
bd setup gemini --check
bd setup cursor --check
bd setup aider --check

# Remove integration
bd setup factory --remove
bd setup codex --remove
bd setup mux --remove
bd setup claude --remove
bd setup gemini --remove
bd setup cursor --remove
bd setup aider --remove
```

**Claude Code options:**
```bash
bd setup claude              # Install globally (~/.claude/settings.json)
bd setup claude --project    # Install for this project only
bd setup claude --stealth    # Use stealth mode (flush only, no git operations)
bd setup gemini              # Install globally (~/.gemini/settings.json)
bd setup gemini --project    # Install for this project only
bd setup gemini --stealth    # Use stealth mode (flush only, no git operations)
bd setup mux --project       # Also install .mux/AGENTS.md workspace layer
bd setup mux --global        # Also install ~/.mux/AGENTS.md global layer
```

**What each setup does:**
- **Factory.ai** (`bd setup factory`): Creates or updates AGENTS.md with beads workflow instructions (full profile — works with multiple AI tools using the AGENTS.md standard)
- **Codex CLI** (`bd setup codex`): Creates or updates AGENTS.md with beads workflow instructions for Codex (full profile)
- **Mux** (`bd setup mux`): Creates or updates AGENTS.md with beads workflow instructions for Mux workspaces (full profile)
- **Claude Code** (`bd setup claude`): Adds hooks to Claude Code's settings.json that run `bd prime` on SessionStart and PreCompact events and manages a minimal-profile beads section in `CLAUDE.md`
- **Gemini CLI** (`bd setup gemini`): Adds hooks to Gemini's settings.json that run `bd prime` on SessionStart and PreCompress events and manages a minimal-profile beads section in `GEMINI.md`
- **Cursor** (`bd setup cursor`): Creates `.cursor/rules/beads.mdc` with workflow instructions
- **Aider** (`bd setup aider`): Creates `.aider.conf.yml` with bd workflow instructions

**`--check` behavior:** For section-based integrations (including Claude/Gemini instruction files), reports status as `current` (up to date), `stale` (legacy or hash mismatch — run setup to update), or `missing` (no beads section). Stale and missing return non-zero exit codes.

See also:
- [INSTALLING.md](INSTALLING.md#ide-and-editor-integrations) - Installation guide
- [AIDER_INTEGRATION.md](AIDER_INTEGRATION.md) - Detailed Aider guide
- [CLAUDE_INTEGRATION.md](CLAUDE_INTEGRATION.md) - Claude integration design

## See Also

- [AGENTS.md](../AGENTS.md) - Main agent workflow guide
- [MOLECULES.md](MOLECULES.md) - Molecular chemistry metaphor (protos, pour, bond, squash, burn)
- [GIT_INTEGRATION.md](GIT_INTEGRATION.md) - Git worktrees and protected branches
- [LABELS.md](../LABELS.md) - Label system guide
- [README.md](../README.md) - User documentation
