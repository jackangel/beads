# Task 1-11: Mark old issue-centric MCP tools as deprecated

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: LOW
- **Phase**: 1
- **Depends On**: None

## Files
- `e:\Projects\BeadsMemory\beads\integrations\beads-mcp\src\beads_mcp\tools.py` (EXISTING)

## Instructions

Add deprecation warnings to the 18 existing issue-centric MCP tools. They will continue to work, but AI assistants should be guided toward the new entity/relationship/episode tools.

**Outcome:** Backward compatibility is preserved, but users are gently nudged toward the v8 knowledge graph API.

**Changes required:**

For each of the 18 existing issue tools (`beads_ready_work`, `beads_list_issues`, `beads_show_issue`, `beads_create_issue`, `beads_update_issue`, `beads_claim_issue`, `beads_close_issue`, `beads_reopen_issue`, `beads_add_dependency`, `beads_quickstart`, `beads_stats`, `beads_blocked`, `beads_inspect_migration`, `beads_get_schema_info`, `beads_repair_deps`, `beads_detect_pollution`, `beads_validate`, `beads_init`):

1. **Add deprecation notice to docstring:**
   ```python
   """
   [Existing description]
   
   DEPRECATED: This tool uses the legacy v7 issue-tracking API. 
   Consider using the v8 knowledge graph API (entity/relationship/episode tools) for new work.
   Legacy tools will be maintained for backward compatibility.
   
   [Existing Args/Returns sections]
   """
   ```

2. **Optional: Add deprecation log (non-intrusive):**
   At the start of each tool function, after getting the client:
   ```python
   # Log deprecation warning (optional, for tracking migration)
   import warnings
   warnings.warn(
       "beads_[tool_name] is deprecated, use v8 knowledge graph tools instead",
       DeprecationWarning,
       stacklevel=2
   )
   ```

**Which tools to deprecate:**
- All tools in the current `tools.py` (18 total) EXCEPT:
  - `beads_quickstart` (still relevant for onboarding)
  - `beads_init` (still relevant for setup)
  - Doctor tools (`beads_inspect_migration`, `beads_repair_deps`, `beads_detect_pollution`, `beads_validate`) - these are diagnostic, not domain-specific

**Recommendation:** Only add docstring deprecation notices, skip the `warnings.warn` (too noisy). Deprecation is informational, not enforcement.

## Architecture Pattern

**Graceful Deprecation** (no breaking changes):
- Old tools continue to work unchanged
- Docstrings guide users toward new tools
- No runtime errors, no forced migrations
- Future major version (v9?) can remove deprecated tools

**Backward Compatibility**:
- v7 (issues) and v8 (entities) coexist indefinitely
- Users can mix issue and entity operations
- No data migration required (v7 data stays in issues table)

## Validation Criteria
- [ ] All 18 issue tools have deprecation notice in docstrings
- [ ] Deprecation text mentions "v7 issue-tracking API" and "v8 knowledge graph API"
- [ ] Deprecation text says "backward compatibility maintained"
- [ ] Exclude `beads_quickstart`, `beads_init`, and doctor tools from deprecation
- [ ] No functional changes (tools still work)
- [ ] No runtime warnings (warnings.warn optional and excluded)
- [ ] No syntax errors

## Impact Analysis
- **Direct impact**: MCP tool docstrings (informational only)
- **Indirect impact**: AI assistants see deprecation notices and prefer v8 tools
- **Dependencies**: None

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Feature 6: MCP Server Update" section for deprecation requirement)
- Philosophy: Graceful deprecation, not forced migration

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
