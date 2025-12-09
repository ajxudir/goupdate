// Package supervision provides tracking for packages that cannot be updated.
//
// This package handles the collection and reporting of packages that are
// "unsupported" for automatic updates due to various reasons such as:
//   - Missing concrete versions in lock files
//   - Floating version constraints (e.g., "5.*", ">=8.0.0")
//   - Packages not configured for update operations
//
// # Core Types
//
// UnsupportedTracker is a thread-safe collector for unsupported packages:
//
//	tracker := supervision.NewUnsupportedTracker()
//	tracker.Add(pkg, "reason for not updating")
//	messages := tracker.Messages()
//
// UnsupportedRuleInfo holds details about why packages in a rule cannot be updated:
//
//	info := &supervision.UnsupportedRuleInfo{
//	    Rule:        "npm-packages",
//	    PackageType: "npm",
//	    Reason:      "No lock file configured",
//	    Count:       5,
//	}
//
// # Helper Functions
//
// ShouldTrackUnsupported determines if a package status indicates tracking:
//
//	if supervision.ShouldTrackUnsupported(pkg.InstallStatus) {
//	    reason := supervision.DeriveUnsupportedReason(pkg, cfg, err, latestMissing)
//	    tracker.Add(pkg, reason)
//	}
//
// DeriveUnsupportedReason generates human-readable explanations:
//
//	reason := supervision.DeriveUnsupportedReason(pkg, cfg, nil, false)
//	// Returns: "No concrete version found in manifest or lock file."
//	// Or: "Floating constraint '>=5.0.0' - update manually or remove constraint."
//
// # Thread Safety
//
// UnsupportedTracker is safe for concurrent use from multiple goroutines.
// All Add operations are serialized, and Messages returns a snapshot.
//
// # Integration with Update Context
//
// The UnsupportedTracker implements the UnsupportedTracker interface from
// pkg/update, allowing seamless integration with update operations:
//
//	ctx := update.NewContext(
//	    update.WithUnsupportedTracker(supervision.NewUnsupportedTracker()),
//	)
package supervision
