// Package lock provides functionality for resolving installed package versions
// from lock files. It supports various lock file formats including package-lock.json,
// pnpm-lock.yaml, yarn.lock, go.sum, custom extraction patterns, and custom commands.
//
// # Design Philosophy
//
// The JSON parsing in this package is intentionally designed to handle generic formats
// rather than package-manager-specific implementations. This approach allows:
//   - Reuse across multiple package managers (npm, pnpm, yarn all share similar formats)
//   - Support for custom tools that output standard JSON structures
//   - Easy extension for new package managers without code changes
//
// Package-manager-specific quirks (like yarn's name@version format) are handled when
// they represent genuinely unique formats that could also be useful for other tools.
package lock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajxudir/goupdate/pkg/cmdexec"
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/utils"
	"github.com/ajxudir/goupdate/pkg/verbose"
)

var (
	findFilesByPatterns   = utils.FindFilesByPatterns
	extractVersionsFromFn = extractVersionsFromLock
)

// ApplyInstalledVersions enriches packages with installed version and status information
// based on lock file configuration.
//
// It performs the following operations:
//   - Groups packages by rule and scope directory
//   - Resolves installed versions from lock files for each scope
//   - Sets InstalledVersion and InstallStatus fields for each package
//   - Handles self-pinning rules where manifest is the lock file
//   - Marks floating constraints that cannot be updated automatically
//
// Parameters:
//   - packages: Slice of packages to enrich with installed version information
//   - cfg: Configuration containing rule definitions and lock file settings
//   - baseDir: Base directory for resolving relative lock file paths
//
// Returns:
//   - []formats.Package: Enriched packages with InstalledVersion and InstallStatus set
//   - error: When lock file resolution fails, returns error; otherwise returns nil
func ApplyInstalledVersions(packages []formats.Package, cfg *config.Config, baseDir string) ([]formats.Package, error) {
	verbose.Printf("Lock resolution: applying installed versions for %d packages\n", len(packages))
	if len(packages) == 0 || cfg == nil {
		verbose.Debugf("Lock resolution: skipping - packages=%d, cfg=%v", len(packages), cfg != nil)
		return packages, nil
	}

	warningDedup := make(map[string]struct{})

	ruleIndexes := make(map[string][]int)
	for idx := range packages {
		ruleIndexes[packages[idx].Rule] = append(ruleIndexes[packages[idx].Rule], idx)
	}

	type scopeKey struct {
		rule string
		dir  string
	}

	scopes := make(map[scopeKey][]int)

	for ruleKey, indexes := range ruleIndexes {
		ruleCfg, ok := cfg.Rules[ruleKey]
		if !ok {
			continue
		}

		if len(ruleCfg.LockFiles) == 0 {
			// Check if this rule uses self-pinning (manifest is its own lock)
			if ruleCfg.SelfPinning {
				verbose.Debugf("Lock resolution: rule %q uses self-pinning (manifest is its own lock)", ruleKey)
				for _, idx := range indexes {
					version := strings.TrimSpace(packages[idx].Version)
					if version == "" || version == "*" {
						// Wildcard versions can't be self-pinned
						verbose.Tracef("Lock resolution: %q has wildcard version, cannot self-pin", packages[idx].Name)
						packages[idx].InstalledVersion = "#N/A"
						packages[idx].InstallStatus = InstallStatusVersionMissing
					} else {
						verbose.Tracef("Lock resolution: %q self-pinned to %q", packages[idx].Name, version)
						packages[idx].InstalledVersion = version
						packages[idx].InstallStatus = InstallStatusSelfPinned
					}
				}
				continue
			}

			verbose.Debugf("Lock resolution: rule %q has no lock files configured", ruleKey)
			for _, idx := range indexes {
				packages[idx].InstalledVersion = "#N/A"
				packages[idx].InstallStatus = InstallStatusNotConfigured

				issueLatestWarning(packages[idx], ruleCfg, warningDedup)
			}
			continue
		}

		for _, idx := range indexes {
			scopeDir := baseDir
			if packages[idx].Source != "" {
				scopeDir = filepath.Dir(packages[idx].Source)
			}

			if scopeDir == "" {
				scopeDir = cfg.WorkingDir
			}
			if scopeDir == "" {
				scopeDir = "."
			}

			scopes[scopeKey{rule: ruleKey, dir: scopeDir}] = append(scopes[scopeKey{rule: ruleKey, dir: scopeDir}], idx)
		}
	}

	for key, indexes := range scopes {
		ruleCfg, ok := cfg.Rules[key.rule]
		if !ok {
			continue
		}

		verbose.Debugf("Lock resolution: resolving versions for rule %q in scope %q (%d packages)", key.rule, key.dir, len(indexes))
		installed, foundLock, err := resolveInstalledVersions(key.dir, ruleCfg.LockFiles)
		if err != nil {
			verbose.Printf("Lock resolution ERROR: failed to resolve lock files for %s: %v\n", key.rule, err)
			return nil, fmt.Errorf("failed to resolve lock files for %s: %w", key.rule, err)
		}

		if !foundLock {
			verbose.Debugf("Lock resolution: no lock files found for rule %q in %q", key.rule, key.dir)
			for _, idx := range indexes {
				packages[idx].InstalledVersion = "#N/A"
				packages[idx].InstallStatus = InstallStatusLockMissing

				issueLatestWarning(packages[idx], ruleCfg, warningDedup)
			}
			continue
		}

		verbose.Debugf("Lock resolution: found %d installed versions from lock files", len(installed))
		for _, idx := range indexes {
			name := packages[idx].Name
			if version, ok := installed[name]; ok && version != "" {
				verbose.Tracef("Lock resolution: %q installed version is %q", name, version)
				packages[idx].InstalledVersion = version
				packages[idx].InstallStatus = InstallStatusLockFound
				continue
			}

			verbose.Tracef("Lock resolution: %q not found in lock file", name)
			packages[idx].InstalledVersion = "#N/A"
			packages[idx].InstallStatus = InstallStatusNotInLock

			issueLatestWarning(packages[idx], ruleCfg, warningDedup)
		}
	}

	for idx := range packages {
		if packages[idx].Version == "*" && packages[idx].InstalledVersion == "#N/A" {
			packages[idx].InstallStatus = InstallStatusVersionMissing
			verbose.Printf("VersionMissing: %s has wildcard %q with no installed version", packages[idx].Name, packages[idx].Version)
		}
	}

	// Mark packages with floating constraints (5.*, >=8.0.0, [8.0.0,9.0.0), etc.)
	// These cannot be updated automatically and require manual handling.
	for idx := range packages {
		if utils.IsFloatingConstraint(packages[idx].Version) {
			packages[idx].InstallStatus = InstallStatusFloating
			verbose.Printf("Floating: %s has constraint %q - manual update required", packages[idx].Name, packages[idx].Version)
		}
	}

	// Mark packages with IgnoreReason as Ignored
	// This takes precedence over other statuses as ignored packages should not be updated
	for idx := range packages {
		if packages[idx].IgnoreReason != "" {
			packages[idx].InstallStatus = InstallStatusIgnored
			verbose.Printf("Ignored: %s - %s", packages[idx].Name, packages[idx].IgnoreReason)
		}
	}

	return packages, nil
}

