package config

// Config is the root configuration structure.
type Config struct {
	Extends         []string                     `yaml:"extends,omitempty"`
	WorkingDir      string                       `yaml:"working_dir,omitempty"`
	Rules           map[string]PackageManagerCfg `yaml:"rules"`
	ExcludeVersions []string                     `yaml:"exclude_versions,omitempty"`
	Groups          map[string]GroupCfg          `yaml:"groups,omitempty"`
	Incremental     []string                     `yaml:"incremental,omitempty"`
	SystemTests     *SystemTestsCfg              `yaml:"system_tests,omitempty"`
	Security        *SecurityCfg                 `yaml:"security,omitempty"`

	// NoTimeout is a runtime flag that disables command timeouts when set to true.
	// It is not persisted to YAML and is set by CLI flags (--no-timeout).
	NoTimeout bool `yaml:"-"`

	// isRootConfig is set to true only for the root config file (not imported configs).
	// Security settings can only be enabled from the root config.
	isRootConfig bool `yaml:"-"`
}

// SecurityCfg holds security-related configuration options.
// These settings can ONLY be enabled from the root config file, not from imported configs.
// This provides a central point of control for security policies.
type SecurityCfg struct {
	// AllowPathTraversal permits the use of ".." in extends paths.
	// Default: false (paths with ".." are rejected for security).
	// Use case: Corporate compliance configs stored in parent directories.
	AllowPathTraversal bool `yaml:"allow_path_traversal,omitempty"`

	// AllowAbsolutePaths permits absolute paths in extends.
	// Default: false (only relative paths are allowed).
	// Use case: Shared configs in /etc/goupdate/ or company directories.
	AllowAbsolutePaths bool `yaml:"allow_absolute_paths,omitempty"`

	// MaxConfigFileSize overrides the default 10MB limit for config files (in bytes).
	// Default: 10485760 (10MB). Set to 0 to use default.
	// Use case: Very large generated configs.
	MaxConfigFileSize int64 `yaml:"max_config_file_size,omitempty"`

	// MaxRegexComplexity sets the maximum allowed regex pattern length.
	// Default: 1000 characters. Set to 0 to use default.
	// Patterns exceeding this limit are rejected to prevent ReDoS attacks.
	MaxRegexComplexity int `yaml:"max_regex_complexity,omitempty"`

	// AllowComplexRegex disables ReDoS protection checks on regex patterns.
	// Default: false (potentially dangerous patterns are rejected).
	// WARNING: Enabling this may expose the tool to ReDoS attacks.
	AllowComplexRegex bool `yaml:"allow_complex_regex,omitempty"`
}

// IsRootConfig returns true if this is the root configuration (not an imported config).
//
// The root config is the primary configuration file loaded by the user, as opposed
// to configs that are imported via the extends mechanism. Only the root config can
// enable security settings.
//
// Returns:
//   - bool: true if this is the root config, false otherwise
func (c *Config) IsRootConfig() bool {
	return c.isRootConfig
}

// SetRootConfig marks this config as the root config.
//
// This should be called when loading the primary user configuration to enable
// security policy enforcement. Imported configs via extends are not root configs.
//
// Parameters:
//   - isRoot: true to mark as root config, false otherwise
func (c *Config) SetRootConfig(isRoot bool) {
	c.isRootConfig = isRoot
}

// GetMaxConfigFileSize returns the configured max file size or the default.
//
// This checks the security configuration for a custom max file size limit.
// If not set, returns the default of 10MB.
//
// Returns:
//   - int64: maximum allowed config file size in bytes
func (c *Config) GetMaxConfigFileSize() int64 {
	if c.Security != nil && c.Security.MaxConfigFileSize > 0 {
		return c.Security.MaxConfigFileSize
	}
	return DefaultMaxConfigFileSize
}

