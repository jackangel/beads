# Research Bundle

*Generated: 2026-03-16*
*Request: Replace Epic/Task/Sub-task hierarchy with a general memory system*

---

## Current Architecture Summary

- **Architecture Type**: Layered CLI Application with Versioned Database Backend
- **Primary Tech Stack**: Go, Dolt (MySQL-compatible versioned SQL), Cobra CLI
- **Key Components**:
  - CLI Layer (`cmd/bd/`): 100+ commands, including issue management
  - Storage Interface (`internal/storage/`): CRUD, dependencies, labels, comments
  - Dolt Database: 14+ tables, including `issues`, `dependencies`, `child_counters`
- **Relevant Tables**:
  - `issues`: Core work items with fields like `issue_type`, `parent_id`
  - `dependencies`: Tracks relationships (blocks, related, parent-child)
  - `child_counters`: Tracks hierarchical IDs (parent-child relationships)

---

## Issue Type System

### Enum and Validation
- **Defined in**: `internal/types/types.go`
- **Issue Types**:
  - `TypeEpic`: Represents large features with subtasks
  - `TypeTask`: Represents standard tasks
  - `TypeSubTask`: Represents subtasks under tasks
- **Validation**:
  - `issue_type` field in `issues` table defaults to `task`
  - Enum enforced in Go code (`types.IssueType`)

### CLI Commands
- **Relevant Commands**:
  - `bd create`: Creates issues with `--type` flag for `epic`, `task`, `sub-task`
  - `bd children`: Lists child issues of a parent
  - `bd update`: Updates issue relationships (e.g., `--parent`)
- **Implementation**:
  - `cmd/bd/children.go`: Implements `bd children` command
  - `cmd/bd/backup_restore_test.go`: Tests for issue type handling

---

## Relationship System

### Dependencies Table
- **Schema**:
  - `issue_id`: ID of the issue
  - `depends_on_id`: ID of the related issue
  - `type`: Relationship type (`blocks`, `related`, `parent-child`, `discovered-from`)
- **Parent-Child Relationships**:
  - Stored as `parent-child` in `dependencies` table
  - `child_counters` table tracks `parent_id` and `last_child`

### Enforcement
- **Code References**:
  - `cmd/bd/children.go`: Lists children of a parent
  - `cmd/bd/children_test.go`: Tests inclusion of closed children
  - `cmd/bd/backup_test.go`: Tests parent-child metadata

---

## Database Schema

### Issues Table
- **Fields**:
  - `id`: Primary key
  - `title`, `description`, `status`, `priority`
  - `issue_type`: Enum (`epic`, `task`, `sub-task`)
  - `parent_id`: Tracks parent issue

### Dependencies Table
- **Fields**:
  - `issue_id`, `depends_on_id`
  - `type`: Relationship type
  - `metadata`: JSON for additional data

### Child Counters Table
- **Fields**:
  - `parent_id`: ID of the parent issue
  - `last_child`: Tracks the last child ID

---

## CLI Commands

### Affected Commands
- `bd create`: Supports `--type` for issue types
- `bd children`: Lists child issues
- `bd update`: Updates parent-child relationships

### Implementation Details
- `cmd/bd/children.go`: Implements `bd children`
- `cmd/bd/children_test.go`: Tests for `bd children`
- `cmd/bd/backup_restore_test.go`: Tests for issue type handling

---

## Test Coverage

### Relevant Test Files
- `cmd/bd/children_test.go`: Tests parent-child listing
- `cmd/bd/backup_restore_test.go`: Tests issue type handling
- `cmd/bd/backup_test.go`: Tests parent-child metadata
- `cmd/bd/agent_routing_test.go`: Tests issue type routing

### Coverage Status
- **Parent-Child Relationships**: Well-covered
- **Issue Types**: Well-covered

---

## Migration Considerations

### Changes Needed
1. Replace `issue_type` enum with a generic `entity_type` field.
2. Replace `parent-child` relationships with a graph-based model.
3. Update CLI commands to support new entity/relationship types.
4. Migrate existing data to new schema.

### Potential Risks
- Breaking existing CLI workflows.
- Data migration complexity.
- Compatibility with existing tests.

---

## Additional Notes
- The current system heavily relies on `issue_type` and `parent-child` relationships.
- Transitioning to a graph-based model will require significant schema and code changes.
- Ensure backward compatibility or provide migration tools for users.