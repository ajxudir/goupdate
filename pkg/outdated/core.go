// Package outdated provides functionality for detecting available package updates.
// It executes configured commands to fetch version information, filters versions
// based on constraints and exclusion patterns, and supports multiple versioning
// strategies (semver, calver, ordered).
package outdated

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/utils"
	"github.com/ajxudir/goupdate/pkg/verbose"
)

var (
	productionSafeVersionPattern   = "(?i)(?:^|[._\\-/])((?:alpha|beta|rc|canary|dev|snapshot|nightly|preview)(?:[._\\-/]?[0-9A-Za-z]+)*)(?:\\+[^\\s]*)?$"
	fallbackExcludeVersionPatterns = []string{productionSafeVersionPattern}
)

var supportedConstraints = map[string]bool{
	"":   true,
	"^":  true,
	"~":  true,
	">=": true,
	"<=": true,
	">":  true,
	"<":  true,
	"=":  true,
	"*":  true,
}

// ListNewerVersions runs the configured command for a package and returns newer versions.
// It prefers installed versions for comparison and falls back to declared constraints.
// The context parameter allows callers to cancel long-running operations.
func ListNewerVersions(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	verbose.Debugf("Checking for updates: %s (current: %s, constraint: %q)",
		p.Name, CurrentVersionForOutdated(p), p.Constraint)

	outdatedCfg, err := resolveOutdatedCfg(p, cfg)
	if err != nil {
		return nil, err
	}

	strategy, err := newVersioningStrategy(outdatedCfg.Versioning)
	if err != nil {
		return nil, err
	}

	if outdatedCfg.Versioning != nil {
		verbose.Debugf("Versioning strategy: format=%q, sort=%q",
			outdatedCfg.Versioning.Format, outdatedCfg.Versioning.Sort)
	}

	scopeDir := resolveOutdatedScope(p, cfg, baseDir)
	verbose.Debugf("Running outdated command in directory: %s", scopeDir)

	output, err := runOutdatedCommand(ctx, outdatedCfg, p, scopeDir)
	if err != nil {
		return nil, err
	}

	versions, err := parseAvailableVersionsForPackage(p.Name, outdatedCfg, output)
	if err != nil {
		return nil, err
	}

	verbose.Debugf("Parsed %d available versions for %s", len(versions), p.Name)
	// Show all retrieved versions at DEBUG level for debugging version issues
	if len(versions) > 0 {
		if len(versions) <= 10 {
			verbose.Debugf("Raw versions for %s: %v", p.Name, versions)
		} else {
			verbose.Debugf("Raw versions for %s: %v... (%d more)", p.Name, versions[:10], len(versions)-10)
		}
	}
	if verbose.IsTrace() && len(versions) > 10 {
		verbose.Tracef("All retrieved tags for %s: %v", p.Name, versions)
	}

	beforeExclusions := len(versions)
	versionsAfterExclusions, err := applyVersionExclusions(versions, outdatedCfg, cfg.Security)
	if err != nil {
		return nil, err
	}

	if beforeExclusions != len(versionsAfterExclusions) {
		excluded := findExcludedVersions(versions, versionsAfterExclusions)
		verbose.Debugf("Excluded %d versions (before: %d, after: %d)",
			beforeExclusions-len(versionsAfterExclusions), beforeExclusions, len(versionsAfterExclusions))
		// Show excluded versions at DEBUG level for debugging
		if len(excluded) > 0 {
			if len(excluded) <= 10 {
				verbose.Debugf("Excluded versions for %s: %v", p.Name, excluded)
			} else {
				verbose.Debugf("Excluded versions for %s: %v... (%d more)", p.Name, excluded[:10], len(excluded)-10)
			}
		}
		if verbose.IsTrace() && len(excluded) > 10 {
			verbose.Tracef("All excluded versions for %s: %v", p.Name, excluded)
		}
	}
	versions = versionsAfterExclusions

	filtered := filterNewerVersionsWithStrategy(CurrentVersionForOutdated(p), versions, strategy)
	verbose.Debugf("Found %d newer versions for %s (current: %s)", len(filtered), p.Name, CurrentVersionForOutdated(p))
	// Show newer versions at DEBUG level
	if len(filtered) > 0 {
		if len(filtered) <= 10 {
			verbose.Debugf("Newer versions for %s: %v", p.Name, filtered)
		} else {
			verbose.Debugf("Newer versions for %s: %v... (%d more)", p.Name, filtered[:10], len(filtered)-10)
		}
	}
	if verbose.IsTrace() && len(filtered) > 10 {
		verbose.Tracef("All newer versions for %s: %v", p.Name, filtered)
	}

	return filtered, nil
}

