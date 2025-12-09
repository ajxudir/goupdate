package filtering

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExactMatcher tests the ExactMatcher struct and its Match method.
//
// It verifies that:
//   - Case-sensitive matching works correctly
//   - Case-insensitive matching works when IgnoreCase is true
//   - Partial matches do not match
//   - String() returns the pattern
func TestExactMatcher(t *testing.T) {
	t.Run("case sensitive match", func(t *testing.T) {
		m := NewExactMatcher("lodash")
		assert.True(t, m.Match("lodash"))
		assert.False(t, m.Match("Lodash"))
		assert.False(t, m.Match("lodash2"))
		assert.Equal(t, "lodash", m.String())
	})

	t.Run("case insensitive match", func(t *testing.T) {
		m := NewExactMatcherIgnoreCase("lodash")
		assert.True(t, m.Match("lodash"))
		assert.True(t, m.Match("Lodash"))
		assert.True(t, m.Match("LODASH"))
		assert.False(t, m.Match("lodash2"))
	})
}

// TestPrefixMatcher tests the PrefixMatcher struct and its Match method.
//
// It verifies that:
//   - Strings starting with prefix match
//   - Strings not starting with prefix don't match
//   - String() contains the prefix
func TestPrefixMatcher(t *testing.T) {
	m := NewPrefixMatcher("@types/")
	assert.True(t, m.Match("@types/react"))
	assert.True(t, m.Match("@types/node"))
	assert.False(t, m.Match("react"))
	assert.False(t, m.Match("types/react"))
	assert.Contains(t, m.String(), "@types/")
}

// TestSuffixMatcher tests the SuffixMatcher struct and its Match method.
//
// It verifies that:
//   - Strings ending with suffix match
//   - Strings not ending with suffix don't match
//   - String() contains the suffix
func TestSuffixMatcher(t *testing.T) {
	m := NewSuffixMatcher("-dev")
	assert.True(t, m.Match("lodash-dev"))
	assert.True(t, m.Match("react-dev"))
	assert.False(t, m.Match("dev-lodash"))
	assert.False(t, m.Match("development"))
	assert.Contains(t, m.String(), "-dev")
}

// TestContainsMatcher tests the ContainsMatcher struct and its Match method.
//
// It verifies that:
//   - Strings containing substring match
//   - Strings not containing substring don't match
//   - String() contains the substring
func TestContainsMatcher(t *testing.T) {
	m := NewContainsMatcher("test")
	assert.True(t, m.Match("testing"))
	assert.True(t, m.Match("test"))
	assert.True(t, m.Match("a-test-package"))
	assert.False(t, m.Match("production"))
	assert.Contains(t, m.String(), "test")
}

// TestGlobMatcher tests the GlobMatcher struct and its Match method.
//
// It verifies that:
//   - Wildcard suffix patterns work (e.g., "@types/*")
//   - Wildcard prefix patterns work (e.g., "*-plugin")
//   - Wildcard middle patterns work (e.g., "@*/core")
func TestGlobMatcher(t *testing.T) {
	t.Run("wildcard suffix", func(t *testing.T) {
		m := NewGlobMatcher("@types/*")
		assert.True(t, m.Match("@types/react"))
		assert.True(t, m.Match("@types/node"))
		assert.False(t, m.Match("@babel/core"))
	})

	t.Run("wildcard prefix", func(t *testing.T) {
		m := NewGlobMatcher("*-plugin")
		assert.True(t, m.Match("babel-plugin"))
		assert.True(t, m.Match("eslint-plugin"))
		assert.False(t, m.Match("plugin-test"))
	})

	t.Run("wildcard middle", func(t *testing.T) {
		m := NewGlobMatcher("@*/core")
		assert.True(t, m.Match("@babel/core"))
		assert.True(t, m.Match("@angular/core"))
		assert.False(t, m.Match("@babel/preset"))
	})
}

// TestRegexMatcher tests the RegexMatcher struct and its Match method.
//
// It verifies that:
//   - Valid regex patterns compile and match correctly
//   - Invalid regex patterns return error
//   - MustRegexMatcher panics on invalid patterns
//   - String() returns pattern with tilde prefix
func TestRegexMatcher(t *testing.T) {
	t.Run("valid regex", func(t *testing.T) {
		m, err := NewRegexMatcher("^lodash.*")
		assert.NoError(t, err)
		assert.True(t, m.Match("lodash"))
		assert.True(t, m.Match("lodash.get"))
		assert.True(t, m.Match("lodash-es"))
		assert.False(t, m.Match("my-lodash"))
	})

	t.Run("invalid regex", func(t *testing.T) {
		m, err := NewRegexMatcher("[invalid")
		assert.Error(t, err)
		assert.Nil(t, m)
	})

	t.Run("MustRegexMatcher", func(t *testing.T) {
		m := MustRegexMatcher("test.*")
		assert.NotNil(t, m)
		assert.True(t, m.Match("testing"))
	})

	t.Run("MustRegexMatcher panics on invalid", func(t *testing.T) {
		assert.Panics(t, func() {
			MustRegexMatcher("[invalid")
		})
	})
}

