package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPackage is a test implementation of PackageRef
type testPackage struct {
	Name string
	Rule string
}

// GetName returns the package name.
//
// Returns:
//   - string: the package name
func (p testPackage) GetName() string { return p.Name }

// GetRule returns the rule name.
//
// Returns:
//   - string: the rule name
func (p testPackage) GetRule() string { return p.Rule }

// TestShouldUpdateIncrementally tests the behavior of ShouldUpdateIncrementally.
//
// It verifies:
//   - Packages matching regex patterns are marked as incremental
//   - Packages matching literal names are marked as incremental
//   - Non-matching packages are not marked as incremental
func TestShouldUpdateIncrementally(t *testing.T) {
	rules := map[string]PackageManagerCfg{}
	for _, ruleName := range jsRules {
		rules[ruleName] = PackageManagerCfg{Incremental: []string{"^service-.*", "legacy", "  "}}
	}

	cfg := &Config{Rules: rules}

	for _, ruleName := range jsRules {
		matches, err := ShouldUpdateIncrementally(testPackage{Name: "service-api", Rule: ruleName}, cfg)
		require.NoError(t, err)
		assert.True(t, matches)

		matches, err = ShouldUpdateIncrementally(testPackage{Name: "legacy", Rule: ruleName}, cfg)
		require.NoError(t, err)
		assert.True(t, matches)

		matches, err = ShouldUpdateIncrementally(testPackage{Name: "other", Rule: ruleName}, cfg)
		require.NoError(t, err)
		assert.False(t, matches)
	}
}

// TestShouldUpdateIncrementallyWithWildcardAndFallback tests the behavior of ShouldUpdateIncrementally with wildcards and fallbacks.
//
// It verifies:
//   - Rule wildcard patterns match all packages
//   - Global fallback patterns work when rule is missing
func TestShouldUpdateIncrementallyWithWildcardAndFallback(t *testing.T) {
	cfg := &Config{
		Rules: map[string]PackageManagerCfg{
			"npm": {Incremental: []string{".*"}},
		},
		Incremental: []string{"^global-"},
	}

	matches, err := ShouldUpdateIncrementally(testPackage{Name: "any", Rule: "npm"}, cfg)
	require.NoError(t, err)
	assert.True(t, matches, "rule wildcard should match all packages")

	matches, err = ShouldUpdateIncrementally(testPackage{Name: "global-service", Rule: "missing"}, cfg)
	require.NoError(t, err)
	assert.True(t, matches, "global fallback should match when rule is missing")
}

// TestShouldUpdateIncrementallyWithPlainName tests the behavior of ShouldUpdateIncrementally with plain package names.
//
// It verifies:
//   - Exact package name matches work
//   - Similar but different names do not match
func TestShouldUpdateIncrementallyWithPlainName(t *testing.T) {
	rules := map[string]PackageManagerCfg{}
	for _, ruleName := range jsRules {
		rules[ruleName] = PackageManagerCfg{Incremental: []string{"nginx"}}
	}

	cfg := &Config{Rules: rules}

	for _, ruleName := range jsRules {
		matches, err := ShouldUpdateIncrementally(testPackage{Name: "nginx", Rule: ruleName}, cfg)
		require.NoError(t, err)
		assert.True(t, matches)

		matches, err = ShouldUpdateIncrementally(testPackage{Name: "nginx-plus", Rule: ruleName}, cfg)
		require.NoError(t, err)
		assert.False(t, matches)
	}
}

// TestShouldUpdateIncrementallyDefaults tests the behavior of ShouldUpdateIncrementally with default values.
//
// It verifies:
//   - Nil config returns false
//   - Empty config returns false
func TestShouldUpdateIncrementallyDefaults(t *testing.T) {
	matches, err := ShouldUpdateIncrementally(testPackage{Name: "service-api"}, nil)
	require.NoError(t, err)
	assert.False(t, matches)

	matches, err = ShouldUpdateIncrementally(testPackage{Name: "service-api"}, &Config{})
	require.NoError(t, err)
	assert.False(t, matches)
}

// TestShouldUpdateIncrementallyInvalidPattern tests the behavior of ShouldUpdateIncrementally with invalid regex patterns.
//
// It verifies:
//   - Invalid regex patterns return an error
func TestShouldUpdateIncrementallyInvalidPattern(t *testing.T) {
	cfg := &Config{Rules: map[string]PackageManagerCfg{
		"npm": {Incremental: []string{"["}},
	}}
	_, err := ShouldUpdateIncrementally(testPackage{Name: "service-api", Rule: "npm"}, cfg)
	assert.Error(t, err)
}
