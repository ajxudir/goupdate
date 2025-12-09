package utils

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrimAndSplit(t *testing.T) {
	tests := []struct {
		input    string
		sep      string
		expected []string
	}{
		{"a,b,c", ",", []string{"a", "b", "c"}},
		{"a, b, c", ",", []string{"a", "b", "c"}},
		{"all", ",", []string{}},
		{"", ",", []string{}},
		{"  a  ,  b  ", ",", []string{"a", "b"}},
	}

	for _, tt := range tests {
		result := TrimAndSplit(tt.input, tt.sep)
		assert.Equal(t, tt.expected, result)
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}
	assert.True(t, Contains(slice, "b"))
	assert.False(t, Contains(slice, "d"))
	assert.False(t, Contains([]string{}, "a"))
}

func TestContainsIgnoreCase(t *testing.T) {
	slice := []string{"Hello", "World", "TEST"}
	assert.True(t, ContainsIgnoreCase(slice, "hello"))
	assert.True(t, ContainsIgnoreCase(slice, "HELLO"))
	assert.True(t, ContainsIgnoreCase(slice, "Hello"))
	assert.True(t, ContainsIgnoreCase(slice, "test"))
	assert.False(t, ContainsIgnoreCase(slice, "foo"))
	assert.False(t, ContainsIgnoreCase([]string{}, "a"))
}

func TestGetConstraintDisplay(t *testing.T) {
	cases := map[string]string{
		"":   "Major",
		"^":  "Compatible (^)",
		"~":  "Patch (~)",
		">=": "Min (>=)",
		"<=": "Max (<=)",
		">":  "Above (>)",
		"<":  "Below (<)",
		"=":  "Exact (=)",
		"*":  "Major (*)",
	}

	var display string
	var ok, warn bool

	for input, expected := range cases {
		display, ok, warn = GetConstraintDisplay(input)
		assert.True(t, ok)
		assert.False(t, warn)
		assert.Equal(t, expected, display)
	}

	display, ok, warn = GetConstraintDisplay("==")
	assert.True(t, ok)
	assert.False(t, warn)
	assert.Equal(t, "Exact (=)", display)

	display, ok, warn = GetConstraintDisplay("~=")
	assert.True(t, ok)
	assert.False(t, warn)
	assert.Equal(t, "Patch (~)", display)

	display, ok, warn = GetConstraintDisplay("??")
	assert.False(t, ok)
	assert.True(t, warn)
	assert.Equal(t, "#N/A", display)
}

func TestNormalizeConstraintForDisplay(t *testing.T) {
	normalized, ok, warn := normalizeConstraintForDisplay("==")
	assert.True(t, ok)
	assert.False(t, warn)
	assert.Equal(t, "=", normalized)

	normalized, ok, warn = normalizeConstraintForDisplay("~=")
	assert.True(t, ok)
	assert.False(t, warn)
	assert.Equal(t, "~", normalized)

	normalized, ok, warn = normalizeConstraintForDisplay("exact")
	assert.True(t, ok)
	assert.True(t, warn)
	assert.Equal(t, "=", normalized)

	normalized, ok, warn = normalizeConstraintForDisplay("??")
	assert.False(t, ok)
	assert.True(t, warn)
	assert.Equal(t, "??", normalized)
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		path     string
		pattern  string
		expected bool
	}{
		{"package.json", "package.json", true},
		{"src/package.json", "**/package.json", true},
		{"test.txt", "*.txt", true},
		{"test.md", "*.txt", false},
		{"exclude/file.txt", "!exclude/*", false},
		{"include/file.txt", "!exclude/*", true},
		{"deep/nested/file.json", "**/file.json", true},
		{"file.json", "**/file.json", true},
		{"[abc", "[abc", true},
	}

	for _, tt := range tests {
		result := MatchGlob(tt.path, tt.pattern)
		assert.Equal(t, tt.expected, result, "path: %s, pattern: %s", tt.path, tt.pattern)
	}
}

func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		{"*.txt", "^[^/]*\\.txt$"},
		{"**/*.go", "^(?:.*/)?[^/]*\\.go$"},
		{"test?", "^test.$"},
		{"**", "^.*$"},         // Double star at end without trailing slash
		{"dir/**", "^dir/.*$"}, // Double star at end of path
	}

	for _, tt := range tests {
		result := globToRegex(tt.pattern)
		assert.Equal(t, tt.expected, result, "pattern: %s", tt.pattern)
	}
}

