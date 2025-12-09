package systemtest

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/goupdate/pkg/config"
)

func TestRunner_HasTests(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.SystemTestsCfg
		expected bool
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: false,
		},
		{
			name: "empty tests",
			cfg: &config.SystemTestsCfg{
				Tests: []config.SystemTestCfg{},
			},
			expected: false,
		},
		{
			name: "has tests",
			cfg: &config.SystemTestsCfg{
				Tests: []config.SystemTestCfg{
					{Name: "test1", Commands: "echo hello"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner(tt.cfg, "/tmp", false, false)
			assert.Equal(t, tt.expected, runner.HasTests())
		})
	}
}

func TestRunner_ShouldRunPreflight(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		cfg      *config.SystemTestsCfg
		expected bool
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: false,
		},
		{
			name: "default preflight (true)",
			cfg: &config.SystemTestsCfg{
				Tests: []config.SystemTestCfg{
					{Name: "test1", Commands: "echo hello"},
				},
			},
			expected: true,
		},
		{
			name: "preflight disabled",
			cfg: &config.SystemTestsCfg{
				RunPreflight: &falseVal,
				Tests: []config.SystemTestCfg{
					{Name: "test1", Commands: "echo hello"},
				},
			},
			expected: false,
		},
		{
			name: "preflight enabled explicitly",
			cfg: &config.SystemTestsCfg{
				RunPreflight: &trueVal,
				Tests: []config.SystemTestCfg{
					{Name: "test1", Commands: "echo hello"},
				},
			},
			expected: true,
		},
		{
			name: "no tests",
			cfg: &config.SystemTestsCfg{
				RunPreflight: &trueVal,
				Tests:        []config.SystemTestCfg{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner(tt.cfg, "/tmp", false, false)
			assert.Equal(t, tt.expected, runner.ShouldRunPreflight())
		})
	}
}

func TestRunner_ShouldRunAfterEach(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.SystemTestsCfg
		expected bool
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: false,
		},
		{
			name: "default mode (after_all)",
			cfg: &config.SystemTestsCfg{
				Tests: []config.SystemTestCfg{
					{Name: "test1", Commands: "echo hello"},
				},
			},
			expected: false,
		},
		{
			name: "after_each mode",
			cfg: &config.SystemTestsCfg{
				RunMode: config.SystemTestRunModeAfterEach,
				Tests: []config.SystemTestCfg{
					{Name: "test1", Commands: "echo hello"},
				},
			},
			expected: true,
		},
		{
			name: "no tests with after_each mode",
			cfg: &config.SystemTestsCfg{
				RunMode: config.SystemTestRunModeAfterEach,
				Tests:   []config.SystemTestCfg{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner(tt.cfg, "/tmp", false, false)
			assert.Equal(t, tt.expected, runner.ShouldRunAfterEach())
		})
	}
}

func TestRunner_ShouldRunAfterAll(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.SystemTestsCfg
		expected bool
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: false,
		},
		{
			name: "default mode (after_all)",
			cfg: &config.SystemTestsCfg{
				Tests: []config.SystemTestCfg{
					{Name: "test1", Commands: "echo hello"},
				},
			},
			expected: true,
		},
		{
			name: "after_each mode",
			cfg: &config.SystemTestsCfg{
				RunMode: config.SystemTestRunModeAfterEach,
				Tests: []config.SystemTestCfg{
					{Name: "test1", Commands: "echo hello"},
				},
			},
			expected: false,
		},
		{
			name: "none mode",
			cfg: &config.SystemTestsCfg{
				RunMode: config.SystemTestRunModeNone,
				Tests: []config.SystemTestCfg{
					{Name: "test1", Commands: "echo hello"},
				},
			},
			expected: false,
		},
		{
			name: "no tests with after_all mode",
			cfg: &config.SystemTestsCfg{
				RunMode: config.SystemTestRunModeAfterAll,
				Tests:   []config.SystemTestCfg{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner(tt.cfg, "/tmp", false, false)
			assert.Equal(t, tt.expected, runner.ShouldRunAfterAll())
		})
	}
}