// resolveOutdatedCfg builds the effective outdated configuration for a package.
//
// It performs the following operations:
//   - Retrieves base configuration from the package's rule
//   - Applies package-specific overrides if configured
//   - Merges default version exclusions
//   - Applies NoTimeout flag from runtime config
//
// Parameters:
//   - p: The package to resolve configuration for
//   - cfg: The global configuration containing rules and overrides
//
// Returns:
//   - *config.OutdatedCfg: The effective outdated configuration with all overrides applied
//   - error: When rule is missing or outdated config is not defined; returns nil on success
func resolveOutdatedCfg(p formats.Package, cfg *config.Config) (*config.OutdatedCfg, error) {
	ruleCfg, ok := cfg.Rules[p.Rule]
	if !ok {
		return nil, fmt.Errorf("rule configuration missing for %s", p.Rule)
	}

	if ruleCfg.Outdated == nil {
		return nil, &errors.UnsupportedError{Reason: fmt.Sprintf("outdated configuration missing for %s", p.Rule)}
	}

	verbose.Tracef("Using outdated config from rule %q for package %s", p.Rule, p.Name)

	effective := cloneOutdatedCfg(ruleCfg.Outdated)

	var overrideCfg *config.OutdatedOverrideCfg
	if ruleCfg.PackageOverrides != nil {
		if override, ok := ruleCfg.PackageOverrides[p.Name]; ok {
			overrideCfg = override.Outdated
			verbose.Tracef("Package %s has package_overrides configured", p.Name)
		}
	}

	if overrideCfg != nil {
		if overrideCfg.Versioning != nil {
			effective.Versioning = overrideCfg.Versioning
			verbose.Tracef("Package %s: using custom versioning from override", p.Name)
		}

		if overrideCfg.ExcludeVersions != nil {
			effective.ExcludeVersions = cloneStringSlice(overrideCfg.ExcludeVersions)
			verbose.Tracef("Package %s: using custom exclude_versions from override", p.Name)
		}

		if overrideCfg.ExcludeVersionPatterns != nil {
			effective.ExcludeVersionPatterns = cloneStringSlice(overrideCfg.ExcludeVersionPatterns)
			verbose.Tracef("Package %s: using custom exclude_version_patterns from override", p.Name)
		}

		if overrideCfg.TimeoutSeconds != nil {
			effective.TimeoutSeconds = *overrideCfg.TimeoutSeconds
			verbose.Tracef("Package %s: using custom timeout %ds from override", p.Name, *overrideCfg.TimeoutSeconds)
		}
	}

	applyDefaultExclusions(effective, resolveDefaultExclusions(cfg, ruleCfg))

	// Apply NoTimeout flag from runtime config
	if cfg.NoTimeout {
		effective.TimeoutSeconds = 0
	}

	return effective, nil
}

// resolveDefaultExclusions determines which default exclusion patterns to use.
//
// Parameters:
//   - cfg: The global configuration
//   - ruleCfg: The package manager rule configuration
//
// Returns:
//   - []string: Default exclusion patterns from rule config, or global config if rule has none
func resolveDefaultExclusions(cfg *config.Config, ruleCfg config.PackageManagerCfg) []string {
	if ruleCfg.ExcludeVersions != nil {
		return ruleCfg.ExcludeVersions
	}
	return cfg.ExcludeVersions
}