func TestMatchPatterns(t *testing.T) {
	includes := []string{"*.go", "*.txt"}
	excludes := []string{"*_test.go"}

	assert.True(t, MatchPatterns("main.go", includes, excludes))
	assert.False(t, MatchPatterns("main_test.go", includes, excludes))
	assert.False(t, MatchPatterns("README.md", includes, excludes))
}

func TestMapConstraint(t *testing.T) {
	mappings := map[string]string{
		"~=": "~",
		"==": "=",
	}

	assert.Equal(t, "~", MapConstraint("~=", mappings))
	assert.Equal(t, "=", MapConstraint("==", mappings))
	assert.Equal(t, ">=", MapConstraint(">=", mappings))
}

func TestExtractNamedGroups(t *testing.T) {
	pattern := `(?P<name>\w+)\s+(?P<version>\d+\.\d+)`
	text := "package 1.2"

	result, err := ExtractNamedGroups(pattern, text)
	assert.NoError(t, err)
	assert.Equal(t, "package", result["name"])
	assert.Equal(t, "1.2", result["version"])

	// Test no match
	result, err = ExtractNamedGroups(pattern, "no match")
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Test invalid pattern
	_, err = ExtractNamedGroups(`(?P<name`, text)
	assert.Error(t, err)
}

func TestExtractAllMatches(t *testing.T) {
	pattern := `(?P<name>\w+)\s+(?P<version>\d+\.\d+)`
	text := "package1 1.0\npackage2 2.0"

	results, err := ExtractAllMatches(pattern, text)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "package1", results[0]["name"])
	assert.Equal(t, "package2", results[1]["name"])

	// Test no matches
	results, err = ExtractAllMatches(pattern, "no matches")
	assert.NoError(t, err)
	assert.Nil(t, results)

	// Test invalid regex
	_, err = ExtractAllMatches("[invalid", "text")
	assert.Error(t, err)

	// Test pattern with no named groups - matches but result is empty
	results, err = ExtractAllMatches(`\d+`, "123 456")
	assert.NoError(t, err)
	// Results should be nil or empty since there are no named groups to capture
	assert.Empty(t, results)
}

func TestXMLFunctions(t *testing.T) {
	root := &XMLNode{
		XMLName: xml.Name{Local: "root"},
		Nodes: []XMLNode{
			{
				XMLName: xml.Name{Local: "child"},
				Content: "  text  ",
				Attrs: []xml.Attr{
					{Name: xml.Name{Local: "id"}, Value: "123"},
				},
				Nodes: []XMLNode{
					{XMLName: xml.Name{Local: "nested"}, Content: "nested text"},
				},
			},
		},
	}

	// Test FindXMLNodes
	nodes := FindXMLNodes(root, "child")
	assert.Len(t, nodes, 1)

	nodes = FindXMLNodes(root, "child/nested")
	assert.Len(t, nodes, 1)

	nodes = FindXMLNodes(root, "notfound")
	assert.Len(t, nodes, 0)

	// Test GetXMLNodeText
	node := &root.Nodes[0]
	assert.Equal(t, "text", GetXMLNodeText(node))
	assert.Equal(t, "", GetXMLNodeText(nil))

	// Test GetXMLAttr
	assert.Equal(t, "123", GetXMLAttr(node, "id"))
	assert.Equal(t, "", GetXMLAttr(node, "notfound"))

	// Test with empty path - should return the input nodes
	nodes = findNodesRecursive([]*XMLNode{root}, []string{})
	assert.Len(t, nodes, 1)
	assert.Equal(t, "root", nodes[0].XMLName.Local)

	// Test with empty nodes - should return empty
	nodes = findNodesRecursive([]*XMLNode{}, []string{"child"})
	assert.Empty(t, nodes)
}

