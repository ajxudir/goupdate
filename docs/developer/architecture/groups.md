# Package Groups Architecture

> Package groups enable atomic updates where multiple packages are updated together with a single lock command.

## Table of Contents

- [Key Files](#key-files)
- [Purpose](#purpose)
- [Configuration Syntax](#configuration-syntax)
- [Group Parsing](#group-parsing)
- [Group Validation](#group-validation)
- [Group Assignment](#group-assignment)
- [Update Group Key](#update-group-key)
- [Group Templates](#group-templates)
- [Group-Level Locking](#group-level-locking)
- [Group Rollback](#group-rollback)
- [Floating Constraint Restriction](#floating-constraint-restriction)
- [Group Display](#group-display)
- [Group Sorting](#group-sorting)
- [Global Groups](#global-groups)
- [Rule-Level Groups](#rule-level-groups)
- [Testing](#testing)
- [Related Documentation](#related-documentation)

---

## Key Files

| File | Purpose |
|------|---------|
| `pkg/config/groups.go` | Group parsing and validation |
| `pkg/update/group.go` | Group key resolution |
| `cmd/update.go` | Group-level processing |
| `cmd/list.go` | Group assignment |

## Purpose

Groups are used for:
1. **Atomic updates:** Update related packages together
2. **Lock efficiency:** Run lock command once for multiple packages
3. **Rollback scope:** Rollback entire group on failure

## Configuration Syntax

### Simple Sequence

```yaml
rules:
  npm:
    groups:
      react-ecosystem:
        - react
        - react-dom
        - react-router
```

### Map with Packages Key

```yaml
rules:
  npm:
    groups:
      react-ecosystem:
        packages:
          - react
          - react-dom
```

### Object Format with Name

```yaml
rules:
  npm:
    groups:
      react-ecosystem:
        - name: react
        - name: react-dom
```

## Group Parsing

**Location:** `pkg/config/groups.go:12-51`

```go
func (g *GroupCfg) UnmarshalYAML(value *yaml.Node) error {
    switch value.Kind {
    case yaml.SequenceNode:
        // ["react", "react-dom"]
        packages, err := parseGroupSequence(value.Content)
    case yaml.MappingNode:
        // {packages: [...]}
        for key, node := range pairs {
            if key == "packages" || key == "members" {
                parsed, err := parseGroupSequence(node.Content)
            }
        }
    }
}
```

## Group Validation

**Location:** `pkg/config/groups.go:87-131`

```go
func validateGroupMembership(cfg *Config) error
```

**Validation Rules:**
1. Package cannot be in multiple groups within same rule
2. Reports all conflicts with group names

**Error Example:**
```
rule npm has packages assigned to multiple groups: lodash (group-a, group-b)
```

## Group Assignment

**Location:** `cmd/list.go:217-243`

```go
func applyPackageGroups(packages []formats.Package, cfg *config.Config) []formats.Package
```

**Process:**
1. For each package, check if name appears in any group
2. Assign group name to `pkg.Group`
3. Used for display and update coordination

## Update Group Key

**Location:** `pkg/update/group.go:41-51`

```go
func UpdateGroupKey(cfg *config.UpdateCfg, pkg formats.Package) string {
    // 1. Use package's pre-assigned group
    if strings.TrimSpace(pkg.Group) != "" {
        return pkg.Group
    }

    // 2. Use config group template
    if group, ok := ResolveUpdateGroup(cfg, pkg); ok {
        return group
    }

    // 3. Fall back to package name (isolated updates)
    return pkg.Name
}
```

## Group Templates

**Location:** `pkg/update/group.go:24-36`

```go
func ResolveUpdateGroup(cfg *config.UpdateCfg, pkg formats.Package) (string, bool) {
    replacer := strings.NewReplacer(
        "{{package}}", pkg.Name,
        "{{rule}}", pkg.Rule,
        "{{type}}", pkg.Type,
    )
    return replacer.Replace(cfg.Group), true
}
```

**Template Placeholders:**

| Placeholder | Value |
|-------------|-------|
| `{{package}}` | Package name |
| `{{rule}}` | Rule name |
| `{{type}}` | Package type (prod/dev) |

**Example:**
```yaml
update:
  group: "{{rule}}-deps"  # Groups all packages by rule
```

## Group-Level Locking

**Location:** `cmd/update.go:393-457`

When multiple packages share the same group (via the `group` field):

```go
if useGroupLock && !dryRun {
    // 1. Update all declared versions (skipLock=true)
    for _, plan := range plans {
        updatePackageFunc(plan.res.pkg, plan.res.target, cfg, workDir, dryRun, true)
    }

    // 2. Run lock command once
    update.RunGroupLockCommand(groupUpdateCfg, workDir)

    // 3. Validate all packages
    for _, plan := range applied {
        validateUpdatedPackage(plan, reloadList, baseline)
    }
}
```

## Group Rollback

**Location:** `cmd/update.go:598-611`

On failure, entire group is rolled back:

```go
func rollbackPlans(plans []*plannedUpdate, cfg *config.Config, workDir string, failures *[]error, groupErr error) {
    for _, plan := range plans {
        // Restore original version
        updatePackageFunc(plan.res.pkg, plan.original, cfg, workDir, dryRun, skipLock)
        plan.res.status = "Failed"
    }
}
```

## Floating Constraint Restriction

**Location:** `cmd/update.go:211-220`

Floating constraints cannot be in groups:

```go
if utils.IsFloatingConstraint(p.Version) {
    if groupDisplay != "" && groupDisplay != "-" {
        floatingGroupErr := fmt.Errorf(
            "floating constraint '%s' cannot be used with group updates; "+
            "remove from group or use exact version", p.Version)
    }
}
```

**Reason:** Groups run a single lock command that expects specific versions.

## Group Display

In output, the GROUP column shows:
- Group name (if assigned)
- Empty or `-` (if not grouped)

```
RULE   PM  TYPE  VERSION  STATUS    GROUP           NAME
npm    js  prod  ^4.17    Updated   react-ecosystem react
npm    js  prod  ^4.17    Updated   react-ecosystem react-dom
npm    js  prod  ^4.17    Updated   -               lodash
```

## Group Sorting

**Location:** `cmd/update.go:159-167`

Packages are sorted to process groups together:

```go
sort.Slice(resolved, func(i, j int) bool {
    // 1. By rule
    // 2. By package type
    // 3. By group
    // 4. By type
    // 5. By name
})
```

## Global Groups

Groups can be defined globally (applies to all rules):

```yaml
groups:
  core-libs:
    packages:
      - lodash
      - express
```

## Rule-Level Groups

Groups defined in rules apply only to that rule:

```yaml
rules:
  npm:
    groups:
      react-ecosystem:
        - react
        - react-dom
```

## Testing

**Test File:** `pkg/config/groups_test.go`

Key test scenarios:
- Group parsing (all syntax variants)
- Validation (conflict detection)
- Group assignment
- Template resolution

## Related Documentation

- [update.md](./update.md) - Update command with groups
- [configuration.md](./configuration.md) - Group configuration
- [floating-constraints.md](./floating-constraints.md) - Floating + group restriction
