package filtering

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ajxudir/goupdate/pkg/formats"
)

// TestParseFileFilterPatterns tests the behavior of ParseFileFilterPatterns.
//
// It verifies:
//   - Empty filter returns nil slices
//   - Single include pattern is parsed correctly
//   - Multiple include patterns are parsed correctly
//   - Exclude patterns (prefixed with !) are parsed correctly
//   - Mixed include and exclude patterns are handled correctly
//   - Whitespace is trimmed from patterns
func TestParseFileFilterPatterns(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		wantInclude []string
		wantExclude []string
	}{
		{
			name:        "empty filter",
			filter:      "",
			wantInclude: nil,
			wantExclude: nil,
		},
		{
			name:        "single include pattern",
			filter:      "go.mod",
			wantInclude: []string{"go.mod"},
			wantExclude: nil,
		},
		{
			name:        "multiple include patterns",
			filter:      "go.mod,package.json",
			wantInclude: []string{"go.mod", "package.json"},
			wantExclude: nil,
		},
		{
			name:        "single exclude pattern",
			filter:      "!**/testdata/**",
			wantInclude: nil,
			wantExclude: []string{"**/testdata/**"},
		},
		{
			name:        "mixed include and exclude",
			filter:      "go.mod,!**/testdata/**,package.json",
			wantInclude: []string{"go.mod", "package.json"},
			wantExclude: []string{"**/testdata/**"},
		},
		{
			name:        "handles whitespace",
			filter:      " go.mod , !test/** , package.json ",
			wantInclude: []string{"go.mod", "package.json"},
			wantExclude: []string{"test/**"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := ParseFileFilterPatterns(tt.filter)
			assert.Equal(t, tt.wantInclude, patterns.Include)
			assert.Equal(t, tt.wantExclude, patterns.Exclude)
		})
	}
}

// TestMatchesFileFilter tests the behavior of MatchesFileFilter.
//
// It verifies:
//   - No patterns matches all files
//   - Include patterns match correctly
//   - Include patterns reject non-matching files
//   - Exclude patterns reject matching files
//   - Exclude patterns allow non-matching files
//   - Glob patterns with ** work correctly
//   - Exclude takes priority over include
func TestMatchesFileFilter(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		patterns FileFilterPatterns
		want     bool
	}{
		{
			name:     "no patterns matches all",
			path:     "any/path/file.json",
			patterns: FileFilterPatterns{},
			want:     true,
		},
		{
			name:     "include pattern matches",
			path:     "go.mod",
			patterns: FileFilterPatterns{Include: []string{"go.mod"}},
			want:     true,
		},
		{
			name:     "include pattern no match",
			path:     "package.json",
			patterns: FileFilterPatterns{Include: []string{"go.mod"}},
			want:     false,
		},
		{
			name:     "exclude pattern rejects",
			path:     "pkg/testdata/go.mod",
			patterns: FileFilterPatterns{Exclude: []string{"**/testdata/**"}},
			want:     false,
		},
		{
			name:     "exclude pattern allows",
			path:     "go.mod",
			patterns: FileFilterPatterns{Exclude: []string{"**/testdata/**"}},
			want:     true,
		},
		{
			name:     "glob pattern with **",
			path:     "src/deep/nested/file.json",
			patterns: FileFilterPatterns{Include: []string{"**/*.json"}},
			want:     true,
		},
		{
			name:     "exclude takes priority over include",
			path:     "testdata/go.mod",
			patterns: FileFilterPatterns{Include: []string{"*.mod"}, Exclude: []string{"testdata/*"}},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesFileFilter(tt.path, tt.patterns)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestFilterDetectedFiles tests the behavior of FilterDetectedFiles.
//
// It verifies:
//   - No filter returns all detected files
//   - Include filter selects only matching files
//   - Exclude filter removes matching files
//   - Empty file path uses basename fallback
//   - Rules are removed when all files are filtered out
func TestFilterDetectedFiles(t *testing.T) {
	baseDir := "/project"
	detected := map[string][]string{
		"mod": {"/project/go.mod", "/project/pkg/testdata/go.mod"},
		"npm": {"/project/package.json", "/project/examples/package.json"},
	}

	t.Run("no filter returns all", func(t *testing.T) {
		result := FilterDetectedFiles(detected, "", baseDir)
		assert.Equal(t, detected, result)
	})

	t.Run("include filter", func(t *testing.T) {
		result := FilterDetectedFiles(detected, "go.mod", baseDir)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "mod")
		assert.Len(t, result["mod"], 1)
	})

	t.Run("exclude filter", func(t *testing.T) {
		result := FilterDetectedFiles(detected, "!**/testdata/**,!**/examples/**", baseDir)
		assert.Len(t, result["mod"], 1)
		assert.Len(t, result["npm"], 1)
	})

	t.Run("handles empty file path with basename fallback", func(t *testing.T) {
		emptyPathFiles := map[string][]string{
			"mod": {""}, // empty path triggers the fallback
		}
		// Empty string passed to filepath.Base returns ".", so this tests the fallback path
		result := FilterDetectedFiles(emptyPathFiles, ".", "/project")
		assert.Len(t, result, 1)
		assert.Contains(t, result["mod"], "")
	})

	t.Run("rule removed when all files filtered out", func(t *testing.T) {
		result := FilterDetectedFiles(detected, "!**/*", baseDir)
		assert.Empty(t, result)
	})
}

// TestFilterPackagesByFile tests the behavior of FilterPackagesByFile.
//
// It verifies:
//   - No filter returns all packages
//   - Include filter selects only matching packages
//   - Exclude filter removes matching packages
//   - Glob patterns work correctly
//   - Empty source path uses basename fallback
func TestFilterPackagesByFile(t *testing.T) {
	baseDir := "/project"
	pkgs := []formats.Package{
		{Name: "pkg1", Source: "/project/go.mod"},
		{Name: "pkg2", Source: "/project/testdata/go.mod"},
		{Name: "pkg3", Source: "/project/package.json"},
	}

	t.Run("no filter returns all", func(t *testing.T) {
		result := FilterPackagesByFile(pkgs, "", baseDir)
		assert.Len(t, result, 3)
	})

	t.Run("include filter", func(t *testing.T) {
		result := FilterPackagesByFile(pkgs, "go.mod", baseDir)
		assert.Len(t, result, 1)
		assert.Equal(t, "pkg1", result[0].Name)
	})

	t.Run("exclude filter", func(t *testing.T) {
		result := FilterPackagesByFile(pkgs, "!**/testdata/**", baseDir)
		assert.Len(t, result, 2)
	})

	t.Run("glob pattern", func(t *testing.T) {
		result := FilterPackagesByFile(pkgs, "**/*.json", baseDir)
		assert.Len(t, result, 1)
		assert.Equal(t, "pkg3", result[0].Name)
	})

	t.Run("empty source path uses basename fallback", func(t *testing.T) {
		// Package with empty Source to trigger relPath == "" fallback
		pkgsWithEmpty := []formats.Package{
			{Name: "pkg1", Source: ""},
		}
		// When relPath is empty after filepath.Rel, it uses filepath.Base
		result := FilterPackagesByFile(pkgsWithEmpty, ".", "/project")
		// filepath.Base("") returns ".", so matching "." pattern works
		assert.Len(t, result, 1)
	})
}
