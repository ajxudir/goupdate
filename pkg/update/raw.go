package update

import (
	"fmt"
	"strings"

	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/errors"
	"github.com/user/goupdate/pkg/formats"
	"github.com/user/goupdate/pkg/utils"
)

// extractAllMatchesWithIndexFunc is a variable that holds the utils.ExtractAllMatchesWithIndex function.
// This allows for dependency injection during testing.
var extractAllMatchesWithIndexFunc = utils.ExtractAllMatchesWithIndex

// updateRawVersion updates the version of a package in raw text content using regex extraction.
//
// It performs the following operations:
//   - Step 1: Validate extraction pattern is configured
//   - Step 2: Extract all matches with named groups from content
//   - Step 3: Find the match for the target package by name
//   - Step 4: Locate the version capture group within the match
//   - Step 5: Determine replacement version (with or without constraint prefix)
//   - Step 6: Replace version at the exact position in the content
//
// Parameters:
//   - content: The original raw file content as bytes
//   - p: The package to update, containing name and constraint information
//   - ruleCfg: Package manager configuration with regex extraction pattern
//   - target: The target version to update to (without constraint prefix)
//
// Returns:
//   - []byte: Updated raw content with version replaced
//   - error: Returns error if extraction pattern missing, package not found, version group missing, or invalid position; returns nil on success
func updateRawVersion(content []byte, p formats.Package, ruleCfg config.PackageManagerCfg, target string) ([]byte, error) {
	if ruleCfg.Extraction == nil || ruleCfg.Extraction.Pattern == "" {
		return nil, &errors.UnsupportedError{Reason: "missing extraction pattern"}
	}

	text := string(content)
	matches, err := extractAllMatchesWithIndexFunc(ruleCfg.Extraction.Pattern, text)
	if err != nil {
		return nil, err
	}

	// Find the match for the target package
	var targetMatch *utils.MatchWithIndex
	for i := range matches {
		match := &matches[i]
		name := match.Groups["name"]
		if name == "" {
			name = match.Groups["n"]
		}
		if name == "" || !strings.EqualFold(strings.TrimSpace(name), p.Name) {
			continue
		}
		targetMatch = match
		break
	}

	if targetMatch == nil {
		return nil, fmt.Errorf("package %s not found in raw content", p.Name)
	}

	// Check if we have a version group with index
	versionIdx, hasVersionIdx := targetMatch.GroupIndex["version"]
	if !hasVersionIdx || versionIdx[0] < 0 {
		// Try version_alt for some formats
		versionIdx, hasVersionIdx = targetMatch.GroupIndex["version_alt"]
	}

	if !hasVersionIdx || versionIdx[0] < 0 {
		return nil, fmt.Errorf("no version found for package %s", p.Name)
	}

	// Bounds check to prevent panic
	if versionIdx[0] > len(text) || versionIdx[1] > len(text) || versionIdx[0] > versionIdx[1] {
		return nil, fmt.Errorf("invalid version position for package %s", p.Name)
	}

	// Determine the replacement version
	// If the pattern has a separate constraint group, don't include constraint in version
	// If not, check if the captured version includes a constraint-like prefix
	_, hasConstraintGroup := targetMatch.GroupIndex["constraint"]

	var newVersion string
	if hasConstraintGroup {
		// Constraint is captured separately, just replace the version number
		newVersion = target
	} else {
		// Check if the captured version starts with constraint characters
		oldVersion := targetMatch.Groups["version"]
		if oldVersion == "" {
			oldVersion = targetMatch.Groups["version_alt"]
		}
		if len(oldVersion) > 0 && strings.ContainsAny(string(oldVersion[0]), "^~<>=!") {
			// Version includes constraint, include it in replacement
			newVersion = fmt.Sprintf("%s%s", p.Constraint, target)
		} else {
			// Just the version number (constraint is outside capture group)
			newVersion = target
		}
	}

	// Replace at the exact position
	result := text[:versionIdx[0]] + newVersion + text[versionIdx[1]:]

	return []byte(result), nil
}
