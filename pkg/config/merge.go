package config

import "github.com/ajxudir/goupdate/pkg/verbose"

// mergeConfigs merges two configurations with custom taking precedence.
//
// This performs a deep merge of two Config structures, where custom settings
// override base settings. Used for implementing the extends inheritance chain.
//
// Parameters:
//   - base: the base configuration
//   - custom: the custom configuration that overrides base
//
// Returns:
//   - *Config: the merged configuration
func mergeConfigs(base, custom *Config) *Config {
	if custom == nil {
		return base
	}

	merged := &Config{
		WorkingDir:      base.WorkingDir,
		Rules:           make(map[string]PackageManagerCfg),
		ExcludeVersions: base.ExcludeVersions,
		Groups:          make(map[string]GroupCfg),
		Incremental:     base.Incremental,
		SystemTests:     base.SystemTests,
	}

	for key, rule := range base.Rules {
		merged.Rules[key] = rule
	}

	for key, group := range base.Groups {
		merged.Groups[key] = group
	}

	for key, rule := range custom.Rules {
		if existingRule, exists := merged.Rules[key]; exists {
			mergedRule := mergeRules(existingRule, rule)
			merged.Rules[key] = mergedRule
			verbose.Printf("Rule %q: merged with existing rule\n", key)
		} else {
			merged.Rules[key] = rule
			verbose.Printf("Rule %q: added new rule (include=%v)\n", key, rule.Include)
		}
	}

	for key, group := range custom.Groups {
		if existing, exists := merged.Groups[key]; exists {
			merged.Groups[key] = mergeGroup(existing, group)
		} else {
			merged.Groups[key] = group
		}
	}

	merged.ExcludeVersions = mergeVersionPatterns(base.ExcludeVersions, custom.ExcludeVersions)
	merged.Incremental = mergeStringLists(base.Incremental, custom.Incremental)

	// Merge system_tests by test name (keyed merge)
	if custom.SystemTests != nil {
		merged.SystemTests = mergeSystemTests(merged.SystemTests, custom.SystemTests)
	}

	return merged
}

// mergeSystemTests merges system test configurations by test name.
//
// Tests are merged by their Name field (unique key). Custom tests override
// base tests with the same name, and new tests are appended.
// Top-level settings (RunPreflight, RunMode, StopOnFail) use override if set.
//
// Parameters:
//   - base: the base system tests configuration
//   - override: the override system tests configuration
//
// Returns:
//   - *SystemTestsCfg: the merged configuration
func mergeSystemTests(base, override *SystemTestsCfg) *SystemTestsCfg {
	if override == nil {
		return base
	}
	if base == nil {
		return override
	}

	merged := &SystemTestsCfg{
		RunPreflight: base.RunPreflight,
		RunMode:      base.RunMode,
		StopOnFail:   base.StopOnFail,
	}

	// Override top-level settings if set in override
	if override.RunPreflight != nil {
		merged.RunPreflight = override.RunPreflight
	}
	if override.RunMode != "" {
		merged.RunMode = override.RunMode
	}
	if override.StopOnFail != nil {
		merged.StopOnFail = override.StopOnFail
	}

	// Merge tests by name
	testsByName := make(map[string]SystemTestCfg)
	testOrder := make([]string, 0)

	// Add base tests
	for _, test := range base.Tests {
		testsByName[test.Name] = test
		testOrder = append(testOrder, test.Name)
	}

	// Override or add custom tests
	for _, test := range override.Tests {
		if _, exists := testsByName[test.Name]; !exists {
			testOrder = append(testOrder, test.Name)
		}
		testsByName[test.Name] = test
	}

	// Build final test list maintaining order
	merged.Tests = make([]SystemTestCfg, 0, len(testOrder))
	for _, name := range testOrder {
		merged.Tests = append(merged.Tests, testsByName[name])
	}

	return merged
}

// mergeGroup merges two group configurations.
//
// If custom has packages defined, they completely replace the base packages.
//
// Parameters:
//   - base: the base group configuration
//   - custom: the custom group configuration that overrides base
//
// Returns:
//   - GroupCfg: the merged group configuration
func mergeGroup(base, custom GroupCfg) GroupCfg {
	merged := base
	if custom.Packages != nil {
		merged.Packages = custom.Packages
	}
	return merged
}

// mergeGroupMaps merges two group configuration maps.
//
// This merges groups by name, with override groups taking precedence.
// New groups from override are added to the result.
//
// Parameters:
//   - base: the base group map
//   - override: the override group map
//
// Returns:
//   - map[string]GroupCfg: the merged group map, or nil if both inputs are nil
func mergeGroupMaps(base, override map[string]GroupCfg) map[string]GroupCfg {
	if base == nil && override == nil {
		return nil
	}

	merged := make(map[string]GroupCfg)

	for key, group := range base {
		merged[key] = group
	}

	for key, group := range override {
		if existing, ok := merged[key]; ok {
			merged[key] = mergeGroup(existing, group)
			continue
		}

		merged[key] = group
	}

	return merged
}

// mergeVersionPatterns overwrites base patterns with override.
//
// When extending configs, list fields are completely overwritten by the extending
// config to allow full customization. If override is nil, base is returned unchanged.
// If override is an empty slice, it clears the patterns.
//
// Parameters:
//   - base: the base pattern list
//   - override: the override pattern list (replaces base when not nil)
//
// Returns:
//   - []string: override if not nil, otherwise base
func mergeVersionPatterns(base, override []string) []string {
	if override == nil {
		return base
	}
	return override
}

