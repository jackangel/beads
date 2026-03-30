package extraction

import (
	"context"
	"testing"
)

// TestBuildExtractionPrompt verifies that the prompt is correctly formatted.
func TestBuildExtractionPrompt(t *testing.T) {
	text := "Alice leads Team X and uses Kubernetes for deployment."

	prompt := buildExtractionPrompt(text)

	if len(prompt) == 0 {
		t.Fatal("prompt is empty")
	}

	// Verify the input text is included
	if !contains(prompt, text) {
		t.Errorf("prompt does not contain input text")
	}

	// Verify JSON structure instructions are present
	if !contains(prompt, "entities") || !contains(prompt, "relationships") {
		t.Errorf("prompt does not describe expected JSON structure")
	}

	// Verify it asks for JSON-only output
	if !contains(prompt, "ONLY valid JSON") {
		t.Errorf("prompt does not emphasize JSON-only output")
	}
}

// TestParseExtractionResponse validates JSON parsing with mock responses.
func TestParseExtractionResponse(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		wantEntities   int
		wantRelations  int
		wantErr        bool
	}{
		{
			name: "valid response with entities and relationships",
			response: `{
				"entities": [
					{"name": "Alice", "type": "person", "summary": "Team lead"},
					{"name": "Team X", "type": "organization", "summary": "Engineering team"}
				],
				"relationships": [
					{"source": "Alice", "type": "leads", "target": "Team X", "confidence": 0.95}
				]
			}`,
			wantEntities:  2,
			wantRelations: 1,
			wantErr:       false,
		},
		{
			name: "response with markdown code block",
			response: "```json\n" + `{
				"entities": [{"name": "Bob", "type": "person", "summary": "Developer"}],
				"relationships": []
			}` + "\n```",
			wantEntities:  1,
			wantRelations: 0,
			wantErr:       false,
		},
		{
			name: "empty entities and relationships",
			response: `{
				"entities": [],
				"relationships": []
			}`,
			wantEntities:  0,
			wantRelations: 0,
			wantErr:       false,
		},
		{
			name: "missing fields defaults to empty slices",
			response: `{
			}`,
			wantEntities:  0,
			wantRelations: 0,
			wantErr:       false,
		},
		{
			name:     "invalid JSON",
			response: `{"entities": [invalid json`,
			wantErr:  true,
		},
		{
			name:     "non-JSON text",
			response: `This is not JSON at all`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result ExtractionResult
			err := parseExtractionResponse(tt.response, &result)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseExtractionResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // Skip further checks if we expected an error
			}

			if len(result.Entities) != tt.wantEntities {
				t.Errorf("got %d entities, want %d", len(result.Entities), tt.wantEntities)
			}

			if len(result.Relationships) != tt.wantRelations {
				t.Errorf("got %d relationships, want %d", len(result.Relationships), tt.wantRelations)
			}
		})
	}
}

// TestParseExtractionResponse_ValidatesStructure ensures parsed data has expected fields.
func TestParseExtractionResponse_ValidatesStructure(t *testing.T) {
	response := `{
		"entities": [
			{"name": "Charlie", "type": "person", "summary": "Engineer", "metadata": {"team": "backend"}}
		],
		"relationships": [
			{"source": "Charlie", "type": "works_on", "target": "API", "confidence": 0.8}
		]
	}`

	var result ExtractionResult
	err := parseExtractionResponse(response, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Validate entity fields
	if len(result.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(result.Entities))
	}
	entity := result.Entities[0]
	if entity.Name != "Charlie" {
		t.Errorf("entity name = %q, want %q", entity.Name, "Charlie")
	}
	if entity.EntityType != "person" {
		t.Errorf("entity type = %q, want %q", entity.EntityType, "person")
	}
	if entity.Summary != "Engineer" {
		t.Errorf("entity summary = %q, want %q", entity.Summary, "Engineer")
	}
	if entity.Metadata == nil {
		t.Error("entity metadata is nil")
	} else if entity.Metadata["team"] != "backend" {
		t.Errorf("entity metadata[team] = %v, want %q", entity.Metadata["team"], "backend")
	}

	// Validate relationship fields
	if len(result.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(result.Relationships))
	}
	rel := result.Relationships[0]
	if rel.SourceName != "Charlie" {
		t.Errorf("relationship source = %q, want %q", rel.SourceName, "Charlie")
	}
	if rel.RelationshipType != "works_on" {
		t.Errorf("relationship type = %q, want %q", rel.RelationshipType, "works_on")
	}
	if rel.TargetName != "API" {
		t.Errorf("relationship target = %q, want %q", rel.TargetName, "API")
	}
	if rel.Confidence != 0.8 {
		t.Errorf("relationship confidence = %f, want %f", rel.Confidence, 0.8)
	}
}

// TestExtractFromText_EmptyInput verifies handling of empty text.
func TestExtractFromText_EmptyInput(t *testing.T) {
	// Create extractor with dummy config (won't make API calls for empty input)
	extractor := NewExtractor(ExtractionConfig{
		APIKey: "dummy-key-for-test",
	})

	ctx := context.Background()
	result, err := extractor.ExtractFromText(ctx, "")

	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}

	if len(result.Entities) != 0 {
		t.Errorf("expected 0 entities for empty input, got %d", len(result.Entities))
	}

	if len(result.Relationships) != 0 {
		t.Errorf("expected 0 relationships for empty input, got %d", len(result.Relationships))
	}
}

// TestNewExtractor_DefaultConfig verifies default configuration values.
func TestNewExtractor_DefaultConfig(t *testing.T) {
	extractor := NewExtractor(ExtractionConfig{
		APIKey: "test-key",
	})

	if extractor.config.Model != "claude-3-5-haiku-latest" {
		t.Errorf("default model = %q, want %q", extractor.config.Model, "claude-3-5-haiku-latest")
	}

	if extractor.config.MaxTokens != 4096 {
		t.Errorf("default max tokens = %d, want %d", extractor.config.MaxTokens, 4096)
	}

	extractor2 := NewExtractor(ExtractionConfig{
		APIKey:    "test-key",
		Model:     "custom-model",
		MaxTokens: 2048,
	})

	if extractor2.config.Model != "custom-model" {
		t.Errorf("custom model not preserved: got %q", extractor2.config.Model)
	}

	if extractor2.config.MaxTokens != 2048 {
		t.Errorf("custom max tokens not preserved: got %d", extractor2.config.MaxTokens)
	}
}

// contains is a helper to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
