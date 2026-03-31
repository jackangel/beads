# How to Use Knowledge Graph Features

**Status**: 🚧 **NOT YET IMPLEMENTED** - This is a design specification for planned features.

## Overview: From Issues to Knowledge Graphs

Currently, beads uses a hierarchical model (Epic → Task → Sub-task). The plan proposes migrating to a **flexible knowledge graph** with:

- **Entities** - General-purpose nodes (not just tasks)
- **Relationships** - Temporal edges with validity windows
- **Episodes** - Immutable provenance log
- **Custom Ontology** - Define your own types
- **Graph Exploration** - Traverse and visualize

---

## 1. Entities (replaces Issues)

General-purpose nodes that can represent anything - bugs, features, people, documents, etc.

### Examples

```bash
# Create a bug entity
bd entity create --entity-type bug --name "Login fails on Safari" \
  --description "Users can't log in on Safari 17+" --json

# Create a feature entity
bd entity create --entity-type feature --name "Dark mode" \
  --description "Add dark mode support" --json

# Create a person entity (custom type)
bd entity create --entity-type person --name "Alice" \
  --description "Frontend developer" --json

# List all entities of a type
bd entity list --entity-type feature --json

# Show entity details
bd entity show bd-42 --json

# Update entity
bd entity update bd-42 --name "Dark mode v2" --summary "Updated scope" --json

# Delete entity
bd entity delete bd-42 --json
```

---

## 2. Relationships (replaces Dependencies)

Temporal edges between entities with validity windows. Relationships can expire or be time-bounded.

### Examples

```bash
# Create a "blocks" relationship
bd relationship create --from bd-100 --type blocks --to bd-101 --json

# Create a "works-on" relationship with time bounds
bd relationship create --from person-alice --type works-on --to bd-42 \
  --valid-from "2026-03-01" --valid-until "2026-03-31" --json

# List outgoing relationships from an entity
bd relationship list --from bd-100 --json

# List incoming relationships to an entity
bd relationship list --to bd-100 --json

# Show relationship details
bd relationship show rel-50 --json

# Close a relationship (set end date)
bd relationship update rel-50 --valid-until "2026-03-31" --json

# Delete relationship (sets valid_until to now)
bd relationship delete rel-50 --json
```

### Temporal Validity

Relationships have `valid_from` and `valid_until` timestamps:

- **Active relationship**: `valid_until` is NULL or in the future
- **Expired relationship**: `valid_until` is in the past
- **Future relationship**: `valid_from` is in the future

Queries can filter by temporal validity:

```bash
# Get relationships active at specific time
bd relationship list --from bd-100 --at "2026-02-15" --json

# Get relationships active now
bd relationship list --from bd-100 --active --json
```

---

## 3. Episodes (provenance layer)

Immutable log of raw data sources. Tracks where entities came from.

### Examples

```bash
# Record a GitHub issue as an episode
bd episode create --source "github:issue:123" --file issue-123.json --json

# Record a meeting transcript
bd episode create --source "meeting:standup" --file transcript.txt --json

# Record a Slack conversation
bd episode create --source "slack:thread:456" --file thread.json --json

# List episodes from a source
bd episode list --source "github" --since "2026-03-01" --json

# Show episode details
bd episode show ep-100 --json
```

### Use Cases

- **Provenance tracking**: Know where each entity originated
- **Audit trail**: Immutable log of all data sources
- **Re-extraction**: Reprocess raw data if entity extraction logic changes
- **Debugging**: Trace entity back to original source

---

## 4. Custom Ontology (flexible types)

Define your own entity and relationship types with JSON schemas (Pydantic-like validation).

### Examples

**Register custom entity type:**

