package update

import (
	"fmt"
	"strings"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/formats"
)

// ResolveUpdateCfg returns the effective update configuration for a package.
//
// It performs the following operations:
//   - Step 1: Validate rule configuration exists for the package
//   - Step 2: Check that update configuration is defined for the rule
//   - Step 3: Create a copy of the base update configuration
//   - Step 4: Apply package-specific overrides if they exist
//   - Step 5: Merge commands, environment, group, and timeout settings from overrides
//
// Parameters:
//   - p: The package to resolve configuration for
//   - cfg: Global configuration containing rules and package-specific overrides
//
// Returns:
//   - *config.UpdateCfg: Effective update configuration with overrides applied
//   - error: Returns error if rule is missing; returns UnsupportedError if update config is missing; returns nil on success
func ResolveUpdateCfg(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
	ruleCfg, ok := cfg.Rules[p.Rule]
	if !ok {
		return nil, fmt.Errorf("rule configuration missing for %s", p.Rule)
	}

	if ruleCfg.Update == nil {
		return nil, &errors.UnsupportedError{Reason: fmt.Sprintf("update configuration missing for %s", p.Rule)}
	}

	effective := *ruleCfg.Update

	if ruleCfg.PackageOverrides != nil {
		if override, ok := ruleCfg.PackageOverrides[p.Name]; ok && override.Update != nil {
			if override.Update.Commands != nil {
				effective.Commands = strings.TrimSpace(*override.Update.Commands)
			}
			if override.Update.Env != nil {
				effective.Env = make(map[string]string, len(override.Update.Env))
				for k, v := range override.Update.Env {
					effective.Env[k] = v
				}
			}
			if override.Update.Group != nil {
				effective.Group = strings.TrimSpace(*override.Update.Group)
			}
			if override.Update.TimeoutSeconds != nil {
				effective.TimeoutSeconds = *override.Update.TimeoutSeconds
			}
		}
	}

	return &effective, nil
}