// cloneOutdatedCfg creates a deep copy of an outdated configuration.
//
// It performs the following operations:
//   - Copies the config structure
//   - Clones all slice fields to prevent mutation
//   - Clones map fields (Env) to prevent mutation
//   - Clones nested Extraction config
//
// Parameters:
//   - cfg: The outdated configuration to clone; may be nil
//
// Returns:
//   - *config.OutdatedCfg: A deep copy of the configuration, or nil if input is nil
func cloneOutdatedCfg(cfg *config.OutdatedCfg) *config.OutdatedCfg {
	if cfg == nil {
		return nil
	}

	cloned := *cfg
	cloned.ExcludeVersions = cloneStringSlice(cfg.ExcludeVersions)
	cloned.ExcludeVersionPatterns = cloneStringSlice(cfg.ExcludeVersionPatterns)
	cloned.TimeoutSeconds = cfg.TimeoutSeconds

	// Clone new fields
	if cfg.Env != nil {
		cloned.Env = make(map[string]string, len(cfg.Env))
		for k, v := range cfg.Env {
			cloned.Env[k] = v
		}
	}

	if cfg.Extraction != nil {
		extraction := *cfg.Extraction
		cloned.Extraction = &extraction
	}

	return &cloned
}

// cloneStringSlice creates a deep copy of a string slice.
//
// Parameters:
//   - values: The string slice to clone; may be nil
//
// Returns:
//   - []string: A copy of the slice, or nil if input is nil
func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}

	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

// resolveOutdatedScope determines the working directory for executing outdated commands.
//
// It performs the following operations:
//   - Prefers the directory containing the package's source file
//   - Falls back to the provided baseDir
//   - Falls back to the config's working directory
//   - Defaults to current directory "." if all else fails
//
// Parameters:
//   - p: The package to resolve scope for
//   - cfg: The global configuration
//   - baseDir: The base directory from the package listing
//
// Returns:
//   - string: The resolved working directory path
func resolveOutdatedScope(p formats.Package, cfg *config.Config, baseDir string) string {
	scopeDir := baseDir

	if p.Source != "" {
		scopeDir = filepath.Dir(p.Source)
	}

	if scopeDir == "" {
		scopeDir = cfg.WorkingDir
	}

	if scopeDir == "" {
		scopeDir = "."
	}

	return scopeDir
}

// applyDefaultExclusions merges default exclusion patterns into the configuration.
//
// It performs the following operations:
//   - Uses provided defaults, or fallback patterns if defaults are empty
//   - Skips merge if config patterns are explicitly empty (user opt-out)
//   - Deduplicates patterns when merging
//
// Parameters:
//   - cfg: The outdated configuration to modify (modified in place)
//   - defaults: Default exclusion patterns to merge in
func applyDefaultExclusions(cfg *config.OutdatedCfg, defaults []string) {
	defaultPatterns := defaults
	if len(defaultPatterns) == 0 {
		defaultPatterns = fallbackExcludeVersionPatterns
	}

	if cfg.ExcludeVersionPatterns == nil {
		cfg.ExcludeVersionPatterns = append([]string{}, defaultPatterns...)
		return
	}

	if len(cfg.ExcludeVersionPatterns) == 0 {
		return
	}

	patternSet := make(map[string]struct{})
	for _, pattern := range cfg.ExcludeVersionPatterns {
		patternSet[pattern] = struct{}{}
	}

	for _, pattern := range defaultPatterns {
		if _, exists := patternSet[pattern]; !exists {
			cfg.ExcludeVersionPatterns = append(cfg.ExcludeVersionPatterns, pattern)
			patternSet[pattern] = struct{}{}
		}
	}
}

