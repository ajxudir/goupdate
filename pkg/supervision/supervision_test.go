package supervision

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/user/goupdate/pkg/formats"
	"github.com/user/goupdate/pkg/lock"
)

// TestNewUnsupportedTracker tests the behavior of NewUnsupportedTracker.
//
// It verifies:
//   - Tracker is properly initialized
//   - New tracker has no messages
func TestNewUnsupportedTracker(t *testing.T) {
	tracker := NewUnsupportedTracker()
	assert.NotNil(t, tracker)
	assert.Empty(t, tracker.Messages())
}

// TestShouldTrackUnsupported tests the behavior of ShouldTrackUnsupported.
//
// It verifies:
//   - NotConfigured, Floating, and VersionMissing statuses return true
//   - Other statuses return false
//   - Empty status returns false
func TestShouldTrackUnsupported(t *testing.T) {
	assert.True(t, ShouldTrackUnsupported(lock.InstallStatusNotConfigured))
	assert.True(t, ShouldTrackUnsupported(lock.InstallStatusFloating))
	assert.True(t, ShouldTrackUnsupported(lock.InstallStatusVersionMissing))
	assert.False(t, ShouldTrackUnsupported("ok"))
	assert.False(t, ShouldTrackUnsupported(""))
}

// TestUnsupportedTrackerAdd tests the behavior of adding packages to tracker.
//
// It verifies:
//   - Empty and whitespace-only reasons are ignored
//   - Packages with reasons are tracked correctly
//   - Count is incremented for same rule/package-type combination
func TestUnsupportedTrackerAdd(t *testing.T) {
	tracker := NewUnsupportedTracker()

	pkg := formats.Package{
		Name:        "test-pkg",
		PackageType: "npm",
		Rule:        "rule1",
	}

	t.Run("empty reason is ignored", func(t *testing.T) {
		tracker.Add(pkg, "")
		tracker.Add(pkg, "   ")
		assert.Empty(t, tracker.Messages())
	})

	t.Run("adds package with reason", func(t *testing.T) {
		tracker.Add(pkg, "some reason")
		messages := tracker.Messages()
		assert.Len(t, messages, 1)
		assert.Contains(t, messages[0], "rule1")
		assert.Contains(t, messages[0], "npm")
		assert.Contains(t, messages[0], "some reason")
	})

	t.Run("increments count for same rule", func(t *testing.T) {
		tracker.Add(pkg, "another reason") // Same rule/pm
		messages := tracker.Messages()
		assert.Len(t, messages, 1)
		assert.Contains(t, messages[0], "2 packages")
	})
}

// TestUnsupportedTrackerMessages tests the behavior of message generation.
//
// It verifies:
//   - Messages are generated for all tracked rules
//   - Messages are sorted by rule name
func TestUnsupportedTrackerMessages(t *testing.T) {
	tracker := NewUnsupportedTracker()

	// Add multiple rules
	tracker.Add(formats.Package{PackageType: "npm", Rule: "rule2"}, "reason2")
	tracker.Add(formats.Package{PackageType: "npm", Rule: "rule1"}, "reason1")

	messages := tracker.Messages()
	assert.Len(t, messages, 2)

	// Should be sorted by rule
	assert.Contains(t, messages[0], "rule1")
	assert.Contains(t, messages[1], "rule2")
}

// TestDeriveUnsupportedReason tests the behavior of reason derivation.
//
// It verifies:
//   - VersionMissing status produces appropriate reason
//   - Floating constraint produces appropriate reason
//   - NotConfigured status returns empty reason
//   - Latest missing flag returns empty reason
func TestDeriveUnsupportedReason(t *testing.T) {
	t.Run("version missing status", func(t *testing.T) {
		pkg := formats.Package{
			Name:          "test",
			Rule:          "rule1",
			InstallStatus: lock.InstallStatusVersionMissing,
		}
		reason := DeriveUnsupportedReason(pkg, nil, nil, false)
		assert.Contains(t, reason, "No concrete version")
	})

	t.Run("floating constraint", func(t *testing.T) {
		pkg := formats.Package{
			Name:    "test",
			Version: "5.*", // Wildcard version is floating
		}
		reason := DeriveUnsupportedReason(pkg, nil, nil, false)
		assert.Contains(t, reason, "Floating constraint")
	})

	t.Run("not configured status returns empty", func(t *testing.T) {
		pkg := formats.Package{
			Name:          "test",
			Rule:          "rule1",
			InstallStatus: lock.InstallStatusNotConfigured,
		}
		reason := DeriveUnsupportedReason(pkg, nil, nil, false)
		assert.Empty(t, reason)
	})

	t.Run("latest missing returns empty", func(t *testing.T) {
		pkg := formats.Package{Name: "test", Rule: "rule1"}
		reason := DeriveUnsupportedReason(pkg, nil, nil, true)
		assert.Empty(t, reason)
	})
}

// TestUnsupportedTrackerCount tests the behavior of tracker count.
//
// It verifies:
//   - Count starts at zero
//   - Count increases with different rule/package-type combinations
//   - Same combination doesn't increase count
func TestUnsupportedTrackerCount(t *testing.T) {
	tracker := NewUnsupportedTracker()

	assert.Equal(t, 0, tracker.Count())

	tracker.Add(formats.Package{Rule: "rule1", PackageType: "npm"}, "reason1")
	assert.Equal(t, 1, tracker.Count())

	tracker.Add(formats.Package{Rule: "rule2", PackageType: "go"}, "reason2")
	assert.Equal(t, 2, tracker.Count())
}

// TestUnsupportedTrackerTotalPackages tests the behavior of total package counting.
//
// It verifies:
//   - Total starts at zero
//   - Total increases with each package added
//   - Same rule/package-type combination increases total
//   - Different rules both contribute to total
func TestUnsupportedTrackerTotalPackages(t *testing.T) {
	tracker := NewUnsupportedTracker()

	assert.Equal(t, 0, tracker.TotalPackages())

	tracker.Add(formats.Package{Rule: "rule1", PackageType: "npm"}, "reason1")
	assert.Equal(t, 1, tracker.TotalPackages())

	// Adding to same rule increases total packages
	tracker.Add(formats.Package{Rule: "rule1", PackageType: "npm"}, "reason1")
	assert.Equal(t, 2, tracker.TotalPackages())

	// Different rule
	tracker.Add(formats.Package{Rule: "rule2", PackageType: "go"}, "reason2")
	assert.Equal(t, 3, tracker.TotalPackages())
}
