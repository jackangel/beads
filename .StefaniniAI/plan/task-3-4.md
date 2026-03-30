# Task 3-4: Add --extract flag to episode create command

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: MODIFY
- **Risk Level**: LOW
- **Phase**: 3
- **Depends On**: 3-1

## Files
- `e:\Projects\BeadsMemory\beads\cmd\bd\episode_create.go` (EXISTING)

## Instructions
Add `--extract` flag to `bd episode create` for automatic post-creation extraction.

**Changes:**
```go
var autoExtract bool

func init() {
    episodeCreateCmd.Flags().BoolVar(&autoExtract, "extract", false, "Auto-extract entities after creation")
}

// In runEpisodeCreate, after store.CreateEpisode:
if autoExtract {
    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    if apiKey == "" {
        fmt.Fprintln(os.Stderr, "Warning: --extract requires ANTHROPIC_API_KEY, skipping extraction")
    } else {
        _, err := extraction.ExtractFromEpisode(ctx, store, episode.ID, apiKey)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Warning: extraction failed: %v\n", err)
        }
    }
}
```

## Validation Criteria
- [ ] `--extract` flag added
- [ ] Calls `extraction.ExtractFromEpisode` if flag set
- [ ] Warns if API key missing (doesn't fail)
- [ ] Warns if extraction fails (doesn't fail episode creation)
- [ ] No compilation errors

## Impact Analysis
- **Direct impact**: episode_create.go (optional auto-extraction)
- **Indirect impact**: Users can create + extract in one command
- **Dependencies**: Task 3-1 (extraction package)

## Context
- Pattern: Optional post-operation hooks

## User Feedback
*(Empty)*
