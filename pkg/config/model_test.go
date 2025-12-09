package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsEnabled tests the behavior of PackageManagerCfg.IsEnabled.
//
// It verifies:
//   - Nil enabled returns true
//   - Enabled true
//   - Enabled false
func TestIsEnabled(t *testing.T) {
	t.Run("nil enabled returns true", func(t *testing.T) {
		cfg := &PackageManagerCfg{}
		assert.True(t, cfg.IsEnabled())
	})

	t.Run("enabled true", func(t *testing.T) {
		enabled := true
		cfg := &PackageManagerCfg{Enabled: &enabled}
		assert.True(t, cfg.IsEnabled())
	})

	t.Run("enabled false", func(t *testing.T) {
		enabled := false
		cfg := &PackageManagerCfg{Enabled: &enabled}
		assert.False(t, cfg.IsEnabled())
	})
}

// TestIsRootConfig tests the behavior of Config.IsRootConfig and SetRootConfig.
//
// It verifies:
//   - Default is false
//   - Returns true when set
//   - Can toggle back to false
func TestIsRootConfig(t *testing.T) {
	t.Run("default is false", func(t *testing.T) {
		cfg := &Config{}
		assert.False(t, cfg.IsRootConfig())
	})

	t.Run("returns true when set", func(t *testing.T) {
		cfg := &Config{}
		cfg.SetRootConfig(true)
		assert.True(t, cfg.IsRootConfig())
	})

	t.Run("can toggle back to false", func(t *testing.T) {
		cfg := &Config{}
		cfg.SetRootConfig(true)
		cfg.SetRootConfig(false)
		assert.False(t, cfg.IsRootConfig())
	})
}

// TestGetMaxConfigFileSize tests the behavior of Config.GetMaxConfigFileSize.
//
// It verifies:
//   - Returns default when Security is nil
//   - Returns default when MaxConfigFileSize is zero
//   - Returns configured value when set
func TestGetMaxConfigFileSize(t *testing.T) {
	t.Run("returns default when Security is nil", func(t *testing.T) {
		cfg := &Config{}
		assert.Equal(t, int64(DefaultMaxConfigFileSize), cfg.GetMaxConfigFileSize())
	})

	t.Run("returns default when MaxConfigFileSize is zero", func(t *testing.T) {
		cfg := &Config{Security: &SecurityCfg{MaxConfigFileSize: 0}}
		assert.Equal(t, int64(DefaultMaxConfigFileSize), cfg.GetMaxConfigFileSize())
	})

	t.Run("returns configured value when set", func(t *testing.T) {
		cfg := &Config{Security: &SecurityCfg{MaxConfigFileSize: 5000}}
		assert.Equal(t, int64(5000), cfg.GetMaxConfigFileSize())
	})
}

// TestGetMaxRegexComplexity tests the behavior of Config.GetMaxRegexComplexity.
//
// It verifies:
//   - Returns default when Security is nil
//   - Returns default when MaxRegexComplexity is zero
//   - Returns configured value when set
func TestGetMaxRegexComplexity(t *testing.T) {
	t.Run("returns default when Security is nil", func(t *testing.T) {
		cfg := &Config{}
		assert.Equal(t, DefaultMaxRegexComplexity, cfg.GetMaxRegexComplexity())
	})

	t.Run("returns default when MaxRegexComplexity is zero", func(t *testing.T) {
		cfg := &Config{Security: &SecurityCfg{MaxRegexComplexity: 0}}
		assert.Equal(t, DefaultMaxRegexComplexity, cfg.GetMaxRegexComplexity())
	})

	t.Run("returns configured value when set", func(t *testing.T) {
		cfg := &Config{Security: &SecurityCfg{MaxRegexComplexity: 500}}
		assert.Equal(t, 500, cfg.GetMaxRegexComplexity())
	})
}

// TestAllowsComplexRegex tests the behavior of Config.AllowsComplexRegex.
//
// It verifies:
//   - Returns false when Security is nil
//   - Returns false when not set
//   - Returns true when set
func TestAllowsComplexRegex(t *testing.T) {
	t.Run("returns false when Security is nil", func(t *testing.T) {
		cfg := &Config{}
		assert.False(t, cfg.AllowsComplexRegex())
	})

	t.Run("returns false when not set", func(t *testing.T) {
		cfg := &Config{Security: &SecurityCfg{}}
		assert.False(t, cfg.AllowsComplexRegex())
	})

	t.Run("returns true when set", func(t *testing.T) {
		cfg := &Config{Security: &SecurityCfg{AllowComplexRegex: true}}
		assert.True(t, cfg.AllowsComplexRegex())
	})
}

