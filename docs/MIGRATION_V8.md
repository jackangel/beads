# Database v8 Migration Guide

This guide walks you through migrating your beads database from v7 (Epic/Task hierarchy) to v8 (Knowledge Graph entity-relationship model).

## Quick Navigation

- [Overview](#overview)
- [Pre-Migration Checklist](#pre-migration-checklist)
- [Migration Steps](#migration-steps)
- [Rollback Procedures](#rollback-procedures)
- [Post-Migration Verification](#post-migration-verification)
- [Common Errors and Solutions](#common-errors-and-solutions)
- [FAQ](#faq)

## Overview

### What is v8 Migration?

The v8 migration transforms beads from a hierarchical issue tracking system (Epic → Task → Sub-task) into a flexible Knowledge Graph based on entities and relationships. This enables:

- **Flexible entity types**: Not limited to "epic", "task", "sub-task"
- **Rich relationship modeling**: Beyond parent-child (e.g., "implements", "tests", "documents")
- **Generalized memory system**: Store any type of work artifact with custom relationships
- **Future extensibility**: Add new entity types and relationship types without schema changes

### Epic/Task Model (v7) vs. Knowledge Graph (v8)

**v7 (Before):**
```
Epic (bd-1)
  ├── Task (bd-2)
  │   └── Sub-task (bd-3)
  └── Task (bd-4)
```
- Rigid 3-level hierarchy
- Fixed issue types: `epic`, `task`, `sub-task`
- Parent-child relationships only

**v8 (After):**
```
Feature (bd-1) ──implements──> Spec (bd-5)
  │                               │
  ├──contains──> Task (bd-2)     │
  │   └──tests──> Test (bd-3)    │
  │                               │
  └──contains──> Task (bd-4)     │
                  └──documents──> Doc (bd-6)
```
- Flexible entity graph
- Custom entity types and relationships
- Multi-dimensional relationships

### Why Migrate?

**Benefits of v8:**
- ✅ **Model complex relationships**: Link design docs, tests, implementations
- ✅ **Custom entity types**: Create domain-specific entities beyond "task"
- ✅ **Rich queries**: Find all tests for a feature, all docs for a component
- ✅ **Future-proof**: Extensible without breaking schema changes
- ✅ **Backward compatible**: Old issue types still work, mapped to entities

**Considerations:**
- ⚠️ **Learning curve**: New commands (`bd entity`, `bd relationship`)
- ⚠️ **Migration time**: Proportional to database size (typically < 1 minute)
- ⚠️ **Disk space**: Requires 2x current database size during migration

### Who Should Migrate?

**Migrate now if:**
- 🟢 You need to model complex relationships beyond parent-child
- 🟢 You want to track non-issue entities (specs, docs, tests)
- 🟢 You're starting a new project with beads
- 🟢 Your team is ready to adopt the new model

**Wait to migrate if:**
- 🔴 Your project is in production and stability is critical
- 🔴 Your team is not ready for new workflows
- 🔴 You only use simple task lists (v7 is sufficient)

**Migration is optional.** v7 will continue to be supported with deprecation warnings.

---

## Pre-Migration Checklist

Complete these steps **before** starting migration to ensure a safe process:

### 1. Backup Your Database

**CRITICAL:** Always backup before migration.

```bash
# Option 1: Dolt commit + push (recommended)
bd dolt commit -m "Pre-v8 migration backup"
bd dolt push

# Option 2: JSONL export backup
bd backup export backup-pre-v8.jsonl

# Option 3: Filesystem copy
cp -r .beads/dolt .beads/dolt.backup-$(date +%Y%m%d-%H%M%S)
```

**Verify backup:**
```bash
# Check Dolt commit history
bd dolt log --oneline | head -5

# Check JSONL export (if used)
wc -l backup-pre-v8.jsonl
```

### 2. Check Current Status

```bash
# Check database schema version
bd migrate status

# Example output:
# Current schema version: 7
# Latest available version: 8
# Migration status: ready
```

### 3. Validate Data Integrity

```bash
# Run pre-migration validation
bd migrate validate

# Example output:
# ✓ All issues have valid types
# ✓ All dependencies are valid
# ✓ No orphaned relationships
# ✓ Child counters consistent
# Ready for migration
```

### 4. Check Disk Space

The migration requires temporary space for schema transformation.

```bash
# Check current database size
du -sh .beads/dolt/

# Ensure you have 2x available space
df -h .beads/dolt/
```

**Rule of thumb:** If `.beads/dolt/` is 500MB, ensure 1GB free space.

### 5. Ensure No Concurrent Writers

**CRITICAL:** Stop all other beads processes before migration.

```bash
# Check for running Dolt server
bd info --json | grep -i server

# If server is running, stop it:
bd server stop

# Ensure no other bd commands are running
ps aux | grep 'bd ' | grep -v grep
```

### 6. Review Breaking Changes

- `bd children` command is deprecated → Use `bd relationship list --type contains`
- Issue type field (`epic`/`task`/`sub-task`) mapped to entity type
- Parent-child dependencies become `contains` relationships

---

## Migration Steps

Follow these steps in order. Each step is reversible until Step 5.

### Step 1: Check Current Status

Verify you're ready to migrate:

```bash
bd migrate status
```

**Expected output:**
```
Current schema version: 7
Latest available version: 8
Migration status: ready
Estimated migration time: 30 seconds
Estimated disk usage: +250MB temporary
```

**If status is not "ready":**
- Ensure you've completed the [Pre-Migration Checklist](#pre-migration-checklist)
- Check for error messages and resolve them first

### Step 2: Dry-Run Migration (Preview Changes)

Preview what will change without modifying data:

```bash
bd migrate to-v8 --dry-run
```

**Expected output:**
```
[DRY RUN] Migration plan for v7 → v8:

Entities to create:
  - 150 issues → entities (types: epic, task, sub-task)
  
Relationships to create:
  - 45 parent-child dependencies → "contains" relationships
  - 30 blocks dependencies → "blocks" relationships
  - 25 related dependencies → "related" relationships
  
Tables to create:
  - entities
  - relationships
  - entity_types
  - relationship_types
  
Tables to preserve:
  - issues (for backward compatibility)
  - dependencies (for backward compatibility)
  - labels, comments, events (unchanged)
  
Estimated time: 30 seconds
Disk space required: +250MB (temporary)

[DRY RUN] No changes made.
```

**Review the output carefully:**
- Verify entity/relationship counts match expectations
- Check for any warnings or notes
- Ensure disk space is sufficient

### Step 3: Run Migration

Execute the migration:

```bash
bd migrate to-v8
```

**Expected output:**
```
Starting migration v7 → v8...

[1/6] Creating backup checkpoint...
  ✓ Checkpoint created: dolt_migration_v8_20260316_143022

[2/6] Creating entity schema...
  ✓ Created table: entities
  ✓ Created table: relationships
  ✓ Created table: entity_types
  ✓ Created table: relationship_types

[3/6] Migrating issues to entities...
  ✓ Migrated 150 issues → entities
  
[4/6] Migrating dependencies to relationships...
  ✓ Migrated 45 parent-child → contains
  ✓ Migrated 30 blocks → blocks
  ✓ Migrated 25 related → related

[5/6] Validating migration...
  ✓ Entity count: 150 (matches issue count)
  ✓ Relationship count: 100 (matches dependency count)
  ✓ Data integrity: OK

[6/6] Updating schema version...
  ✓ Schema version: 7 → 8

Migration completed successfully in 28 seconds.

Next steps:
  1. Run: bd migrate validate
  2. Run: bd compat set v8  
  3. Test entity/relationship commands
```

**If migration fails:** See [Common Errors and Solutions](#common-errors-and-solutions)

### Step 4: Validate Migration

Verify data integrity after migration:

```bash
bd migrate validate
```

**Expected output:**
```
Validating v8 migration...

Entity validation:
  ✓ Entity count: 150
  ✓ All entities have valid types
  ✓ All entities have source issues
  
Relationship validation:
  ✓ Relationship count: 100
  ✓ All relationships have valid entities
  ✓ No orphaned relationships
  
Data integrity:
  ✓ Issue data preserved
  ✓ Dependency data preserved
  ✓ Labels preserved
  ✓ Comments preserved
  
Backward compatibility:
  ✓ Legacy commands still work (with deprecation warnings)
  
Migration validation: PASSED
```

**If validation fails:** Use rollback procedure in Step 7

### Step 5: Switch to v8 Compatibility Mode

Enable v8 entity/relationship commands:

```bash
bd compat set v8
```

**Expected output:**
```
Compatibility mode set to: v8

Enabled commands:
  - bd entity list
  - bd entity create
  - bd entity update
  - bd relationship list
  - bd relationship create
  - bd relationship delete
  
Legacy commands (with deprecation warnings):
  - bd create (use: bd entity create)
  - bd children (use: bd relationship list --type contains)
  
Configuration saved to: .beads/config.toml
```

### Step 6: Verify Functionality

Test both new and legacy commands:

```bash
# Test entity commands
bd entity list --json | head -5
bd entity show bd-1 --json

# Test relationship commands  
bd relationship list --json | head -5
bd relationship list --source bd-1 --json

# Test legacy commands (should work with warnings)
bd list --json | head -5  
bd show bd-1 --json

# Test creating new entities
bd entity create "New feature" --type task --description "Test v8 creation"

# Test creating relationships
bd relationship create bd-1 implements bd-2
```

**All commands should work.** Legacy commands show deprecation warnings.

### Step 7: Update Your Workflows

Update scripts and documentation to use v8 commands:

**Old (v7):**
```bash
bd create "Task" --type task --parent bd-1
bd children bd-1
```

**New (v8):**
```bash
bd entity create "Task" --type task
bd relationship create bd-1 contains bd-2
bd relationship list --source bd-1 --type contains
```

See [CLI_REFERENCE.md](CLI_REFERENCE.md) for complete v8 command reference.

---

## Rollback Procedures

If migration fails or you need to revert to v7:

### Automatic Rollback (Recommended)

If migration fails mid-process, it auto-rolls back:

```bash
# Migration failure triggers automatic rollback
bd migrate to-v8
# Error: Migration failed at step 4/6
# Auto-rolling back to checkpoint...
# ✓ Rolled back to: dolt_migration_v8_20260316_143022
# Database restored to v7
```

### Manual Rollback

If you completed migration but want to revert:

```bash
# Step 1: Roll back to v7 schema
bd migrate rollback

# Expected output:
# Rolling back migration v8 → v7...
# [1/4] Restoring from checkpoint...
# [2/4] Dropping v8 tables...
# [3/4] Updating schema version...
# [4/4] Validating rollback...
# ✓ Rollback completed successfully
```

```bash
# Step 2: Verify rollback
bd migrate status

# Expected output:
# Current schema version: 7
# Latest available version: 8
# Migration status: ready (not migrated)
```

```bash
# Step 3: Switch back to v7 mode
bd compat set v7

# Expected output:
# Compatibility mode set to: v7
# Using legacy issue commands
```

### Manual Checkpoint Restore (Last Resort)

If automated rollback fails:

```bash
# List available checkpoints
bd dolt log --oneline | grep checkpoint

# Restore specific checkpoint
bd dolt checkout dolt_migration_v8_20260316_143022

# Reset to checkpoint
bd dolt reset --hard

# Verify restoration
bd migrate status
```

### JSONL Backup Restore (Nuclear Option)

If all else fails, restore from JSONL backup:

```bash
# Clear current database (DESTRUCTIVE)
rm -rf .beads/dolt/

# Re-initialize
bd init

# Restore from backup
bd backup restore backup-pre-v8.jsonl

# Verify restoration
bd list --json | wc -l
```

---

## Post-Migration Verification

After migration, verify everything works correctly:

### 1. Row Count Checks

Verify all data was migrated:

```bash
# Count entities (should match issue count)
bd entity list --json | wc -l

# Count relationships (should match dependency count)  
bd relationship list --json | wc -l

# Compare to legacy tables
bd dolt sql -q "SELECT COUNT(*) FROM issues"
bd dolt sql -q "SELECT COUNT(*) FROM dependencies"
```

**Expected:** Entity count = Issue count, Relationship count = Dependency count

### 2. Sample Data Validation

Spot-check specific issues:

```bash
# Check a known issue exists as entity
bd entity show bd-1 --json

# Check its relationships
bd relationship list --source bd-1 --json

# Compare to legacy data
bd show bd-1 --json
```

**Expected:** Entity data matches issue data

### 3. Test Entity Commands

Verify full CRUD operations:

```bash
# Create new entity
bd entity create "Post-migration test" --type task

# Update entity
bd entity update bd-150 --description "Updated post-migration"

# Create relationship
bd relationship create bd-150 tests bd-1

# Query relationships
bd relationship list --source bd-150

# Delete relationship
bd relationship delete bd-150 tests bd-1

# Close entity
bd entity close bd-150 --reason "Test complete"
```

**Expected:** All operations succeed without errors

### 4. Test Legacy Commands

Verify backward compatibility:

```bash
# Legacy list (should still work)
bd list --status open --json

# Legacy show (should still work)
bd show bd-1 --json

# Legacy create (should show deprecation warning)
bd create "Legacy test" --type task
# Warning: 'bd create' is deprecated. Use 'bd entity create' instead.

# Legacy children (should show deprecation warning)
bd children bd-1
# Warning: 'bd children' is deprecated. Use 'bd relationship list --type contains' instead.
```

**Expected:** Commands work with deprecation warnings

### 5. Test Filtering and Queries

Verify complex queries work:

```bash
# Filter by entity type
bd entity list --type task --json

# Filter by relationship type
bd relationship list --type contains --json

# Combined filters
bd entity list --type task --status open --priority 1 --json

# Search relationships
bd relationship list --source bd-1 --target bd-5 --json
```

**Expected:** Filters return correct results

### 6. Performance Check

Verify queries remain fast:

```bash
# Benchmark ready query
time bd ready --json

# Benchmark entity list
time bd entity list --json

# Benchmark relationship list
time bd relationship list --json
```

**Expected:** Queries complete in < 1 second for typical databases

---

## Common Errors and Solutions

### Error: "Migration failed: row count mismatch"

**Symptom:**
```
[5/6] Validating migration...
  ✗ Entity count: 148 (expected: 150)
  Error: Migration failed: row count mismatch
```

**Cause:** Concurrent writes during migration or corrupted data

**Solution:**
```bash
# 1. Automatic rollback will trigger
# (wait for it to complete)

# 2. Stop all other bd processes
ps aux | grep 'bd ' | grep -v grep

# 3. Validate data integrity
bd migrate validate

# 4. Retry migration
bd migrate to-v8
```

### Error: "Schema version already 8"

**Symptom:**
```
bd migrate to-v8
Error: Schema version already 8. Database already migrated.
```

**Cause:** Database was already migrated

**Solution:**
```bash
# Check current status
bd migrate status

# If you want to force re-migration:
bd migrate rollback
bd migrate to-v8

# If migration is correct:
# (No action needed, already on v8)
```

### Error: "Not enough disk space"

**Symptom:**
```
[2/6] Creating entity schema...
Error: Not enough disk space (need 500MB, have 200MB)
```

**Cause:** Insufficient disk space for temporary tables

**Solution:**
```bash
# Option 1: Free up space
df -h .beads/dolt/
# Delete old backups, temporary files, etc.

# Option 2: Use external storage
mv .beads/dolt /path/to/larger/disk/dolt
ln -s /path/to/larger/disk/dolt .beads/dolt

# Option 3: Compress Dolt database
bd dolt gc

# Retry migration
bd migrate to-v8
```

### Error: "Migration timeout"

**Symptom:**
```
[3/6] Migrating issues to entities...
Error: Operation timed out after 300 seconds
```

**Cause:** Very large database (> 100k issues) or slow disk

**Solution:**
```bash
# Increase timeout
bd migrate to-v8 --timeout 1800  # 30 minutes

# Or migrate in steps (if supported):
bd migrate to-v8 --batch-size 1000
```

### Error: "Checkpoint restore failed"

**Symptom:**
```
bd migrate rollback
[1/4] Restoring from checkpoint...
Error: Checkpoint not found: dolt_migration_v8_...
```

**Cause:** Checkpoint was deleted or corrupted

**Solution:**
```bash
# Option 1: Restore from Dolt history
bd dolt log --oneline
bd dolt checkout <commit-before-migration>
bd dolt reset --hard

# Option 2: Restore from JSONL backup
bd backup restore backup-pre-v8.jsonl

# Option 3: Pull from remote (if available)
bd dolt pull
```

### Error: "Invalid dependency type"

**Symptom:**
```
[4/6] Migrating dependencies to relationships...
Error: Unknown dependency type: 'custom-type'
```

**Cause:** Custom dependency types not yet supported in v8

**Solution:**
```bash
# Option 1: Rollback and clean up custom types
bd migrate rollback

# Find custom dependency types
bd dolt sql -q "SELECT DISTINCT type FROM dependencies"

# Remove or convert custom types to standard ones
bd dolt sql -q "UPDATE dependencies SET type='related' WHERE type='custom-type'"

# Retry migration
bd migrate to-v8

# Option 2: Contact support for custom type migration
# See: https://github.com/your-org/beads/issues
```

### Warning: "Legacy command deprecated"

**Symptom:**
```
bd create "Task" --type task
Warning: 'bd create' is deprecated. Use 'bd entity create' instead.
[Issue created successfully]
```

**Cause:** Using v7 commands in v8 mode

**Solution:**
```bash
# Update to v8 commands:
# Old: bd create "Task" --type task
# New: bd entity create "Task" --type task

# Old: bd children bd-1
# New: bd relationship list --source bd-1 --type contains

# See CLI_REFERENCE.md for complete command mapping
```

---

## FAQ

### General Questions

**Q: Will my old issues be deleted?**

A: No. Both v7 and v8 schemas coexist. The `issues` table is preserved for backward compatibility, and a new `entities` table is created. You can query both.

**Q: Can I roll back after switching to v8?**

A: Yes. Use `bd migrate rollback` to revert to v7 schema. However, any entities/relationships created **after** migration will be lost.

**Q: Do I need to re-clone from remote?**

A: No. Migration is local and commits to your Dolt history. Use `bd dolt push` to share migrated database with collaborators.

**Q: What happens to my labels and comments?**

A: Labels and comments are preserved unchanged. They work with both v7 (issues) and v8 (entities).

**Q: How long does migration take?**

A: Typical migration times:
- 100 issues: ~10 seconds
- 1,000 issues: ~30 seconds
- 10,000 issues: ~3 minutes
- 100,000 issues: ~30 minutes

**Q: Can I migrate incrementally?**

A: No. Migration is atomic (all-or-nothing). This ensures data consistency.

### Compatibility Questions

**Q: Do old commands still work after migration?**

A: Yes. Old commands like `bd create`, `bd list`, `bd show` still work but show deprecation warnings. They operate on the `issues` table.

**Q: Can I use v7 and v8 commands together?**

A: Yes. You can mix commands, but it's recommended to adopt v8 commands fully for consistency.

**Q: What happens if a collaborator hasn't migrated?**

A: They'll see the old v7 schema until they run `bd migrate to-v8` locally. Dolt will merge both schemas correctly, but v8 entities won't be visible to v7 users.

**Q: Can I use v8 entities in v7 mode?**

A: No. You must set `bd compat set v8` to use entity/relationship commands.

### Workflow Questions

**Q: How do I create parent-child relationships in v8?**

A:
```bash
# v7 (deprecated):
bd create "Child task" --parent bd-1

# v8 (recommended):
bd entity create "Child task" --type task
bd relationship create bd-1 contains bd-2
```

**Q: How do I list children in v8?**

A:
```bash
# v7 (deprecated):
bd children bd-1

# v8 (recommended):
bd relationship list --source bd-1 --type contains
```

**Q: Can I create custom entity types?**

A: Yes! v8 is extensible:
```bash
bd entity create "Design doc" --type document
bd entity create "Test suite" --type test
bd relationship create bd-1 documents bd-2
bd relationship create bd-3 tests bd-1
```

**Q: Do epics still exist in v8?**

A: Yes. "Epic" is now an entity type. Old epics are automatically migrated to entities with `type=epic`.

### Troubleshooting Questions

**Q: Migration is taking too long. Is it stuck?**

A: Check progress:
```bash
# In another terminal:
bd dolt sql -q "SELECT COUNT(*) FROM entities"

# If count is increasing, migration is progressing
# If count is static for > 5 minutes, migration may be stuck
```

**Q: Can I cancel migration mid-process?**

A: Not recommended. If you must:
```bash
# Ctrl+C will trigger automatic rollback
# Wait for rollback to complete before running bd commands
```

**Q: Migration succeeded but queries are slow. Why?**

A: Indexes may need rebuilding:
```bash
# Rebuild indexes
bd dolt sql -q "ANALYZE TABLE entities"
bd dolt sql -q "ANALYZE TABLE relationships"

# Compact database
bd dolt gc
```

**Q: I get "foreign key constraint" errors after migration. What now?**

A: Validate data integrity:
```bash
bd migrate validate

# If validation fails, rollback and fix issues:
bd migrate rollback
bd dolt sql -q "SELECT * FROM issues WHERE parent_id NOT IN (SELECT id FROM issues)"
# (fix orphaned references)

# Retry migration
bd migrate to-v8
```

### Remote Sync Questions

**Q: Can I push v8 database to remote?**

A: Yes:
```bash
bd dolt push
```

**Q: What happens if remote is still v7?**

A: Dolt will merge schemas. v8 tables (`entities`, `relationships`) will be added to remote. Remote users need to migrate locally to access v8 features.

**Q: Can I pull updates from v7 remote after migrating?**

A: Yes. Dolt merges both schemas. New v7 issues will be visible in your v8 database via backward compatibility layer.

---

## Additional Resources

- [CLI Reference](CLI_REFERENCE.md) - Complete v8 command reference
- [Architecture](ARCHITECTURE.md) - v8 entity-relationship model architecture
- [Advanced Guide](ADVANCED.md) - Custom entity types and complex queries
- [Troubleshooting](TROUBLESHOOTING.md) - Detailed error diagnosis

**Need help?** Open an issue: https://github.com/your-org/beads/issues

---

**Migration checklist recap:**

- [ ] Backup database (`bd dolt commit && bd dolt push`)
- [ ] Run `bd migrate status` (verify ready)
- [ ] Run `bd migrate to-v8 --dry-run` (preview)
- [ ] Run `bd migrate to-v8` (execute)
- [ ] Run `bd migrate validate` (verify)
- [ ] Run `bd compat set v8` (enable v8 mode)
- [ ] Test entity/relationship commands
- [ ] Update workflows and scripts
- [ ] Push to remote (`bd dolt push`)
- [ ] Notify team to migrate

**Welcome to beads v8!** 🎉