// TestAnyMatcher tests the AnyMatcher struct (OR composition).
//
// It verifies that:
//   - Match succeeds if any sub-matcher matches
//   - Match fails if no sub-matchers match
//   - String() shows "any(" composition
func TestAnyMatcher(t *testing.T) {
	m := NewAnyMatcher(
		NewExactMatcher("lodash"),
		NewPrefixMatcher("@types/"),
	)
	assert.True(t, m.Match("lodash"))
	assert.True(t, m.Match("@types/react"))
	assert.False(t, m.Match("react"))
	assert.Contains(t, m.String(), "any(")
}

// TestAllMatcher tests the AllMatcher struct (AND composition).
//
// It verifies that:
//   - Match succeeds only if all sub-matchers match
//   - Match fails if any sub-matcher fails
//   - String() shows "all(" composition
func TestAllMatcher(t *testing.T) {
	m := NewAllMatcher(
		NewPrefixMatcher("@"),
		NewSuffixMatcher("/core"),
	)
	assert.True(t, m.Match("@angular/core"))
	assert.True(t, m.Match("@babel/core"))
	assert.False(t, m.Match("@babel/preset"))
	assert.False(t, m.Match("core"))
	assert.Contains(t, m.String(), "all(")
}

// TestNotMatcher tests the NotMatcher struct (negation).
//
// It verifies that:
//   - Match returns inverse of wrapped matcher
//   - String() shows "!" negation prefix
func TestNotMatcher(t *testing.T) {
	m := NewNotMatcher(NewExactMatcher("test"))
	assert.False(t, m.Match("test"))
	assert.True(t, m.Match("test2"))
	assert.True(t, m.Match("production"))
	assert.Contains(t, m.String(), "!")
}

// TestParseMatcher tests the ParseMatcher function.
//
// Parameters:
//   - pattern: String pattern to parse into a Matcher
//
// It verifies that:
//   - Exact patterns create ExactMatcher
//   - Glob patterns with "*" create GlobMatcher
//   - Patterns with "~" prefix create RegexMatcher
//   - Patterns with "!" prefix create NotMatcher
//   - Invalid regex patterns return error
func TestParseMatcher(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		m, err := ParseMatcher("lodash")
		assert.NoError(t, err)
		assert.True(t, m.Match("lodash"))
		assert.False(t, m.Match("lodash2"))
	})

	t.Run("glob with wildcard suffix", func(t *testing.T) {
		m, err := ParseMatcher("@types/*")
		assert.NoError(t, err)
		assert.True(t, m.Match("@types/react"))
		assert.False(t, m.Match("react"))
	})

	t.Run("glob with wildcard prefix", func(t *testing.T) {
		m, err := ParseMatcher("*.js")
		assert.NoError(t, err)
		assert.True(t, m.Match("main.js"))
		assert.False(t, m.Match("main.ts"))
	})

	t.Run("regex with tilde prefix", func(t *testing.T) {
		m, err := ParseMatcher("~^test.*")
		assert.NoError(t, err)
		assert.True(t, m.Match("testing"))
		assert.False(t, m.Match("mytest"))
	})

	t.Run("negation", func(t *testing.T) {
		m, err := ParseMatcher("!test")
		assert.NoError(t, err)
		assert.False(t, m.Match("test"))
		assert.True(t, m.Match("production"))
	})

	t.Run("invalid regex with tilde", func(t *testing.T) {
		_, err := ParseMatcher("~[invalid")
		assert.Error(t, err)
	})

	t.Run("glob with wildcard in middle", func(t *testing.T) {
		// Pattern like "a*b" where * is in the middle (not simple prefix/suffix)
		m, err := ParseMatcher("@*/core")
		assert.NoError(t, err)
		assert.True(t, m.Match("@angular/core"))
		assert.True(t, m.Match("@babel/core"))
		assert.False(t, m.Match("@babel/preset"))
	})

	t.Run("glob with question mark", func(t *testing.T) {
		// Pattern with ? wildcard creates GlobMatcher
		m, err := ParseMatcher("test?")
		assert.NoError(t, err)
		assert.True(t, m.Match("test1"))
		assert.True(t, m.Match("testa"))
		assert.False(t, m.Match("test"))
		assert.False(t, m.Match("test12"))
	})

	t.Run("negation with invalid inner pattern", func(t *testing.T) {
		// Negation of invalid pattern should return error
		_, err := ParseMatcher("!~[invalid")
		assert.Error(t, err)
	})
}

// TestParseMatchers tests the ParseMatchers function.
//
// Parameters:
//   - patterns: Slice of pattern strings to parse
//
// It verifies that:
//   - Multiple patterns are parsed correctly
//   - Invalid patterns cause error to be returned
func TestParseMatchers(t *testing.T) {
	matchers, err := ParseMatchers([]string{"lodash", "@types/*", "~^test"})
	assert.NoError(t, err)
	assert.Len(t, matchers, 3)

	t.Run("error on invalid pattern", func(t *testing.T) {
		_, err := ParseMatchers([]string{"valid", "~[invalid"})
		assert.Error(t, err)
	})
}