```bash
# Create schema file (design-asset-schema.json)
{
  "type": "object",
  "properties": {
    "file_path": {"type": "string"},
    "file_type": {"enum": ["figma", "sketch", "svg", "png"]},
    "designer": {"type": "string"},
    "version": {"type": "number"}
  },
  "required": ["file_path", "file_type"]
}

# Register the type
bd ontology register-entity-type --name "design-asset" \
  --schema design-asset-schema.json --json
```

**Register custom relationship type:**

```bash
# Create schema file (references-schema.json)
{
  "type": "object",
  "properties": {
    "reference_type": {"enum": ["imports", "uses", "extends"]},
    "line_number": {"type": "number"}
  },
  "required": ["reference_type"]
}

# Register the type
bd ontology register-relationship-type --name "references" \
  --schema references-schema.json --json
```

**Use custom types:**

```bash
# Create entity with custom type
bd entity create --entity-type design-asset \
  --name "Login screen mockup" \
  --metadata '{"file_path": "designs/login.fig", "file_type": "figma"}' \
  --json

# Create relationship with custom type
bd relationship create --from component-a --type references --to component-b \
  --metadata '{"reference_type": "imports", "line_number": 42}' \
  --json

# List all registered types
bd ontology list --json
```

---

## 5. Graph Exploration (the killer feature)

Traverse and visualize the knowledge graph.

### Examples

**Explore neighborhood:**

```bash
# Explore entities within 2 hops of bd-42
bd graph explore bd-42 --depth 2 --json

# Output shows:
# - Direct relationships (depth 1)
# - Relationships of relationships (depth 2)
# - Entity details for each node
```

**Find shortest path:**

```bash
# Find shortest path from bd-100 to bd-200
bd graph traverse bd-100 bd-200 --json

# Output shows:
# - Path: [bd-100, bd-150, bd-180, bd-200]
# - Relationships connecting each node
```

**Visualize graph:**

```bash
# Generate Graphviz DOT format
bd graph visualize bd-42 --format dot > graph.dot

# Render as PNG
dot -Tpng graph.dot -o graph.png

# Render as SVG
dot -Tsvg graph.dot -o graph.svg
```

---

## Migration Process

When this feature is implemented, you'll migrate existing data from v7 (issues) to v8 (knowledge graph):

### Preview Migration

```bash
# Dry-run (shows what will happen without applying)
bd migrate to-v8 --dry-run

# Output shows:
# - How many issues → entities
# - How many dependencies → relationships
# - How many events → episodes
# - Schema changes
```

### Run Migration

```bash
# Execute migration
bd migrate to-v8

# Progress shows:
# [1000/5000] Migrating issues to entities...
# [1000/2000] Migrating dependencies to relationships...
# [500/500] Migrating events to episodes...
# ✓ Migration complete
```

### Validate Migration

```bash
# Check migration status
bd migrate status

# Validate data integrity
bd migrate validate

# Output shows:
# ✓ All issues migrated (5000/5000)
# ✓ All dependencies migrated (2000/2000)
# ✓ No data loss (checksums match)
# ✓ Schema version: 8
```

### Rollback if Needed

```bash
# Rollback to v7 schema
bd migrate rollback

# Restores:
# - Issues table
# - Dependencies table
# - Events table
# - Schema version: 7
```

---

## Use Cases

### 1. Track a Feature Across Teams

```bash
# Create feature entity
bd entity create --entity-type feature --name "Payment integration" \
  --description "Stripe payment integration" --json
# Returns: payment-feature (id)

# Link to designers
bd relationship create --from payment-feature --type designed-by --to person-alice --json

# Link to engineers
bd relationship create --from payment-feature --type implemented-by --to person-bob --json

# Link to dependencies
bd relationship create --from payment-feature --type depends-on --to api-gateway --json

# Link to pull requests
bd relationship create --from payment-feature --type implemented-in --to pr-1234 --json

# Explore entire feature graph
bd graph explore payment-feature --depth 2 --json
```

### 2. Model a Bug Investigation