func TestNormalizePath(t *testing.T) {
	assert.Equal(t, "path/to/file", NormalizePath("path/to/file"))
	assert.Equal(t, "path/to/file", NormalizePath("./path/to/file"))
	assert.Equal(t, "path/to/file", NormalizePath("path//to//file"))
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input              string
		expectedVersion    string
		expectedConstraint string
	}{
		{"^1.2.3", "1.2.3", "^"},
		{"~2.0.0", "2.0.0", "~"},
		{">=1.0.0", "1.0.0", ">="},
		{"1.2.3", "1.2.3", ""},
		{"*", "*", "*"},
		{"latest", "latest", ""},
		{"v1.2.3", "1.2.3", ""},
		{"${VERSION:-2.3.4}", "2.3.4", ""},
		{"1.0.0 - 2.0.0", "1.0.0 - 2.0.0", ""},
		{"1.0.0 || 2.0.0", "1.0.0 || 2.0.0", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseVersion(tt.input)
			assert.Equal(t, tt.expectedVersion, result.Version)
			assert.Equal(t, tt.expectedConstraint, result.Constraint)
		})
	}
}

func TestFindFilesByPatterns(t *testing.T) {
	baseDir, _ := filepath.Abs("../testdata")
	files, err := FindFilesByPatterns(baseDir, []string{"**/package-lock.json"})
	require.NoError(t, err)
	assert.NotEmpty(t, files)

	found := false
	for _, f := range files {
		if filepath.Base(f) == "package-lock.json" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestFindFilesByPatternsOptions(t *testing.T) {
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(originalWD) }()

	require.NoError(t, os.WriteFile("root.log", []byte("root"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "nested"), 0o755))
	nestedFile := filepath.Join(tmpDir, "nested", "file.txt")
	require.NoError(t, os.WriteFile(nestedFile, []byte("nested"), 0o644))

	skipDir := filepath.Join(tmpDir, "node_modules")
	require.NoError(t, os.MkdirAll(skipDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skipDir, "ignored.txt"), []byte("ignore"), 0o644))

	patterns := []string{"", "*.txt", "root.log"}
	matches, err := FindFilesByPatterns("", patterns)
	require.NoError(t, err)

	assert.Contains(t, matches, filepath.Join("nested", "file.txt"))
	assert.Contains(t, matches, "root.log")
	for _, match := range matches {
		assert.NotContains(t, match, "node_modules")
	}
}

func TestExtractAllMatchesWithIndex(t *testing.T) {
	t.Run("valid pattern with multiple matches", func(t *testing.T) {
		text := `"lodash": {"version": "4.17.21"}, "express": {"version": "4.18.2"}`
		pattern := `"(?P<name>[^"]+)":\s*\{[^}]*"version":\s*"(?P<version>[^"]+)"`
		results, err := ExtractAllMatchesWithIndex(pattern, text)
		require.NoError(t, err)
		require.Len(t, results, 2)

		assert.Equal(t, "lodash", results[0].Groups["name"])
		assert.Equal(t, "4.17.21", results[0].Groups["version"])
		assert.True(t, results[0].Start >= 0)
		assert.True(t, results[0].End > results[0].Start)
		assert.NotEmpty(t, results[0].FullMatch)

		assert.Equal(t, "express", results[1].Groups["name"])
		assert.Equal(t, "4.18.2", results[1].Groups["version"])
	})

	t.Run("invalid regex pattern", func(t *testing.T) {
		_, err := ExtractAllMatchesWithIndex("[invalid", "text")
		assert.Error(t, err)
	})

	t.Run("no matches", func(t *testing.T) {
		results, err := ExtractAllMatchesWithIndex(`(?P<name>\d+)`, "no numbers here")
		require.NoError(t, err)
		assert.Nil(t, results)
	})

	t.Run("match with position indices", func(t *testing.T) {
		text := "version: 1.2.3"
		pattern := `version:\s*(?P<ver>[\d.]+)`
		results, err := ExtractAllMatchesWithIndex(pattern, text)
		require.NoError(t, err)
		require.Len(t, results, 1)

		assert.Equal(t, "1.2.3", results[0].Groups["ver"])
		// Check that the index is correct
		verIdx := results[0].GroupIndex["ver"]
		assert.Equal(t, "1.2.3", text[verIdx[0]:verIdx[1]])
	})
}

