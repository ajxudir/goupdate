package filtering

import (
	"regexp"
	"strings"

	"github.com/user/goupdate/pkg/utils"
)

// Matcher defines the interface for string matching strategies.
//
// Implementations can provide different matching algorithms:
// exact, prefix, suffix, glob, or regex matching.
//
// Example:
//
//	matcher := filtering.NewGlobMatcher("*.go")
//	if matcher.Match("main.go") {
//	    fmt.Println("matched!")
//	}
type Matcher interface {
	// Match tests if the given value matches the pattern.
	//
	// Parameters:
	//   - value: String to test against the pattern
	//
	// Returns:
	//   - bool: true if value matches the pattern
	Match(value string) bool

	// String returns a string representation of the matcher.
	//
	// Returns:
	//   - string: Description of the pattern
	String() string
}

// ExactMatcher matches strings that exactly equal the pattern.
//
// Fields:
//   - Pattern: The exact string to match
//   - IgnoreCase: If true, performs case-insensitive matching
//
// Example:
//
//	matcher := &filtering.ExactMatcher{Pattern: "lodash", IgnoreCase: true}
//	matcher.Match("Lodash")  // returns true
//	matcher.Match("lodash")  // returns true
//	matcher.Match("lodash2") // returns false
type ExactMatcher struct {
	// Pattern is the exact string to match.
	Pattern string

	// IgnoreCase enables case-insensitive matching.
	IgnoreCase bool
}

// Match tests if value exactly equals the pattern.
//
// Parameters:
//   - value: String to test
//
// Returns:
//   - bool: true if value equals pattern (respecting IgnoreCase)
func (m *ExactMatcher) Match(value string) bool {
	if m.IgnoreCase {
		return strings.EqualFold(value, m.Pattern)
	}
	return value == m.Pattern
}

// String returns the pattern string.
//
// Returns:
//   - string: The exact pattern being matched
func (m *ExactMatcher) String() string {
	return m.Pattern
}

// PrefixMatcher matches strings that start with the pattern.
//
// Fields:
//   - Prefix: The prefix string to match
//   - IgnoreCase: If true, performs case-insensitive matching
//
// Example:
//
//	matcher := &filtering.PrefixMatcher{Prefix: "@angular/"}
//	matcher.Match("@angular/core")    // returns true
//	matcher.Match("@angular/common")  // returns true
//	matcher.Match("react")            // returns false
type PrefixMatcher struct {
	// Prefix is the string that values must start with.
	Prefix string

	// IgnoreCase enables case-insensitive matching.
	IgnoreCase bool
}

// Match tests if value starts with the prefix.
//
// Parameters:
//   - value: String to test
//
// Returns:
//   - bool: true if value starts with prefix
func (m *PrefixMatcher) Match(value string) bool {
	if m.IgnoreCase {
		return strings.HasPrefix(strings.ToLower(value), strings.ToLower(m.Prefix))
	}
	return strings.HasPrefix(value, m.Prefix)
}

// String returns the prefix pattern.
//
// Returns:
//   - string: The prefix with a trailing asterisk (e.g., "prefix*")
func (m *PrefixMatcher) String() string {
	return m.Prefix + "*"
}

// SuffixMatcher matches strings that end with the pattern.
//
// Fields:
//   - Suffix: The suffix string to match
//   - IgnoreCase: If true, performs case-insensitive matching
//
// Example:
//
//	matcher := &filtering.SuffixMatcher{Suffix: "-plugin"}
//	matcher.Match("babel-plugin")    // returns true
//	matcher.Match("webpack-plugin")  // returns true
//	matcher.Match("plugin-babel")    // returns false
type SuffixMatcher struct {
	// Suffix is the string that values must end with.
	Suffix string

	// IgnoreCase enables case-insensitive matching.
	IgnoreCase bool
}

// Match tests if value ends with the suffix.
//
// Parameters:
//   - value: String to test
//
// Returns:
//   - bool: true if value ends with suffix
func (m *SuffixMatcher) Match(value string) bool {
	if m.IgnoreCase {
		return strings.HasSuffix(strings.ToLower(value), strings.ToLower(m.Suffix))
	}
	return strings.HasSuffix(value, m.Suffix)
}