// issueLatestWarning checks if a package uses a latest indicator without a lock file
// and tracks warning deduplication.
//
// It performs the following operations:
//   - Skips packages that were found in lock files
//   - Checks if the package version is a latest indicator (e.g., "latest", "current")
//   - Deduplicates warnings using the seen map to avoid repeated warnings
//
// Parameters:
//   - pkg: Package to check for latest indicator usage
//   - ruleCfg: Package manager configuration containing latest indicator patterns
//   - seen: Map tracking which package warnings have already been issued
//
// Returns: This function does not return any values.
func issueLatestWarning(pkg formats.Package, ruleCfg config.PackageManagerCfg, seen map[string]struct{}) {
	if pkg.InstallStatus == InstallStatusLockFound {
		return
	}

	if !utils.IsLatestIndicator(pkg.Version, pkg.Name, &ruleCfg) {
		return
	}

	key := fmt.Sprintf("%s:%s", pkg.Rule, pkg.Name)
	if _, exists := seen[key]; exists {
		return
	}
	seen[key] = struct{}{}

	// Rely on the Unsupported status instead of emitting noisy warnings when the
	// installed version is unknown for packages declaring a latest indicator.
}

// resolveInstalledVersions extracts package versions from configured lock files.
//
// It performs the following operations:
//   - Searches for lock files matching configured patterns
//   - Extracts package name-version mappings from each found lock file
//   - Aggregates results across multiple lock files
//   - Tracks whether any lock files were found
//
// Parameters:
//   - baseDir: Base directory to search for lock files
//   - lockCfgs: Slice of lock file configurations specifying files and extraction methods
//
// Returns:
//   - map[string]string: Map of package names to installed versions
//   - bool: True if any lock files were found, false otherwise
//   - error: When file search or version extraction fails, returns error; otherwise returns nil
func resolveInstalledVersions(baseDir string, lockCfgs []config.LockFileCfg) (map[string]string, bool, error) {
	verbose.Debugf("Lock resolution: searching for lock files in %q (%d lock configs)", baseDir, len(lockCfgs))
	installed := make(map[string]string)
	foundAny := false

	for i, lockCfg := range lockCfgs {
		if len(lockCfg.Files) == 0 {
			verbose.Tracef("Lock resolution: config %d has no file patterns, skipping", i)
			continue
		}

		verbose.Tracef("Lock resolution: searching for patterns %v", lockCfg.Files)
		files, err := findFilesByPatterns(baseDir, lockCfg.Files)
		if err != nil {
			verbose.Printf("Lock resolution ERROR: failed to find lock files: %v\n", err)
			return nil, false, fmt.Errorf("failed to find lock files in %s: %w", baseDir, err)
		}

		if len(files) == 0 {
			verbose.Tracef("Lock resolution: no files matched patterns %v", lockCfg.Files)
			continue
		}

		verbose.Debugf("Lock resolution: found %d lock files: %v", len(files), files)
		foundAny = true
		for _, file := range files {
			matches, err := extractVersionsFromFn(file, &lockCfg)
			if err != nil {
				verbose.Printf("Lock resolution ERROR: failed to extract versions from %s: %v\n", file, err)
				return nil, false, fmt.Errorf("failed to extract versions from %s: %w", file, err)
			}

			verbose.Debugf("Lock extraction: %s → %d packages", filepath.Base(file), len(matches))

			// Note: If len(matches) == 0 with a configured extraction pattern,
			// this could indicate a misconfigured pattern. Consumers can check
			// the returned map size if they need to handle this case.

			for name, version := range matches {
				if version == "" {
					continue
				}
				installed[name] = version
			}
		}
	}

	verbose.Debugf("Lock resolution: total %d installed versions resolved", len(installed))
	return installed, foundAny, nil
}

