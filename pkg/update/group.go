package update

import (
	"strings"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
)

// NormalizeUpdateGroup returns the configured group identifier for a package update.
// It first checks if the package already has a group assigned (e.g., from ApplyPackageGroups),
// then falls back to the update config's group template. When no group is configured,
// it returns an empty string to indicate the package is not grouped with others.
func NormalizeUpdateGroup(cfg *config.UpdateCfg, pkg formats.Package) string {
	// Preserve existing group from ApplyPackageGroups (rule-level or top-level groups)
	if strings.TrimSpace(pkg.Group) != "" {
		return pkg.Group
	}

	group, ok := ResolveUpdateGroup(cfg, pkg)
	if !ok {
		return ""
	}

	return group
}

// ResolveUpdateGroup returns the resolved group identifier for a package and reports
// whether a group has been configured.
func ResolveUpdateGroup(cfg *config.UpdateCfg, pkg formats.Package) (string, bool) {
	if cfg == nil || strings.TrimSpace(cfg.Group) == "" {
		return "", false
	}

	replacer := strings.NewReplacer(
		"{{package}}", pkg.Name,
		"{{rule}}", pkg.Rule,
		"{{type}}", pkg.Type,
	)

	return replacer.Replace(cfg.Group), true
}

// UpdateGroupKey returns the key used to coordinate grouped updates. When no group is
// configured, it falls back to the package name to ensure isolation between
// ungrouped packages without populating the display column.
func UpdateGroupKey(cfg *config.UpdateCfg, pkg formats.Package) string {
	if strings.TrimSpace(pkg.Group) != "" {
		return pkg.Group
	}

	if group, ok := ResolveUpdateGroup(cfg, pkg); ok {
		return group
	}

	return pkg.Name
}
