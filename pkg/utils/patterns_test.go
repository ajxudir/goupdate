package utils

import (
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
)

func TestSelectPatterns_NilConfig(t *testing.T) {
	result := SelectPatterns("content", nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestSelectPatterns_SinglePatternOnly(t *testing.T) {
	cfg := &config.ExtractionCfg{
		Pattern: `(?P<name>\w+)`,
	}

	result := SelectPatterns("content", cfg)
	if len(result) != 1 || result[0] != cfg.Pattern {
		t.Errorf("expected single pattern, got %v", result)
	}
}

func TestSelectPatterns_EmptyConfig(t *testing.T) {
	cfg := &config.ExtractionCfg{}

	result := SelectPatterns("content", cfg)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestSelectPatterns_NoDetect_AlwaysInclude(t *testing.T) {
	cfg := &config.ExtractionCfg{
		Patterns: []config.PatternCfg{
			{Name: "always", Pattern: `pattern1`},
			{Name: "always2", Pattern: `pattern2`},
		},
	}

	result := SelectPatterns("any content", cfg)
	if len(result) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(result))
	}
	if result[0] != "pattern1" || result[1] != "pattern2" {
		t.Errorf("unexpected patterns: %v", result)
	}
}

func TestSelectPatterns_WithDetect_OnlyMatchingIncluded(t *testing.T) {
	cfg := &config.ExtractionCfg{
		Patterns: []config.PatternCfg{
			{Name: "v9", Detect: `lockfileVersion:\s*'9`, Pattern: `v9_pattern`},
			{Name: "v6", Detect: `lockfileVersion:\s*'6`, Pattern: `v6_pattern`},
		},
	}

	content := `lockfileVersion: '9.0'`
	result := SelectPatterns(content, cfg)

	if len(result) != 1 {
		t.Errorf("expected 1 pattern, got %d: %v", len(result), result)
	}
	if result[0] != "v9_pattern" {
		t.Errorf("expected v9_pattern, got %s", result[0])
	}
}

func TestSelectPatterns_MultipleDetectMatches_AllIncluded(t *testing.T) {
	cfg := &config.ExtractionCfg{
		Patterns: []config.PatternCfg{
			{Name: "has_foo", Detect: `foo`, Pattern: `foo_pattern`},
			{Name: "has_bar", Detect: `bar`, Pattern: `bar_pattern`},
			{Name: "no_match", Detect: `not_present`, Pattern: `nomatch_pattern`},
		},
	}

	content := `foo and bar are both here`
	result := SelectPatterns(content, cfg)

	// Should include both foo and bar patterns, not nomatch
	if len(result) != 2 {
		t.Errorf("expected 2 patterns, got %d: %v", len(result), result)
	}
	if result[0] != "foo_pattern" || result[1] != "bar_pattern" {
		t.Errorf("unexpected patterns: %v", result)
	}
}

func TestSelectPatterns_MixedDetectAndNoDetect(t *testing.T) {
	cfg := &config.ExtractionCfg{
		Patterns: []config.PatternCfg{
			{Name: "always", Pattern: `always_pattern`},                          // No detect = always
			{Name: "v9", Detect: `lockfileVersion:\s*'9`, Pattern: `v9_pattern`}, // Conditional
			{Name: "v6", Detect: `lockfileVersion:\s*'6`, Pattern: `v6_pattern`}, // Won't match
		},
	}

	content := `lockfileVersion: '9.0'`
	result := SelectPatterns(content, cfg)

	// Should include always_pattern and v9_pattern
	if len(result) != 2 {
		t.Errorf("expected 2 patterns, got %d: %v", len(result), result)
	}
	if result[0] != "always_pattern" || result[1] != "v9_pattern" {
		t.Errorf("unexpected patterns: %v", result)
	}
}

func TestSelectPatterns_NoMatchesFallbackToSinglePattern(t *testing.T) {
	cfg := &config.ExtractionCfg{
		Pattern: `fallback_pattern`,
		Patterns: []config.PatternCfg{
			{Name: "v9", Detect: `lockfileVersion:\s*'9`, Pattern: `v9_pattern`},
		},
	}

	content := `lockfileVersion: '6.0'` // Won't match v9
	result := SelectPatterns(content, cfg)

	if len(result) != 1 {
		t.Errorf("expected 1 pattern (fallback), got %d: %v", len(result), result)
	}
	if result[0] != "fallback_pattern" {
		t.Errorf("expected fallback_pattern, got %s", result[0])
	}
}

func TestSelectPatterns_EmptyPatternSkipped(t *testing.T) {
	cfg := &config.ExtractionCfg{
		Patterns: []config.PatternCfg{
			{Name: "empty", Pattern: ""},
			{Name: "valid", Pattern: `valid_pattern`},
		},
	}

	result := SelectPatterns("content", cfg)
	if len(result) != 1 || result[0] != "valid_pattern" {
		t.Errorf("expected only valid_pattern, got %v", result)
	}
}

func TestSelectPatternsWithNames(t *testing.T) {
	cfg := &config.ExtractionCfg{
		Patterns: []config.PatternCfg{
			{Name: "v9", Detect: `version:\s*9`, Pattern: `v9_pattern`},
			{Name: "always", Pattern: `always_pattern`},
		},
	}

	content := `version: 9`
	result := SelectPatternsWithNames(content, cfg)

	if len(result) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(result))
	}
	if result[0].Name != "v9" || result[1].Name != "always" {
		t.Errorf("unexpected pattern names: %v, %v", result[0].Name, result[1].Name)
	}
}