// extractVersionsFromLock extracts package versions from a single lock file.
//
// It performs the following operations:
//   - Checks if custom commands are configured and delegates to extractVersionsFromCommand
//   - Reads the lock file content from disk
//   - Applies configured extraction pattern to parse package names and versions
//   - Normalizes package names (removes prefixes like "node_modules/")
//   - Filters out entries with empty names or versions
//
// Parameters:
//   - path: Absolute or relative path to the lock file
//   - cfg: Lock file configuration containing extraction patterns or commands
//
// Returns:
//   - map[string]string: Map of package names to versions extracted from the lock file
//   - error: When cfg is nil, file read fails, or pattern parsing fails, returns error; otherwise returns nil
func extractVersionsFromLock(path string, cfg *config.LockFileCfg) (map[string]string, error) {
	verbose.Tracef("Lock extraction: processing %s", path)

	// Handle nil config
	if cfg == nil {
		verbose.Printf("Lock extraction ERROR: config is nil for %s\n", path)
		return nil, fmt.Errorf("lock file extraction config missing for %s", path)
	}

	// Check if custom commands are configured for this lock file
	if cfg.Commands != "" {
		verbose.Tracef("Lock extraction: using custom commands for %s", path)
		return extractVersionsFromCommand(path, cfg)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		verbose.Printf("Lock extraction ERROR: failed to read %s: %v\n", path, err)
		return nil, err
	}

	// Validate extraction configuration - must have either Pattern or Patterns
	if cfg.Extraction == nil || (cfg.Extraction.Pattern == "" && len(cfg.Extraction.Patterns) == 0) {
		verbose.Printf("Lock extraction ERROR: no extraction pattern configured for %s\n", path)
		return nil, fmt.Errorf("lock file extraction pattern missing for %s", path)
	}

	// Use multi-pattern extraction for maximum flexibility across lock file versions
	matches, err := utils.ExtractWithPatterns(string(content), cfg.Extraction)
	if err != nil {
		verbose.Printf("Lock extraction ERROR: pattern matching failed for %s: %v\n", filepath.Base(path), err)
		return nil, fmt.Errorf("failed to parse lock file %s: %w", filepath.Base(path), err)
	}
	verbose.Tracef("Lock extraction: pattern matched %d entries from %s", len(matches), filepath.Base(path))

	results := make(map[string]string)
	for _, match := range matches {
		name := normalizeLockPackageName(match["name"], match["n"])
		version := strings.TrimSpace(match["version"])

		version = strings.TrimSuffix(version, "/go.mod")

		if name == "" || version == "" {
			continue
		}

		results[name] = version
	}

	return results, nil
}