// String returns the suffix pattern.
//
// Returns:
//   - string: The suffix with a leading asterisk (e.g., "*suffix")
func (m *SuffixMatcher) String() string {
	return "*" + m.Suffix
}

// ContainsMatcher matches strings that contain the pattern.
//
// Fields:
//   - Substring: The substring to search for
//   - IgnoreCase: If true, performs case-insensitive matching
//
// Example:
//
//	matcher := &filtering.ContainsMatcher{Substring: "test"}
//	matcher.Match("test-utils")   // returns true
//	matcher.Match("jest-test")    // returns true
//	matcher.Match("production")   // returns false
type ContainsMatcher struct {
	// Substring is the string to search for within values.
	Substring string

	// IgnoreCase enables case-insensitive matching.
	IgnoreCase bool
}

// Match tests if value contains the substring.
//
// Parameters:
//   - value: String to test
//
// Returns:
//   - bool: true if value contains substring
func (m *ContainsMatcher) Match(value string) bool {
	if m.IgnoreCase {
		return strings.Contains(strings.ToLower(value), strings.ToLower(m.Substring))
	}
	return strings.Contains(value, m.Substring)
}

// String returns the contains pattern.
//
// Returns:
//   - string: The substring wrapped in asterisks (e.g., "*substring*")
func (m *ContainsMatcher) String() string {
	return "*" + m.Substring + "*"
}

// GlobMatcher matches strings using glob patterns.
//
// Supports:
//   - * matches any sequence of characters (except /)
//   - ** matches any sequence including /
//   - ? matches any single character
//   - ! prefix negates the match
//
// Fields:
//   - Pattern: The glob pattern
//
// Example:
//
//	matcher := filtering.NewGlobMatcher("@types/*")
//	matcher.Match("@types/node")   // returns true
//	matcher.Match("@types/react")  // returns true
//	matcher.Match("@babel/core")   // returns false
type GlobMatcher struct {
	// Pattern is the glob pattern string.
	Pattern string
}

// Match tests if value matches the glob pattern.
//
// Parameters:
//   - value: String to test
//
// Returns:
//   - bool: true if value matches the glob pattern
func (m *GlobMatcher) Match(value string) bool {
	return utils.MatchGlob(value, m.Pattern)
}

// String returns the glob pattern.
//
// Returns:
//   - string: The glob pattern string (e.g., "*.go", "**/test_*.go")
func (m *GlobMatcher) String() string {
	return m.Pattern
}

// RegexMatcher matches strings using regular expressions.
//
// Fields:
//   - Pattern: The regex pattern string
//   - regex: Compiled regex (set by NewRegexMatcher)
//
// Example:
//
//	matcher, _ := filtering.NewRegexMatcher(`^@\w+/`)
//	matcher.Match("@angular/core")  // returns true
//	matcher.Match("lodash")         // returns false
type RegexMatcher struct {
	// Pattern is the original regex pattern string.
	Pattern string

	// regex is the compiled regular expression.
	regex *regexp.Regexp
}

// Match tests if value matches the regex pattern.
//
// Parameters:
//   - value: String to test
//
// Returns:
//   - bool: true if value matches the regex
func (m *RegexMatcher) Match(value string) bool {
	if m.regex == nil {
		return false
	}
	return m.regex.MatchString(value)
}

// String returns the regex pattern.
//
// Returns:
//   - string: The pattern prefixed with tilde (e.g., "~^@\\w+/")
func (m *RegexMatcher) String() string {
	return "~" + m.Pattern
}

// NewExactMatcher creates a case-sensitive exact matcher.
//
// Parameters:
//   - pattern: Exact string to match
//
// Returns:
//   - Matcher: An ExactMatcher instance
//
// Example:
//
//	matcher := filtering.NewExactMatcher("lodash")
func NewExactMatcher(pattern string) Matcher {
	return &ExactMatcher{Pattern: pattern, IgnoreCase: false}
}