func TestRunner_StopOnFail(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		cfg      *config.SystemTestsCfg
		expected bool
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: true, // Default is true
		},
		{
			name: "default stop_on_fail (true)",
			cfg: &config.SystemTestsCfg{
				Tests: []config.SystemTestCfg{
					{Name: "test1", Commands: "echo hello"},
				},
			},
			expected: true,
		},
		{
			name: "stop_on_fail disabled",
			cfg: &config.SystemTestsCfg{
				StopOnFail: &falseVal,
				Tests: []config.SystemTestCfg{
					{Name: "test1", Commands: "echo hello"},
				},
			},
			expected: false,
		},
		{
			name: "stop_on_fail enabled explicitly",
			cfg: &config.SystemTestsCfg{
				StopOnFail: &trueVal,
				Tests: []config.SystemTestCfg{
					{Name: "test1", Commands: "echo hello"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner(tt.cfg, "/tmp", false, false)
			assert.Equal(t, tt.expected, runner.StopOnFail())
		})
	}
}

func TestRunner_RunPreflight(t *testing.T) {
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{
			{
				Name:           "passing-test",
				Commands:       "echo hello",
				TimeoutSeconds: 10,
			},
		},
	}

	runner := NewRunner(cfg, "/tmp", false, false)
	result := runner.RunPreflight()

	assert.NotNil(t, result)
	assert.Equal(t, PhasePreflight, result.Phase)
	assert.Len(t, result.Tests, 1)
	assert.True(t, result.Tests[0].Passed)
	assert.Equal(t, "passing-test", result.Tests[0].Name)
	assert.True(t, result.Passed())
}

func TestRunner_RunAfterUpdate(t *testing.T) {
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{
			{
				Name:           "passing-test",
				Commands:       "echo hello",
				TimeoutSeconds: 10,
			},
		},
	}

	runner := NewRunner(cfg, "/tmp", false, false)
	result := runner.RunAfterUpdate()

	assert.NotNil(t, result)
	assert.Equal(t, PhaseAfterEach, result.Phase)
	assert.Len(t, result.Tests, 1)
	assert.True(t, result.Tests[0].Passed)
}

func TestRunner_RunValidation(t *testing.T) {
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{
			{
				Name:           "passing-test",
				Commands:       "echo hello",
				TimeoutSeconds: 10,
			},
		},
	}

	runner := NewRunner(cfg, "/tmp", false, false)
	result := runner.RunValidation()

	assert.NotNil(t, result)
	assert.Equal(t, PhaseAfterAll, result.Phase)
	assert.Len(t, result.Tests, 1)
	assert.True(t, result.Tests[0].Passed)
}

func TestRunner_RunTests_EmptyConfig(t *testing.T) {
	// Test with nil config
	runner := NewRunner(nil, "/tmp", false, false)
	result := runner.RunPreflight()
	assert.NotNil(t, result)
	assert.Empty(t, result.Tests)
	assert.Equal(t, PhasePreflight, result.Phase)

	// Test with empty tests
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{},
	}
	runner = NewRunner(cfg, "/tmp", false, false)
	result = runner.RunPreflight()
	assert.NotNil(t, result)
	assert.Empty(t, result.Tests)
}

func TestRunner_FailingTest(t *testing.T) {
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{
			{
				Name:           "failing-test",
				Commands:       "exit 1",
				TimeoutSeconds: 10,
			},
		},
	}

	runner := NewRunner(cfg, "/tmp", false, false)
	result := runner.RunPreflight()

	assert.NotNil(t, result)
	assert.Len(t, result.Tests, 1)
	assert.False(t, result.Tests[0].Passed)
	assert.NotNil(t, result.Tests[0].Error)
	assert.False(t, result.Passed())
	assert.True(t, result.HasCriticalFailure())
}

func TestRunner_ContinueOnFail(t *testing.T) {
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{
			{
				Name:           "failing-test",
				Commands:       "exit 1",
				TimeoutSeconds: 10,
				ContinueOnFail: true,
			},
		},
	}

	runner := NewRunner(cfg, "/tmp", false, false)
	result := runner.RunPreflight()

	assert.NotNil(t, result)
	assert.Len(t, result.Tests, 1)
	assert.False(t, result.Tests[0].Passed)
	assert.True(t, result.Tests[0].ContinueOnFail)
	assert.False(t, result.HasCriticalFailure()) // No critical failure because continue_on_fail is true
}

func TestRunner_NoTimeout(t *testing.T) {
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{
			{
				Name:           "test-no-timeout",
				Commands:       "echo hello",
				TimeoutSeconds: 1,
			},
		},
	}

	// With noTimeout=true, the timeout should be ignored
	runner := NewRunner(cfg, "/tmp", true, false)
	result := runner.RunPreflight()

	assert.NotNil(t, result)
	assert.Len(t, result.Tests, 1)
	assert.True(t, result.Tests[0].Passed)
}

