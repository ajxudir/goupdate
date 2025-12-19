// Package systemtest provides functionality for running system tests after updates.
// System tests validate that packages work correctly after version changes by
// executing configured test commands.
package systemtest

import (
	"fmt"
	"time"

	"github.com/ajxudir/goupdate/pkg/cmdexec"
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/verbose"
)

// DefaultTimeoutSeconds is the default timeout for system tests (5 minutes).
const DefaultTimeoutSeconds = 300

// Runner executes system tests based on configuration and provides control over test execution timing.
//
// Fields:
//   - cfg: System tests configuration containing test definitions
//   - workDir: Working directory where test commands will be executed
//   - noTimeout: When true, disables timeout enforcement for all tests
//   - verbose: When true, enables verbose output during test execution
type Runner struct {
	cfg       *config.SystemTestsCfg
	workDir   string
	noTimeout bool
	verbose   bool
}

// NewRunner creates a new system test runner with the specified configuration.
//
// Parameters:
//   - cfg: System tests configuration, can be nil if no tests are configured
//   - workDir: Working directory where test commands will be executed
//   - noTimeout: When true, disables timeout enforcement for all tests
//   - verbose: When true, enables verbose output during test execution
//
// Returns:
//   - *Runner: A new runner instance ready to execute tests
func NewRunner(cfg *config.SystemTestsCfg, workDir string, noTimeout bool, verbose bool) *Runner {
	return &Runner{
		cfg:       cfg,
		workDir:   workDir,
		noTimeout: noTimeout,
		verbose:   verbose,
	}
}

// HasTests returns true if there are tests configured and returns false otherwise.
//
// Returns:
//   - bool: true if configuration is non-nil and contains at least one test; false otherwise
func (r *Runner) HasTests() bool {
	return r.cfg != nil && len(r.cfg.Tests) > 0
}

// ShouldRunPreflight returns true if preflight tests should be run before any updates.
//
// Preflight tests validate the system state before making any changes.
//
// Returns:
//   - bool: true if configuration enables preflight tests and has tests defined; false otherwise
func (r *Runner) ShouldRunPreflight() bool {
	if r.cfg == nil {
		return false
	}
	return r.cfg.IsRunPreflight() && len(r.cfg.Tests) > 0
}

// ShouldRunAfterEach returns true if tests should run after each package update.
//
// Returns:
//   - bool: true if run_mode is "after_each" and tests are configured; false otherwise
func (r *Runner) ShouldRunAfterEach() bool {
	if r.cfg == nil {
		return false
	}
	return r.cfg.GetRunMode() == config.SystemTestRunModeAfterEach && len(r.cfg.Tests) > 0
}

// ShouldRunAfterAll returns true if tests should run once after all updates complete.
//
// Returns:
//   - bool: true if run_mode is "after_all" and tests are configured; false otherwise
func (r *Runner) ShouldRunAfterAll() bool {
	if r.cfg == nil {
		return false
	}
	return r.cfg.GetRunMode() == config.SystemTestRunModeAfterAll && len(r.cfg.Tests) > 0
}

// StopOnFail returns true if updates should stop on test failure.
//
// Returns:
//   - bool: true if stop_on_fail is enabled in configuration (default); false otherwise
func (r *Runner) StopOnFail() bool {
	if r.cfg == nil {
		return true
	}
	return r.cfg.IsStopOnFail()
}

// RunPreflight executes all system tests as a preflight check before any updates.
//
// Returns:
//   - *Result: Test execution results with Phase set to PhasePreflight
func (r *Runner) RunPreflight() *Result {
	return r.runTests(PhasePreflight)
}

// RunAfterUpdate executes all system tests after a package update.
//
// Returns:
//   - *Result: Test execution results with Phase set to PhaseAfterEach
func (r *Runner) RunAfterUpdate() *Result {
	return r.runTests(PhaseAfterEach)
}

// RunValidation executes all system tests as final validation after all updates.
//
// Returns:
//   - *Result: Test execution results with Phase set to PhaseAfterAll
func (r *Runner) RunValidation() *Result {
	return r.runTests(PhaseAfterAll)
}