// mergeStringLists overwrites base list with override.
//
// When extending configs, list fields are completely overwritten by the extending
// config to allow full customization. If override is nil, base is returned unchanged.
// If override is an empty slice, it clears the list.
//
// Parameters:
//   - base: the base string list
//   - override: the override string list (replaces base when not nil)
//
// Returns:
//   - []string: override if not nil, otherwise base
func mergeStringLists(base, override []string) []string {
	if override == nil {
		return base
	}
	return override
}

// mergeRules merges two package manager rule configurations.
//
// This performs a field-by-field merge where custom fields override base fields.
// List fields (include, exclude, etc.) are completely overwritten by custom config.
//
// Parameters:
//   - base: the base rule configuration
//   - custom: the custom rule configuration that overrides base
//
// Returns:
//   - PackageManagerCfg: the merged rule configuration
func mergeRules(base, custom PackageManagerCfg) PackageManagerCfg {
	merged := base

	// Enabled field must be merged first - if custom explicitly sets enabled, use it
	if custom.Enabled != nil {
		merged.Enabled = custom.Enabled
	}

	if custom.Manager != "" {
		merged.Manager = custom.Manager
	}
	// List fields: custom overwrites base (last config wins)
	if custom.Include != nil {
		merged.Include = mergeStringLists(merged.Include, custom.Include)
	}
	if custom.Exclude != nil {
		merged.Exclude = mergeStringLists(merged.Exclude, custom.Exclude)
	}
	if len(custom.Groups) > 0 {
		merged.Groups = mergeGroupMaps(merged.Groups, custom.Groups)
	}
	if len(custom.Packages) > 0 {
		merged.Packages = mergePackageSettings(merged.Packages, custom.Packages)
	}
	if custom.Format != "" {
		merged.Format = custom.Format
	}
	if len(custom.Fields) > 0 {
		merged.Fields = custom.Fields
	}
	if custom.Ignore != nil {
		merged.Ignore = mergeStringLists(merged.Ignore, custom.Ignore)
	}
	if custom.ExcludeVersions != nil {
		merged.ExcludeVersions = mergeVersionPatterns(merged.ExcludeVersions, custom.ExcludeVersions)
	}
	if len(custom.ConstraintMapping) > 0 {
		merged.ConstraintMapping = custom.ConstraintMapping
	}
	if custom.LatestMapping != nil {
		merged.LatestMapping = mergeLatestMappingCfg(merged.LatestMapping, custom.LatestMapping)
	}
	if len(custom.PackageOverrides) > 0 {
		merged.PackageOverrides = custom.PackageOverrides
	}
	if custom.Extraction != nil {
		merged.Extraction = custom.Extraction
	}
	if custom.Outdated != nil {
		merged.Outdated = custom.Outdated
	}
	if custom.Update != nil {
		merged.Update = custom.Update
	}
	if custom.LockFiles != nil {
		merged.LockFiles = mergeLockFiles(merged.LockFiles, custom.LockFiles)
	}
	if custom.Metadata != nil {
		merged.Metadata = custom.Metadata
	}
	if custom.Incremental != nil {
		merged.Incremental = mergeStringLists(merged.Incremental, custom.Incremental)
	}

	return merged
}

// mergeLockFiles merges lock file configurations.
//
// If override is nil, base is returned unchanged.
// If override is an empty slice, it clears the list.
// Otherwise, lock files are merged by their first file pattern, with override
// configurations replacing matching base configurations.
//
// Parameters:
//   - base: the base lock file configurations
//   - override: the override lock file configurations
//
// Returns:
//   - []LockFileCfg: the merged lock file configurations
func mergeLockFiles(base, override []LockFileCfg) []LockFileCfg {
	if override == nil {
		return base
	}

	if len(override) == 0 {
		return []LockFileCfg{}
	}

	// Create a map by first file pattern for deduplication
	seen := make(map[string]int, len(base)+len(override))
	merged := make([]LockFileCfg, 0, len(base)+len(override))

	// Add base configs
	for _, cfg := range base {
		if len(cfg.Files) > 0 {
			key := cfg.Files[0]
			seen[key] = len(merged)
			merged = append(merged, cfg)
		}
	}

	// Add or override from custom
	for _, cfg := range override {
		if len(cfg.Files) > 0 {
			key := cfg.Files[0]
			if idx, exists := seen[key]; exists {
				// Override existing
				merged[idx] = cfg
			} else {
				// Add new
				seen[key] = len(merged)
				merged = append(merged, cfg)
			}
		} else {
			// No files - just append
			merged = append(merged, cfg)
		}
	}

	return merged
}

// mergePackageSettings merges package settings maps.
// Custom settings override base settings for the same package.
//
// Parameters:
//   - base: the base package settings
//   - custom: the custom package settings that override base
//
// Returns:
//   - map[string]PackageSettings: the merged package settings
func mergePackageSettings(base, custom map[string]PackageSettings) map[string]PackageSettings {
	if base == nil {
		result := make(map[string]PackageSettings, len(custom))
		for k, v := range custom {
			result[k] = v
		}
		return result
	}

	result := make(map[string]PackageSettings, len(base)+len(custom))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range custom {
		result[k] = v
	}
	return result
}