func TestRunner_DefaultTimeout(t *testing.T) {
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{
			{
				Name:     "test-default-timeout",
				Commands: "echo hello",
				// TimeoutSeconds not set, should use default
			},
		},
	}

	runner := NewRunner(cfg, "/tmp", false, false)
	result := runner.RunPreflight()

	assert.NotNil(t, result)
	assert.Len(t, result.Tests, 1)
	assert.True(t, result.Tests[0].Passed)
}

func TestRunner_MultipleTests(t *testing.T) {
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{
			{
				Name:           "test1",
				Commands:       "echo test1",
				TimeoutSeconds: 10,
			},
			{
				Name:           "test2",
				Commands:       "echo test2",
				TimeoutSeconds: 10,
			},
			{
				Name:           "test3",
				Commands:       "exit 1",
				TimeoutSeconds: 10,
				ContinueOnFail: true,
			},
		},
	}

	runner := NewRunner(cfg, "/tmp", false, false)
	result := runner.RunPreflight()

	assert.NotNil(t, result)
	assert.Len(t, result.Tests, 3)
	assert.True(t, result.Tests[0].Passed)
	assert.True(t, result.Tests[1].Passed)
	assert.False(t, result.Tests[2].Passed)
	assert.Equal(t, 2, result.PassedCount())
	assert.Equal(t, 1, result.FailedCount())
	assert.False(t, result.HasCriticalFailure()) // The failing test has continue_on_fail
}

func TestResult_FailedTests(t *testing.T) {
	result := &Result{
		Tests: []TestResult{
			{Name: "test1", Passed: true},
			{Name: "test2", Passed: false, Error: fmt.Errorf("test error")},
			{Name: "test3", Passed: false, Error: fmt.Errorf("another error"), ContinueOnFail: true},
		},
	}

	failed := result.FailedTests()
	assert.Len(t, failed, 2)
	assert.Equal(t, "test2", failed[0].Name)
	assert.Equal(t, "test3", failed[1].Name)
}

func TestResult_CriticalFailures(t *testing.T) {
	result := &Result{
		Tests: []TestResult{
			{Name: "test1", Passed: true},
			{Name: "test2", Passed: false, ContinueOnFail: false},
			{Name: "test3", Passed: false, ContinueOnFail: true},
		},
	}

	critical := result.CriticalFailures()
	assert.Len(t, critical, 1)
	assert.Equal(t, "test2", critical[0].Name)
}

func TestResult_Summary(t *testing.T) {
	tests := []struct {
		name     string
		result   *Result
		expected string
	}{
		{
			name: "all passed",
			result: &Result{
				Tests: []TestResult{
					{Name: "test1", Passed: true},
					{Name: "test2", Passed: true},
				},
			},
			expected: "All 2 system tests passed",
		},
		{
			name: "some failed",
			result: &Result{
				Tests: []TestResult{
					{Name: "test1", Passed: true},
					{Name: "test2", Passed: false},
				},
			},
			expected: "1/2 system tests passed (1 failed)",
		},
		{
			name: "all failed",
			result: &Result{
				Tests: []TestResult{
					{Name: "test1", Passed: false},
					{Name: "test2", Passed: false},
				},
			},
			expected: "0/2 system tests passed (2 failed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.Summary())
		})
	}
}

func TestResult_FormatResults(t *testing.T) {
	result := &Result{
		Phase: PhasePreflight,
		Tests: []TestResult{
			{Name: "passing-test", Passed: true, Duration: 500 * time.Millisecond},
			{Name: "failing-test", Passed: false, Duration: 2 * time.Second, Error: fmt.Errorf("test failed")},
		},
	}

	output := result.FormatResults()

	assert.Contains(t, output, "System Tests (Preflight)")
	assert.Contains(t, output, "passing-test")
	assert.Contains(t, output, "failing-test")
	assert.Contains(t, output, "✓")
	assert.Contains(t, output, "✗")
	assert.Contains(t, output, "test failed")
}

func TestResult_FormatResults_NoError(t *testing.T) {
	result := &Result{
		Phase: PhaseAfterAll,
		Tests: []TestResult{
			{Name: "passing-test", Passed: true, Duration: 100 * time.Millisecond},
		},
	}

	output := result.FormatResults()

	assert.Contains(t, output, "System Tests (Validation)")
	assert.Contains(t, output, "passing-test")
	assert.Contains(t, output, "✓")
}

