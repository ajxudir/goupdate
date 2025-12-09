package supervision

import (
	"github.com/ajxudir/goupdate/pkg/formats"
)

// Tracker defines the interface for tracking unsupported packages.
//
// This interface is designed to match the UnsupportedTracker interface
// defined in pkg/update/context.go, allowing the supervision package
// to provide a standard implementation while other code depends only
// on the interface.
//
// Standard implementation: *UnsupportedTracker
//
// Example:
//
//	var tracker Tracker = supervision.NewUnsupportedTracker()
//	tracker.Add(pkg, "reason")
//	messages := tracker.Messages()
type Tracker interface {
	// Add tracks an unsupported package with a reason.
	//
	// Parameters:
	//   - p: Package to track
	//   - reason: Human-readable reason for not supporting updates
	Add(p formats.Package, reason string)

	// Messages returns formatted messages for all tracked unsupported rules.
	//
	// Returns:
	//   - []string: Formatted messages, or nil if no packages tracked
	Messages() []string
}

// Verify that UnsupportedTracker implements the Tracker interface.
// This is a compile-time check.
var _ Tracker = (*UnsupportedTracker)(nil)
