// Package lock provides functionality for resolving installed package versions
// from lock files. It supports various lock file formats including package-lock.json,
// pnpm-lock.yaml, yarn.lock, go.sum, custom extraction patterns, and custom commands.
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
	if len(packages) == 0 || cfg == nil {
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
				for _, idx := range indexes {
					version := strings.TrimSpace(packages[idx].Version)
					if version == "" || version == "*" {
						// Wildcard versions can't be self-pinned
						packages[idx].InstalledVersion = "#N/A"
						packages[idx].InstallStatus = InstallStatusVersionMissing
					} else {
						packages[idx].InstalledVersion = version
						packages[idx].InstallStatus = InstallStatusSelfPinned
					}
				}
				continue
			}

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

		installed, foundLock, err := resolveInstalledVersions(key.dir, ruleCfg.LockFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve lock files for %s: %w", key.rule, err)
		}

		if !foundLock {
			for _, idx := range indexes {
				packages[idx].InstalledVersion = "#N/A"
				packages[idx].InstallStatus = InstallStatusLockMissing

				issueLatestWarning(packages[idx], ruleCfg, warningDedup)
			}
			continue
		}

		for _, idx := range indexes {
			name := packages[idx].Name
			if version, ok := installed[name]; ok && version != "" {
				packages[idx].InstalledVersion = version
				packages[idx].InstallStatus = InstallStatusLockFound
				continue
			}

			packages[idx].InstalledVersion = "#N/A"
			packages[idx].InstallStatus = InstallStatusNotInLock

			issueLatestWarning(packages[idx], ruleCfg, warningDedup)
		}
	}

	for idx := range packages {
		if packages[idx].Version == "*" && packages[idx].InstalledVersion == "#N/A" {
			packages[idx].InstallStatus = InstallStatusVersionMissing
		}
	}

	// Mark packages with floating constraints (5.*, >=8.0.0, [8.0.0,9.0.0), etc.)
	// These cannot be updated automatically and require manual handling.
	for idx := range packages {
		if utils.IsFloatingConstraint(packages[idx].Version) {
			packages[idx].InstallStatus = InstallStatusFloating
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
	installed := make(map[string]string)
	foundAny := false

	for _, lockCfg := range lockCfgs {
		if len(lockCfg.Files) == 0 {
			continue
		}

		files, err := findFilesByPatterns(baseDir, lockCfg.Files)
		if err != nil {
			return nil, false, fmt.Errorf("failed to find lock files in %s: %w", baseDir, err)
		}

		if len(files) == 0 {
			continue
		}

		foundAny = true
		for _, file := range files {
			matches, err := extractVersionsFromFn(file, &lockCfg)
			if err != nil {
				return nil, false, fmt.Errorf("failed to extract versions from %s: %w", file, err)
			}

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
	// Handle nil config
	if cfg == nil {
		return nil, fmt.Errorf("lock file extraction config missing for %s", path)
	}

	// Check if custom commands are configured for this lock file
	if cfg.Commands != "" {
		return extractVersionsFromCommand(path, cfg)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if cfg == nil || cfg.Extraction == nil || cfg.Extraction.Pattern == "" {
		return nil, fmt.Errorf("lock file extraction pattern missing for %s", path)
	}

	matches, err := utils.ExtractAllMatches(cfg.Extraction.Pattern, string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse lock file %s: %w", filepath.Base(path), err)
	}

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
// across different lock file versions. The command output should be JSON in one
// of these formats:
//   - Object format: {"package-name": "version", ...}
//   - Array format: [{"name": "package-name", "version": "1.0.0"}, ...]
//
// Or raw format with regex extraction via CommandExtraction.
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

	verbose.Infof("Executing lock file command in %s: %s", baseDir, cfg.Commands)

	// Execute the command
	output, cmdErr := cmdexec.Execute(cfg.Commands, cfg.Env, baseDir, cfg.GetTimeoutSeconds(), replacements)

	verbose.Infof("Lock file command output length: %d bytes", len(output))

	// Try to parse output even if command failed (e.g., npm ls returns exit 1 when
	// packages are missing but still outputs valid JSON with version info).
	if len(output) > 0 {
		results, parseErr := parseLockCommandOutput(output, cfg.CommandExtraction)
		if parseErr == nil && len(results) > 0 {
			// Successfully parsed versions from output
			if cmdErr != nil {
				verbose.Infof("Lock command exited with error but output contained %d packages", len(results))
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
// It performs the following operations:
//   - Attempts to parse as object format: {"package-name": "version"}
//   - Falls back to array format: [{"name": "pkg", "version": "ver"}]
//   - Handles nested format (npm ls): {"dependencies": {"pkg": {"version": "ver"}}}
//   - Supports configurable JSON keys via extraction config
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

	// Try object format first: {"package-name": "version", ...}
	var objFormat map[string]interface{}
	if err := json.Unmarshal(output, &objFormat); err == nil {
		for name, val := range objFormat {
			if version, ok := val.(string); ok && version != "" {
				results[name] = version
			}
		}
		if len(results) > 0 {
			return results, nil
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
