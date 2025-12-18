package testutil

import (
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/systemtest"
)

// PackageBuilder provides a fluent API for building test packages.
//
// Use this builder to construct Package objects for testing purposes
// without needing to set all fields manually.
type PackageBuilder struct {
	pkg formats.Package
}

// NewPackage creates a new PackageBuilder with the given name.
//
// Initializes a builder with the package name set and all other
// fields empty or at their zero values.
//
// Parameters:
//   - name: Package name to set
//
// Returns:
//   - *PackageBuilder: New builder instance ready for method chaining
func NewPackage(name string) *PackageBuilder {
	return &PackageBuilder{
		pkg: formats.Package{
			Name: name,
		},
	}
}

// WithRule sets the rule for the package.
//
// Parameters:
//   - rule: Rule identifier (e.g., "npm", "pip", "mod")
//
// Returns:
//   - *PackageBuilder: Self for method chaining
func (b *PackageBuilder) WithRule(rule string) *PackageBuilder {
	b.pkg.Rule = rule
	return b
}

// WithType sets the type for the package.
//
// Parameters:
//   - t: Dependency type (e.g., "prod", "dev", "optional")
//
// Returns:
//   - *PackageBuilder: Self for method chaining
func (b *PackageBuilder) WithType(t string) *PackageBuilder {
	b.pkg.Type = t
	return b
}

// WithPackageType sets the package manager type.
//
// Parameters:
//   - pt: Package manager type (e.g., "js", "golang", "dotnet", "python")
//
// Returns:
//   - *PackageBuilder: Self for method chaining
func (b *PackageBuilder) WithPackageType(pt string) *PackageBuilder {
	b.pkg.PackageType = pt
	return b
}

// WithVersion sets the declared version for the package.
//
// Parameters:
//   - v: Version string as declared in package file (e.g., "1.0.0", "^2.0.0")
//
// Returns:
//   - *PackageBuilder: Self for method chaining
func (b *PackageBuilder) WithVersion(v string) *PackageBuilder {
	b.pkg.Version = v
	return b
}

// WithInstalledVersion sets the installed version for the package.
//
// Parameters:
//   - v: Version string as installed (from lock file)
//
// Returns:
//   - *PackageBuilder: Self for method chaining
func (b *PackageBuilder) WithInstalledVersion(v string) *PackageBuilder {
	b.pkg.InstalledVersion = v
	return b
}

// WithConstraint sets the constraint for the package.
//
// Parameters:
//   - c: Version constraint symbol (e.g., "^", "~", ">=", "=")
//
// Returns:
//   - *PackageBuilder: Self for method chaining
func (b *PackageBuilder) WithConstraint(c string) *PackageBuilder {
	b.pkg.Constraint = c
	return b
}

// WithSource sets the source file path for the package.
//
// Parameters:
//   - s: Path to the file where the package was defined
//
// Returns:
//   - *PackageBuilder: Self for method chaining
func (b *PackageBuilder) WithSource(s string) *PackageBuilder {
	b.pkg.Source = s
	return b
}

// WithGroup sets the group for the package.
//
// Parameters:
//   - g: Group name for batch updates
//
// Returns:
//   - *PackageBuilder: Self for method chaining
func (b *PackageBuilder) WithGroup(g string) *PackageBuilder {
	b.pkg.Group = g
	return b
}

// Build returns the built package.
//
// Returns the constructed Package. The builder can be reused after
// calling Build.
//
// Returns:
//   - formats.Package: The built package instance
func (b *PackageBuilder) Build() formats.Package {
	return b.pkg
}

// NPMPackage creates a typical NPM package for testing.
//
// Creates a JavaScript package with NPM defaults including caret constraint.
//
// Parameters:
//   - name: Package name
//   - version: Declared version
//   - installed: Installed version from lock file
//
// Returns:
//   - formats.Package: Configured NPM package
func NPMPackage(name, version, installed string) formats.Package {
	return NewPackage(name).
		WithRule("npm").
		WithPackageType("js").
		WithType("prod").
		WithVersion(version).
		WithInstalledVersion(installed).
		WithConstraint("^").
		Build()
}

// GoPackage creates a typical Go module package for testing.
//
// Creates a Go module dependency with standard defaults.
//
// Parameters:
//   - name: Module path
//   - version: Declared version
//   - installed: Installed version from go.sum
//
// Returns:
//   - formats.Package: Configured Go module package
func GoPackage(name, version, installed string) formats.Package {
	return NewPackage(name).
		WithRule("mod").
		WithPackageType("golang").
		WithType("prod").
		WithVersion(version).
		WithInstalledVersion(installed).
		Build()
}

// DotNetPackage creates a typical .NET package for testing.
//
// Creates a NuGet package with standard .NET defaults.
//
// Parameters:
//   - name: NuGet package ID
//   - version: Declared version
//   - installed: Installed version from lock file
//
// Returns:
//   - formats.Package: Configured .NET package
func DotNetPackage(name, version, installed string) formats.Package {
	return NewPackage(name).
		WithRule("nuget").
		WithPackageType("dotnet").
		WithType("prod").
		WithVersion(version).
		WithInstalledVersion(installed).
		Build()
}

// PythonPackage creates a typical Python package for testing.
//
// Creates a pip package with standard Python defaults.
//
// Parameters:
//   - name: PyPI package name
//   - version: Declared version
//   - installed: Installed version
//
// Returns:
//   - formats.Package: Configured Python package
func PythonPackage(name, version, installed string) formats.Package {
	return NewPackage(name).
		WithRule("pip").
		WithPackageType("python").
		WithType("prod").
		WithVersion(version).
		WithInstalledVersion(installed).
		Build()
}

// ComposerPackage creates a typical Composer (PHP) package for testing.
//
// Creates a Composer package with standard PHP defaults.
//
// Parameters:
//   - name: Composer package name (e.g., "vendor/package")
//   - version: Declared version
//   - installed: Installed version from composer.lock
//
// Returns:
//   - formats.Package: Configured Composer package
func ComposerPackage(name, version, installed string) formats.Package {
	return NewPackage(name).
		WithRule("composer").
		WithPackageType("php").
		WithType("prod").
		WithVersion(version).
		WithInstalledVersion(installed).
		WithConstraint("^").
		Build()
}

// CreateSystemTestRunner creates a system test runner for testing.
//
// If cfg is nil, returns a runner with no tests configured which will
// return a passed result when run.
//
// Parameters:
//   - cfg: System tests configuration, or nil for empty runner
//   - noTimeout: Whether to disable test timeouts
//   - verbose: Whether to enable verbose output
//
// Returns:
//   - *systemtest.Runner: Configured test runner
func CreateSystemTestRunner(cfg *config.SystemTestsCfg, noTimeout, verbose bool) *systemtest.Runner {
	return systemtest.NewRunner(cfg, "/test", noTimeout, verbose)
}
