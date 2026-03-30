# Task 3-1: Create LLM extraction package with Anthropic SDK integration

## Assignment
- **Agent**: EditMode_Coder
- **Operation**: CREATE
- **Risk Level**: MEDIUM
- **Phase**: 3
- **Depends On**: None (uses existing anthropic-sdk-go dependency)

## Files
- `e:\Projects\BeadsMemory\beads\internal\extraction\llm.go` (NEW)
- `e:\Projects\BeadsMemory\beads\internal\extraction\llm_test.go` (NEW)

## Instructions
Create an LLM-powered extraction package that processes episode raw data and extracts entities + relationships using Anthropic Claude.

**Outcome:** Episodes (conversation logs, meeting transcripts, documents) can be processed to automatically extract structured knowledge graph data.

**File: `internal/extraction/llm.go`**
```go
package extraction

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
    "github.com/steveyegge/beads/internal/storage"
    "github.com/steveyegge/beads/internal/types"
)

// ExtractionResult holds entities and relationships extracted from an episode.
type ExtractionResult struct {
    Entities      []*types.Entity      `json:"entities"`
    Relationships []*types.Relationship `json:"relationships"`
    RawResponse   string               `json:"raw_response"`
}

// ExtractFromEpisode processes an episode's raw data and extracts entities + relationships using Claude.
func ExtractFromEpisode(ctx context.Context, store storage.Storage, episodeID, apiKey string) (*ExtractionResult, error) {
    // Fetch episode
    episode, err := store.GetEpisode(ctx, episodeID)
    if err != nil {
        return nil, fmt.Errorf("fetching episode: %w", err)
    }
    
    // Prepare prompt
    prompt := buildExtractionPrompt(string(episode.RawData))
    
    // Call Claude
    client := anthropic.NewClient(option.WithAPIKey(apiKey))
    msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     anthropic.F(anthropic.ModelClaude_3_5_Sonnet_20241022),
        MaxTokens: anthropic.F(int64(4096)),
        Messages: anthropic.F([]anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
        }),
    })
    if err != nil {
        return nil, fmt.Errorf("calling Claude API: %w", err)
    }
    
    responseText := msg.Content[0].AsText().Text
    
    // Parse JSON response
    var result ExtractionResult
    if err := json.Unmarshal([]byte(responseText), &result); err != nil {
        return nil, fmt.Errorf("parsing extraction result: %w (response: %s)", err, responseText)
    }
    
    result.RawResponse = responseText
    return &result, nil
}

func buildExtractionPrompt(rawData string) string {
    return fmt.Sprintf(`Extract entities and relationships from the following text.

Return a JSON object with:
- "entities": array of {entity_type, name, summary}
- "relationships": array of {source_entity_name, relationship_type, target_entity_name, confidence}

Text:
%s

Return ONLY valid JSON, no markdown formatting.`, rawData)
}
```

**File: `internal/extraction/llm_test.go`**
```go
package extraction

import (
    "testing"
)

func TestBuildExtractionPrompt(t *testing.T) {
    prompt := buildExtractionPrompt("Alice leads Team X")
    if len(prompt) == 0 {
        t.Error("Expected prompt, got empty string")
    }
}

// Integration test with real API: skip in CI (requires API key)
func TestExtractFromEpisode_Integration(t *testing.T) {
    t.Skip("Integration test: requires live API and database")
}
```

**Extraction Workflow:**
1. Fetch episode raw data (BLOB)
2. Build extraction prompt
3. Call Claude API with prompt
4. Parse JSON response (entities + relationships)
5. Create entities in storage
6. Create relationships in storage (link by name→ID resolution)
7. Update episode.extracted_at timestamp

**Edge cases:**
- Handle API rate limits (use backoff/retry from cenkalti/backoff)
- Handle malformed JSON responses (retry with clarifying prompt)
- Handle entity name ambiguity (first pass: create duplicates, user merges later)

## Architecture Pattern
**LLM Integration Pattern** (from find_duplicates.go):
- Use anthropic-sdk-go client
- Structured prompts with clear JSON schema
- Parse responses with error handling
- Store raw response for debugging

**API Key Management**:
- Accept API key as parameter (caller reads from env or config)
- Use `option.WithAPIKey(apiKey)` for client initialization

## Validation Criteria
- [ ] `internal/extraction/llm.go` created with `ExtractFromEpisode` function
- [ ] Uses anthropic-sdk-go (Claude 3.5 Sonnet)
- [ ] Prompt instructs Claude to return JSON with entities + relationships
- [ ] Parses JSON response into `ExtractionResult`
- [ ] Basic test validates prompt building
- [ ] No compilation errors
- [ ] Integration test skipped (requires API key)

## Impact Analysis
- **Direct impact**: New extraction package
- **Indirect impact**: Tasks 3-2, 3-3, 3-4 use this for episode processing
- **Dependencies**: anthropic-sdk-go (already in go.mod)

## Context
- Research Bundle: `cmd/bd/find_duplicates.go` (Claude API pattern)
- API: Anthropic SDK (github.com/anthropics/anthropic-sdk-go)

## User Feedback
*(Empty)*
