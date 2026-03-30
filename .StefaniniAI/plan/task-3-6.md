# Task 3-6: Add RetrieveMemory method to storage interface

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: HIGH
- **Phase**: 3
- **Depends On**: 3-5

## Files
- `e:\Projects\BeadsMemory\beads\internal\storage\storage.go` (EXISTING)
- `e:\Projects\BeadsMemory\beads\internal\storage\dolt\retrieval.go` (NEW)

## Instructions
Add `RetrieveMemory` method to the Storage interface and implement it in DoltStore by wrapping the retrieval package.

**Changes to `internal/storage/storage.go`:**
```go
// Add to Storage interface (after existing methods):
RetrieveMemory(ctx context.Context, query retrieval.MemoryQuery) (*retrieval.MemoryContext, error)
```

**New file: `internal/storage/dolt/retrieval.go`**
```go
package dolt

import (
    "context"
    "github.com/steveyegge/beads/internal/retrieval"
)

func (d *DoltStore) RetrieveMemory(ctx context.Context, query retrieval.MemoryQuery) (*retrieval.MemoryContext, error) {
    // Delegate to retrieval package
    return retrieval.RetrieveMemory(ctx, d, query)
}
```

**Note:** The retrieval package handles all logic. DoltStore just provides the interface wrapper.

## Validation Criteria
- [ ] `Storage` interface has `RetrieveMemory` method
- [ ] `DoltStore` implements method in `retrieval.go`
- [ ] Method delegates to `retrieval.RetrieveMemory`
- [ ] No compilation errors

## Impact Analysis
- **Direct impact**: Storage interface (HIGH risk: 100+ commands depend on it)
- **Indirect impact**: Task 3-7 (CLI) calls this method
- **Dependencies**: Task 3-5 (retrieval package)

## Context
- Pattern: Thin wrapper delegating to specialized package

## User Feedback
*(Empty)*