func TestResult_FormatResultsQuiet_AllPassed(t *testing.T) {
	result := &Result{
		Phase: PhasePreflight,
		Tests: []TestResult{
			{Name: "passing-test-1", Passed: true, Duration: 100 * time.Millisecond},
			{Name: "passing-test-2", Passed: true, Duration: 200 * time.Millisecond},
		},
	}

	// When all tests pass, FormatResultsQuiet should return empty string
	output := result.FormatResultsQuiet()
	assert.Empty(t, output)
}

func TestResult_FormatResultsQuiet_WithFailures(t *testing.T) {
	result := &Result{
		Phase: PhaseAfterAll,
		Tests: []TestResult{
			{Name: "passing-test", Passed: true, Duration: 100 * time.Millisecond},
			{Name: "failing-test", Passed: false, Duration: 2 * time.Second, Error: fmt.Errorf("test failed")},
		},
	}

	// When there are failures, FormatResultsQuiet should show only failures
	output := result.FormatResultsQuiet()

	assert.Contains(t, output, "System Tests (Validation)")
	assert.NotContains(t, output, "passing-test") // Passing tests should be hidden
	assert.Contains(t, output, "failing-test")    // Failing tests should be shown
	assert.Contains(t, output, "✗")
	assert.Contains(t, output, "test failed")
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500ms",
		},
		{
			name:     "seconds",
			duration: 2 * time.Second,
			expected: "2.0s",
		},
		{
			name:     "seconds with decimal",
			duration: 2500 * time.Millisecond,
			expected: "2.5s",
		},
		{
			name:     "very short",
			duration: 50 * time.Millisecond,
			expected: "50ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatDuration(tt.duration))
		})
	}
}

func TestRun(t *testing.T) {
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{
			{
				Name:           "test1",
				Commands:       "echo hello",
				TimeoutSeconds: 10,
			},
		},
	}

	tests := []struct {
		name          string
		phase         string
		expectedPhase string
	}{
		{
			name:          "preflight phase",
			phase:         PhasePreflight,
			expectedPhase: PhasePreflight,
		},
		{
			name:          "after_each phase",
			phase:         PhaseAfterEach,
			expectedPhase: PhaseAfterEach,
		},
		{
			name:          "after_all phase",
			phase:         PhaseAfterAll,
			expectedPhase: PhaseAfterAll,
		},
		{
			name:          "custom phase",
			phase:         "Custom",
			expectedPhase: "Custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Run(cfg, "/tmp", false, tt.phase)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedPhase, result.Phase)
			assert.Len(t, result.Tests, 1)
			assert.True(t, result.Tests[0].Passed)
		})
	}
}

func TestValidateCommands(t *testing.T) {
	// Test with nil config
	missing := ValidateCommands(nil)
	assert.Nil(t, missing)

	// Test with empty tests
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{},
	}
	missing = ValidateCommands(cfg)
	assert.Nil(t, missing)

	// Test with tests (currently returns nil as validation is not implemented)
	cfg = &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{
			{Name: "test1", Commands: "npm test"},
		},
	}
	missing = ValidateCommands(cfg)
	assert.Nil(t, missing)
}

func TestRunner_WithEnvironmentVariables(t *testing.T) {
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{
			{
				Name:           "env-test",
				Commands:       "echo $TEST_VAR",
				TimeoutSeconds: 10,
				Env: map[string]string{
					"TEST_VAR": "hello",
				},
			},
		},
	}

	runner := NewRunner(cfg, "/tmp", false, false)
	result := runner.RunPreflight()

	assert.NotNil(t, result)
	assert.Len(t, result.Tests, 1)
	assert.True(t, result.Tests[0].Passed)
	assert.Contains(t, result.Tests[0].Output, "hello")
}

func TestResult_Passed_EmptyTests(t *testing.T) {
	result := &Result{
		Tests: []TestResult{},
	}
	assert.True(t, result.Passed())
}

func TestResult_TotalDuration(t *testing.T) {
	cfg := &config.SystemTestsCfg{
		Tests: []config.SystemTestCfg{
			{
				Name:           "test1",
				Commands:       "echo hello",
				TimeoutSeconds: 10,
			},
		},
	}

	runner := NewRunner(cfg, "/tmp", false, false)
	result := runner.RunPreflight()

	assert.NotNil(t, result)
	assert.True(t, result.TotalDuration > 0)
}