// extractVersionsFromCommand executes custom commands to extract installed versions.
//
// This is useful for complex lock files or when maximum compatibility is needed
// across different lock file versions.
//
// # Output Formats
//
// The command output can be in JSON format (default) or raw text format.
// See [parseLockCommandJSON] for detailed JSON format support including:
//   - Simple object: {"package-name": "version", ...}
//   - Array format: [{"name": "package-name", "version": "1.0.0"}, ...]
//   - npm ls format: {"dependencies": {"pkg": {"version": "ver"}}}
//   - pnpm ls format: [{"dependencies": {"pkg": {"version": "ver"}}}]
//   - yarn list format: {"data": {"trees": [{"name": "pkg@version"}]}}
//   - package-lock.json v3: {"packages": {"node_modules/pkg": {"version": "ver"}}}
//
// For raw format, use CommandExtraction with Format="raw" and a regex Pattern.
//
// # Template Variables
//
// Commands support these template variables:
//   - {{lock_file}}: Full path to the lock file
//   - {{base_dir}}: Directory containing the lock file
//
// # Error Handling
//
// If a command exits with non-zero status but produces valid output (common with
// npm ls when packages are missing), the output is still parsed successfully.
//
// Parameters:
//   - path: Absolute or relative path to the lock file
//   - cfg: Lock file configuration containing commands and extraction settings
//
// Returns:
//   - map[string]string: Map of package names to versions from command output
//   - error: When command fails and output cannot be parsed, returns error; otherwise returns nil
func extractVersionsFromCommand(path string, cfg *config.LockFileCfg) (map[string]string, error) {
	baseDir := filepath.Dir(path)

	// Build replacements for command templates
	replacements := map[string]string{
		"lock_file": path,
		"base_dir":  baseDir,
	}

	verbose.Debugf("Executing lock file command in %s: %s", baseDir, cfg.Commands)

	// Execute the command
	output, cmdErr := cmdexec.Execute(cfg.Commands, cfg.Env, baseDir, cfg.GetTimeoutSeconds(), replacements)

	// Try to parse output even if command failed (e.g., npm ls returns exit 1 when
	// packages are missing but still outputs valid JSON with version info).
	if len(output) > 0 {
		results, parseErr := parseLockCommandOutput(output, cfg.CommandExtraction)
		if parseErr == nil && len(results) > 0 {
			// Successfully parsed versions from output
			if cmdErr != nil {
				verbose.Debugf("Lock command exited with error but output contained %d packages", len(results))
			}
			return results, nil
		}
	}

	// If we couldn't parse output and command failed, return the command error
	if cmdErr != nil {
		return nil, fmt.Errorf("lock file command failed: %w", cmdErr)
	}

	// Parse the output based on configuration
	return parseLockCommandOutput(output, cfg.CommandExtraction)
}