// runOutdatedCommand executes the configured outdated command and handles errors.
//
// It performs the following operations:
//   - Validates that command is configured
//   - Executes command with package information and context
//   - Normalizes known errors to UnsupportedError when appropriate
//   - Extracts command name for error reporting
//
// Parameters:
//   - ctx: Context for cancellation support
//   - cfg: The outdated configuration with command to execute
//   - p: The package to check for updates
//   - dir: The working directory for command execution
//
// Returns:
//   - []byte: Raw output from the command execution
//   - error: When command is empty, execution fails, or parsing fails; returns nil on success
func runOutdatedCommand(ctx context.Context, cfg *config.OutdatedCfg, p formats.Package, dir string) ([]byte, error) {
	if strings.TrimSpace(cfg.Commands) == "" {
		return nil, fmt.Errorf("outdated command is empty")
	}

	output, err := execOutdatedFunc(ctx, cfg, p.Name, CurrentVersionForOutdated(p), p.Constraint, dir)
	if err != nil {
		// Extract first command name for error message
		commandName := ""
		// Normalize line endings for cross-platform compatibility (CRLF -> LF)
		lines := strings.Split(strings.ReplaceAll(cfg.Commands, "\r\n", "\n"), "\n")
		if len(lines) > 0 {
			parts := strings.Fields(strings.TrimSpace(lines[0]))
			if len(parts) > 0 {
				commandName = parts[0]
			}
		}

		if normalized := normalizeOutdatedError(err, commandName); normalized != err {
			return nil, normalized
		}

		return nil, fmt.Errorf("failed to execute outdated command: %w", err)
	}

	return output, nil
}

// ensureGoModFlag adds -mod=mod flag to Go commands if not already present.
//
// It performs the following operations:
//   - Checks if command is "go" (case-insensitive)
//   - Skips if -mod flag already exists
//   - Inserts -mod=mod after the subcommand
//
// Parameters:
//   - command: The command name to check
//   - args: The argument list for the command
//
// Returns:
//   - []string: Modified arguments with -mod=mod added if needed, or original args unchanged
func ensureGoModFlag(command string, args []string) []string {
	if !strings.EqualFold(command, "go") {
		return args
	}

	for _, arg := range args {
		if arg == "-mod" || strings.HasPrefix(arg, "-mod=") {
			return args
		}
	}

	insertAt := 0
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		insertAt = 1
	}

	withMod := make([]string, 0, len(args)+1)
	withMod = append(withMod, args[:insertAt]...)
	withMod = append(withMod, "-mod=mod")
	withMod = append(withMod, args[insertAt:]...)

	return withMod
}

// normalizeOutdatedError converts known command-specific errors to UnsupportedError.
//
// It performs the following operations:
//   - Checks if command is "dotnet" (case-insensitive)
//   - Detects known unsupported scenarios (missing assets, multiple projects)
//   - Wraps detected errors as UnsupportedError
//
// Parameters:
//   - err: The error to potentially normalize
//   - command: The command name that was executed
//
// Returns:
//   - error: UnsupportedError if error matches known patterns; original error otherwise
func normalizeOutdatedError(err error, command string) error {
	if err == nil {
		return nil
	}

	if !strings.EqualFold(command, "dotnet") {
		return err
	}

	message := err.Error()
	if strings.Contains(message, "No assets file was found") || strings.Contains(message, "Found more than one project") {
		return &errors.UnsupportedError{Reason: message}
	}

	return err
}

// applyVersionExclusions filters out versions matching exclusion rules.
//
// It performs the following operations:
//   - Builds exclusion set from exact version matches
//   - Compiles and validates regex patterns for safety
//   - Filters versions against both exact matches and patterns
//
// Parameters:
//   - versions: List of available versions to filter
//   - cfg: Outdated configuration with exclusion rules
//
// Returns:
//   - []string: Filtered versions with exclusions removed
//   - error: When regex pattern is invalid or unsafe; returns nil on success
func applyVersionExclusions(versions []string, cfg *config.OutdatedCfg, secCfg *config.SecurityCfg) ([]string, error) {
	if cfg == nil || (len(cfg.ExcludeVersions) == 0 && len(cfg.ExcludeVersionPatterns) == 0) {
		return versions, nil
	}

	exclude := make(map[string]struct{}, len(cfg.ExcludeVersions))
	for _, v := range cfg.ExcludeVersions {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			exclude[trimmed] = struct{}{}
		}
	}

	// Build regex validation options from security config
	regexOpts := utils.RegexValidationOptions{}
	if secCfg != nil {
		regexOpts.SkipComplexityCheck = secCfg.AllowComplexRegex
		if secCfg.MaxRegexComplexity > 0 {
			regexOpts.MaxLength = secCfg.MaxRegexComplexity
		}
	}

	var regexes []*regexp.Regexp
	for _, pattern := range cfg.ExcludeVersionPatterns {
		if strings.TrimSpace(pattern) == "" {
			continue
		}

		// Validate regex safety to prevent ReDoS attacks (configurable via security settings)
		if err := utils.ValidateRegexSafetyWithOptions(pattern, regexOpts); err != nil {
			return nil, fmt.Errorf("unsafe exclude_version_patterns entry '%s': %w", pattern, err)
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude_version_patterns entry '%s': %w", pattern, err)
		}
		regexes = append(regexes, re)
	}

	filtered := make([]string, 0, len(versions))

	for _, version := range versions {
		trimmed := strings.TrimSpace(version)
		if trimmed == "" {
			continue
		}

		if _, found := exclude[trimmed]; found {
			continue
		}

		excluded := false
		for _, re := range regexes {
			if re.MatchString(trimmed) {
				excluded = true
				break
			}
		}

		if !excluded {
			filtered = append(filtered, trimmed)
		}
	}

	return filtered, nil
}

