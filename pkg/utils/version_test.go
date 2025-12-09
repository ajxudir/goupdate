package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/user/goupdate/pkg/config"
)

func strPtr(s string) *string {
	return &s
}

func TestValidateConstraint(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		expected   string
	}{
		{"empty constraint", "", ""},
		{"caret", "^", "^"},
		{"tilde", "~", "~"},
		{"greater than or equal", ">=", ">="},
		{"less than or equal", "<=", "<="},
		{"greater than", ">", ">"},
		{"less than", "<", "<"},
		{"exact", "=", "="},
		{"wildcard", "*", "*"},
		{"invalid single char", "x", ""},
		{"invalid multi char", ">>>", ""},
		{"invalid tilde-greater", "~>", ""},
		{"invalid random", "invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateConstraint(tt.constraint, "test-pkg")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyPackageOverride(t *testing.T) {
	t.Run("no config", func(t *testing.T) {
		vInfo := VersionInfo{Version: "1.0.0", Constraint: "^"}
		result := ApplyPackageOverride("pkg", vInfo, nil)
		assert.Equal(t, vInfo, result)
	})

	t.Run("no overrides map", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{Manager: "js"}
		vInfo := VersionInfo{Version: "1.0.0", Constraint: "^"}
		result := ApplyPackageOverride("pkg", vInfo, cfg)
		assert.Equal(t, vInfo, result)
	})

	t.Run("package not in overrides", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{
			Manager: "js",
			PackageOverrides: map[string]config.PackageOverrideCfg{
				"other": {Version: "2.0.0"},
			},
		}
		vInfo := VersionInfo{Version: "1.0.0", Constraint: "^"}
		result := ApplyPackageOverride("pkg", vInfo, cfg)
		assert.Equal(t, vInfo, result)
	})

	t.Run("override constraint only", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{
			Manager: "js",
			PackageOverrides: map[string]config.PackageOverrideCfg{
				"pkg": {Constraint: strPtr("~")},
			},
		}
		vInfo := VersionInfo{Version: "1.0.0", Constraint: "^"}
		result := ApplyPackageOverride("pkg", vInfo, cfg)
		assert.Equal(t, "1.0.0", result.Version)
		assert.Equal(t, "~", result.Constraint)
	})

	t.Run("override version only", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{
			Manager: "js",
			PackageOverrides: map[string]config.PackageOverrideCfg{
				"pkg": {Version: "2.0.0"},
			},
		}
		vInfo := VersionInfo{Version: "1.0.0", Constraint: "^"}
		result := ApplyPackageOverride("pkg", vInfo, cfg)
		assert.Equal(t, "2.0.0", result.Version)
		assert.Equal(t, "^", result.Constraint)
	})

	t.Run("override both version and constraint", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{
			Manager: "js",
			PackageOverrides: map[string]config.PackageOverrideCfg{
				"pkg": {Version: "2.0.0", Constraint: strPtr("")},
			},
		}
		vInfo := VersionInfo{Version: "1.0.0", Constraint: "^"}
		result := ApplyPackageOverride("pkg", vInfo, cfg)
		assert.Equal(t, "2.0.0", result.Version)
		assert.Equal(t, "", result.Constraint)
	})

	t.Run("invalid constraint validates to exact", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{
			Manager: "js",
			PackageOverrides: map[string]config.PackageOverrideCfg{
				"pkg": {Constraint: strPtr("invalid")},
			},
		}
		vInfo := VersionInfo{Version: "1.0.0", Constraint: "^"}
		result := ApplyPackageOverride("pkg", vInfo, cfg)
		assert.Equal(t, "1.0.0", result.Version)
		assert.Equal(t, "", result.Constraint) // invalid becomes exact
	})
}

func TestNormalizeDeclaredVersion(t *testing.T) {
	t.Run("empty version defaults to latest", func(t *testing.T) {
		vInfo := NormalizeDeclaredVersion("pkg", VersionInfo{}, nil)
		assert.Equal(t, "*", vInfo.Version)
	})

	t.Run("maps configured latest token", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{LatestMapping: &config.LatestMappingCfg{Default: map[string]string{"rolling": "*"}}}
		vInfo := NormalizeDeclaredVersion("pkg", VersionInfo{Version: "rolling"}, cfg)
		assert.Equal(t, "*", vInfo.Version)
	})

	t.Run("normalizes default latest value", func(t *testing.T) {
		vInfo := NormalizeDeclaredVersion("pkg", VersionInfo{Version: "latest"}, nil)
		assert.Equal(t, "*", vInfo.Version)
	})

	t.Run("honors custom empty mapping", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{LatestMapping: &config.LatestMappingCfg{Default: map[string]string{"": "latest"}}}
		vInfo := NormalizeDeclaredVersion("pkg", VersionInfo{}, cfg)
		assert.Equal(t, "latest", vInfo.Version)
	})

	t.Run("applies package scoped mappings", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{LatestMapping: &config.LatestMappingCfg{Packages: map[string]map[string]string{"special": {"edge": "latest"}}}}
		vInfo := NormalizeDeclaredVersion("special", VersionInfo{Version: "edge"}, cfg)
		assert.Equal(t, "latest", vInfo.Version)

		other := NormalizeDeclaredVersion("other", VersionInfo{Version: "edge"}, cfg)
		assert.Equal(t, "edge", other.Version)
	})

	t.Run("supports multiple tokens per package", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{LatestMapping: &config.LatestMappingCfg{Packages: map[string]map[string]string{"duo": {"edge": "*", "rolling": "*"}}}}
		vInfo := NormalizeDeclaredVersion("duo", VersionInfo{Version: "rolling"}, cfg)
		assert.Equal(t, "*", vInfo.Version)
	})
}

