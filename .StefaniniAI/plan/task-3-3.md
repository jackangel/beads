# Task 3-3: Add CLI command for batch episode extraction

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Risk Level**: MEDIUM
- **Phase**: 3
- **Depends On**: 3-1, 1-1

## Files
- `e:\Projects\BeadsMemory\beads\cmd\bd\episode_extract_all.go` (NEW)

## Instructions
Create `bd episode extract-all` command for batch processing unextracted episodes.

**Implementation pattern:**
```go
var (
    extractSince string
    extractLimit int
)

var episodeExtractAllCmd = &cobra.Command{
    Use:   "extract-all",
    Short: "Batch extract entities from unprocessed episodes",
    Long: `Process all episodes where extracted_at is NULL.
    
Optionally filter by episodes created after --since timestamp.`,
    RunE: runEpisodeExtractAll,
}

func init() {
    episodeExtractAllCmd.Flags().StringVar(&extractSince, "since", "", "Process episodes since timestamp (ISO 8601)")
    episodeExtractAllCmd.Flags().IntVar(&extractLimit, "limit", 10, "Maximum episodes to process")
    episodeCmd.AddCommand(episodeExtractAllCmd)
}

func runEpisodeExtractAll(cmd *cobra.Command, args []string) error {
    CheckReadonly("extract-all")
    
    // Fetch unprocessed episodes (extracted_at IS NULL)
    // Use SearchEpisodes with filter (TBD: add ExtractedAt filter)
    
    // For each episode:
    //   - Call extraction.ExtractFromEpisode
    //   - Create entities/relationships
    //   - Update extracted_at
    
    // Output: summary of processed episodes
}
```

## Validation Criteria
- [ ] Command created: `bd episode extract-all`
- [ ] `--since` and `--limit` flags work
- [ ] Filters episodes by NULL extracted_at
- [ ] Processes episodes in batch (respects limit)
- [ ] `--json` output includes counts per episode
- [ ] No compilation errors

## Impact Analysis
- **Direct impact**: New CLI command
- **Indirect impact**: Task 3-8 (MCP tool) wraps this
- **Dependencies**: Task 3-1 (extraction), Task 1-1 (extracted_at column)

## Context
- Pattern: Task 3-2 (episode extract)

## User Feedback
*(Empty)*