// CurrentVersionForOutdated returns the version to use for outdated comparison.
func CurrentVersionForOutdated(p formats.Package) string {
	current := strings.TrimSpace(p.InstalledVersion)
	if current != "" && current != "#N/A" {
		return current
	}

	return strings.TrimSpace(p.Version)
}

// SummarizeAvailableVersions returns the best major, minor, and patch candidates.
//
// It categorizes available versions into major, minor, and patch update candidates based on
// the current version. For each category, it selects either the newest (non-incremental) or
// nearest (incremental) version.
//
// Special handling:
//   - Pre-release to stable transitions (e.g., 1.0.0-rc03 → 1.0.0) are detected as patch updates
//     when the major.minor.patch numbers are identical but the stable release is newer
//   - Non-semver versions (4+ segments, calver, ordered) use extracted numeric parts for comparison
//
// Parameters:
//   - current: The current version to compare against
//   - versions: List of available versions to evaluate
//   - cfg: Versioning configuration (nil uses semver defaults)
//   - incremental: When true, selects nearest version; when false, selects newest
//
// Returns:
//   - string: Best major update candidate (or "#N/A" if none)
//   - string: Best minor update candidate (or "#N/A" if none)
//   - string: Best patch update candidate (or "#N/A" if none)
//   - error: When versioning strategy creation fails; nil on success
func SummarizeAvailableVersions(current string, versions []string, cfg *config.VersioningCfg, incremental bool) (string, string, string, error) {
	strategy, err := newVersioningStrategy(cfg)
	if err != nil {
		return "#N/A", "#N/A", "#N/A", err
	}

	base, ok := strategy.parseVersion(current)
	if !ok {
		verbose.Debugf("Version summarization: could not parse current version %q", current)
		return "#N/A", "#N/A", "#N/A", nil
	}

	var majorCandidate, minorCandidate, patchCandidate *parsedVersion

	isBetterCandidate := func(candidate *parsedVersion, parsed parsedVersion) bool {
		if candidate == nil {
			return true
		}

		if incremental {
			return strategy.compare(parsed, *candidate) < 0
		}

		return strategy.compare(parsed, *candidate) > 0
	}

	for _, version := range versions {
		parsed, valid := strategy.parseVersion(version)
		if !valid {
			continue
		}

		switch {
		case parsed.major > base.major:
			if isBetterCandidate(majorCandidate, parsed) {
				copy := parsed
				majorCandidate = &copy
			}
		case parsed.major == base.major && parsed.minor > base.minor:
			if isBetterCandidate(minorCandidate, parsed) {
				copy := parsed
				minorCandidate = &copy
			}
		case parsed.major == base.major && parsed.minor == base.minor && parsed.patch > base.patch:
			if isBetterCandidate(patchCandidate, parsed) {
				copy := parsed
				patchCandidate = &copy
			}
		case parsed.major == base.major && parsed.minor == base.minor && parsed.patch == base.patch:
			// Same major.minor.patch - check if candidate is newer via full semver comparison.
			// This handles prerelease → stable transitions (e.g., 1.0.0-rc03 → 1.0.0)
			// and prerelease → newer prerelease (e.g., 1.0.0-alpha → 1.0.0-beta).
			if strategy.compare(parsed, base) > 0 {
				if isBetterCandidate(patchCandidate, parsed) {
					copy := parsed
					patchCandidate = &copy
				}
			}
		}
	}

	major := "#N/A"
	minor := "#N/A"
	patch := "#N/A"

	if majorCandidate != nil {
		major = majorCandidate.raw
	}

	if minorCandidate != nil {
		minor = minorCandidate.raw
	}

	if patchCandidate != nil {
		patch = patchCandidate.raw
	}

	verbose.Debugf("Version candidates: major=%s, minor=%s, patch=%s (incremental=%v)",
		major, minor, patch, incremental)

	return major, minor, patch, nil
}