// parseLockCommandOutput parses the output of a lock file command based on configured format.
//
// It performs the following operations:
//   - Determines output format (json or raw) from extraction config
//   - Delegates to format-specific parser (JSON or raw regex)
//   - Returns parsed package name-version mappings
//
// Parameters:
//   - output: Raw output bytes from the executed lock file command
//   - extraction: Optional extraction configuration specifying format and parsing rules
//
// Returns:
//   - map[string]string: Map of package names to versions parsed from command output
//   - error: When format is unsupported or parsing fails, returns error; otherwise returns nil
func parseLockCommandOutput(output []byte, extraction *config.LockCommandExtractionCfg) (map[string]string, error) {
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return make(map[string]string), nil
	}

	// Determine format
	format := "json"
	if extraction != nil && extraction.Format != "" {
		format = extraction.Format
	}

	switch format {
	case "json":
		return parseLockCommandJSON(output, extraction)
	case "raw":
		if extraction == nil || extraction.Pattern == "" {
			return nil, fmt.Errorf("raw format requires command_extraction.pattern")
		}
		return parseLockCommandRaw(outputStr, extraction)
	default:
		return nil, fmt.Errorf("unsupported lock command output format: %s", format)
	}
}

// parseLockCommandJSON parses JSON output from a lock file command supporting multiple formats.
//
// # Supported JSON Formats
//
// The function attempts to parse JSON in the following order, stopping at the first successful match:
//
// 1. Simple Object Format (generic):
//
//	{"package-name": "version", "other-package": "1.0.0"}
//
//	This format is skipped if any value is a nested object/array, as that indicates
//	a more complex format (like npm/yarn/pnpm output) that should be handled below.
//
// 2. Array Format (generic, pnpm ls):
//
//	[{"name": "package-name", "version": "1.0.0"}, ...]
//
//	Custom JSON keys can be configured via extraction.JSONNameKey and extraction.JSONVersionKey.
//	Also handles pnpm ls --json output which wraps dependencies in array elements:
//	[{"name": "project", "dependencies": {"pkg": {"version": "ver"}}, "devDependencies": {...}}]
//
// 3. Nested Object Format (npm ls, package-lock.json v3, yarn list):
//
//	npm ls --json:           {"dependencies": {"pkg": {"version": "ver", "dependencies": {...}}}}
//	package-lock.json v3:    {"packages": {"node_modules/pkg": {"version": "ver"}}}
//	yarn list --json:        {"type": "tree", "data": {"trees": [{"name": "pkg@version"}]}}
//
// # Package Manager Compatibility
//
// These formats are designed to be generic and reusable across package managers.
// The specific commands that produce compatible output include:
//   - npm: npm ls --json --all
//   - pnpm: pnpm ls --json --depth=Infinity
//   - yarn v1: yarn list --json --depth=0
//   - Direct parsing of package-lock.json v3
//
// Parameters:
//   - output: Raw JSON output bytes from the lock file command
//   - extraction: Optional extraction config with custom JSON key names
//
// Returns:
//   - map[string]string: Map of package names to versions extracted from JSON
//   - error: When all parsing attempts fail, returns error; otherwise returns nil
func parseLockCommandJSON(output []byte, extraction *config.LockCommandExtractionCfg) (map[string]string, error) {
	results := make(map[string]string)

	// Format detection order rationale:
	// 1. Simple object - most specific constraint (all values must be strings)
	// 2. Array - supports both generic [{name, version}] and pnpm's nested format
	// 3. Nested object - catches npm ls, package-lock.json v3, and yarn list
	//
	// This order ensures that simpler custom tool outputs are matched first,
	// while complex package manager outputs fall through to appropriate parsers.

	// Try object format first: {"package-name": "version", ...}
	// Skip this format if any value is a nested object/array (indicates complex format like yarn/npm)
	var objFormat map[string]interface{}
	if err := json.Unmarshal(output, &objFormat); err == nil {
		hasNestedStructures := false
		for _, val := range objFormat {
			switch val.(type) {
			case map[string]interface{}, []interface{}:
				hasNestedStructures = true
			}
			if hasNestedStructures {
				break
			}
		}
		if !hasNestedStructures && len(objFormat) > 0 {
			for name, val := range objFormat {
				if version, ok := val.(string); ok && version != "" {
					results[name] = version
				}
			}
			if len(results) > 0 {
				return results, nil
			}
		}
	}

	// Try array format: [{"name": "package-name", "version": "1.0.0"}, ...]
	// Also handles pnpm ls format: [{dependencies: {pkg: {version: "ver"}}, devDependencies: {...}}]
	var arrFormat []map[string]interface{}
	if err := json.Unmarshal(output, &arrFormat); err == nil {
		nameKey := "name"
		versionKey := "version"
		if extraction != nil {
			if extraction.JSONNameKey != "" {
				nameKey = extraction.JSONNameKey
			}
			if extraction.JSONVersionKey != "" {
				versionKey = extraction.JSONVersionKey
			}
		}

		for _, item := range arrFormat {
			name, nameOK := item[nameKey].(string)
			version, verOK := item[versionKey].(string)
			if nameOK && verOK && name != "" && version != "" {
				results[name] = version
			}

			// Handle pnpm ls format: [{dependencies: {pkg: {version: "ver"}}, devDependencies: {...}}]
			if deps, ok := item["dependencies"].(map[string]interface{}); ok {
				extractNestedDependencies(deps, results)
			}
			if devDeps, ok := item["devDependencies"].(map[string]interface{}); ok {
				extractNestedDependencies(devDeps, results)
			}
		}
		// Return results (even if empty) since valid array format was parsed
		return results, nil
	}

	// Try nested format (npm ls style): {"dependencies": {"pkg": {"version": "ver"}}}
	var nestedFormat map[string]interface{}
	if err := json.Unmarshal(output, &nestedFormat); err == nil {
		// Check for "dependencies" key (npm ls format)
		if deps, ok := nestedFormat["dependencies"].(map[string]interface{}); ok {
			extractNestedDependencies(deps, results)
		}
		// Check for "packages" key (package-lock.json v3 format)
		if pkgs, ok := nestedFormat["packages"].(map[string]interface{}); ok {
			for name, val := range pkgs {
				if pkgInfo, ok := val.(map[string]interface{}); ok {
					if version, ok := pkgInfo["version"].(string); ok && version != "" {
						// Normalize package name (remove node_modules/ prefix)
						cleanName := strings.TrimPrefix(name, "node_modules/")
						if cleanName != "" && cleanName != name || !strings.HasPrefix(name, "node_modules/") {
							if cleanName == "" {
								cleanName = name
							}
							results[cleanName] = version
						}
					}
				}
			}
		}
		// Check for yarn list format: {"type":"tree","data":{"trees":[{"name":"pkg@version"}]}}
		if data, ok := nestedFormat["data"].(map[string]interface{}); ok {
			if trees, ok := data["trees"].([]interface{}); ok {
				for _, tree := range trees {
					if treeObj, ok := tree.(map[string]interface{}); ok {
						if nameAtVersion, ok := treeObj["name"].(string); ok && nameAtVersion != "" {
							// Parse "package@version" or "@scope/package@version" format
							name, version := parseYarnNameVersion(nameAtVersion)
							if name != "" && version != "" {
								results[name] = version
							}
						}
					}
				}
			}
		}
		if len(results) > 0 {
			return results, nil
		}
	}

	return nil, fmt.Errorf("failed to parse lock command JSON output: unrecognized format")
}