// TestAllowsPathTraversal tests the behavior of Config.AllowsPathTraversal.
//
// It verifies:
//   - Returns false when Security is nil
//   - Returns false when not set
//   - Returns true when set
func TestAllowsPathTraversal(t *testing.T) {
	t.Run("returns false when Security is nil", func(t *testing.T) {
		cfg := &Config{}
		assert.False(t, cfg.AllowsPathTraversal())
	})

	t.Run("returns false when not set", func(t *testing.T) {
		cfg := &Config{Security: &SecurityCfg{}}
		assert.False(t, cfg.AllowsPathTraversal())
	})

	t.Run("returns true when set", func(t *testing.T) {
		cfg := &Config{Security: &SecurityCfg{AllowPathTraversal: true}}
		assert.True(t, cfg.AllowsPathTraversal())
	})
}

// TestAllowsAbsolutePaths tests the behavior of Config.AllowsAbsolutePaths.
//
// It verifies:
//   - Returns false when Security is nil
//   - Returns false when not set
//   - Returns true when set
func TestAllowsAbsolutePaths(t *testing.T) {
	t.Run("returns false when Security is nil", func(t *testing.T) {
		cfg := &Config{}
		assert.False(t, cfg.AllowsAbsolutePaths())
	})

	t.Run("returns false when not set", func(t *testing.T) {
		cfg := &Config{Security: &SecurityCfg{}}
		assert.False(t, cfg.AllowsAbsolutePaths())
	})

	t.Run("returns true when set", func(t *testing.T) {
		cfg := &Config{Security: &SecurityCfg{AllowAbsolutePaths: true}}
		assert.True(t, cfg.AllowsAbsolutePaths())
	})
}

// TestLockFileCfgGetTimeoutSeconds tests the behavior of LockFileCfg.GetTimeoutSeconds.
//
// It verifies:
//   - Returns default when TimeoutSeconds is zero
//   - Returns configured value when set
func TestLockFileCfgGetTimeoutSeconds(t *testing.T) {
	t.Run("returns default when TimeoutSeconds is zero", func(t *testing.T) {
		cfg := &LockFileCfg{}
		assert.Equal(t, 60, cfg.GetTimeoutSeconds())
	})

	t.Run("returns configured value when set", func(t *testing.T) {
		cfg := &LockFileCfg{TimeoutSeconds: 120}
		assert.Equal(t, 120, cfg.GetTimeoutSeconds())
	})
}

// TestSystemTestsCfgDefaults tests the default values of SystemTestsCfg methods.
//
// It verifies:
//   - IsRunPreflight defaults to true
//   - IsStopOnFail defaults to true
//   - GetRunMode defaults to after_all
func TestSystemTestsCfgDefaults(t *testing.T) {
	t.Run("IsRunPreflight defaults to true", func(t *testing.T) {
		cfg := &SystemTestsCfg{}
		assert.True(t, cfg.IsRunPreflight())
	})

	t.Run("IsRunPreflight returns configured value", func(t *testing.T) {
		f := false
		cfg := &SystemTestsCfg{RunPreflight: &f}
		assert.False(t, cfg.IsRunPreflight())
	})

	t.Run("IsStopOnFail defaults to true", func(t *testing.T) {
		cfg := &SystemTestsCfg{}
		assert.True(t, cfg.IsStopOnFail())
	})

	t.Run("IsStopOnFail returns configured value", func(t *testing.T) {
		f := false
		cfg := &SystemTestsCfg{StopOnFail: &f}
		assert.False(t, cfg.IsStopOnFail())
	})

	t.Run("GetRunMode defaults to after_all", func(t *testing.T) {
		cfg := &SystemTestsCfg{}
		assert.Equal(t, "after_all", cfg.GetRunMode())
	})

	t.Run("GetRunMode returns configured value", func(t *testing.T) {
		cfg := &SystemTestsCfg{RunMode: "after_each"}
		assert.Equal(t, "after_each", cfg.GetRunMode())
	})
}
