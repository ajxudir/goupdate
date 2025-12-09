package config

import (
	"fmt"
	"regexp"
	"strings"
)

// PackageRef is an interface for package reference used by incremental logic.
// This allows the config package to work with packages without circular imports.
type PackageRef interface {
	GetName() string
	GetRule() string
}

// ShouldUpdateIncrementally reports whether a package should select the nearest available version instead of the latest.
//
// This checks both rule-level and global incremental patterns to determine
// if the package should use incremental updates. Incremental updates prefer
// the nearest compatible version rather than jumping to the latest version.
//
// Parameters:
//   - p: the package reference containing name and rule information
//   - cfg: the configuration containing incremental patterns
//
// Returns:
//   - bool: true if package should use incremental updates, false otherwise
//   - error: error if a pattern is invalid (malformed regex)
func ShouldUpdateIncrementally(p PackageRef, cfg *Config) (bool, error) {
	if cfg == nil {
		return false, nil
	}

	patterns := collectIncrementalPatterns(p, cfg)

	if len(patterns) == 0 {
		return false, nil
	}

	for _, pattern := range patterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" {
			continue
		}

		matcher, err := compileIncrementalPattern(trimmed)
		if err != nil {
			return false, fmt.Errorf("invalid incremental package pattern %q: %w", pattern, err)
		}

		if matcher.MatchString(p.GetName()) {
			return true, nil
		}
	}

	return false, nil
}

// collectIncrementalPatterns gathers all incremental patterns for a package.
//
// This collects patterns from both the package's rule configuration
// and the global configuration.
//
// Parameters:
//   - p: the package reference containing name and rule information
//   - cfg: the configuration containing incremental patterns
//
// Returns:
//   - []string: list of incremental patterns to check
func collectIncrementalPatterns(p PackageRef, cfg *Config) []string {
	patterns := make([]string, 0)

	if p.GetRule() != "" {
		if rule, ok := cfg.Rules[p.GetRule()]; ok {
			patterns = append(patterns, rule.Incremental...)
		}
	}

	patterns = append(patterns, cfg.Incremental...)

	return patterns
}

// compileIncrementalPattern compiles a pattern into a regex matcher.
//
// If the pattern contains regex metacharacters, it's compiled as-is.
// Otherwise, it's treated as a literal string with anchors (exact match).
//
// Parameters:
//   - pattern: the pattern string to compile
//
// Returns:
//   - *regexp.Regexp: compiled regex matcher
//   - error: error if pattern is invalid regex
func compileIncrementalPattern(pattern string) (*regexp.Regexp, error) {
	if usesRegexMeta(pattern) {
		return regexp.Compile(pattern)
	}

	return regexp.Compile("^" + regexp.QuoteMeta(pattern) + "$")
}

// usesRegexMeta checks if a pattern contains regex metacharacters.
//
// This determines whether a pattern should be compiled as regex or
// treated as a literal string.
//
// Parameters:
//   - pattern: the pattern string to check
//
// Returns:
//   - bool: true if pattern contains regex metacharacters, false otherwise
func usesRegexMeta(pattern string) bool {
	return strings.ContainsAny(pattern, ".*+?{}()|[]^$\\")
}