// extractNestedDependencies recursively extracts dependencies from npm ls format.
//
// It performs the following operations:
//   - Iterates through dependency entries
//   - Extracts version from each package info object
//   - Recursively processes nested dependencies
//   - Accumulates results in the provided map
//
// Parameters:
//   - deps: Map of package names to package info objects from npm ls output
//   - results: Map to accumulate extracted package name-version pairs
//
// Returns: This function does not return any values; it modifies results in place.
func extractNestedDependencies(deps map[string]interface{}, results map[string]string) {
	for name, val := range deps {
		if pkgInfo, ok := val.(map[string]interface{}); ok {
			if version, ok := pkgInfo["version"].(string); ok && version != "" {
				results[name] = version
			}
			// Recursively handle nested dependencies
			if nestedDeps, ok := pkgInfo["dependencies"].(map[string]interface{}); ok {
				extractNestedDependencies(nestedDeps, results)
			}
		}
	}
}

// parseLockCommandRaw parses raw text output from a lock file command using regex patterns.
//
// It performs the following operations:
//   - Applies configured regex pattern to extract package names and versions
//   - Normalizes package names (removes prefixes and suffixes)
//   - Filters out entries with empty names or versions
//   - Returns extracted package name-version mappings
//
// Parameters:
//   - output: Raw text output from the lock file command
//   - extraction: Extraction configuration containing the regex pattern with named groups
//
// Returns:
//   - map[string]string: Map of package names to versions extracted from raw output
//   - error: When pattern matching fails, returns error; otherwise returns nil
func parseLockCommandRaw(output string, extraction *config.LockCommandExtractionCfg) (map[string]string, error) {
	matches, err := utils.ExtractAllMatches(extraction.Pattern, output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lock command output: %w", err)
	}

	results := make(map[string]string)
	for _, match := range matches {
		name := normalizeLockPackageName(match["name"], match["n"])
		version := strings.TrimSpace(match["version"])

		if name == "" || version == "" {
			continue
		}

		results[name] = version
	}

	return results, nil
}