func TestSelectPatternsWithNames_FallbackToDefault(t *testing.T) {
	cfg := &config.ExtractionCfg{
		Pattern: `single_pattern`,
	}

	result := SelectPatternsWithNames("content", cfg)
	if len(result) != 1 || result[0].Name != "default" {
		t.Errorf("expected default name, got %v", result)
	}
}

func TestExtractWithPatterns(t *testing.T) {
	cfg := &config.ExtractionCfg{
		Patterns: []config.PatternCfg{
			{Name: "v9", Detect: `v9`, Pattern: `(?P<name>\w+)@(?P<version>[\d.]+)`},
		},
	}

	content := `v9 lodash@4.17.21 express@4.18.2`
	result, err := ExtractWithPatterns(content, cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 matches, got %d: %v", len(result), result)
	}
	if result[0]["name"] != "lodash" || result[0]["version"] != "4.17.21" {
		t.Errorf("unexpected first match: %v", result[0])
	}
}

func TestExtractWithPatterns_MultiplePatterns(t *testing.T) {
	cfg := &config.ExtractionCfg{
		Patterns: []config.PatternCfg{
			{Name: "format1", Pattern: `pkg:(?P<name>\w+):(?P<version>[\d.]+)`},
			{Name: "format2", Pattern: `(?P<name>\w+)@(?P<version>[\d.]+)`},
		},
	}

	content := `pkg:lodash:4.17.21 express@4.18.2`
	result, err := ExtractWithPatterns(content, cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 matches, got %d: %v", len(result), result)
	}
}

func TestMatchesAnyExcludePattern_SimplePatterns(t *testing.T) {
	patterns := []string{`(?i)-alpha`, `(?i)-beta`}

	tests := []struct {
		version string
		matches bool
	}{
		{"1.0.0-alpha", true},
		{"2.0.0-BETA", true},
		{"1.0.0", false},
		{"1.0.0-rc1", false},
	}

	for _, tt := range tests {
		matched, _ := MatchesAnyExcludePattern(tt.version, patterns, nil, "")
		if matched != tt.matches {
			t.Errorf("version %q: expected matched=%v, got %v", tt.version, tt.matches, matched)
		}
	}
}

func TestMatchesAnyExcludePattern_PatternCfgs(t *testing.T) {
	patternCfgs := []config.PatternCfg{
		{Name: "alpha", Pattern: `(?i)-alpha`},
		{Name: "beta", Pattern: `(?i)-beta`},
	}

	matched, name := MatchesAnyExcludePattern("1.0.0-alpha", nil, patternCfgs, "")
	if !matched {
		t.Error("expected match")
	}
	if name != "alpha" {
		t.Errorf("expected name 'alpha', got %q", name)
	}
}

func TestMatchesAnyExcludePattern_WithDetect(t *testing.T) {
	patternCfgs := []config.PatternCfg{
		{Name: "npm-prerelease", Detect: `"lockfileVersion"`, Pattern: `(?i)-alpha`},
		{Name: "pnpm-prerelease", Detect: `lockfileVersion:`, Pattern: `(?i)-rc`},
	}

	// With npm-style content, only npm-prerelease should apply
	npmContent := `{"lockfileVersion": 3}`
	matched, name := MatchesAnyExcludePattern("1.0.0-alpha", nil, patternCfgs, npmContent)
	if !matched || name != "npm-prerelease" {
		t.Errorf("expected npm-prerelease match, got matched=%v, name=%q", matched, name)
	}

	// -rc should not match because pnpm detect doesn't match npm content
	matched, _ = MatchesAnyExcludePattern("1.0.0-rc1", nil, patternCfgs, npmContent)
	if matched {
		t.Error("expected no match for -rc1 with npm content")
	}

	// With pnpm-style content, pnpm-prerelease should apply
	pnpmContent := `lockfileVersion: '9.0'`
	matched, name = MatchesAnyExcludePattern("1.0.0-rc1", nil, patternCfgs, pnpmContent)
	if !matched || name != "pnpm-prerelease" {
		t.Errorf("expected pnpm-prerelease match, got matched=%v, name=%q", matched, name)
	}
}

