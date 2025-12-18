package utils

import (
	"regexp"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/verbose"
)

// SelectPatterns returns ALL applicable patterns for the given content.
//
// This function implements multi-pattern extraction with conditional detection.
// It supports both single pattern (Pattern field) and multi-pattern (Patterns field)
// configurations.
//
// Logic:
//   - If Patterns is empty, falls back to single Pattern field
//   - For each pattern in Patterns:
//   - If Detect is empty → ALWAYS include (default = true)
//   - If Detect is set → Include ONLY if detect regex matches content
//   - Returns ALL matching patterns (additive, not exclusive)
//   - If no Patterns match but Pattern is set, returns Pattern as fallback
//
// Parameters:
//   - content: The file content to check against detect patterns
//   - cfg: The extraction configuration with Pattern and/or Patterns fields
//
// Returns:
//   - []string: All applicable extraction patterns; empty slice if none match
func SelectPatterns(content string, cfg *config.ExtractionCfg) []string {
	if cfg == nil {
		return nil
	}

	// If no multi-pattern config, use single pattern
	if len(cfg.Patterns) == 0 {
		if cfg.Pattern != "" {
			return []string{cfg.Pattern}
		}
		return nil
	}

	// Collect all matching patterns
	var result []string
	for _, p := range cfg.Patterns {
		if p.Pattern == "" {
			continue
		}

		// No detect = always include
		if p.Detect == "" {
			result = append(result, p.Pattern)
			continue
		}

		// With detect = include only if detect matches
		if matchesDetect(content, p.Detect) {
			result = append(result, p.Pattern)
		}
	}

	// If no patterns matched, fall back to single Pattern field
	if len(result) == 0 && cfg.Pattern != "" {
		return []string{cfg.Pattern}
	}

	return result
}

// SelectPatternsWithNames returns ALL applicable patterns with their names.
//
// This is similar to SelectPatterns but returns PatternCfg structs
// to preserve pattern names for logging and debugging.
//
// Parameters:
//   - content: The file content to check against detect patterns
//   - cfg: The extraction configuration with Pattern and/or Patterns fields
//
// Returns:
//   - []config.PatternCfg: All applicable pattern configs; empty slice if none match
func SelectPatternsWithNames(content string, cfg *config.ExtractionCfg) []config.PatternCfg {
	if cfg == nil {
		return nil
	}

	// If no multi-pattern config, create single pattern entry
	if len(cfg.Patterns) == 0 {
		if cfg.Pattern != "" {
			return []config.PatternCfg{{Name: "default", Pattern: cfg.Pattern}}
		}
		return nil
	}

	// Collect all matching patterns
	var result []config.PatternCfg
	for _, p := range cfg.Patterns {
		if p.Pattern == "" {
			continue
		}

		// No detect = always include
		if p.Detect == "" {
			result = append(result, p)
			continue
		}

		// With detect = include only if detect matches
		if matchesDetect(content, p.Detect) {
			result = append(result, p)
		}
	}

	// If no patterns matched, fall back to single Pattern field
	if len(result) == 0 && cfg.Pattern != "" {
		return []config.PatternCfg{{Name: "fallback", Pattern: cfg.Pattern}}
	}

	return result
}

// matchesDetect checks if content matches a detection pattern.
//
// Parameters:
//   - content: The content to check
//   - detectPattern: The regex pattern to match against
//
// Returns:
//   - bool: true if pattern matches content, false otherwise (including on error)
func matchesDetect(content, detectPattern string) bool {
	if detectPattern == "" {
		return true // No detect = always match
	}

	re, err := getOrCompileRegex(detectPattern)
	if err != nil {
		return false
	}

	return re.MatchString(content)
}

