package packages

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/utils"
	"github.com/ajxudir/goupdate/pkg/verbose"
	"github.com/ajxudir/goupdate/pkg/warnings"
)

// DetectFiles discovers all manifest files matching configured include/exclude patterns.
//
// It performs the following operations:
//   - Validates the configuration
//   - Iterates through each enabled package manager rule
//   - Walks the directory tree to find matching files
//   - Resolves conflicts when multiple rules match the same file
//
// Parameters:
//   - cfg: Configuration containing package manager rules with include/exclude patterns
//   - baseDir: Base directory to search from; uses cfg.WorkingDir if empty, or "." if both empty
//
// Returns:
//   - map[string][]string: Map of rule names to lists of absolute file paths detected for each rule
//   - error: When cfg is nil, returns error; when no rules are configured, returns error;
//     when directory access fails, returns error; otherwise returns nil
func DetectFiles(cfg *config.Config, baseDir string) (map[string][]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	if len(cfg.Rules) == 0 {
		return nil, fmt.Errorf("no package manager rules configured")
	}

	detected := make(map[string][]string)

	if baseDir == "" {
		baseDir = cfg.WorkingDir
	}
	if baseDir == "" {
		baseDir = "."
	}

	verbose.Printf("Starting file detection in base directory: %s\n", baseDir)
	verbose.Debugf("Processing %d configured rules", len(cfg.Rules))

	for ruleKey, rule := range cfg.Rules {
		// Skip disabled rules
		if !rule.IsEnabled() {
			verbose.Tracef("Rule %q: skipped (disabled)", ruleKey)
			continue
		}

		if len(rule.Include) == 0 {
			warnings.Warnf("⚠️ rule %s has no include patterns; skipping detection\n", ruleKey)
			continue
		}

		verbose.Tracef("Rule %q: scanning with include=%v, exclude=%v", ruleKey, rule.Include, rule.Exclude)

		files, err := detectForRule(baseDir, rule)
		if err != nil {
			return nil, err
		}

		if len(files) > 0 {
			detected[ruleKey] = files
			verbose.Debugf("Rule %q: found %d matching files", ruleKey, len(files))
			if verbose.IsEnabled() {
				for _, f := range files {
					verbose.Printf("  - %s", f)
				}
			}
		} else {
			verbose.Printf("Rule %q: no matching files found", ruleKey)
		}
	}

	return resolveRuleConflicts(cfg, detected), nil
}