```bash
# Create bug entity
bd entity create --entity-type bug --name "Checkout crash" \
  --description "App crashes on checkout button" --json
# Returns: checkout-bug

# Link to root cause
bd entity create --entity-type issue --name "Race condition in payment flow" --json
# Returns: race-condition

bd relationship create --from checkout-bug --type caused-by --to race-condition --json

# Link to fix
bd entity create --entity-type pull-request --name "Fix race in payment" --json
# Returns: pr-1234

bd relationship create --from race-condition --type fixed-by --to pr-1234 --json

# Visualize investigation
bd graph visualize checkout-bug --format dot > investigation.dot
```

### 3. Track Temporal Relationships

```bash
# Alice works on feature this month
bd relationship create --from person-alice --type works-on --to feature-123 \
  --valid-from "2026-03-01" --valid-until "2026-03-31" --json

# Bob takes over next month
bd relationship create --from person-bob --type works-on --to feature-123 \
  --valid-from "2026-04-01" --json

# Query: Who worked on feature-123 on March 15?
bd relationship list --to feature-123 --type works-on --at "2026-03-15" --json
# Returns: person-alice

# Query: Who is working on feature-123 now?
bd relationship list --to feature-123 --type works-on --active --json
# Returns: person-bob
```

### 4. Model Code Relationships

```bash
# Register custom types
bd ontology register-entity-type --name "code-file" --schema code-file-schema.json
bd ontology register-relationship-type --name "imports" --schema imports-schema.json

# Create entities for code files
bd entity create --entity-type code-file --name "auth.ts" \
  --metadata '{"path": "src/auth.ts", "language": "typescript"}' --json

bd entity create --entity-type code-file --name "user.ts" \
  --metadata '{"path": "src/models/user.ts", "language": "typescript"}' --json

# Create import relationship
bd relationship create --from auth-ts --type imports --to user-ts \
  --metadata '{"line_number": 3}' --json

# Find all files that import user.ts
bd relationship list --to user-ts --type imports --json
```

---

## Compatibility Mode

During migration, both v7 (issues) and v8 (knowledge graph) can coexist:

```bash
# Check current mode
bd compat status

# Set to v7 mode (use old issue schema)
bd compat set v7

# Set to v8 mode (use new knowledge graph schema)
bd compat set v8
```

Legacy commands show deprecation warnings in v8 mode:

```bash
bd create "New task"
# ⚠ Warning: This command is deprecated. Use 'bd entity create' instead.
# ✓ Created: bd-123
```

---

## Current Status

**Implementation Status**: 🚧 **NOT YET IMPLEMENTED**

This document describes **planned features** from [Plan.md](Plan.md). Key details:

- **Complexity**: 6-8 weeks of development work
- **Risk Level**: HIGH (major architectural change)
- **Current Phase**: Planning (Phase 0)
- **Schema Version**: Currently v7 (issues), migrating to v8 (knowledge graph)

### Implementation Phases

1. **Phase 1**: Foundation - Type system & schema design
2. **Phase 2**: Storage interface extension
3. **Phase 3**: Dolt storage implementation
4. **Phase 4**: CLI command migration
5. **Phase 5**: Data migration tools & cutover
6. **Phase 6**: Tests, validation & documentation

### Tracking Implementation

To track when this gets implemented, create a beads issue:

```bash
# Using current beads (not the new system)
bd create "Implement knowledge graph migration" \
  --description="Track progress on Plan.md implementation" \
  -t epic -p 1 --json
```

---

## Questions?

- **Design Plan**: See [Plan.md](Plan.md)
- **Architecture**: See [docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md)
- **Current CLI**: See [docs/CLI_REFERENCE.md](../docs/CLI_REFERENCE.md)
- **Agent Instructions**: See [AGENT_INSTRUCTIONS.md](../AGENT_INSTRUCTIONS.md)

---

**Last Updated**: March 31, 2026
**Plan Version**: v1.0 (from Plan.md)
