package outdated

import (
	"fmt"
	"testing"

	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/formats"
)

// BenchmarkFilterNewerVersionsSemver benchmarks filtering semver versions.
//
// It measures performance with 100 version strings across multiple minor versions.
func BenchmarkFilterNewerVersionsSemver(b *testing.B) {
	versions := make([]string, 100)
	for i := 0; i < 100; i++ {
		versions[i] = fmt.Sprintf("%d.%d.%d", i/10, i%10, i%5)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FilterNewerVersions("1.0.0", versions, nil)
	}
}

// BenchmarkFilterNewerVersionsLargeSet benchmarks filtering with a large version set.
//
// It measures performance with 1000 version strings to test scaling behavior.
func BenchmarkFilterNewerVersionsLargeSet(b *testing.B) {
	versions := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		versions[i] = fmt.Sprintf("%d.%d.%d", i/100, (i/10)%10, i%10)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FilterNewerVersions("5.0.0", versions, nil)
	}
}

// BenchmarkSummarizeAvailableVersions benchmarks version summarization.
//
// It measures performance of categorizing versions into major, minor, and patch updates.
func BenchmarkSummarizeAvailableVersions(b *testing.B) {
	versions := []string{
		"1.0.1", "1.0.2", "1.1.0", "1.2.0", "2.0.0", "2.1.0", "3.0.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _ = SummarizeAvailableVersions("1.0.0", versions, nil, false)
	}
}

// BenchmarkFilterVersionsByConstraint benchmarks constraint-based version filtering.
//
// It measures performance of filtering 100 versions with caret constraint.
func BenchmarkFilterVersionsByConstraint(b *testing.B) {
	pkg := formats.Package{
		Name:       "test-pkg",
		Version:    "1.0.0",
		Constraint: "^",
	}
	versions := make([]string, 100)
	for i := 0; i < 100; i++ {
		versions[i] = fmt.Sprintf("1.%d.%d", i/10, i%10)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
	}
}

// BenchmarkVersionParsingRegex benchmarks regex-based version parsing.
//
// It measures performance of parsing versions with prerelease and build metadata.
func BenchmarkVersionParsingRegex(b *testing.B) {
	cfg := &config.VersioningCfg{
		Format: "regex",
		Regex:  `^v?(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[^+]+))?(?:\+(?P<build>.+))?$`,
	}

	strategy, _ := newVersioningStrategy(cfg)
	versions := []string{
		"v1.2.3", "2.0.0-alpha", "3.0.0+build.123", "v4.5.6-rc.1+meta",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range versions {
			strategy.parseVersion(v)
		}
	}
}