// NewExactMatcherIgnoreCase creates a case-insensitive exact matcher.
//
// Parameters:
//   - pattern: Exact string to match (case-insensitive)
//
// Returns:
//   - Matcher: An ExactMatcher with IgnoreCase=true
//
// Example:
//
//	matcher := filtering.NewExactMatcherIgnoreCase("Lodash")
func NewExactMatcherIgnoreCase(pattern string) Matcher {
	return &ExactMatcher{Pattern: pattern, IgnoreCase: true}
}

// NewPrefixMatcher creates a prefix matcher.
//
// Parameters:
//   - prefix: Prefix string to match
//
// Returns:
//   - Matcher: A PrefixMatcher instance
//
// Example:
//
//	matcher := filtering.NewPrefixMatcher("@angular/")
func NewPrefixMatcher(prefix string) Matcher {
	return &PrefixMatcher{Prefix: prefix, IgnoreCase: false}
}

// NewSuffixMatcher creates a suffix matcher.
//
// Parameters:
//   - suffix: Suffix string to match
//
// Returns:
//   - Matcher: A SuffixMatcher instance
//
// Example:
//
//	matcher := filtering.NewSuffixMatcher("-plugin")
func NewSuffixMatcher(suffix string) Matcher {
	return &SuffixMatcher{Suffix: suffix, IgnoreCase: false}
}

// NewContainsMatcher creates a substring matcher.
//
// Parameters:
//   - substring: Substring to search for
//
// Returns:
//   - Matcher: A ContainsMatcher instance
//
// Example:
//
//	matcher := filtering.NewContainsMatcher("test")
func NewContainsMatcher(substring string) Matcher {
	return &ContainsMatcher{Substring: substring, IgnoreCase: false}
}

// NewGlobMatcher creates a glob pattern matcher.
//
// Parameters:
//   - pattern: Glob pattern string
//
// Returns:
//   - Matcher: A GlobMatcher instance
//
// Example:
//
//	matcher := filtering.NewGlobMatcher("@types/*")
func NewGlobMatcher(pattern string) Matcher {
	return &GlobMatcher{Pattern: pattern}
}

// NewRegexMatcher creates a regex matcher.
//
// Parameters:
//   - pattern: Regular expression pattern
//
// Returns:
//   - Matcher: A RegexMatcher instance
//   - error: Compilation error if pattern is invalid
//
// Example:
//
//	matcher, err := filtering.NewRegexMatcher(`^@\w+/`)
func NewRegexMatcher(pattern string) (Matcher, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexMatcher{Pattern: pattern, regex: regex}, nil
}

// MustRegexMatcher creates a regex matcher, panicking on invalid pattern.
//
// Use this only for compile-time constant patterns that are known valid.
//
// Parameters:
//   - pattern: Regular expression pattern
//
// Returns:
//   - Matcher: A RegexMatcher instance
//
// Example:
//
//	var scopedPackage = filtering.MustRegexMatcher(`^@\w+/`)
func MustRegexMatcher(pattern string) Matcher {
	m, err := NewRegexMatcher(pattern)
	if err != nil {
		panic("invalid regex pattern: " + err.Error())
	}
	return m
}

// AnyMatcher matches if any of the contained matchers match.
//
// This implements OR logic across multiple matchers.
//
// Fields:
//   - Matchers: Slice of matchers to test
//
// Example:
//
//	matcher := filtering.NewAnyMatcher(
//	    filtering.NewPrefixMatcher("@angular/"),
//	    filtering.NewPrefixMatcher("@types/"),
//	)
type AnyMatcher struct {
	// Matchers is the list of matchers (OR logic).
	Matchers []Matcher
}

// Match returns true if any matcher matches.
//
// Parameters:
//   - value: String to test
//
// Returns:
//   - bool: true if any matcher matches
func (m *AnyMatcher) Match(value string) bool {
	for _, matcher := range m.Matchers {
		if matcher.Match(value) {
			return true
		}
	}
	return false
}

