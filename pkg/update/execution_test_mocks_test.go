package update

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