// ExtractWithPatterns applies all matching patterns and returns combined results.
//
// This function selects applicable patterns based on content and applies each one,
// combining all matches into a single result set. Duplicates are preserved as they
// may represent different package entries.
//
// Parameters:
//   - content: The file content to extract from
//   - cfg: The extraction configuration with Pattern and/or Patterns fields
//
// Returns:
//   - []map[string]string: Combined matches from all applicable patterns
//   - error: Returns error if any pattern fails to extract
func ExtractWithPatterns(content string, cfg *config.ExtractionCfg) ([]map[string]string, error) {
	patterns := SelectPatterns(content, cfg)
	if len(patterns) == 0 {
		verbose.Printf("Pattern extraction: no patterns selected from config\n")
		return nil, nil
	}

	verbose.Printf("Pattern extraction: applying %d pattern(s)\n", len(patterns))
	var allMatches []map[string]string
	for i, pattern := range patterns {
		verbose.Printf("Pattern extraction: applying pattern %d/%d\n", i+1, len(patterns))
		matches, err := ExtractAllMatches(pattern, content)
		if err != nil {
			verbose.Printf("Pattern extraction ERROR: pattern %d failed: %v\n", i+1, err)
			return nil, err
		}
		verbose.Printf("Pattern extraction: pattern %d matched %d entries\n", i+1, len(matches))
		allMatches = append(allMatches, matches...)
	}

	verbose.Printf("Pattern extraction: total %d matches from all patterns\n", len(allMatches))
	return allMatches, nil
}

// ExtractWithPatternsIndexed applies all matching patterns and returns matches with indices.
//
// This is useful when precise replacement positions are needed.
//
// Parameters:
//   - content: The file content to extract from
//   - cfg: The extraction configuration with Pattern and/or Patterns fields
//
// Returns:
//   - []MatchWithIndex: Combined matches with positions from all applicable patterns
//   - error: Returns error if any pattern fails to extract
func ExtractWithPatternsIndexed(content string, cfg *config.ExtractionCfg) ([]MatchWithIndex, error) {
	patterns := SelectPatterns(content, cfg)
	if len(patterns) == 0 {
		return nil, nil
	}

	var allMatches []MatchWithIndex
	for _, pattern := range patterns {
		matches, err := ExtractAllMatchesWithIndex(pattern, content)
		if err != nil {
			return nil, err
		}
		allMatches = append(allMatches, matches...)
	}

	return allMatches, nil
}

// MatchesAnyExcludePattern checks if a version matches any exclusion pattern.
//
// This function supports both simple patterns (legacy) and PatternCfg patterns
// with conditional detection.
//
// Parameters:
//   - version: The version string to check
//   - patterns: Simple regex patterns to match against
//   - patternCfgs: PatternCfg entries with optional detect conditions
//   - detectContent: Content to evaluate detect conditions against (can be empty)
//
// Returns:
//   - bool: true if version matches any applicable exclusion pattern
//   - string: The pattern name that matched (empty if no match)
func MatchesAnyExcludePattern(version string, patterns []string, patternCfgs []config.PatternCfg, detectContent string) (bool, string) {
	verbose.Printf("Exclude check: checking version %q against %d simple patterns, %d pattern configs\n",
		version, len(patterns), len(patternCfgs))

	// Check simple patterns first
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, version); matched {
			verbose.Printf("Exclude check: version %q matched simple pattern %q\n", version, pattern)
			return true, pattern
		}
	}

	// Check PatternCfg patterns with detect conditions
	for _, p := range patternCfgs {
		if p.Pattern == "" {
			continue
		}

		// Check detect condition if set
		if p.Detect != "" && !matchesDetect(detectContent, p.Detect) {
			verbose.Printf("Exclude check: pattern %q skipped (detect condition not met)\n", p.Name)
			continue
		}

		if matched, _ := regexp.MatchString(p.Pattern, version); matched {
			name := p.Name
			if name == "" {
				name = p.Pattern
			}
			verbose.Printf("Exclude check: version %q matched pattern config %q\n", version, name)
			return true, name
		}
	}

	verbose.Printf("Exclude check: version %q not excluded\n", version)
	return false, ""
}