func TestIsLatestIndicator(t *testing.T) {
	t.Run("default latest is asterisk", func(t *testing.T) {
		assert.True(t, IsLatestIndicator("*", "pkg", nil))
		assert.False(t, IsLatestIndicator("1.0.0", "pkg", nil))
	})

	t.Run("custom latest value", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{LatestMapping: &config.LatestMappingCfg{Default: map[string]string{"": "latest"}}}
		assert.True(t, IsLatestIndicator("latest", "pkg", cfg))
		assert.False(t, IsLatestIndicator("*", "pkg", cfg))
	})
}

func TestIsFloatingConstraint(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		// Not floating - exact versions
		{"empty string", "", false},
		{"exact version", "1.0.0", false},
		{"exact semver", "2.3.4", false},
		{"version with v prefix", "v1.2.3", false},
		{"version with build metadata", "1.0.0+build123", false},
		{"prerelease version", "1.0.0-alpha.1", false},

		// Floating - pure wildcard
		{"pure wildcard", "*", true},

		// Floating - embedded wildcards
		{"major wildcard", "5.*", true},
		{"minor wildcard", "5.4.*", true},
		{"patch wildcard", "1.2.*", true},
		{"x notation major", "5.x", true},
		{"x notation minor", "5.4.x", true},
		{"trailing wildcard without dot", "5*", true},

		// Floating - NuGet/MSBuild ranges
		{"nuget inclusive range", "[8.0.0,9.0.0)", true},
		{"nuget exclusive range", "(1.0,2.0]", true},
		{"nuget inclusive both", "[1.0.0,2.0.0]", true},
		{"nuget exclusive both", "(1.0.0,2.0.0)", true},

		// Floating - compound constraints (min AND max)
		{"npm compound range", ">=1.0.0 <2.0.0", true},
		{"composer range", ">=3.0,<4.0", true},
		{"pip compatible with upper", ">1.0.0,<=2.0.0", true},

		// NOT floating - single-sided constraints (can be updated normally)
		{"greater than only", ">1.0.0", false},
		{"greater equal only", ">=1.0.0", false},
		{"less than only", "<2.0.0", false},
		{"less equal only", "<=2.0.0", false},

		// Floating - OR constraints
		{"or constraint pipe", "^2.0|^3.0", true},
		{"or constraint double pipe", ">=1.0 || <0.5", true},

		// Edge cases
		{"whitespace only", "   ", false},
		{"version with spaces", " 1.0.0 ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFloatingConstraint(tt.version)
			assert.Equal(t, tt.expected, result, "IsFloatingConstraint(%q) = %v, want %v", tt.version, result, tt.expected)
		})
	}
}

func TestIsFloatingConstraintRealWorldExamples(t *testing.T) {
	// Test real-world examples from various package managers
	tests := []struct {
		manager  string
		version  string
		floating bool
	}{
		// npm/yarn examples
		{"npm", "^1.0.0", false},        // caret is NOT floating - it's a constraint type
		{"npm", "~1.0.0", false},        // tilde is NOT floating
		{"npm", "1.x", true},            // x-range IS floating
		{"npm", "1.2.x", true},          // x-range IS floating
		{"npm", "*", true},              // wildcard IS floating
		{"npm", ">=1.0.0 <2.0.0", true}, // range IS floating

		// composer examples
		{"composer", "^1.0", false},
		{"composer", "~1.0", false},
		{"composer", "1.*", true},
		{"composer", ">=1.0,<2.0", true},

		// NuGet/MSBuild examples
		{"nuget", "8.0.0", false},
		{"nuget", "8.*", true},
		{"nuget", "[8.0.0,9.0.0)", true},
		{"nuget", ">=8.0.0", false}, // single-sided, can be updated

		// Go mod examples (go doesn't really have floating, but test edge cases)
		{"go", "v1.2.3", false},
		{"go", "v0.0.0-20210101000000-abcdef123456", false},

		// pip examples
		{"pip", ">=1.0.0,<2.0.0", true},
		{"pip", "~=1.4.2", false}, // compatible release, not floating
		{"pip", "==1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.manager+"_"+tt.version, func(t *testing.T) {
			result := IsFloatingConstraint(tt.version)
			assert.Equal(t, tt.floating, result, "%s: IsFloatingConstraint(%q) = %v, want %v", tt.manager, tt.version, result, tt.floating)
		})
	}
}