// FilterNewerVersions returns versions newer than current using the provided versioning config.
func FilterNewerVersions(current string, versions []string, cfg *config.VersioningCfg) ([]string, error) {
	strategy, err := newVersioningStrategy(cfg)
	if err != nil {
		return nil, err
	}

	return filterNewerVersionsWithStrategy(current, versions, strategy), nil
}

// filterNewerVersionsWithStrategy filters and sorts versions newer than current using a strategy.
//
// It performs the following operations:
//   - Uses position-based filtering for ordered format
//   - Parses and compares versions for other formats
//   - Deduplicates by normalized key
//   - Separates comparable and passthrough versions
//   - Sorts results according to strategy
//
// Parameters:
//   - current: The current version to compare against
//   - versions: List of available versions to filter
//   - strategy: The versioning strategy for parsing and comparison
//
// Returns:
//   - []string: Filtered and sorted versions newer than current
func filterNewerVersionsWithStrategy(current string, versions []string, strategy versioningStrategy) []string {
	if strategy.format == versionFormatOrdered {
		return strategy.filterOrdered(current, versions)
	}

	base, baseOK := strategy.parseVersion(current)
	seen := make(map[string]struct{})

	var comparable []parsedVersion
	var passthrough []string

	for _, version := range versions {
		cleaned := strings.TrimSpace(version)
		if cleaned == "" {
			continue
		}

		parsed, ok := strategy.parseVersion(cleaned)
		key := strategy.keyFor(parsed, cleaned)

		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		if ok {
			if !baseOK || strategy.compare(parsed, base) > 0 {
				comparable = append(comparable, parsed)
			}
			continue
		}

		if !baseOK {
			passthrough = append(passthrough, cleaned)
		}
	}

	strategy.sortComparable(comparable)
	sort.Strings(passthrough)

	filtered := make([]string, 0, len(comparable)+len(passthrough))
	for _, entry := range comparable {
		filtered = append(filtered, entry.raw)
	}

	filtered = append(filtered, passthrough...)
	return filtered
}

// UpdateSelectionFlags controls which version upgrades to consider.
type UpdateSelectionFlags struct {
	Major bool
	Minor bool
	Patch bool
}

