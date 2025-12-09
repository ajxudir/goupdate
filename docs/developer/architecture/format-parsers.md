# Format Parsers Architecture

> Format parsers extract package information from manifest files in various formats (JSON, YAML, XML, Raw).

## Table of Contents

- [Key Files](#key-files)
- [Parser Interface](#parser-interface)
- [Parser Factory](#parser-factory)
- [JSON Parser](#json-parser)
- [YAML Parser](#yaml-parser)
- [XML Parser](#xml-parser)
- [Raw Parser](#raw-parser)
- [Version Parsing](#version-parsing)
- [Constraint Mapping](#constraint-mapping)
- [Package Overrides](#package-overrides)
- [Ignore Patterns](#ignore-patterns)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [Related Documentation](#related-documentation)

---

## Key Files

| File | Purpose |
|------|---------|
| `pkg/formats/parser.go` | Parser factory |
| `pkg/formats/model.go` | Package model |
| `pkg/formats/json.go` | JSON parser |
| `pkg/formats/yaml.go` | YAML parser |
| `pkg/formats/xml.go` | XML parser |
| `pkg/formats/raw.go` | Regex-based parser |

## Parser Interface

**Location:** `pkg/formats/model.go`

```go
type FormatParser interface {
    Parse(content []byte, cfg *config.PackageManagerCfg) ([]Package, error)
}

type Package struct {
    Name             string
    Version          string
    Constraint       string
    Type             string  // prod, dev
    PackageType      string  // js, php, python, etc.
    Source           string  // File path
    Rule             string
    Group            string
    InstalledVersion string
    InstallStatus    string
}
```

## Parser Factory

**Location:** `pkg/formats/parser.go`

```go
func GetFormatParser(format string) (FormatParser, error) {
    switch format {
    case "json":
        return &JSONParser{}, nil
    case "yaml":
        return &YAMLParser{}, nil
    case "xml":
        return &XMLParser{}, nil
    case "raw":
        return &RawParser{}, nil
    default:
        return nil, fmt.Errorf("unsupported format: %s", format)
    }
}
```

## JSON Parser

**Location:** `pkg/formats/json.go`

**Supported Structures:**

1. **Object with version values (npm style):**
```json
{
  "dependencies": {
    "lodash": "^4.17.21",
    "express": "~4.18.0"
  }
}
```

2. **Array of objects:**
```json
{
  "dependencies": [
    {"name": "lodash", "version": "4.17.21"}
  ]
}
```

**Configuration:**
```yaml
format: json
fields:
  dependencies: prod
  devDependencies: dev
```

**Processing:**
1. Parse JSON into ordered map (preserves key order)
2. Iterate through configured fields
3. Extract name and version
4. Parse constraint from version string

## YAML Parser

**Location:** `pkg/formats/yaml.go`

**Supported Structures:**

```yaml
dependencies:
  lodash: "^4.17.21"
  express: "~4.18.0"
```

**Configuration:**
```yaml
format: yaml
fields:
  dependencies: prod
  dev-dependencies: dev
```

**Processing:**
1. Parse YAML into map structure
2. Navigate to configured fields
3. Extract name and version pairs

## XML Parser

**Location:** `pkg/formats/xml.go`

**Supported Structures (MSBuild/NuGet):**

```xml
<Project>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>
```

**Configuration:**
```yaml
format: xml
fields:
  ItemGroup/PackageReference: prod
extraction:
  path: "PackageReference"
  name_attr: "Include"
  version_attr: "Version"
```

**Processing:**
1. Parse XML document
2. Navigate using path expression
3. Extract attributes for name/version

## Raw Parser

**Location:** `pkg/formats/raw.go`

**For non-structured formats (requirements.txt, go.mod):**

```
lodash>=4.17.21
express~=4.18.0
```

**Configuration:**
```yaml
format: raw
extraction:
  pattern: '(?m)^(?P<n>[\w\-\.]+)(?:[ \t]*(?P<constraint>[><=~!]+)[ \t]*(?P<version>[\w\.\-\+]+))?'
```

**Required Named Groups:**

| Group | Description |
|-------|-------------|
| `n` or `name` | Package name |
| `version` | Version string |
| `constraint` | Optional constraint prefix |
| `version_alt` | Alternative version location |

**Processing:**
1. Apply regex pattern to content
2. Extract named groups from matches
3. Build package list from matches

## Version Parsing

All parsers use `utils.ParseVersion` to split constraint from version:

```go
vInfo := utils.ParseVersion(versionStr)
// vInfo.Version = "4.17.21"
// vInfo.Constraint = "^"
```

## Constraint Mapping

After parsing, constraints can be normalized:

```go
if cfg.ConstraintMapping != nil {
    vInfo.Constraint = utils.MapConstraint(vInfo.Constraint, cfg.ConstraintMapping)
}
```

**Example mapping:**
```yaml
constraint_mapping:
  "~=": "~"
  "==": "="
```

## Package Overrides

Applied after parsing:

```go
vInfo = utils.ApplyPackageOverride(name, vInfo, cfg)
```

## Ignore Patterns

Packages matching ignore patterns are skipped:

```go
if shouldIgnorePackage(name, cfg) {
    continue
}
```

**Configuration:**
```yaml
ignore:
  - php
  - ext-*
```

## Error Handling

Each parser returns specific errors:

| Parser | Error Types |
|--------|-------------|
| JSON | Invalid JSON, missing fields |
| YAML | Invalid YAML, missing fields |
| XML | Invalid XML, path not found |
| Raw | Invalid regex, no matches |

## Testing

**Test File:** `pkg/formats/formats_test.go`

Key test patterns:
- Valid content parsing
- Invalid content handling
- Field mapping
- Constraint extraction
- Ignore patterns

## Related Documentation

- [scan.md](./scan.md) - File detection
- [list.md](./list.md) - Package listing
- [configuration.md](./configuration.md) - Parser configuration