// normalizeLockPackageName normalizes package names extracted from lock files.
//
// It performs the following operations:
//   - Uses primary name if available, falls back to alternative name
//   - Removes "node_modules/" prefix for npm packages
//   - Removes "/go.mod" suffix for Go modules
//   - Returns empty string if no valid name is found
//
// Parameters:
//   - name: Primary package name from regex named group "name"
//   - alt: Alternative package name from regex named group "n"
//
// Returns:
//   - string: Normalized package name, or empty string if both inputs are empty
func normalizeLockPackageName(name, alt string) string {
	resolved := strings.TrimSpace(name)
	if resolved == "" {
		resolved = strings.TrimSpace(alt)
	}

	if resolved == "" {
		return ""
	}

	resolved = strings.TrimPrefix(resolved, "node_modules/")
	resolved = strings.TrimSuffix(resolved, "/go.mod")

	return resolved
}

// parseYarnNameVersion parses yarn's "name@version" format into separate name and version.
//
// It handles both regular packages and scoped packages:
//   - "lodash@4.17.21" -> ("lodash", "4.17.21")
//   - "@babel/core@7.26.0" -> ("@babel/core", "7.26.0")
//   - "@vue/reactivity@3.5.13" -> ("@vue/reactivity", "3.5.13")
//
// Parameters:
//   - nameAtVersion: Combined name@version string from yarn list output
//
// Returns:
//   - string: Package name
//   - string: Package version
func parseYarnNameVersion(nameAtVersion string) (string, string) {
	if nameAtVersion == "" {
		return "", ""
	}

	// Handle scoped packages: @scope/package@version
	// Find the last @ that's not at position 0 (which would be a scope)
	lastAt := strings.LastIndex(nameAtVersion, "@")
	if lastAt <= 0 {
		// No @ found or @ is at position 0 (just a scope, no version)
		return nameAtVersion, ""
	}

	// Check if this @ is part of a scoped package name
	// If there's a / before the last @, and the string starts with @, we need to be careful
	if strings.HasPrefix(nameAtVersion, "@") {
		// Scoped package: @scope/package@version
		// Find the @ after the scope (after the /)
		slashIdx := strings.Index(nameAtVersion, "/")
		if slashIdx > 0 && lastAt > slashIdx {
			// The last @ is after the /, so it's the version separator
			return nameAtVersion[:lastAt], nameAtVersion[lastAt+1:]
		}
		// The @ is part of the scope, no version found
		return nameAtVersion, ""
	}

	// Regular package: package@version
	return nameAtVersion[:lastAt], nameAtVersion[lastAt+1:]
}
