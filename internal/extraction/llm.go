package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
)

// ExtractionConfig holds configuration for the LLM-powered extractor.
type ExtractionConfig struct {
	APIKey    string // Anthropic API key
	Model     string // Claude model to use (default: "claude-3-5-haiku-latest")
	MaxTokens int    // Maximum tokens for response (default: 4096)
}

// ExtractedEntity represents a simplified entity extracted from text.
// This is an intermediate format before creating a full types.Entity.
type ExtractedEntity struct {
	Name       string                 `json:"name"`
	EntityType string                 `json:"type"`
	Summary    string                 `json:"summary"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ExtractedRelationship represents a relationship extracted from text.
// This is an intermediate format before creating a full types.Relationship.
type ExtractedRelationship struct {
	SourceName       string  `json:"source"`       // Source entity name
	RelationshipType string  `json:"type"`         // Relationship type
	TargetName       string  `json:"target"`       // Target entity name
	Confidence       float64 `json:"confidence"`   // Confidence score (0.0-1.0)
}

// ExtractionResult holds the complete extraction output from Claude.
type ExtractionResult struct {
	Entities      []ExtractedEntity      `json:"entities"`
	Relationships []ExtractedRelationship `json:"relationships"`
	RawResponse   string                  `json:"raw_response"` // Full LLM response for debugging
}

// Extractor wraps the Anthropic client for entity extraction.
type Extractor struct {
	config ExtractionConfig
	client anthropic.Client
}

// NewExtractor creates a new LLM-powered extractor with the given configuration.
func NewExtractor(config ExtractionConfig) *Extractor {
	// Set defaults
	if config.Model == "" {
		config.Model = "claude-3-5-haiku-latest"
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}

	client := anthropic.NewClient(option.WithAPIKey(config.APIKey))

	return &Extractor{
		config: config,
		client: client,
	}
}

// ExtractFromText sends text to Claude and extracts entities and relationships.
// The LLM returns structured JSON that is parsed into an ExtractionResult.
//
// Example usage:
//
//	extractor := NewExtractor(ExtractionConfig{APIKey: "sk-..."})
//	result, err := extractor.ExtractFromText(ctx, "Alice works on Project X...")
func (e *Extractor) ExtractFromText(ctx context.Context, text string) (*ExtractionResult, error) {
	if strings.TrimSpace(text) == "" {
		return &ExtractionResult{
			Entities:      []ExtractedEntity{},
			Relationships: []ExtractedRelationship{},
			RawResponse:   "",
		}, nil
	}

	prompt := buildExtractionPrompt(text)

	// Call Anthropic API (following pattern from find_duplicates.go)
	msg, err := e.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(e.config.Model),
		MaxTokens: int64(e.config.MaxTokens),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("calling Claude API: %w", err)
	}

	// Extract text from response
	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("empty response from Claude")
	}

	responseText := msg.Content[0].Text

	// Parse JSON response
	var result ExtractionResult
	if err := parseExtractionResponse(responseText, &result); err != nil {
		return nil, fmt.Errorf("parsing extraction result: %w", err)
	}

	result.RawResponse = responseText
	return &result, nil
}

// buildExtractionPrompt creates the system prompt for entity extraction.
// The prompt instructs Claude to return structured JSON with entities and relationships.
func buildExtractionPrompt(text string) string {
	return fmt.Sprintf(`Extract entities and relationships from the following text.

Return a JSON object with this exact structure:
{
  "entities": [
    {
      "name": "Entity Name",
      "type": "entity_type",
      "summary": "Brief description of the entity"
    }
  ],
  "relationships": [
    {
      "source": "Source Entity Name",
      "type": "relationship_type",
      "target": "Target Entity Name",
      "confidence": 0.9
    }
  ]
}

Guidelines:
- Entity types: person, organization, product, concept, technology, location, event, document
- Relationship types: works_on, leads, uses, implements, depends_on, reports_to, owns, mentions, participates_in
- Confidence should be between 0.0 (uncertain) and 1.0 (certain)
- Use entity names consistently across entities and relationships
- Extract only entities explicitly mentioned in the text
- Return ONLY valid JSON, no markdown formatting, no code blocks, no explanations

Text:
%s`, text)
}

// ExtractFromEpisode fetches an episode by ID and extracts entities/relationships from it.
// This is a convenience wrapper around ExtractFromText for episode-based extraction.
//
// Parameters:
//   - ctx: Context for cancellation
//   - store: Storage interface providing episode access
//   - episodeID: ID of the episode to extract from
//   - apiKey: Anthropic API key for LLM calls
//
// Returns:
//   - ExtractionResult with extracted entities and relationships
//   - error if episode not found, API call fails, or parsing fails
func ExtractFromEpisode(ctx context.Context, store storage.Storage, episodeID string, apiKey string) (*ExtractionResult, error) {
	// Fetch episode
	episode, err := store.GetEpisode(ctx, episodeID)
	if err != nil {
		return nil, fmt.Errorf("fetching episode %s: %w", episodeID, err)
	}

	// Check if raw data exists
	if len(episode.RawData) == 0 {
		return &ExtractionResult{
			Entities:      []ExtractedEntity{},
			Relationships: []ExtractedRelationship{},
			RawResponse:   "",
		}, nil
	}

	// Convert raw data to text (assume UTF-8 encoded text)
	text := string(episode.RawData)

	// Create extractor and run extraction
	extractor := NewExtractor(ExtractionConfig{APIKey: apiKey})
	result, err := extractor.ExtractFromText(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("extracting from episode %s: %w", episodeID, err)
	}

	return result, nil
}

// ConvertToTypesEntity converts an ExtractedEntity to a full types.Entity.
// This helper is used by CLI commands to convert extraction results to storage-ready entities.
func ConvertToTypesEntity(extracted ExtractedEntity, createdBy string) *types.Entity {
	return &types.Entity{
		EntityType: extracted.EntityType,
		Name:       extracted.Name,
		Summary:    extracted.Summary,
		Metadata:   extracted.Metadata,
		CreatedBy:  createdBy,
		UpdatedBy:  createdBy,
	}
}

// ConvertToTypesRelationship converts an ExtractedRelationship to a full types.Relationship.
// This helper is used by CLI commands to convert extraction results to storage-ready relationships.
//
// Note: Caller must resolve entity names to IDs before using this function, as the storage layer
// requires entity IDs, not names.
func ConvertToTypesRelationship(extracted ExtractedRelationship, sourceID, targetID, createdBy string) *types.Relationship {
	confidence := extracted.Confidence
	return &types.Relationship{
		SourceEntityID:   sourceID,
		RelationshipType: extracted.RelationshipType,
		TargetEntityID:   targetID,
		Confidence:       &confidence,
		CreatedBy:        createdBy,
	}
}

// parseExtractionResponse parses Claude's JSON response into an ExtractionResult.
// It handles common formatting issues like markdown code blocks.
func parseExtractionResponse(responseText string, result *ExtractionResult) error {
	// Strip markdown code blocks if present (```json ... ```)
	cleaned := strings.TrimSpace(responseText)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	// Parse JSON
	if err := json.Unmarshal([]byte(cleaned), result); err != nil {
		return fmt.Errorf("invalid JSON response: %w (response: %s)", err, responseText)
	}

	// Initialize empty slices if nil (for consistent behavior)
	if result.Entities == nil {
		result.Entities = []ExtractedEntity{}
	}
	if result.Relationships == nil {
		result.Relationships = []ExtractedRelationship{}
	}

	return nil
}