// TestMatchAny tests the MatchAny convenience function.
//
// Parameters:
//   - value: String value to match against
//   - patterns: Slice of pattern strings
//
// It verifies that:
//   - Returns true if any pattern matches
//   - Returns false if no patterns match
//   - Empty patterns returns false
//   - Invalid patterns return error
func TestMatchAny(t *testing.T) {
	patterns := []string{"lodash", "@types/*"}

	matched, err := MatchAny("lodash", patterns)
	assert.NoError(t, err)
	assert.True(t, matched)

	matched, err = MatchAny("@types/react", patterns)
	assert.NoError(t, err)
	assert.True(t, matched)

	matched, err = MatchAny("react", patterns)
	assert.NoError(t, err)
	assert.False(t, matched)

	// Empty patterns should return false
	matched, err = MatchAny("anything", nil)
	assert.NoError(t, err)
	assert.False(t, matched)

	// Invalid pattern returns error
	_, err = MatchAny("value", []string{"~[invalid"})
	assert.Error(t, err)
}

// TestMatchAll tests the MatchAll convenience function.
//
// Parameters:
//   - value: String value to match against
//   - patterns: Slice of pattern strings
//
// It verifies that:
//   - Returns true only if all patterns match
//   - Returns false if any pattern fails
//   - Empty patterns returns true
//   - Invalid patterns return error
func TestMatchAll(t *testing.T) {
	// Use patterns that match together
	patterns := []string{"@*", "*core"}

	matched, err := MatchAll("@angular/core", patterns)
	assert.NoError(t, err)
	assert.True(t, matched)

	matched, err = MatchAll("@babel/preset", patterns)
	assert.NoError(t, err)
	assert.False(t, matched)

	matched, err = MatchAll("core", patterns)
	assert.NoError(t, err)
	assert.False(t, matched)

	// Empty patterns should return true
	matched, err = MatchAll("anything", nil)
	assert.NoError(t, err)
	assert.True(t, matched)

	// Invalid pattern returns error
	_, err = MatchAll("value", []string{"~[invalid"})
	assert.Error(t, err)
}

// Tests for case-insensitive matching

// TestPrefixMatcherIgnoreCase tests the PrefixMatcher with IgnoreCase=true.
//
// It verifies that:
//   - Prefix matching ignores case differences
//   - Various case combinations all match
func TestPrefixMatcherIgnoreCase(t *testing.T) {
	m := &PrefixMatcher{Prefix: "@Types/", IgnoreCase: true}
	assert.True(t, m.Match("@types/react"))
	assert.True(t, m.Match("@TYPES/node"))
	assert.True(t, m.Match("@Types/lodash"))
	assert.False(t, m.Match("react"))
}

// TestSuffixMatcherIgnoreCase tests the SuffixMatcher with IgnoreCase=true.
//
// It verifies that:
//   - Suffix matching ignores case differences
//   - Various case combinations all match
func TestSuffixMatcherIgnoreCase(t *testing.T) {
	m := &SuffixMatcher{Suffix: "-DEV", IgnoreCase: true}
	assert.True(t, m.Match("lodash-dev"))
	assert.True(t, m.Match("lodash-DEV"))
	assert.True(t, m.Match("lodash-Dev"))
	assert.False(t, m.Match("dev-lodash"))
}

// TestContainsMatcherIgnoreCase tests the ContainsMatcher with IgnoreCase=true.
//
// It verifies that:
//   - Contains matching ignores case differences
//   - Various case combinations all match
func TestContainsMatcherIgnoreCase(t *testing.T) {
	m := &ContainsMatcher{Substring: "TEST", IgnoreCase: true}
	assert.True(t, m.Match("testing"))
	assert.True(t, m.Match("TESTING"))
	assert.True(t, m.Match("TeSt"))
	assert.True(t, m.Match("my-test-package"))
	assert.False(t, m.Match("production"))
}

// Tests for String() methods

// TestGlobMatcherString tests the String method of GlobMatcher.
//
// It verifies that:
//   - String() returns the original pattern
func TestGlobMatcherString(t *testing.T) {
	m := &GlobMatcher{Pattern: "@types/*"}
	assert.Equal(t, "@types/*", m.String())
}

// TestRegexMatcherString tests the String method of RegexMatcher.
//
// It verifies that:
//   - String() returns pattern prefixed with "~"
func TestRegexMatcherString(t *testing.T) {
	m, err := NewRegexMatcher("^test.*")
	assert.NoError(t, err)
	assert.Equal(t, "~^test.*", m.String())
}

// TestRegexMatcherNilRegex tests RegexMatcher behavior with nil regex.
//
// It verifies that:
//   - Match returns false when regex is nil (edge case)
func TestRegexMatcherNilRegex(t *testing.T) {
	// Test when regex is nil (edge case)
	m := &RegexMatcher{Pattern: "test", regex: nil}
	assert.False(t, m.Match("test"))
}