// FilterVersionsByConstraint narrows available versions to those permitted by the package constraint or flag overrides.
//
// When flags are provided, they override constraint semantics to limit the scope of acceptable upgrades.
//
// Special handling:
//   - Non-semver versions (4+ segments like 1.0.0.0, calver like 2024.01.15) are passed through
//     when there's no constraint or when the reference version is also non-semver
//   - This ensures 4-segment and other non-standard version formats are not silently dropped
//
// Parameters:
//   - p: Package with version and constraint information
//   - versions: List of available versions to filter
//   - flags: Override flags for major/minor/patch scope
//
// Returns:
//   - []string: Versions permitted by the constraint or flags
func FilterVersionsByConstraint(p formats.Package, versions []string, flags UpdateSelectionFlags) []string {
	constraint := NormalizeConstraint(p.Constraint)
	constraintSegments := countConstraintSegments(p.Version)

	originalConstraint := constraint
	switch {
	case flags.Major:
		constraint = ""
	case flags.Minor:
		constraint = "^"
	case flags.Patch:
		constraint = "~"
	}

	currentVersion := CurrentVersionForOutdated(p)
	reference := canonicalSemver(p.Version)
	if reference == "" {
		reference = canonicalSemver(currentVersion)
		if constraintSegments == 0 {
			constraintSegments = countConstraintSegments(currentVersion)
		}
	}

	if (flags.Major || flags.Minor || flags.Patch) && canonicalSemver(currentVersion) != "" {
		reference = canonicalSemver(currentVersion)
		constraintSegments = countConstraintSegments(currentVersion)
	}

	if constraint == "*" {
		constraint = ""
	}

	// Log constraint filtering details
	verbose.Debugf("FilterVersionsByConstraint for %s: input=%d versions, constraint=%q (original=%q), reference=%s, flags={major=%v, minor=%v, patch=%v}",
		p.Name, len(versions), constraint, originalConstraint, reference, flags.Major, flags.Minor, flags.Patch)

	allowed := make([]string, 0, len(versions))

	for _, raw := range versions {
		canonical := canonicalSemver(raw)

		// For non-semver versions (4+ segments, calver, etc.), allow them through
		// when there's no constraint or when the reference itself is non-semver
		if canonical == "" {
			if constraint == "" || reference == "" {
				// No constraint or reference is also non-semver - pass through
				allowed = append(allowed, raw)
			}
			continue
		}

		switch constraint {
		case "^":
			if reference == "" || semver.Major(reference) == semver.Major(canonical) {
				allowed = append(allowed, raw)
			}
		case "~":
			if reference == "" {
				allowed = append(allowed, raw)
				continue
			}

			if semver.Major(reference) == semver.Major(canonical) && semver.MajorMinor(reference) == semver.MajorMinor(canonical) {
				allowed = append(allowed, raw)
			}
		case ">=":
			if reference == "" || semver.Compare(canonical, reference) >= 0 {
				allowed = append(allowed, raw)
			}
		case ">":
			if reference == "" || semver.Compare(canonical, reference) > 0 {
				allowed = append(allowed, raw)
			}
		case "<=":
			if reference == "" || semver.Compare(canonical, reference) <= 0 {
				allowed = append(allowed, raw)
			}
		case "<":
			if reference == "" || semver.Compare(canonical, reference) < 0 {
				allowed = append(allowed, raw)
			}
		case "=":
			if reference == "" {
				allowed = append(allowed, raw)
				continue
			}

			if matchesExactConstraint(reference, canonical, constraintSegments) {
				allowed = append(allowed, raw)
			}
		default:
			allowed = append(allowed, raw)
		}
	}

	// Log filtering results
	filtered := len(versions) - len(allowed)
	if filtered > 0 {
		verbose.Debugf("FilterVersionsByConstraint for %s: filtered out %d versions, %d remaining", p.Name, filtered, len(allowed))
		if verbose.IsTrace() && len(allowed) > 0 {
			verbose.Tracef("Allowed versions for %s after constraint filter: %v", p.Name, allowed)
		}
	} else {
		verbose.Debugf("FilterVersionsByConstraint for %s: all %d versions allowed", p.Name, len(allowed))
	}

	return allowed
}

// matchesExactConstraint checks if a candidate matches reference at the specified precision.
//
// It performs the following operations:
//   - For 1 segment: matches major version only
//   - For 2 segments: matches major.minor
//   - For 3+ segments: matches exact version
//
// Parameters:
//   - reference: The reference version to match against
//   - candidate: The candidate version to check
//   - segments: Number of version segments to consider (1, 2, or 3)
//
// Returns:
//   - bool: True if candidate matches reference at specified precision, false otherwise
func matchesExactConstraint(reference, candidate string, segments int) bool {
	if reference == "" || candidate == "" {
		return false
	}

	switch {
	case segments <= 1:
		return semver.Major(reference) == semver.Major(candidate)
	case segments == 2:
		return semver.MajorMinor(reference) == semver.MajorMinor(candidate)
	default:
		return semver.Compare(candidate, reference) == 0
	}
}

// NormalizeConstraint returns a canonical symbol for a constraint string.
func NormalizeConstraint(constraint string) string {
	trimmed := strings.TrimSpace(strings.ToLower(constraint))

	switch trimmed {
	case "==":
		return "="
	case "~=":
		return "~"
	case "exact":
		return "="
	}

	if supportedConstraints[trimmed] {
		return trimmed
	}

	return "="
}