// GetMaxRegexComplexity returns the configured max regex complexity or the default.
//
// This checks the security configuration for a custom max regex complexity limit.
// If not set, returns the default of 1000 characters to prevent ReDoS attacks.
//
// Returns:
//   - int: maximum allowed regex pattern length in characters
func (c *Config) GetMaxRegexComplexity() int {
	if c.Security != nil && c.Security.MaxRegexComplexity > 0 {
		return c.Security.MaxRegexComplexity
	}
	return DefaultMaxRegexComplexity
}

// AllowsPathTraversal returns true if path traversal is allowed in extends.
//
// Path traversal using ".." in extends paths is disabled by default for security.
// This can be enabled in the root config's security settings.
//
// Returns:
//   - bool: true if path traversal is allowed, false otherwise
func (c *Config) AllowsPathTraversal() bool {
	return c.Security != nil && c.Security.AllowPathTraversal
}

// AllowsAbsolutePaths returns true if absolute paths are allowed in extends.
//
// Absolute paths in extends are disabled by default for security.
// This can be enabled in the root config's security settings.
//
// Returns:
//   - bool: true if absolute paths are allowed, false otherwise
func (c *Config) AllowsAbsolutePaths() bool {
	return c.Security != nil && c.Security.AllowAbsolutePaths
}

// AllowsComplexRegex returns true if complex regex patterns are allowed.
//
// Complex regex patterns that could cause ReDoS attacks are rejected by default.
// This can be disabled in the root config's security settings.
//
// Returns:
//   - bool: true if complex regex patterns are allowed, false otherwise
func (c *Config) AllowsComplexRegex() bool {
	return c.Security != nil && c.Security.AllowComplexRegex
}

// DefaultMaxConfigFileSize is the default maximum config file size (10MB).
const DefaultMaxConfigFileSize = 10 * 1024 * 1024

// DefaultMaxRegexComplexity is the default maximum regex pattern length.
const DefaultMaxRegexComplexity = 1000

// PackageSettings holds per-package configuration options at the package manager level.
type PackageSettings struct {
	// WithAllDependencies enables updating with all dependencies (-W flag for composer).
	// When true, the update command includes transitive dependencies.
	WithAllDependencies bool `yaml:"with_all_dependencies,omitempty"`
}

// GroupCfg holds group configuration for package grouping.
type GroupCfg struct {
	// Packages is the list of package names in this group.
	Packages []string `yaml:"-"`

	// WithAllDependencies enables updating with all dependencies for the entire group.
	// This applies -W flag (or equivalent) for all packages in the group.
	WithAllDependencies bool `yaml:"-"`
}

