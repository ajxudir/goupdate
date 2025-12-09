package supervision

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/constants"
	"github.com/user/goupdate/pkg/formats"
	"github.com/user/goupdate/pkg/lock"
	"github.com/user/goupdate/pkg/utils"
	"github.com/user/goupdate/pkg/verbose"
)

// UnsupportedRuleInfo holds information about an unsupported rule.
//
// Fields:
//   - Rule: Configuration rule name (e.g., "npm-packages")
//   - PackageType: Package manager type (e.g., "npm", "go")
//   - Reason: Human-readable explanation
//   - Count: Number of packages affected
type UnsupportedRuleInfo struct {
	Rule        string
	PackageType string
	Reason      string
	Count       int
}

// UnsupportedTracker collects unique unsupported reasons grouped by rule.
//
// It is safe for concurrent use. Packages are grouped by their rule and
// package type combination, with counts aggregated for each unique reason.
type UnsupportedTracker struct {
	mu    sync.RWMutex
	rules map[string]*UnsupportedRuleInfo
}

// NewUnsupportedTracker creates a new UnsupportedTracker.
//
// Returns:
//   - *UnsupportedTracker: Initialized tracker ready for use
//
// Example:
//
//	tracker := supervision.NewUnsupportedTracker()
func NewUnsupportedTracker() *UnsupportedTracker {
	return &UnsupportedTracker{rules: make(map[string]*UnsupportedRuleInfo)}
}

// ShouldTrackUnsupported returns true if the status indicates the package should be tracked.
//
// Trackable statuses include:
//   - InstallStatusNotConfigured: Lock file not configured
//   - InstallStatusFloating: Floating version constraint
//   - InstallStatusVersionMissing: No concrete version found
//
// Parameters:
//   - status: Package install status string
//
// Returns:
//   - bool: true if package should be tracked as unsupported
//
// Example:
//
//	if supervision.ShouldTrackUnsupported(pkg.InstallStatus) {
//	    tracker.Add(pkg, reason)
//	}
func ShouldTrackUnsupported(status string) bool {
	return strings.EqualFold(status, lock.InstallStatusNotConfigured) ||
		strings.EqualFold(status, lock.InstallStatusFloating) ||
		strings.EqualFold(status, lock.InstallStatusVersionMissing)
}

// Add tracks an unsupported package with a reason.
//
// Packages are grouped by their rule and package type combination.
// If a package with the same combination already exists, the count is
// incremented. Empty reasons are ignored.
//
// Parameters:
//   - p: Package to track
//   - reason: Human-readable reason for not supporting updates
//
// Example:
//
//	tracker.Add(pkg, "No lock file configured")
func (t *UnsupportedTracker) Add(p formats.Package, reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return
	}

	key := fmt.Sprintf("%s|%s", p.PackageType, p.Rule)

	t.mu.Lock()
	defer t.mu.Unlock()

	if info, exists := t.rules[key]; exists {
		info.Count++
		return
	}

	t.rules[key] = &UnsupportedRuleInfo{
		Rule:        p.Rule,
		PackageType: p.PackageType,
		Reason:      reason,
		Count:       1,
	}
}

// Messages returns formatted messages for all tracked unsupported rules.
//
// Messages are sorted by rule name, then by package type. Each message
// includes an icon, rule name, package type, reason, and package count.
//
// Returns:
//   - []string: Formatted messages, or nil if no packages tracked
//
// Example:
//
//	messages := tracker.Messages()
//	for _, msg := range messages {
//	    fmt.Println(msg)
//	}
//	// Output: 🚫 npm-packages (npm): No lock file configured (5 packages)
func (t *UnsupportedTracker) Messages() []string {
	t.mu.RLock()
	if len(t.rules) == 0 {
		t.mu.RUnlock()
		return nil
	}

	entries := make([]*UnsupportedRuleInfo, 0, len(t.rules))
	for _, info := range t.rules {
		entries = append(entries, info)
	}
	t.mu.RUnlock()

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Rule != entries[j].Rule {
			return entries[i].Rule < entries[j].Rule
		}
		return entries[i].PackageType < entries[j].PackageType
	})

	messages := make([]string, 0, len(entries))
	for _, entry := range entries {
		messages = append(messages, fmt.Sprintf("%s %s (%s): %s (%d packages)",
			constants.IconBlocked, entry.Rule, entry.PackageType, entry.Reason, entry.Count))
	}

	return messages
}

// Count returns the total number of unique rule/package-type combinations tracked.
//
// Returns:
//   - int: Number of tracked entries
func (t *UnsupportedTracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.rules)
}

// TotalPackages returns the total number of packages tracked across all rules.
//
// Returns:
//   - int: Sum of all package counts
func (t *UnsupportedTracker) TotalPackages() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	total := 0
	for _, info := range t.rules {
		total += info.Count
	}
	return total
}

// DeriveUnsupportedReason determines the reason why a package cannot be updated.
//
// It returns a human-readable message for unsupported packages based on their
// status and version constraints. Returns empty string if no specific reason
// can be determined.
//
// Parameters:
//   - p: Package to analyze
//   - cfg: Configuration (reserved for future use)
//   - err: Error from previous operations (reserved for future use)
//   - latestMissing: true if outdated commands are not available
//
// Returns:
//   - string: Human-readable reason, or empty string
//
// Example:
//
//	reason := supervision.DeriveUnsupportedReason(pkg, cfg, nil, false)
//	if reason != "" {
//	    tracker.Add(pkg, reason)
//	}
func DeriveUnsupportedReason(p formats.Package, _ *config.Config, _ error, latestMissing bool) string {
	// VersionMissing status - no concrete version could be determined
	if strings.EqualFold(p.InstallStatus, lock.InstallStatusVersionMissing) {
		verbose.UnsupportedHelp(p.Rule, "lock")
		return "No concrete version found in manifest or lock file."
	}

	// Floating constraints (5.*, >=8.0.0, [8.0.0,9.0.0), etc.) cannot be updated automatically
	if utils.IsFloatingConstraint(p.Version) {
		verbose.Infof("Package '%s' has floating constraint '%s' - cannot auto-update", p.Name, p.Version)
		return fmt.Sprintf("Floating constraint '%s' - update manually or remove constraint.", p.Version)
	}

	// NotConfigured status - provide verbose help for configuration
	if strings.EqualFold(p.InstallStatus, lock.InstallStatusNotConfigured) {
		verbose.UnsupportedHelp(p.Rule, "lock")
	}

	// Check if outdated commands are missing
	if latestMissing {
		verbose.UnsupportedHelp(p.Rule, "outdated")
	}

	// NotConfigured status is self-explanatory - no extra message needed
	// Only show messages for cases that require explanation (VersionMissing, Floating)
	return ""
}
