package utils

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/ajxudir/goupdate/pkg/verbose"
)

// regexCache stores compiled regex patterns to avoid recompilation.
// This improves performance when the same pattern is used multiple times.
var regexCache sync.Map

// getOrCompileRegex retrieves a compiled regex from cache or compiles and caches it.
//
// It performs the following operations:
//   - Step 1: Checks if pattern exists in cache with type-safe assertion
//   - Step 2: Returns cached regex if found and valid
//   - Step 3: Compiles new regex if not cached or cache entry is invalid
//   - Step 4: Stores compiled regex in cache for future use
//
// Parameters:
//   - pattern: The regex pattern string to compile
//
// Returns:
//   - *regexp.Regexp: The compiled regular expression
//   - error: Returns nil on success; returns compilation error if pattern is invalid
func getOrCompileRegex(pattern string) (*regexp.Regexp, error) {
	if cached, ok := regexCache.Load(pattern); ok {
		// Use safe type assertion to prevent panic on unexpected types
		if re, typeOK := cached.(*regexp.Regexp); typeOK {
			return re, nil
		}
		// Cache contained unexpected type - fall through to recompile
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	regexCache.Store(pattern, re)
	return re, nil
}

// ErrRegexTooComplex is returned when a regex pattern is potentially vulnerable to ReDoS.
var ErrRegexTooComplex = errors.New("regex pattern too complex: potential ReDoS vulnerability")

// DefaultMaxRegexPatternLength is the default maximum allowed regex pattern length to prevent DoS.
const DefaultMaxRegexPatternLength = 1000

// maxRegexQuantifiers is the maximum number of quantifiers (+, *) allowed in a single
// regex pattern. Excessive quantifiers can lead to exponential backtracking (ReDoS).
const maxRegexQuantifiers = 15

// RegexValidationOptions configures regex safety validation.
//
// This type allows fine-grained control over regex pattern safety checks
// to prevent ReDoS (Regular Expression Denial of Service) attacks.
//
// Fields:
//   - MaxLength: The maximum allowed pattern length; 0 uses DefaultMaxRegexPatternLength (1000)
//   - SkipComplexityCheck: If true, disables ReDoS pattern detection (use with caution)
type RegexValidationOptions struct {
	// MaxLength is the maximum allowed pattern length. 0 uses default (1000).
	MaxLength int
	// SkipComplexityCheck disables ReDoS pattern detection (dangerous).
	SkipComplexityCheck bool
}

// ValidateRegexSafety performs light ReDoS protection by checking for common vulnerable patterns.
//
// It validates regex patterns to prevent Regular Expression Denial of Service attacks
// by detecting nested quantifiers, excessive complexity, and other dangerous patterns.
// This is not comprehensive but catches the most common issues.
//
// Parameters:
//   - pattern: The regex pattern string to validate
//
// Returns:
//   - error: Returns nil if pattern is safe; returns ErrRegexTooComplex with helpful message if pattern is potentially dangerous
func ValidateRegexSafety(pattern string) error {
	return ValidateRegexSafetyWithOptions(pattern, RegexValidationOptions{})
}

// ValidateRegexSafetyWithOptions performs regex safety validation with configurable limits.
//
// It performs the following operations:
//   - Step 1: Validates pattern length against configured or default maximum
//   - Step 2: Checks for nested quantifiers (e.g., (a+)+, (.*)+)
//   - Step 3: Checks for simple nested patterns (e.g., (x*)+)
//   - Step 4: Checks for overlapping alternatives with quantifiers
//   - Step 5: Checks for excessive quantifier count
//
// Parameters:
//   - pattern: The regex pattern string to validate
//   - opts: Configuration options for validation (MaxLength, SkipComplexityCheck)
//
// Returns:
//   - error: Returns nil if pattern is safe; returns ErrRegexTooComplex with configuration suggestions if pattern is potentially dangerous
func ValidateRegexSafetyWithOptions(pattern string, opts RegexValidationOptions) error {
	maxLength := opts.MaxLength
	if maxLength <= 0 {
		maxLength = DefaultMaxRegexPatternLength
	}

	// Check pattern length
	if len(pattern) > maxLength {
		verbose.Printf("Regex validation ERROR: pattern length %d exceeds maximum %d\n", len(pattern), maxLength)
		return fmt.Errorf("%w: pattern length %d exceeds maximum %d\n\n"+
			"💡 To increase this limit, add to your root config:\n"+
			"   security:\n"+
			"     max_regex_complexity: %d  # or larger value",
			ErrRegexTooComplex, len(pattern), maxLength, len(pattern)+100)
	}

	// Skip complexity checks if configured (dangerous but sometimes needed)
	if opts.SkipComplexityCheck {
		return nil
	}

	// Check for nested quantifiers - common ReDoS patterns
	// Patterns like (a+)+, (a*)+, (.*)+, (.+)*, etc.
	// This specifically looks for quantifiers on groups that contain quantified wildcards
	// We need to be careful not to flag legitimate patterns like (?P<x>[a-z]+)?
	nestedQuantifiers := regexp.MustCompile(`\([^)]*(?:\.\*|\.\+|\\w\*|\\w\+|\\s\*|\\s\+)[^)]*\)[+*]`)
	if nestedQuantifiers.MatchString(pattern) {
		verbose.Printf("Regex validation ERROR: nested quantifiers detected (potential ReDoS)\n")
		return fmt.Errorf("%w: nested quantifiers detected - "+
			"to allow complex regex, add security.allow_complex_regex: true to your root config",
			ErrRegexTooComplex)
	}

	// Check for simple nested quantifier patterns like (a+)+, (x*)+
	simpleNested := regexp.MustCompile(`\([a-zA-Z][+*]\)[+*]`)
	if simpleNested.MatchString(pattern) {
		verbose.Printf("Regex validation ERROR: simple nested quantifiers detected (potential ReDoS)\n")
		return fmt.Errorf("%w: simple nested quantifiers detected - "+
			"to allow complex regex, add security.allow_complex_regex: true to your root config",
			ErrRegexTooComplex)
	}

	// Check for overlapping alternatives with quantifiers that share a common prefix
	// Patterns like (a|aa)+, (ab|abc)+ are dangerous
	overlappingAlts := regexp.MustCompile(`\(([^|)]+)\|([^)]+)\)[+*]`)
	if matches := overlappingAlts.FindStringSubmatch(pattern); len(matches) >= 3 {
		alt1, alt2 := matches[1], matches[2]
		// Check if one is a prefix of the other
		if strings.HasPrefix(alt1, alt2) || strings.HasPrefix(alt2, alt1) {
			verbose.Printf("Regex validation ERROR: overlapping alternatives with quantifiers detected\n")
			return fmt.Errorf("%w: overlapping alternatives with quantifiers detected - "+
				"to allow complex regex, add security.allow_complex_regex: true to your root config",
				ErrRegexTooComplex)
		}
	}

	// Check for excessive quantifiers
	quantifierCount := strings.Count(pattern, "+") + strings.Count(pattern, "*")
	if quantifierCount > maxRegexQuantifiers {
		verbose.Printf("Regex validation ERROR: excessive quantifiers (%d, max %d)\n", quantifierCount, maxRegexQuantifiers)
		return fmt.Errorf("%w: excessive quantifiers (%d, max %d) - "+
			"to allow complex regex, add security.allow_complex_regex: true to your root config",
			ErrRegexTooComplex, quantifierCount, maxRegexQuantifiers)
	}

	return nil
}

// TrimAndSplit splits a string by separator and trims whitespace from each part.
//
// It performs the following operations:
//   - Step 1: Returns empty slice if input is "" or "all"
//   - Step 2: Splits string by separator
//   - Step 3: Trims whitespace from each part
//   - Step 4: Filters out empty strings after trimming
//
// Parameters:
//   - s: The string to split and trim
//   - sep: The separator to split on
//
// Returns:
//   - []string: Slice of trimmed non-empty strings; empty slice if input is "" or "all"
func TrimAndSplit(s string, sep string) []string {
	if s == "" || s == "all" {
		return []string{}
	}

	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// Contains checks if a string slice contains an item.
//
// Performs case-sensitive exact match comparison.
//
// Parameters:
//   - slice: The slice of strings to search
//   - item: The string to search for
//
// Returns:
//   - bool: true if item is found in slice, false otherwise
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ContainsIgnoreCase checks if a string slice contains an item (case-insensitive).
//
// Performs case-insensitive comparison using strings.EqualFold.
//
// Parameters:
//   - slice: The slice of strings to search
//   - item: The string to search for (case-insensitive)
//
// Returns:
//   - bool: true if item is found in slice (case-insensitive), false otherwise
func ContainsIgnoreCase(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// GetConstraintDisplay returns a human-readable constraint display string.
//
// It normalizes and converts constraint operators into user-friendly display names.
//
// Parameters:
//   - constraint: The constraint operator (e.g., "^", "~", ">=", "<=", ">", "<", "=", "*", "")
//
// Returns:
//   - string: Human-readable display name (e.g., "Compatible (^)", "Patch (~)", "Major")
//   - bool: true if the constraint is valid, false if invalid
//   - bool: true if a warning was issued during normalization, false otherwise
func GetConstraintDisplay(constraint string) (string, bool, bool) {
	normalized, ok, warn := normalizeConstraintForDisplay(constraint)

	switch normalized {
	case "":
		return "Major", ok, warn
	case "^":
		return "Compatible (^)", ok, warn
	case "~":
		return "Patch (~)", ok, warn
	case ">=":
		return "Min (>=)", ok, warn
	case "<=":
		return "Max (<=)", ok, warn
	case ">":
		return "Above (>)", ok, warn
	case "<":
		return "Below (<)", ok, warn
	case "=":
		return "Exact (=)", ok, warn
	case "*":
		return "Major (*)", ok, warn
	}

	return "#N/A", false, warn
}

// normalizeConstraintForDisplay normalizes constraint syntax for display purposes.
//
// It handles common variations and invalid constraints, converting them to
// standard forms. Returns (normalized, isValid, hasWarning).
//
// Parameters:
//   - constraint: The raw constraint string to normalize
//
// Returns:
//   - string: The normalized constraint operator
//   - bool: true if the constraint is valid, false if unrecognized
//   - bool: true if normalization triggered a warning (e.g., "exact" -> "="), false otherwise
func normalizeConstraintForDisplay(constraint string) (string, bool, bool) {
	normalized := strings.TrimSpace(strings.ToLower(constraint))

	switch normalized {
	case "", "^", "~", ">=", "<=", ">", "<", "=", "*":
		return normalized, true, false
	case "==":
		return "=", true, false
	case "~=":
		return "~", true, false
	case "exact":
		return "=", true, true
	default:
		return normalized, false, true
	}
}

// MatchGlob matches a path against a glob pattern.
//
// It performs the following operations:
//   - Step 1: Checks for ! prefix to determine negation
//   - Step 2: Normalizes path and pattern to use forward slashes
//   - Step 3: Uses regex matching for ** patterns, filepath.Match for simple patterns
//   - Step 4: Negates result if ! prefix was present
//
// Supported patterns:
//   - * matches any sequence of characters within a path segment
//   - ** matches zero or more path segments recursively
//   - ? matches a single character
//   - ! prefix negates the match
//
// Parameters:
//   - path: The file path to match against
//   - pattern: The glob pattern (supports **, *, ?, and ! prefix)
//
// Returns:
//   - bool: true if path matches pattern (or doesn't match if negated), false otherwise
func MatchGlob(path, pattern string) bool {
	negate := false
	if strings.HasPrefix(pattern, "!") {
		negate = true
		pattern = pattern[1:]
	}

	path = filepath.ToSlash(path)
	pattern = filepath.ToSlash(pattern)

	var matched bool

	if strings.Contains(pattern, "**") {
		regexPattern := globToRegex(pattern)
		matched, _ = regexp.MatchString(regexPattern, path)
	} else {
		var err error
		matched, err = filepath.Match(pattern, path)
		if err != nil {
			regexPattern := globToRegex(pattern)
			matched, _ = regexp.MatchString(regexPattern, path)
		}
	}

	if negate {
		return !matched
	}
	return matched
}

// globToRegex converts a glob pattern to a regular expression pattern.
//
// It performs the following conversions:
//   - **/ becomes (?:.*/)?  (optional path segments)
//   - ** becomes .*         (any characters including /)
//   - * becomes [^/]*       (any characters except /)
//   - ? becomes .           (single character)
//   - Other characters are escaped with regexp.QuoteMeta
//
// Parameters:
//   - pattern: The glob pattern to convert
//
// Returns:
//   - string: The equivalent regular expression pattern
func globToRegex(pattern string) string {
	pattern = filepath.ToSlash(pattern)
	var builder strings.Builder
	builder.WriteString("^")

	for i := 0; i < len(pattern); {
		if strings.HasPrefix(pattern[i:], "**/") {
			builder.WriteString("(?:.*/)?")
			i += 3
			continue
		}
		if strings.HasPrefix(pattern[i:], "**") {
			builder.WriteString(".*")
			i += 2
			continue
		}
		switch pattern[i] {
		case '*':
			builder.WriteString("[^/]*")
		case '?':
			builder.WriteString(".")
		default:
			builder.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
		i++
	}

	builder.WriteString("$")
	return builder.String()
}

// MatchPatterns checks if a path matches any include pattern and no exclude pattern.
//
// It performs the following operations:
//   - Step 1: Checks all exclude patterns first - returns false if any match
//   - Step 2: Checks include patterns - returns true if any match
//   - Step 3: Returns false if no include patterns matched
//
// Parameters:
//   - path: The file path to match against patterns
//   - includes: Glob patterns that should match (empty means no inclusions)
//   - excludes: Glob patterns that should not match (takes priority over includes)
//
// Returns:
//   - bool: true if path matches at least one include pattern and no exclude patterns, false otherwise
func MatchPatterns(path string, includes, excludes []string) bool {
	for _, pattern := range excludes {
		if MatchGlob(path, pattern) {
			return false
		}
	}

	for _, pattern := range includes {
		if MatchGlob(path, pattern) {
			return true
		}
	}

	return false
}

// VersionInfo holds parsed version constraint and version string.
//
// This type represents a version specification that may include both a constraint
// operator and a version number. For example, "^1.2.3" would be parsed into
// Constraint: "^" and Version: "1.2.3".
//
// Fields:
//   - Constraint: The version constraint operator (e.g., "^", "~", ">=", "<=", ">", "<", "=", "*", or "")
//   - Version: The version number or version specifier (e.g., "1.2.3", "latest", "*")
type VersionInfo struct {
	Constraint string
	Version    string
}

var (
	constraintRegex = regexp.MustCompile(`^([~^*]|>=?|<=?|=)?[\s]*(.+)$`)
	templateRegex   = regexp.MustCompile(`\$\{[^:}]+:-([^}]+)\}`)
	// Support various version formats:
	// - Standard semver: 1.0.0, v1.0.0, 1.0.0-rc1, 1.0.0+build
	// - Multi-segment: 1.0.0.0, 2024.01.15 (CalVer)
	// - Prefixed versions: next-14.0.3, release-2024.01.15
	// - Named versions: latest, bookworm
	versionRegex = regexp.MustCompile(`^v?(\d+(?:\.\d+)*(?:\.[xX*])?(?:-[\w.]+)?(?:\+[\w.]+)?|latest|[\w-]+(?:[._]\d+(?:\.\d+)*)?)`)
)

// ParseVersion extracts constraint and version from a version string.
//
// It performs the following operations:
//   - Step 1: Extracts version from template syntax ${VAR:-version}
//   - Step 2: Handles range syntax (e.g., "1.0.0 - 2.0.0", "||")
//   - Step 3: Handles wildcard "*" as both constraint and version
//   - Step 4: Extracts constraint operator (^, ~, >=, <=, >, <, =)
//   - Step 5: Normalizes version string, removing "v" prefix if present
//
// Parameters:
//   - versionStr: The version string to parse (e.g., "^1.2.3", ">=2.0.0", "1.0.0 - 2.0.0")
//
// Returns:
//   - VersionInfo: Parsed constraint and version components
func ParseVersion(versionStr string) VersionInfo {
	if matches := templateRegex.FindStringSubmatch(versionStr); len(matches) > 1 {
		versionStr = matches[1]
	}

	if strings.Contains(versionStr, " - ") || strings.Contains(versionStr, "||") {
		return VersionInfo{
			Constraint: "",
			Version:    versionStr,
		}
	}

	if versionStr == "*" {
		return VersionInfo{
			Constraint: "*",
			Version:    "*",
		}
	}

	matches := constraintRegex.FindStringSubmatch(versionStr)
	if len(matches) < 3 {
		return VersionInfo{Version: versionStr}
	}

	constraint := matches[1]
	version := strings.TrimSpace(matches[2])

	if versionMatches := versionRegex.FindStringSubmatch(version); len(versionMatches) > 1 {
		version = versionMatches[1]
	}

	return VersionInfo{
		Constraint: constraint,
		Version:    version,
	}
}

// MapConstraint maps a constraint using provided mappings, or returns original if not found.
//
// This is useful for normalizing package manager-specific constraint syntax
// to a standard form.
//
// Parameters:
//   - original: The original constraint string to map
//   - mappings: A map from constraint strings to their normalized equivalents
//
// Returns:
//   - string: The mapped constraint if found in mappings, otherwise the original constraint
func MapConstraint(original string, mappings map[string]string) string {
	if mapped, ok := mappings[original]; ok {
		return mapped
	}
	return original
}

// ExtractNamedGroups extracts named capture groups from a regex match.
//
// It performs the following operations:
//   - Step 1: Validates regex safety to prevent ReDoS
//   - Step 2: Compiles or retrieves cached regex
//   - Step 3: Finds first match in text
//   - Step 4: Extracts all named groups into a map
//
// Parameters:
//   - pattern: The regex pattern with named groups (e.g., "(?P<name>\\w+)")
//   - text: The text to match against
//
// Returns:
//   - map[string]string: Map of group names to matched values; nil if no match found
//   - error: Returns nil on success; returns validation or compilation error if pattern is invalid or unsafe
func ExtractNamedGroups(pattern, text string) (map[string]string, error) {
	if err := ValidateRegexSafety(pattern); err != nil {
		return nil, err
	}
	re, err := getOrCompileRegex(pattern)
	if err != nil {
		return nil, err
	}

	matches := re.FindStringSubmatch(text)
	if matches == nil {
		return nil, nil
	}

	result := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" && i < len(matches) {
			result[name] = matches[i]
		}
	}

	return result, nil
}

// ExtractAllMatches extracts all named capture group matches from text.
//
// It performs the following operations:
//   - Step 1: Validates regex safety to prevent ReDoS
//   - Step 2: Compiles or retrieves cached regex
//   - Step 3: Finds all matches in text
//   - Step 4: Extracts named groups from each match into a map
//
// Parameters:
//   - pattern: The regex pattern with named groups (e.g., "(?P<name>\\w+)")
//   - text: The text to match against
//
// Returns:
//   - []map[string]string: Slice of maps, each containing group names to matched values; nil if no matches found
//   - error: Returns nil on success; returns validation or compilation error if pattern is invalid or unsafe
func ExtractAllMatches(pattern, text string) ([]map[string]string, error) {
	if err := ValidateRegexSafety(pattern); err != nil {
		return nil, err
	}
	re, err := getOrCompileRegex(pattern)
	if err != nil {
		return nil, err
	}

	allMatches := re.FindAllStringSubmatch(text, -1)
	if allMatches == nil {
		return nil, nil
	}

	var results []map[string]string
	names := re.SubexpNames()

	for _, matches := range allMatches {
		result := make(map[string]string)
		for i, name := range names {
			if i != 0 && name != "" && i < len(matches) {
				result[name] = matches[i]
			}
		}
		if len(result) > 0 {
			results = append(results, result)
		}
	}

	return results, nil
}

// MatchWithIndex holds a regex match with its position in the source text.
//
// This type is useful for precise text replacement when multiple matches exist,
// as it provides both the matched content and exact positions in the source.
//
// Fields:
//   - Groups: Map of named capture group names to their matched string values
//   - GroupIndex: Map of named capture group names to their [start, end] indices in text
//   - FullMatch: The entire matched string
//   - Start: Start index of the full match in the source text
//   - End: End index of the full match in the source text
type MatchWithIndex struct {
	Groups     map[string]string // Named capture groups
	GroupIndex map[string][2]int // Start/end indices for each named group
	FullMatch  string            // The entire matched string
	Start      int               // Start index of full match
	End        int               // End index of full match
}

// ExtractAllMatchesWithIndex extracts all matches with their positions.
//
// It performs the following operations:
//   - Step 1: Validates regex safety to prevent ReDoS
//   - Step 2: Compiles or retrieves cached regex
//   - Step 3: Finds all match positions (indices) in text
//   - Step 4: Extracts named groups and their indices for each match
//
// This is useful for precise replacement when multiple packages share the same version,
// as the indices allow for accurate in-place modifications.
//
// Parameters:
//   - pattern: The regex pattern with named groups (e.g., "(?P<name>\\w+)")
//   - text: The text to match against
//
// Returns:
//   - []MatchWithIndex: Slice of matches with groups, indices, and positions; nil if no matches found
//   - error: Returns nil on success; returns validation or compilation error if pattern is invalid or unsafe
func ExtractAllMatchesWithIndex(pattern, text string) ([]MatchWithIndex, error) {
	if err := ValidateRegexSafety(pattern); err != nil {
		return nil, err
	}
	re, err := getOrCompileRegex(pattern)
	if err != nil {
		return nil, err
	}

	allIndices := re.FindAllStringSubmatchIndex(text, -1)
	if allIndices == nil {
		return nil, nil
	}

	names := re.SubexpNames()
	var results []MatchWithIndex

	for _, indices := range allIndices {
		// indices always has at least 2 elements (full match start/end)
		match := MatchWithIndex{
			Groups:     make(map[string]string),
			GroupIndex: make(map[string][2]int),
			FullMatch:  text[indices[0]:indices[1]],
			Start:      indices[0],
			End:        indices[1],
		}

		for i, name := range names {
			if i == 0 || name == "" {
				continue
			}
			startIdx := i * 2
			endIdx := startIdx + 1
			if startIdx < len(indices) && endIdx < len(indices) {
				start, end := indices[startIdx], indices[endIdx]
				if start >= 0 && end >= 0 {
					match.Groups[name] = text[start:end]
					match.GroupIndex[name] = [2]int{start, end}
				}
			}
		}

		if len(match.Groups) > 0 {
			results = append(results, match)
		}
	}

	return results, nil
}

// XMLNode represents a generic XML node for parsing.
//
// This type provides a flexible structure for parsing and traversing arbitrary
// XML documents without requiring predefined schemas.
//
// Fields:
//   - XMLName: The XML element name and namespace
//   - Attrs: Slice of XML attributes on this node
//   - Content: The text content within this node (chardata)
//   - Nodes: Slice of child XML nodes
type XMLNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content string     `xml:",chardata"`
	Nodes   []XMLNode  `xml:",any"`
}

// FindXMLNodes finds nodes at the given path from the root.
//
// It traverses the XML tree following a slash-separated path of element names.
// For example, "Project/ItemGroup/PackageReference" would find all PackageReference
// elements nested under ItemGroup elements under the root Project element.
//
// Parameters:
//   - root: The root XMLNode to start searching from
//   - path: Slash-separated path of element names (e.g., "parent/child/grandchild")
//
// Returns:
//   - []*XMLNode: Slice of pointers to matching nodes; empty slice if no matches found
func FindXMLNodes(root *XMLNode, path string) []*XMLNode {
	parts := strings.Split(path, "/")
	return findNodesRecursive([]*XMLNode{root}, parts)
}

// findNodesRecursive recursively traverses XML nodes following a path.
//
// It matches nodes against the current path element, then recursively
// processes remaining path elements on matching nodes.
//
// Parameters:
//   - nodes: The current set of nodes to search within
//   - path: The remaining path elements to match
//
// Returns:
//   - []*XMLNode: Slice of nodes that match the complete path
func findNodesRecursive(nodes []*XMLNode, path []string) []*XMLNode {
	if len(path) == 0 || len(nodes) == 0 {
		return nodes
	}

	var result []*XMLNode
	currentPath := path[0]
	remainingPath := path[1:]

	for _, node := range nodes {
		for i := range node.Nodes {
			if node.Nodes[i].XMLName.Local == currentPath {
				result = append(result, &node.Nodes[i])
			}
		}
	}

	if len(remainingPath) > 0 {
		return findNodesRecursive(result, remainingPath)
	}

	return result
}

// GetXMLNodeText returns the trimmed text content of an XML node.
//
// Parameters:
//   - node: The XMLNode to extract text from (can be nil)
//
// Returns:
//   - string: The trimmed text content of the node; empty string if node is nil
func GetXMLNodeText(node *XMLNode) string {
	if node == nil {
		return ""
	}
	return strings.TrimSpace(node.Content)
}

// GetXMLAttr returns the value of a named attribute from an XML node.
//
// Parameters:
//   - node: The XMLNode to extract attribute from
//   - name: The local name of the attribute to retrieve
//
// Returns:
//   - string: The attribute value if found; empty string if attribute doesn't exist
func GetXMLAttr(node *XMLNode, name string) string {
	for _, attr := range node.Attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

// FindFilesByPatterns finds files matching any of the given glob patterns.
//
// It performs the following operations:
//   - Step 1: Walks directory tree starting from baseDir
//   - Step 2: Skips common directories (node_modules, vendor, .git, venv, testdata)
//   - Step 3: Matches each file against provided glob patterns
//   - Step 4: Deduplicates matches and returns absolute paths
//
// Parameters:
//   - baseDir: The base directory to search from; uses "." if empty
//   - patterns: Glob patterns to match files against (supports **, *, ?)
//
// Returns:
//   - []string: Slice of absolute file paths matching any pattern; empty slice if none found
//   - error: Returns nil on success; returns error if directory walk fails
func FindFilesByPatterns(baseDir string, patterns []string) ([]string, error) {
	if baseDir == "" {
		baseDir = "."
	}

	seen := make(map[string]struct{})
	var matches []string
	skipDirs := map[string]struct{}{
		"node_modules":    {},
		"vendor":          {},
		".git":            {},
		"venv":            {},
		"testdata":        {},
		"testdata_errors": {},
	}

	absBaseDir, _ := filepath.Abs(baseDir)
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			// Don't skip the baseDir itself, only subdirectories
			absPath, _ := filepath.Abs(path)
			if absPath != absBaseDir {
				if _, skip := skipDirs[info.Name()]; skip {
					return filepath.SkipDir
				}
			}
			return nil
		}

		relPath := path
		if rel, relErr := filepath.Rel(baseDir, path); relErr == nil {
			relPath = rel
		}
		relPath = filepath.ToSlash(relPath)
		base := filepath.Base(relPath)

		for _, pattern := range patterns {
			if pattern == "" {
				continue
			}
			if MatchGlob(relPath, pattern) || MatchGlob(base, pattern) {
				if _, exists := seen[path]; !exists {
					seen[path] = struct{}{}
					matches = append(matches, path)
				}
				break
			}
		}

		return nil
	})

	return matches, err
}

// NormalizePath normalizes a file path.
//
// It cleans the path by removing redundant separators, resolving . and .. elements,
// and converting to the shortest equivalent path.
//
// Parameters:
//   - path: The file path to normalize
//
// Returns:
//   - string: The normalized file path
func NormalizePath(path string) string {
	return filepath.Clean(path)
}
