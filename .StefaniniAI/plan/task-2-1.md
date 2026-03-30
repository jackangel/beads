# Task 2-1: Add CLI flags for relationship confidence (create/update/list)

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: LOW
- **Phase**: 2
- **Depends On**: 1-2

## Files
- `e:\Projects\BeadsMemory\beads\cmd\bd\relationship_create.go` (EXISTING)
- `e:\Projects\BeadsMemory\beads\cmd\bd\relationship_update.go` (EXISTING)
- `e:\Projects\BeadsMemory\beads\cmd\bd\relationship_list.go` (EXISTING)

## Instructions

Add `--confidence` flag to relationship CLI commands to support confidence scoring from the command line.

**Outcome:** Users and AI agents can set/filter relationships by confidence via CLI.

**Changes required:**

**1. In `cmd/bd/relationship_create.go`:**
- Add flag variable: `var relationshipConfidence *float64`
- Add flag definition in `init()`:
  ```go
  relationshipCreateCmd.Flags().Float64Var(&relationshipConfidence, "confidence", 1.0, "Confidence score 0.0-1.0 (default 1.0)")
  ```
- Set confidence when creating relationship:
  ```go
  relationship := &types.Relationship{
      // ... existing fields ...
      Confidence: relationshipConfidence,
  }
  ```
- Add validation: if confidence provided, check range 0.0-1.0

**2. In `cmd/bd/relationship_update.go`:**
- Add flag variable: `var updateConfidence *float64` (use pointer to distinguish "not set" from 0.0)
- Add flag definition in `init()`:
  ```go
  relationshipUpdateCmd.Flags().Float64Var(&updateConfidence, "confidence", 0, "Update confidence score 0.0-1.0")
  ```
- Only update confidence if flag was set:
  ```go
  if cmd.Flags().Changed("confidence") {
      relationship.Confidence = updateConfidence
  }
  ```
- Add validation: if changed, check range 0.0-1.0

**3. In `cmd/bd/relationship_list.go`:**
- Add filter variables:
  ```go
  var minConfidence *float64
  var maxConfidence *float64
  ```
- Add flag definitions in `init()`:
  ```go
  relationshipListCmd.Flags().Float64Var(&minConfidence, "min-confidence", 0, "Minimum confidence threshold (0.0-1.0)")
  relationshipListCmd.Flags().Float64Var(&maxConfidence, "max-confidence", 0, "Maximum confidence threshold (0.0-1.0)")
  ```
- Apply filters when searching:
  ```go
  filters := storage.RelationshipFilters{
      // ... existing filters ...
      MinConfidence: minConfidence,
      MaxConfidence: maxConfidence,
  }
  ```
- Handle flag checking: only set filter if flag was changed:
  ```go
  if cmd.Flags().Changed("min-confidence") {
      filters.MinConfidence = &minConfidence
  }
  ```

**Validation pattern:**
```go
func validateConfidence(confidence *float64) error {
    if confidence != nil && (*confidence < 0.0 || *confidence > 1.0) {
        return fmt.Errorf("confidence must be between 0.0 and 1.0, got %.2f", *confidence)
    }
    return nil
}
```

## Architecture Pattern

**CLI Flag Pattern** (from existing commands):
- Use `cmd.Flags().Changed("flag-name")` to detect if user set flag
- Use pointer types for optional numeric flags (`*float64` not `float64`)
- Validate input before passing to storage layer
- Follow existing flag patterns from priority, status, etc.

**Confidence Defaults**:
- Create: default 1.0 (certain)
- Update: only update if flag changed
- List: no filter applied if flags not set

## Validation Criteria
- [ ] `--confidence` flag added to relationship_create.go (default 1.0)
- [ ] `--confidence` flag added to relationship_update.go (optional, only updates if changed)
- [ ] `--min-confidence` and `--max-confidence` flags added to relationship_list.go
- [ ] All confidence values validated (0.0-1.0 range)
- [ ] Flags only apply filters/updates when changed (use cmd.Flags().Changed)
- [ ] No compilation errors
- [ ] `bd relationship create --confidence 0.8 ...` works
- [ ] `bd relationship list --min-confidence 0.5 --json` works

## Impact Analysis
- **Direct impact**: 3 relationship CLI commands
- **Indirect impact**: MCP tools can now use confidence flags via task 1-8
- **Dependencies**: Task 1-2 provides Confidence field and storage filters

## Context
- Research Bundle: `e:\Projects\BeadsMemory\beads\.StefaniniAI\Research.md` (see "Feature 4: Relationship Confidence" for rationale)
- Existing pattern: `cmd/bd/create.go`, `cmd/bd/update.go`, `cmd/bd/list.go` (follow priority/status flag patterns)
- Confidence field: Task 1-2 adds Confidence to Relationship type

## User Feedback
*(Empty — the Orchestrator appends feedback here if the user requests a fix after reviewing this task's output. Re-read this section each time you are re-invoked for this task.)*