func TestMatchesDetect(t *testing.T) {
	tests := []struct {
		content string
		detect  string
		matches bool
	}{
		{"lockfileVersion: '9.0'", `lockfileVersion:\s*'9`, true},
		{"lockfileVersion: '6.0'", `lockfileVersion:\s*'9`, false},
		{"any content", "", true}, // Empty detect always matches
		{"content", `[invalid(regex`, false},
	}

	for _, tt := range tests {
		result := matchesDetect(tt.content, tt.detect)
		if result != tt.matches {
			t.Errorf("matchesDetect(%q, %q) = %v, want %v", tt.content, tt.detect, result, tt.matches)
		}
	}
}

func TestSelectPatterns_NPMVersions(t *testing.T) {
	// Test real npm lock file version detection patterns
	cfg := &config.ExtractionCfg{
		Patterns: []config.PatternCfg{
			{Name: "v1", Detect: `"lockfileVersion":\s*1[,\s}]`, Pattern: `v1_pattern`},
			{Name: "v2", Detect: `"lockfileVersion":\s*2[,\s}]`, Pattern: `v2_pattern`},
			{Name: "v3", Detect: `"lockfileVersion":\s*3[,\s}]`, Pattern: `v3_pattern`},
		},
	}

	tests := []struct {
		name            string
		content         string
		expectedPattern string
	}{
		{
			name:            "npm_v1",
			content:         `{"lockfileVersion": 1, "dependencies": {}}`,
			expectedPattern: "v1_pattern",
		},
		{
			name:            "npm_v2",
			content:         `{"lockfileVersion": 2, "packages": {}}`,
			expectedPattern: "v2_pattern",
		},
		{
			name:            "npm_v3",
			content:         `{"lockfileVersion": 3, "packages": {}}`,
			expectedPattern: "v3_pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectPatterns(tt.content, cfg)
			if len(result) != 1 {
				t.Errorf("expected 1 pattern, got %d: %v", len(result), result)
				return
			}
			if result[0] != tt.expectedPattern {
				t.Errorf("expected %s, got %s", tt.expectedPattern, result[0])
			}
		})
	}
}

func TestSelectPatterns_PNPMVersions(t *testing.T) {
	// Test real pnpm lock file version detection patterns
	cfg := &config.ExtractionCfg{
		Patterns: []config.PatternCfg{
			{Name: "v6", Detect: `lockfileVersion:\s*'6`, Pattern: `v6_pattern`},
			{Name: "v7", Detect: `lockfileVersion:\s*'7`, Pattern: `v7_pattern`},
			{Name: "v8", Detect: `lockfileVersion:\s*'8`, Pattern: `v8_pattern`},
			{Name: "v9", Detect: `lockfileVersion:\s*'9`, Pattern: `v9_pattern`},
		},
	}

	tests := []struct {
		name            string
		content         string
		expectedPattern string
	}{
		{
			name:            "pnpm_v6",
			content:         "lockfileVersion: '6.0'\nimporters:",
			expectedPattern: "v6_pattern",
		},
		{
			name:            "pnpm_v7",
			content:         "lockfileVersion: '7.0'\nimporters:",
			expectedPattern: "v7_pattern",
		},
		{
			name:            "pnpm_v8",
			content:         "lockfileVersion: '8.0'\nimporters:",
			expectedPattern: "v8_pattern",
		},
		{
			name:            "pnpm_v9",
			content:         "lockfileVersion: '9.0'\nimporters:",
			expectedPattern: "v9_pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectPatterns(tt.content, cfg)
			if len(result) != 1 {
				t.Errorf("expected 1 pattern, got %d: %v", len(result), result)
				return
			}
			if result[0] != tt.expectedPattern {
				t.Errorf("expected %s, got %s", tt.expectedPattern, result[0])
			}
		})
	}
}

func TestSelectPatterns_YarnVersions(t *testing.T) {
	// Test real yarn lock file version detection patterns
	cfg := &config.ExtractionCfg{
		Patterns: []config.PatternCfg{
			{Name: "classic", Detect: `#\s*yarn lockfile v1`, Pattern: `classic_pattern`},
			{Name: "berry", Detect: `__metadata:\s*\n\s+version:`, Pattern: `berry_pattern`},
		},
	}

	classicContent := `# yarn lockfile v1

lodash@^4.17.21:
  version "4.17.21"
`

	berryContent := `__metadata:
  version: 8

"lodash@npm:^4.17.21":
  version: 4.17.21
`

	t.Run("yarn_classic", func(t *testing.T) {
		result := SelectPatterns(classicContent, cfg)
		if len(result) != 1 || result[0] != "classic_pattern" {
			t.Errorf("expected classic_pattern, got %v", result)
		}
	})

	t.Run("yarn_berry", func(t *testing.T) {
		result := SelectPatterns(berryContent, cfg)
		if len(result) != 1 || result[0] != "berry_pattern" {
			t.Errorf("expected berry_pattern, got %v", result)
		}
	})
}
