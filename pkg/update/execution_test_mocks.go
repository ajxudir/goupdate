package update

import (
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
)

// mockTestResultOutput is a mock for testing SystemTestResult output.
type mockTestResultOutput struct {
	resultOutput string
}

// FormatResultsQuiet implements the interface for test result output.
func (m *mockTestResultOutput) FormatResultsQuiet() string {
	return m.resultOutput
}

// mockProgressReporter is a mock for testing progress reporting.
type mockProgressReporter struct {
	count       int
	incrementFn func()
}

// Increment implements the ProgressReporter interface.
func (m *mockProgressReporter) Increment() {
	m.count++
	if m.incrementFn != nil {
		m.incrementFn()
	}
}

// testDeriveReason returns a standard mock derive reason function for tests.
func testDeriveReason() UnsupportedReasonDeriver {
	return func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}
}

// testNoopUpdater returns a no-op updater function for dry-run tests.
func testNoopUpdater() PackageUpdater {
	return func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return nil
	}
}

// testFailingUpdater returns an updater that always returns the given error.
func testFailingUpdater(err error) PackageUpdater {
	return func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return err
	}
}

// testCallCountingUpdater returns an updater that counts calls.
func testCallCountingUpdater(counter *int) PackageUpdater {
	return func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		*counter++
		return nil
	}
}

// newMockCallbacks creates ExecutionCallbacks with default test mocks.
func newMockCallbacks() ExecutionCallbacks {
	return ExecutionCallbacks{
		DeriveReason: testDeriveReason(),
	}
}