// String returns a description of the matchers.
//
// Returns:
//   - string: Description in format "any(pattern1, pattern2, ...)"
func (m *AnyMatcher) String() string {
	var patterns []string
	for _, matcher := range m.Matchers {
		patterns = append(patterns, matcher.String())
	}
	return "any(" + strings.Join(patterns, ", ") + ")"
}

// AllMatcher matches only if all contained matchers match.
//
// This implements AND logic across multiple matchers.
//
// Fields:
//   - Matchers: Slice of matchers to test
//
// Example:
//
//	matcher := filtering.NewAllMatcher(
//	    filtering.NewPrefixMatcher("@angular/"),
//	    filtering.NewSuffixMatcher("-core"),
//	)
type AllMatcher struct {
	// Matchers is the list of matchers (AND logic).
	Matchers []Matcher
}

// Match returns true only if all matchers match.
//
// Parameters:
//   - value: String to test
//
// Returns:
//   - bool: true if all matchers match
func (m *AllMatcher) Match(value string) bool {
	for _, matcher := range m.Matchers {
		if !matcher.Match(value) {
			return false
		}
	}
	return true
}

// String returns a description of the matchers.
//
// Returns:
//   - string: Description in format "all(pattern1, pattern2, ...)"
func (m *AllMatcher) String() string {
	var patterns []string
	for _, matcher := range m.Matchers {
		patterns = append(patterns, matcher.String())
	}
	return "all(" + strings.Join(patterns, ", ") + ")"
}

// NotMatcher negates another matcher's result.
//
// Fields:
//   - Matcher: The matcher to negate
//
// Example:
//
//	matcher := filtering.NewNotMatcher(
//	    filtering.NewPrefixMatcher("@types/"),
//	)
type NotMatcher struct {
	// Matcher is the matcher to negate.
	Matcher Matcher
}

// Match returns the opposite of the wrapped matcher.
//
// Parameters:
//   - value: String to test
//
// Returns:
//   - bool: true if wrapped matcher returns false
func (m *NotMatcher) Match(value string) bool {
	return !m.Matcher.Match(value)
}

// String returns a description of the negation.
//
// Returns:
//   - string: The negated pattern prefixed with "!" (e.g., "!pattern*")
func (m *NotMatcher) String() string {
	return "!" + m.Matcher.String()
}

// NewAnyMatcher creates a matcher that matches if any sub-matcher matches.
//
// Parameters:
//   - matchers: Matchers to combine with OR logic
//
// Returns:
//   - Matcher: An AnyMatcher instance
//
// Example:
//
//	matcher := filtering.NewAnyMatcher(
//	    filtering.NewExactMatcher("lodash"),
//	    filtering.NewExactMatcher("underscore"),
//	)
func NewAnyMatcher(matchers ...Matcher) Matcher {
	return &AnyMatcher{Matchers: matchers}
}

// NewAllMatcher creates a matcher that matches only if all sub-matchers match.
//
// Parameters:
//   - matchers: Matchers to combine with AND logic
//
// Returns:
//   - Matcher: An AllMatcher instance
//
// Example:
//
//	matcher := filtering.NewAllMatcher(
//	    filtering.NewPrefixMatcher("@"),
//	    filtering.NewContainsMatcher("core"),
//	)
func NewAllMatcher(matchers ...Matcher) Matcher {
	return &AllMatcher{Matchers: matchers}
}

// NewNotMatcher creates a matcher that negates another matcher.
//
// Parameters:
//   - matcher: Matcher to negate
//
// Returns:
//   - Matcher: A NotMatcher instance
//
// Example:
//
//	matcher := filtering.NewNotMatcher(filtering.NewContainsMatcher("test"))
func NewNotMatcher(matcher Matcher) Matcher {
	return &NotMatcher{Matcher: matcher}
}

