package systemtest

// TestRunner defines the interface for executing system tests.
//
// This interface enables testing code that depends on system test execution
// by allowing mock implementations to be substituted.
//
// Standard implementation: *Runner
//
// Example:
//
//	var runner TestRunner = systemtest.NewRunner(cfg, workDir, false, false)
//	if runner.HasTests() {
//	    result := runner.RunAfterUpdate()
//	}
type TestRunner interface {
	// HasTests returns true if there are tests configured.
	//
	// Returns:
	//   - bool: true if tests are configured; false otherwise
	HasTests() bool

	// ShouldRunPreflight returns true if preflight tests should be run.
	//
	// Returns:
	//   - bool: true if preflight tests should be executed; false otherwise
	ShouldRunPreflight() bool

	// ShouldRunAfterEach returns true if tests should run after each package update.
	//
	// Returns:
	//   - bool: true if tests should run after each update; false otherwise
	ShouldRunAfterEach() bool

	// ShouldRunAfterAll returns true if tests should run once after all updates.
	//
	// Returns:
	//   - bool: true if tests should run after all updates complete; false otherwise
	ShouldRunAfterAll() bool

	// StopOnFail returns true if updates should stop on test failure.
	//
	// Returns:
	//   - bool: true if update process should halt on test failure; false otherwise
	StopOnFail() bool

	// RunPreflight executes all system tests as a preflight check.
	//
	// Returns:
	//   - *Result: Test execution results
	RunPreflight() *Result

	// RunAfterUpdate executes all system tests after an update.
	//
	// Returns:
	//   - *Result: Test execution results
	RunAfterUpdate() *Result

	// RunValidation executes all system tests as final validation.
	//
	// Returns:
	//   - *Result: Test execution results
	RunValidation() *Result
}

// Verify that Runner implements the TestRunner interface.
// This is a compile-time check.
var _ TestRunner = (*Runner)(nil)