// countConstraintSegments counts the number of version segments in a version string.
//
// It performs the following operations:
//   - Strips "v" prefix and trims whitespace
//   - Splits on dots and counts non-empty parts
//   - Caps at 3 segments maximum
//
// Parameters:
//   - version: The version string to analyze (e.g., "1.2", "v1.2.3")
//
// Returns:
//   - int: Number of segments (0-3); 0 if version is empty or "#N/A"
func countConstraintSegments(version string) int {
	cleaned := strings.TrimPrefix(strings.TrimSpace(version), "v")
	if cleaned == "" || cleaned == "#N/A" {
		return 0
	}

	parts := strings.Split(cleaned, ".")
	count := 0
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			count++
		}
	}

	if count > 3 {
		return 3
	}

	return count
}

// IsExactConstraint reports whether the provided constraint requires an exact match.
func IsExactConstraint(constraint string) bool {
	return NormalizeConstraint(constraint) == "="
}

// IsFullyPinnedVersion reports whether the version is fully pinned (has 3+ segments).
// Versions with fewer segments (e.g., "5.4") allow patch updates within the same major.minor.
// Versions with 3+ segments (e.g., "5.4.1") are considered truly exact and should not be updated.
func IsFullyPinnedVersion(version string) bool {
	return countConstraintSegments(version) >= 3
}

// hasVersion checks if a version string is valid (not empty or placeholder).
func hasVersion(v string) bool {
	return v != "#N/A" && v != ""
}

// selectFirstValid returns the first valid version from the candidates in order.
func selectFirstValid(candidates ...string) (string, bool) {
	for _, v := range candidates {
		if hasVersion(v) {
			return v, true
		}
	}
	return "", false
}

// getVersionCandidates returns version candidates in order based on scope and mode.
// Incremental mode: smallest step first (patch → minor → major)
// Non-incremental mode: largest step first (major → minor → patch)
func getVersionCandidates(major, minor, patch string, scope string, incremental bool) []string {
	switch scope {
	case "major":
		if incremental {
			return []string{patch, minor, major}
		}
		return []string{major, minor, patch}
	case "minor":
		if incremental {
			return []string{patch, minor}
		}
		return []string{minor, patch}
	case "patch":
		return []string{patch}
	}
	return nil
}

// determineScope returns the scope based on flags and constraint.
func determineScope(flags UpdateSelectionFlags, constraint string) string {
	switch {
	case flags.Major:
		return "major"
	case flags.Minor:
		return "minor"
	case flags.Patch:
		return "patch"
	default:
		normalized := NormalizeConstraint(constraint)
		switch normalized {
		case "", "*":
			return "major"
		case "^":
			return "minor"
		case "~":
			return "patch"
		}
	}
	return "major"
}

// SelectTargetVersion selects the appropriate target version based on selection flags and constraint.
// When incremental is true, it prioritizes patch → minor → major (smallest step first),
// while still respecting the scope allowed by flags.
func SelectTargetVersion(major, minor, patch string, flags UpdateSelectionFlags, constraint string, incremental bool) (string, error) {
	scope := determineScope(flags, constraint)
	candidates := getVersionCandidates(major, minor, patch, scope, incremental)

	verbose.Debugf("Target selection: scope=%s, incremental=%v, candidates=%v", scope, incremental, candidates)

	if v, ok := selectFirstValid(candidates...); ok {
		verbose.Debugf("Selected target version: %s", v)
		return v, nil
	}
	return "", fmt.Errorf("no suitable version found")
}

// findExcludedVersions returns versions that were in 'before' but not in 'after'.
//
// Parameters:
//   - before: Original list of versions
//   - after: Filtered list of versions
//
// Returns:
//   - []string: Versions that were excluded (present in before but not in after)
func findExcludedVersions(before, after []string) []string {
	afterSet := make(map[string]struct{}, len(after))
	for _, v := range after {
		afterSet[v] = struct{}{}
	}

	var excluded []string
	for _, v := range before {
		if _, exists := afterSet[v]; !exists {
			excluded = append(excluded, v)
		}
	}
	return excluded
}