func TestValidateRegexSafety(t *testing.T) {
	t.Run("valid safe patterns", func(t *testing.T) {
		safePatterns := []string{
			`(?P<name>\w+)`,
			`(?P<version>\d+\.\d+\.\d+)`,
			`^prefix.*suffix$`,
			`\s+\d+\s+`,
			`[a-zA-Z0-9_]+`,
			// Real-world patterns from the codebase
			`(?m)^(?P<n>[\w\-]+)\s*(?P<constraint>[<>=~]+)?\s*(?P<version>[\d\.]+)`,
			`(?P<constraint>[<>=~]+)?`,
		}

		for _, p := range safePatterns {
			err := ValidateRegexSafety(p)
			assert.NoError(t, err, "pattern should be safe: %s", p)
		}
	})

	t.Run("reject nested quantifiers with wildcards", func(t *testing.T) {
		// These are classic ReDoS patterns with wildcards inside groups
		unsafePatterns := []string{
			`(.*)+`,  // Wildcard .* with outer quantifier
			`(.+)+`,  // Wildcard .+ with outer quantifier
			`(\w*)+`, // Word char wildcard with outer quantifier
			`(\s+)+`, // Whitespace with outer quantifier
		}

		for _, p := range unsafePatterns {
			err := ValidateRegexSafety(p)
			assert.ErrorIs(t, err, ErrRegexTooComplex, "pattern should be rejected: %s", p)
		}
	})

	t.Run("reject simple nested quantifiers", func(t *testing.T) {
		unsafePatterns := []string{
			`(a+)+`, // Simple nested quantifier
			`(x*)+`, // Simple nested quantifier
		}

		for _, p := range unsafePatterns {
			err := ValidateRegexSafety(p)
			assert.ErrorIs(t, err, ErrRegexTooComplex, "pattern should be rejected: %s", p)
		}
	})

	t.Run("reject excessively long patterns", func(t *testing.T) {
		// Create a pattern longer than DefaultMaxRegexPatternLength
		longPattern := make([]byte, DefaultMaxRegexPatternLength+1)
		for i := range longPattern {
			longPattern[i] = 'a'
		}
		err := ValidateRegexSafety(string(longPattern))
		assert.ErrorIs(t, err, ErrRegexTooComplex)
	})

	t.Run("reject excessive quantifiers", func(t *testing.T) {
		// Pattern with more than 15 quantifiers
		excessivePattern := `a+b+c+d+e+f+g+h+i+j+k+l+m+n+o+p+`
		err := ValidateRegexSafety(excessivePattern)
		assert.ErrorIs(t, err, ErrRegexTooComplex)
	})

	t.Run("reject overlapping alternatives with quantifiers", func(t *testing.T) {
		// Overlapping alternatives can cause exponential backtracking
		unsafePatterns := []string{
			`(a|aa)+`,   // Overlapping alternatives
			`(ab|abc)+`, // Prefix overlap
		}

		for _, p := range unsafePatterns {
			err := ValidateRegexSafety(p)
			assert.ErrorIs(t, err, ErrRegexTooComplex, "pattern should be rejected: %s", p)
		}
	})

	t.Run("allow distinct alternatives", func(t *testing.T) {
		// Non-overlapping alternatives should be allowed
		safePatterns := []string{
			`(x|y)+`,     // Distinct alternatives
			`(cat|dog)+`, // No common prefix
		}

		for _, p := range safePatterns {
			err := ValidateRegexSafety(p)
			assert.NoError(t, err, "pattern should be safe: %s", p)
		}
	})
}

func TestExtractFunctionsRejectUnsafePatterns(t *testing.T) {
	unsafePattern := `(a+)+`
	text := "aaaaaa"

	t.Run("ExtractNamedGroups rejects unsafe pattern", func(t *testing.T) {
		_, err := ExtractNamedGroups(unsafePattern, text)
		assert.ErrorIs(t, err, ErrRegexTooComplex)
	})

	t.Run("ExtractAllMatches rejects unsafe pattern", func(t *testing.T) {
		_, err := ExtractAllMatches(unsafePattern, text)
		assert.ErrorIs(t, err, ErrRegexTooComplex)
	})

	t.Run("ExtractAllMatchesWithIndex rejects unsafe pattern", func(t *testing.T) {
		_, err := ExtractAllMatchesWithIndex(unsafePattern, text)
		assert.ErrorIs(t, err, ErrRegexTooComplex)
	})
}