// PackageManagerCfg holds configuration for a package manager rule.
type PackageManagerCfg struct {
	// Enabled controls whether this rule is active. Defaults to true if not specified.
	// Set to false to disable a rule inherited from extends without removing it.
	Enabled           *bool                         `yaml:"enabled,omitempty"`
	Manager           string                        `yaml:"manager"`
	Include           []string                      `yaml:"include"`
	Exclude           []string                      `yaml:"exclude,omitempty"`
	Groups            map[string]GroupCfg           `yaml:"groups,omitempty"`
	// Packages holds per-package settings for individual packages outside of groups.
	// Key is the package name, value is the settings for that package.
	Packages          map[string]PackageSettings    `yaml:"packages,omitempty"`
	Format            string                        `yaml:"format"`
	Fields            map[string]string             `yaml:"fields"`
	Ignore            []string                      `yaml:"ignore,omitempty"`
	ExcludeVersions   []string                      `yaml:"exclude_versions,omitempty"`
	ConstraintMapping map[string]string             `yaml:"constraint_mapping,omitempty"`
	LatestMapping     *LatestMappingCfg             `yaml:"latest_mapping,omitempty"`
	PackageOverrides  map[string]PackageOverrideCfg `yaml:"package_overrides,omitempty"`
	Extraction        *ExtractionCfg                `yaml:"extraction,omitempty"`
	Outdated          *OutdatedCfg                  `yaml:"outdated,omitempty"`
	Update            *UpdateCfg                    `yaml:"update,omitempty"`
	LockFiles         []LockFileCfg                 `yaml:"lock_files,omitempty"`
	// SelfPinning indicates that the manifest file itself acts as the lock file.
	// When true, declared versions are used as installed versions (e.g., requirements.txt, Dockerfile).
	// This avoids "Unsupported" status for package managers without separate lock files.
	SelfPinning bool                   `yaml:"self_pinning,omitempty"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
	Incremental []string               `yaml:"incremental,omitempty"`
}

// IsEnabled returns true if the rule is enabled (defaults to true if not specified).
//
// Rules are enabled by default. The enabled field can be explicitly set to false
// to disable a rule that was inherited from extends without removing it.
//
// Returns:
//   - bool: true if the rule is enabled, false otherwise
func (p *PackageManagerCfg) IsEnabled() bool {
	if p.Enabled == nil {
		return true
	}
	return *p.Enabled
}

// ShouldUpdateWithAllDependencies returns true if the package should be updated
// with all its dependencies (e.g., -W flag for composer).
//
// Resolution order (first match wins):
//  1. Individual package settings in rules.<manager>.packages.<package>
//  2. Group-level with_all_dependencies setting (if package is in a group)
//
// Parameters:
//   - packageName: the name of the package to check
//
// Returns:
//   - bool: true if the package should be updated with all dependencies
func (p *PackageManagerCfg) ShouldUpdateWithAllDependencies(packageName string) bool {
	// Check individual package settings first (highest priority)
	if p.Packages != nil {
		if settings, ok := p.Packages[packageName]; ok {
			if settings.WithAllDependencies {
				return true
			}
		}
	}

	// Check group-level settings
	for _, group := range p.Groups {
		// Check if package is in this group
		inGroup := false
		for _, pkg := range group.Packages {
			if pkg == packageName {
				inGroup = true
				break
			}
		}

		if !inGroup {
			continue
		}

		// Check group-level setting
		if group.WithAllDependencies {
			return true
		}
	}

	return false
}

// LatestMappingCfg holds configuration for mapping version tokens to latest values.
type LatestMappingCfg struct {
	Default  map[string]string            `yaml:"default,omitempty"`
	Packages map[string]map[string]string `yaml:"packages,omitempty"`
}

// LockFileCfg holds configuration for lock file parsing.
// Use EITHER file-based parsing (format + extraction) OR command-based parsing (commands).
type LockFileCfg struct {
	// Files specifies lock file patterns for detection and rule conflict resolution.
	// When multiple rules match the same manifest (e.g., npm/pnpm/yarn all match package.json),
	// the rule with an existing lock file is preferred.
	// Example: ["**/package-lock.json"]
	Files []string `yaml:"files,omitempty"`

	// Format specifies the lock file format for file-based parsing.
	// Use with extraction for regex-based parsing. Not used when commands is set.
	Format string `yaml:"format,omitempty"`

	// Extraction configures regex-based parsing for file-based mode.
	// Not used when commands is set.
	Extraction *ExtractionCfg `yaml:"extraction,omitempty"`

	// Commands is a multiline command string for command-based parsing.
	// When set, format and extraction are ignored - the command output is parsed instead.
	// Use this when lock files have multiple versions or when maximum compatibility is needed.
	//
	// The command should output JSON in one of these formats:
	//   {"package-name": "version", ...}
	// Or:
	//   [{"name": "package-name", "version": "1.0.0"}, ...]
	//
	// Available placeholders:
	//   {{lock_file}} - Path to the lock file being processed
	//   {{base_dir}} - Directory containing the lock file
	Commands string `yaml:"commands,omitempty"`

	// Env holds environment variables to set when executing lock commands.
	Env map[string]string `yaml:"env,omitempty"`

	// TimeoutSeconds sets command execution timeout for lock parsing commands.
	// Default: 60 seconds.
	TimeoutSeconds int `yaml:"timeout_seconds,omitempty"`

	// CommandExtraction configures how to extract versions from command output.
	// If not specified, the command output is expected to be JSON with
	// {"package": "version"} or [{"name": "...", "version": "..."}] format.
	CommandExtraction *LockCommandExtractionCfg `yaml:"command_extraction,omitempty"`
}

// LockCommandExtractionCfg configures how to extract versions from lock command output.
type LockCommandExtractionCfg struct {
	// Format specifies the output format: "json" (default), "raw".
	Format string `yaml:"format,omitempty"`

	// Pattern is a regex with named groups "name" (or "n") and "version" for raw format.
	// Example: `(?P<name>[\w@/-]+)\s+(?P<version>\d+\.\d+\.\d+)`
	Pattern string `yaml:"pattern,omitempty"`

	// JSONNameKey is the JSON key for package name in array format (default: "name").
	JSONNameKey string `yaml:"json_name_key,omitempty"`

	// JSONVersionKey is the JSON key for version in array format (default: "version").
	JSONVersionKey string `yaml:"json_version_key,omitempty"`
}

// GetTimeoutSeconds returns the configured timeout or the default (60 seconds).
//
// This determines how long to wait for lock file parsing commands to complete.
//
// Returns:
//   - int: timeout in seconds (default: 60)
func (l *LockFileCfg) GetTimeoutSeconds() int {
	if l.TimeoutSeconds > 0 {
		return l.TimeoutSeconds
	}
	return 60
}

// PackageOverrideCfg holds per-package override configuration.
type PackageOverrideCfg struct {
	Ignore     bool                 `yaml:"ignore,omitempty"`
	Constraint *string              `yaml:"constraint,omitempty"`
	Version    string               `yaml:"version,omitempty"`
	Outdated   *OutdatedOverrideCfg `yaml:"outdated,omitempty"`
	Update     *UpdateOverrideCfg   `yaml:"update,omitempty"`
}

// PatternCfg defines a conditional pattern for extraction or exclusion.
// Used for multi-pattern extraction where different patterns apply to different
// file formats or versions (e.g., pnpm-lock.yaml v6 vs v9).
type PatternCfg struct {
	// Name is an optional identifier for the pattern (e.g., "v9", "classic").
	Name string `yaml:"name,omitempty"`

	// Detect is a regex pattern to check if this pattern should apply.
	// If empty, the pattern always applies (default = true).
	// If set, the pattern only applies when Detect matches the file content.
	Detect string `yaml:"detect,omitempty"`

	// Pattern is the extraction regex with named groups (e.g., "n" or "name", "version").
	Pattern string `yaml:"pattern"`
}

// ExtractionCfg holds configuration for version extraction from files.
type ExtractionCfg struct {
	// Pattern is a single extraction regex (for simple cases).
	// Use Patterns for multi-pattern extraction with conditional detection.
	Pattern string `yaml:"pattern,omitempty"`

	// Patterns is a list of conditional patterns for multi-format extraction.
	// ALL matching patterns are applied (additive, not exclusive).
	// If a pattern has no Detect, it always applies.
	// If a pattern has Detect, it only applies when Detect matches the content.
	Patterns []PatternCfg `yaml:"patterns,omitempty"`

	Path string `yaml:"path,omitempty"`
	NameAttr       string `yaml:"name_attr,omitempty"`
	VersionAttr    string `yaml:"version_attr,omitempty"`
	NameElement    string `yaml:"name_element,omitempty"`
	VersionElement string `yaml:"version_element,omitempty"`
	// DevAttr specifies an attribute name that indicates a dev dependency (e.g., "developmentDependency" for nuget).
	DevAttr string `yaml:"dev_attr,omitempty"`
	// DevValue specifies the attribute value that marks a dev dependency (e.g., "true").
	DevValue string `yaml:"dev_value,omitempty"`
	// DevElement specifies a child element name that indicates a dev dependency (e.g., "PrivateAssets" for msbuild).
	DevElement string `yaml:"dev_element,omitempty"`
	// DevElementValue specifies the element text value that marks a dev dependency (e.g., "all").
	DevElementValue string `yaml:"dev_element_value,omitempty"`
}

// OutdatedCfg holds configuration for outdated version checking.
type OutdatedCfg struct {
	// Commands is a multiline string supporting piped (|) and sequential (newline) execution.
	// Use {{package}}, {{version}}, {{constraint}} placeholders for substitution.
	Commands string `yaml:"commands,omitempty"`

	// Env holds environment variables to set when executing commands.
	Env map[string]string `yaml:"env,omitempty"`

	// Format specifies the output format: json, yaml, or raw.
	Format string `yaml:"format,omitempty"`

	// Extraction configures how to extract versions from command output.
	Extraction *OutdatedExtractionCfg `yaml:"extraction,omitempty"`

	// Versioning configures version parsing and sorting.
	Versioning *VersioningCfg `yaml:"versioning,omitempty"`

	// ExcludeVersions lists specific versions to exclude.
	ExcludeVersions []string `yaml:"exclude_versions,omitempty"`

	// ExcludeVersionPatterns lists regex patterns for versions to exclude.
	ExcludeVersionPatterns []string `yaml:"exclude_version_patterns,omitempty"`

	// TimeoutSeconds sets command execution timeout.
	TimeoutSeconds int `yaml:"timeout_seconds,omitempty"`
}

// OutdatedExtractionCfg configures how to extract versions from command output.
type OutdatedExtractionCfg struct {
	// Pattern is a regex with named group "version" for raw format extraction.
	Pattern string `yaml:"pattern,omitempty"`

	// JSONKey is a dot-separated path for JSON extraction (e.g., "versions" or "releases.keys").
	JSONKey string `yaml:"json_key,omitempty"`

	// YAMLKey is a path for YAML extraction.
	YAMLKey string `yaml:"yaml_key,omitempty"`
}

// OutdatedOverrideCfg holds per-package outdated override configuration.
type OutdatedOverrideCfg struct {
	// Commands overrides the multiline commands.
	Commands *string `yaml:"commands,omitempty"`

	// Env overrides environment variables.
	Env map[string]string `yaml:"env,omitempty"`

	// Format overrides the output format.
	Format *string `yaml:"format,omitempty"`

	// Extraction overrides the extraction configuration.
	Extraction *OutdatedExtractionCfg `yaml:"extraction,omitempty"`

	// ExcludeVersions lists specific versions to exclude.
	ExcludeVersions []string `yaml:"exclude_versions,omitempty"`

	// ExcludeVersionPatterns lists regex patterns for versions to exclude.
	ExcludeVersionPatterns []string `yaml:"exclude_version_patterns,omitempty"`

	// Versioning overrides version parsing and sorting.
	Versioning *VersioningCfg `yaml:"versioning,omitempty"`

	// TimeoutSeconds overrides the timeout.
	TimeoutSeconds *int `yaml:"timeout_seconds,omitempty"`
}

// UpdateCfg holds configuration for update commands.
type UpdateCfg struct {
	// Commands is a multiline string for lock/install commands.
	// Supports piped (|) and sequential (newline) execution.
	// Use {{package}}, {{version}}, {{constraint}} placeholders for substitution.
	// This command is run after the manifest version is updated to regenerate the lock file.
	Commands string `yaml:"commands,omitempty"`

	// Env holds environment variables to set when executing commands.
	Env map[string]string `yaml:"env,omitempty"`

	// Group associates packages with a named group for atomic updates.
	Group string `yaml:"group,omitempty"`

	// TimeoutSeconds sets command execution timeout.
	TimeoutSeconds int `yaml:"timeout_seconds,omitempty"`
}

// UpdateOverrideCfg holds per-package update override configuration.
type UpdateOverrideCfg struct {
	// Commands overrides the multiline commands.
	Commands *string `yaml:"commands,omitempty"`

	// Env overrides environment variables for commands.
	Env map[string]string `yaml:"env,omitempty"`

	// Group overrides the group name.
	Group *string `yaml:"group,omitempty"`

	// TimeoutSeconds overrides the timeout.
	TimeoutSeconds *int `yaml:"timeout_seconds,omitempty"`
}

// VersioningCfg holds configuration for version parsing and sorting.
type VersioningCfg struct {
	Format string `yaml:"format,omitempty"`
	Regex  string `yaml:"regex,omitempty"`
	Sort   string `yaml:"sort,omitempty"`
}

// SystemTestCfg defines a single system test configuration.
type SystemTestCfg struct {
	// Name is the identifier for this test (e.g., "unit-tests", "e2e-tests").
	Name string `yaml:"name"`

	// Commands is a multiline string of test commands to execute.
	// Supports piped (|) and sequential (newline) execution.
	Commands string `yaml:"commands"`

	// Env holds environment variables to set when executing test commands.
	Env map[string]string `yaml:"env,omitempty"`

	// TimeoutSeconds sets the timeout for test execution (default: 300).
	TimeoutSeconds int `yaml:"timeout_seconds,omitempty"`

	// ContinueOnFail allows the update process to continue even if this test fails.
	// Useful for non-critical tests that shouldn't block updates.
	ContinueOnFail bool `yaml:"continue_on_fail,omitempty"`
}

// SystemTestsCfg defines the system tests configuration for the update command.
type SystemTestsCfg struct {
	// Tests is a list of test configurations to run.
	Tests []SystemTestCfg `yaml:"tests"`

	// RunPreflight determines whether to run tests before any updates begin.
	// This validates the application works before making changes.
	// Default: true
	RunPreflight *bool `yaml:"run_preflight,omitempty"`

	// RunMode determines when tests run after updates:
	// - "after_each": Run after each package update (maximum safety, slower)
	// - "after_all": Run once after all updates complete (faster)
	// - "none": Only run preflight tests (if enabled)
	// Default: "after_all"
	RunMode string `yaml:"run_mode,omitempty"`

	// StopOnFail determines whether to stop updates if a test fails.
	// Default: true
	StopOnFail *bool `yaml:"stop_on_fail,omitempty"`
}

// IsRunPreflight returns whether preflight tests should run (defaults to true).
//
// Preflight tests validate the application works before making any updates.
// This helps ensure a clean baseline before changes are applied.
//
// Returns:
//   - bool: true if preflight tests should run, false otherwise
func (s *SystemTestsCfg) IsRunPreflight() bool {
	if s.RunPreflight == nil {
		return true
	}
	return *s.RunPreflight
}

// IsStopOnFail returns whether to stop on test failure (defaults to true).
//
// When true, the update process will halt if any test fails.
// When false, updates will continue even after test failures.
//
// Returns:
//   - bool: true if updates should stop on test failure, false otherwise
func (s *SystemTestsCfg) IsStopOnFail() bool {
	if s.StopOnFail == nil {
		return true
	}
	return *s.StopOnFail
}

// GetRunMode returns the run mode (defaults to "after_all").
//
// The run mode determines when tests are executed:
//   - "after_each": Run after each package update (maximum safety, slower)
//   - "after_all": Run once after all updates complete (faster)
//   - "none": Only run preflight tests (if enabled)
//
// Returns:
//   - string: the run mode (default: "after_all")
func (s *SystemTestsCfg) GetRunMode() string {
	if s.RunMode == "" {
		return "after_all"
	}
	return s.RunMode
}

// System test run mode constants.
const (
	SystemTestRunModeAfterEach = "after_each"
	SystemTestRunModeAfterAll  = "after_all"
	SystemTestRunModeNone      = "none"
)