// runTests executes all configured tests and returns the aggregate result.
//
// It performs the following operations:
//   - Step 1: Initialize result structure with test phase
//   - Step 2: Execute each configured test sequentially
//   - Step 3: Collect individual test results and total duration
//
// Parameters:
//   - phase: Test phase identifier (e.g., PhasePreflight, PhaseAfterEach, PhaseAfterAll)
//
// Returns:
//   - *Result: Aggregate test results containing all individual test outcomes
func (r *Runner) runTests(phase string) *Result {
	if r.cfg == nil || len(r.cfg.Tests) == 0 {
		return &Result{Phase: phase}
	}

	result := &Result{
		Phase: phase,
		Tests: make([]TestResult, 0, len(r.cfg.Tests)),
	}

	startTime := time.Now()

	for _, test := range r.cfg.Tests {
		testResult := r.runSingleTest(&test)
		result.Tests = append(result.Tests, testResult)
	}

	result.TotalDuration = time.Since(startTime)
	return result
}

// runSingleTest executes a single test and returns its result.
//
// It performs the following operations:
//   - Step 1: Determine timeout value (test-specific, default, or disabled)
//   - Step 2: Execute test commands in the working directory
//   - Step 3: Capture output, duration, and error status
//
// Parameters:
//   - test: Test configuration containing commands, environment, and timeout settings
//
// Returns:
//   - TestResult: Test execution result with passed status, output, duration, and any error
func (r *Runner) runSingleTest(test *config.SystemTestCfg) TestResult {
	startTime := time.Now()

	timeout := test.TimeoutSeconds
	if timeout == 0 {
		timeout = DefaultTimeoutSeconds
	}
	if r.noTimeout {
		timeout = 0
	}

	// Suppress verbose during command execution to avoid duplicate logging
	verbose.Suppress()
	output, err := cmdexec.Execute(test.Commands, test.Env, r.workDir, timeout, nil)
	verbose.Unsuppress()

	duration := time.Since(startTime)

	testResult := TestResult{
		Name:           test.Name,
		Duration:       duration,
		Output:         string(output),
		ContinueOnFail: test.ContinueOnFail,
	}

	if err != nil {
		testResult.Passed = false
		testResult.Error = fmt.Errorf("%s: %w", test.Name, err)
		verbose.Printf("System test %q FAILED: %v\n", test.Name, err)
	} else {
		testResult.Passed = true
		verbose.Debugf("System test %q passed (%v)", test.Name, duration)
	}

	return testResult
}

// Run is a convenience function to run system tests with configuration at a specific phase.
//
// It creates a runner and executes tests based on the specified phase.
//
// Parameters:
//   - cfg: System tests configuration, can be nil if no tests are configured
//   - workDir: Working directory where test commands will be executed
//   - noTimeout: When true, disables timeout enforcement for all tests
//   - phase: Test phase identifier (PhasePreflight, PhaseAfterEach, PhaseAfterAll, or custom)
//
// Returns:
//   - *Result: Test execution results for the specified phase
func Run(cfg *config.SystemTestsCfg, workDir string, noTimeout bool, phase string) *Result {
	runner := NewRunner(cfg, workDir, noTimeout, false)
	switch phase {
	case PhasePreflight:
		return runner.RunPreflight()
	case PhaseAfterEach:
		return runner.RunAfterUpdate()
	case PhaseAfterAll:
		return runner.RunValidation()
	default:
		return runner.runTests(phase)
	}
}

// ValidateCommands checks that all commands required for system tests are available.
//
// Currently, this function does not perform validation as system test commands
// may have complex project-specific dependencies. Actual test execution will
// report any missing commands.
//
// Parameters:
//   - cfg: System tests configuration to validate
//
// Returns:
//   - []string: List of missing commands with installation hints; currently always returns nil
func ValidateCommands(cfg *config.SystemTestsCfg) []string {
	if cfg == nil || len(cfg.Tests) == 0 {
		return nil
	}

	// For now, we don't validate system test commands in preflight
	// since they may have complex dependencies that are project-specific.
	// The actual test execution will report any missing commands.
	return nil
}