// detectForRule finds all files matching a single rule's include/exclude patterns.
//
// It performs the following operations:
//   - Validates the base directory exists and is accessible
//   - Walks the directory tree using filepath.Walk
//   - Skips directories, broken symlinks, and inaccessible paths
//   - Applies include/exclude pattern matching to regular files
//   - Returns absolute paths of all matching files
//
// Parameters:
//   - baseDir: Base directory to search from; must be an existing directory
//   - rule: Package manager configuration with Include and Exclude patterns
//
// Returns:
//   - []string: List of absolute file paths matching the rule's patterns
//   - error: When baseDir doesn't exist, returns error; when baseDir is not a directory,
//     returns error; when walk encounters unrecoverable errors, returns error; otherwise returns nil
func detectForRule(baseDir string, rule config.PackageManagerCfg) ([]string, error) {
	baseInfo, err := os.Stat(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to access base directory: %w", err)
	}
	if !baseInfo.IsDir() {
		return nil, fmt.Errorf("base path is not a directory: %s", baseDir)
	}

	var matches []string

	err = filepath.Walk(baseDir, func(path string, info os.FileInfo, walkErr error) error {
		// Handle walk errors (e.g., permission denied, broken symlinks)
		if walkErr != nil {
			// For broken symlinks or inaccessible files, skip with warning
			if os.IsNotExist(walkErr) || os.IsPermission(walkErr) {
				warnings.Warnf("⚠️ skipping inaccessible path %s: %v\n", path, walkErr)
				return nil
			}
			// For other errors, continue walking
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check for symlinks using Lstat (filepath.Walk uses Stat which follows symlinks)
		linfo, lstatErr := os.Lstat(path)
		if lstatErr == nil && linfo.Mode()&os.ModeSymlink != 0 {
			// This is a symlink - verify the target exists and is a regular file
			realPath, evalErr := filepath.EvalSymlinks(path)
			if evalErr != nil {
				// Broken symlink - skip with warning
				warnings.Warnf("⚠️ skipping broken symlink %s: %v\n", path, evalErr)
				return nil
			}
			// Check if symlink target is a directory (shouldn't match file patterns)
			realInfo, statErr := os.Stat(realPath)
			if statErr != nil {
				warnings.Warnf("⚠️ skipping symlink with inaccessible target %s: %v\n", path, statErr)
				return nil
			}
			if realInfo.IsDir() {
				return nil
			}
		}

		relPath := path
		if rel, relErr := filepath.Rel(baseDir, path); relErr == nil {
			relPath = rel
		}

		if utils.MatchPatterns(relPath, rule.Include, rule.Exclude) {
			matches = append(matches, path)
		}

		return nil
	})

	return matches, err
}

// resolveRuleConflicts handles files matched by multiple rules by selecting one rule per file.
//
// It performs the following operations:
//   - Builds a reverse map from files to the rules that matched them
//   - For each file matched by multiple rules, selects the most appropriate rule
//   - Removes the file from all non-selected rules
//   - Cleans up rules that have no files remaining
//
// Parameters:
//   - cfg: Configuration containing all package manager rules
//   - detected: Map of rule names to matched file lists, potentially with overlaps
//
// Returns:
//   - map[string][]string: Updated map where each file appears under exactly one rule
func resolveRuleConflicts(cfg *config.Config, detected map[string][]string) map[string][]string {
	fileToRules := make(map[string][]string)
	for rule, files := range detected {
		for _, file := range files {
			fileToRules[file] = append(fileToRules[file], rule)
		}
	}

	conflictCount := 0
	for file, rules := range fileToRules {
		if len(rules) < 2 {
			continue
		}
		conflictCount++

		selected := selectRuleForFile(cfg, file, rules)
		verbose.Printf("Conflict: %s matched %v → selected %s\n", filepath.Base(file), rules, selected)

		for _, rule := range rules {
			if rule == selected {
				continue
			}

			detected[rule] = removeFile(detected[rule], file)
			if len(detected[rule]) == 0 {
				delete(detected, rule)
			}
		}
	}

	if conflictCount > 0 {
		verbose.Printf("Resolved %d file conflicts\n", conflictCount)
	}

	return detected
}

// selectRuleForFile chooses which rule should handle a file when multiple rules match.
//
// It performs the following operations:
//   - Prioritizes rules by known package manager order (npm, pnpm, yarn, then alphabetical)
//   - Checks each prioritized rule for the presence of its lock files
//   - Returns the first rule with a lock file present, or the highest priority rule if none found
//
// Parameters:
//   - cfg: Configuration containing rule definitions with lock file patterns
//   - file: Absolute path to the manifest file that multiple rules matched
//   - rules: List of rule names that all matched the file
//
// Returns:
//   - string: Name of the rule that should handle this file
func selectRuleForFile(cfg *config.Config, file string, rules []string) string {
	dir := filepath.Dir(file)

	prioritized := prioritizeRules(rules)
	verbose.Debugf("Prioritized rule order for %q: %v", file, prioritized)

	for _, ruleName := range prioritized {
		rule, ok := cfg.Rules[ruleName]
		if !ok {
			continue
		}
		if hasLockFile(dir, rule.LockFiles) {
			verbose.Debugf("Rule %q selected: lock file found in %s", ruleName, dir)
			return ruleName
		}
	}

	verbose.Debugf("Rule %q selected: no lock files found, using highest priority", prioritized[0])
	return prioritized[0]
}

// ResolveRuleForFile determines which rule should apply to a file when multiple match.
//
// This is a public wrapper around selectRuleForFile for external use.
//
// Parameters:
//   - cfg: Configuration containing rule definitions with lock file patterns
//   - file: Absolute path to the manifest file that multiple rules matched
//   - rules: List of rule names that all matched the file
//
// Returns:
//   - string: Name of the rule that should handle this file based on lock files and priority
func ResolveRuleForFile(cfg *config.Config, file string, rules []string) string {
	return selectRuleForFile(cfg, file, rules)
}

// prioritizeRules orders rule names by known package manager priority and alphabetically.
//
// It performs the following operations:
//   - Creates a copy of the rules list to avoid modifying the input
//   - Applies stable sorting with npm (priority 0), pnpm (priority 1), yarn (priority 2)
//   - Unknown package managers are sorted alphabetically after known ones
//
// Parameters:
//   - rules: List of rule names to prioritize
//
// Returns:
//   - []string: New slice with rules ordered by priority (npm, pnpm, yarn, then alphabetical)
func prioritizeRules(rules []string) []string {
	ordered := make([]string, len(rules))
	copy(ordered, rules)

	priority := map[string]int{"npm": 0, "pnpm": 1, "yarn": 2}
	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i]
		right := ordered[j]
		leftPriority, leftOk := priority[left]
		rightPriority, rightOk := priority[right]
		switch {
		case leftOk && rightOk:
			return leftPriority < rightPriority
		case leftOk:
			return true
		case rightOk:
			return false
		default:
			return left < right
		}
	})

	return ordered
}

// hasLockFile checks if any configured lock files exist in the given directory.
//
// It performs the following operations:
//   - Iterates through all lock file configurations
//   - For each pattern, constructs the expected file path in the directory
//   - Checks if the file exists using os.Stat
//   - Returns true as soon as any lock file is found
//
// Parameters:
//   - dir: Directory path to check for lock files
//   - lockFiles: List of lock file configurations with file patterns
//
// Returns:
//   - bool: true if any lock file exists in the directory, false otherwise
func hasLockFile(dir string, lockFiles []config.LockFileCfg) bool {
	for _, lockFile := range lockFiles {
		for _, pattern := range lockFile.Files {
			candidate := filepath.Join(dir, filepath.Base(pattern))
			if _, err := os.Stat(candidate); err == nil {
				return true
			}
		}
	}

	return false
}

// removeFile filters out all occurrences of a target file from a list.
//
// It performs the following operations:
//   - Reuses the input slice's backing array for efficiency
//   - Iterates through all files, keeping only non-matching ones
//   - Returns a new slice view excluding all occurrences of target
//
// Parameters:
//   - files: List of file paths to filter
//   - target: File path to remove from the list
//
// Returns:
//   - []string: New slice containing all files except target
func removeFile(files []string, target string) []string {
	filtered := files[:0]
	for _, file := range files {
		if file != target {
			filtered = append(filtered, file)
		}
	}

	return filtered
}