// ParseMatcher creates a matcher from a pattern string.
//
// The pattern format is interpreted as follows:
//   - "exact" - exact match
//   - "prefix*" - prefix match
//   - "*suffix" - suffix match
//   - "*contains*" - contains match
//   - "glob/**/*" - glob match (if contains * or ?)
//   - "~regex" - regex match (if starts with ~)
//   - "!pattern" - negated match
//
// Parameters:
//   - pattern: Pattern string to parse
//
// Returns:
//   - Matcher: Appropriate matcher for the pattern
//   - error: Error if pattern is invalid (e.g., bad regex)
//
// Example:
//
//	matcher, _ := filtering.ParseMatcher("@types/*")  // GlobMatcher
//	matcher, _ := filtering.ParseMatcher("~^@\\w+/")  // RegexMatcher
//	matcher, _ := filtering.ParseMatcher("!test*")   // NotMatcher(PrefixMatcher)
func ParseMatcher(pattern string) (Matcher, error) {
	// Handle negation
	if strings.HasPrefix(pattern, "!") {
		inner, err := ParseMatcher(pattern[1:])
		if err != nil {
			return nil, err
		}
		return NewNotMatcher(inner), nil
	}

	// Handle regex
	if strings.HasPrefix(pattern, "~") {
		return NewRegexMatcher(pattern[1:])
	}

	// Handle glob patterns
	if strings.ContainsAny(pattern, "*?") {
		// Check if it's a simple prefix/suffix pattern
		if strings.HasSuffix(pattern, "*") && !strings.ContainsAny(pattern[:len(pattern)-1], "*?") {
			return NewPrefixMatcher(pattern[:len(pattern)-1]), nil
		}
		if strings.HasPrefix(pattern, "*") && !strings.ContainsAny(pattern[1:], "*?") {
			return NewSuffixMatcher(pattern[1:]), nil
		}
		// Full glob pattern
		return NewGlobMatcher(pattern), nil
	}

	// Exact match
	return NewExactMatcher(pattern), nil
}

// ParseMatchers creates matchers from multiple pattern strings.
//
// Parameters:
//   - patterns: Slice of pattern strings
//
// Returns:
//   - []Matcher: Slice of matchers
//   - error: First parse error encountered
//
// Example:
//
//	matchers, err := filtering.ParseMatchers([]string{"@types/*", "lodash", "!test*"})
func ParseMatchers(patterns []string) ([]Matcher, error) {
	matchers := make([]Matcher, 0, len(patterns))
	for _, pattern := range patterns {
		m, err := ParseMatcher(pattern)
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, m)
	}
	return matchers, nil
}

// MatchAny tests if value matches any of the patterns.
//
// Parameters:
//   - value: String to test
//   - patterns: Pattern strings to match against
//
// Returns:
//   - bool: true if value matches any pattern
//   - error: Parse error if any pattern is invalid
//
// Example:
//
//	matched, _ := filtering.MatchAny("@types/node", []string{"@types/*", "lodash"})
func MatchAny(value string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
		m, err := ParseMatcher(pattern)
		if err != nil {
			return false, err
		}
		if m.Match(value) {
			return true, nil
		}
	}
	return false, nil
}

// MatchAll tests if value matches all of the patterns.
//
// Parameters:
//   - value: String to test
//   - patterns: Pattern strings that must all match
//
// Returns:
//   - bool: true if value matches all patterns
//   - error: Parse error if any pattern is invalid
//
// Example:
//
//	matched, _ := filtering.MatchAll("@angular/core", []string{"@*", "*core*"})
func MatchAll(value string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
		m, err := ParseMatcher(pattern)
		if err != nil {
			return false, err
		}
		if !m.Match(value) {
			return false, nil
		}
	}
	return true, nil
}

// Verify interface implementations.
var (
	_ Matcher = (*ExactMatcher)(nil)
	_ Matcher = (*PrefixMatcher)(nil)
	_ Matcher = (*SuffixMatcher)(nil)
	_ Matcher = (*ContainsMatcher)(nil)
	_ Matcher = (*GlobMatcher)(nil)
	_ Matcher = (*RegexMatcher)(nil)
	_ Matcher = (*AnyMatcher)(nil)
	_ Matcher = (*AllMatcher)(nil)
	_ Matcher = (*NotMatcher)(nil)
)
