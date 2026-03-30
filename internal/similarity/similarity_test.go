package similarity

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect map[string]int
	}{
		{
			name:   "simple text",
			input:  "hello world",
			expect: map[string]int{"hello": 1, "world": 1},
		},
		{
			name:   "repeated words",
			input:  "test test test",
			expect: map[string]int{"test": 3},
		},
		{
			name:   "punctuation removal",
			input:  "hello, world!",
			expect: map[string]int{"hello": 1, "world": 1},
		},
		{
			name:   "single-char exclusion",
			input:  "a b c hello",
			expect: map[string]int{"hello": 1},
		},
		{
			name:   "empty string",
			input:  "",
			expect: map[string]int{},
		},
		{
			name:   "mixed case",
			input:  "Hello WORLD",
			expect: map[string]int{"hello": 1, "world": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tokenize(tt.input)
			if len(got) != len(tt.expect) {
				t.Errorf("Tokenize() len = %d, want %d", len(got), len(tt.expect))
			}
			for k, v := range tt.expect {
				if got[k] != v {
					t.Errorf("Tokenize()[%s] = %d, want %d", k, got[k], v)
				}
			}
		})
	}
}

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name      string
		a         map[string]int
		b         map[string]int
		expectMin float64
		expectMax float64
	}{
		{
			name:      "partial overlap",
			a:         map[string]int{"hello": 1, "world": 1},
			b:         map[string]int{"hello": 1, "universe": 1},
			expectMin: 0.3,
			expectMax: 0.4,
		},
		{
			name:      "identical maps",
			a:         map[string]int{"hello": 1, "world": 1},
			b:         map[string]int{"hello": 1, "world": 1},
			expectMin: 1.0,
			expectMax: 1.0,
		},
		{
			name:      "no overlap",
			a:         map[string]int{"hello": 1},
			b:         map[string]int{"world": 1},
			expectMin: 0.0,
			expectMax: 0.0,
		},
		{
			name:      "both empty",
			a:         map[string]int{},
			b:         map[string]int{},
			expectMin: 1.0,
			expectMax: 1.0,
		},
		{
			name:      "one empty",
			a:         map[string]int{"hello": 1},
			b:         map[string]int{},
			expectMin: 0.0,
			expectMax: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := JaccardSimilarity(tt.a, tt.b)
			if sim < tt.expectMin || sim > tt.expectMax {
				t.Errorf("JaccardSimilarity() = %.2f, want [%.2f, %.2f]", sim, tt.expectMin, tt.expectMax)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name      string
		a         map[string]int
		b         map[string]int
		expectMin float64
		expectMax float64
	}{
		{
			name:      "partial overlap",
			a:         map[string]int{"hello": 1, "world": 1},
			b:         map[string]int{"hello": 1, "universe": 1},
			expectMin: 0.4,
			expectMax: 0.6,
		},
		{
			name:      "identical maps",
			a:         map[string]int{"hello": 1, "world": 1},
			b:         map[string]int{"hello": 1, "world": 1},
			expectMin: 1.0,
			expectMax: 1.0,
		},
		{
			name:      "no overlap",
			a:         map[string]int{"hello": 1},
			b:         map[string]int{"world": 1},
			expectMin: 0.0,
			expectMax: 0.0,
		},
		{
			name:      "one empty",
			a:         map[string]int{"hello": 1},
			b:         map[string]int{},
			expectMin: 0.0,
			expectMax: 0.0,
		},
		{
			name:      "different frequencies",
			a:         map[string]int{"hello": 2, "world": 1},
			b:         map[string]int{"hello": 1, "world": 2},
			expectMin: 0.7,
			expectMax: 0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := CosineSimilarity(tt.a, tt.b)
			const epsilon = 1e-9
			if sim < (tt.expectMin-epsilon) || sim > (tt.expectMax+epsilon) {
				t.Errorf("CosineSimilarity() = %.10f, want [%.10f, %.10f]", sim, tt.expectMin, tt.expectMax)
			}
		})
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "already normalized",
			input:  "hello world",
			expect: "hello world",
		},
		{
			name:   "mixed case",
			input:  "Hello World",
			expect: "hello world",
		},
		{
			name:   "extra whitespace",
			input:  "  hello   world  ",
			expect: "hello world",
		},
		{
			name:   "newlines and tabs",
			input:  "hello\tworld\ntest",
			expect: "hello world test",
		},
		{
			name:   "empty string",
			input:  "",
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeText(tt.input)
			if got != tt.expect {
				t.Errorf("NormalizeText() = %q, want %q", got, tt.expect)
			}
		})
	}
}
