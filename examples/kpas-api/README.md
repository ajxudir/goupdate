# kpas-api Example

This example demonstrates goupdate configuration for a Laravel PHP API with Vue.js frontend assets, based on the [matematikk-mooc/kpas-api](https://github.com/matematikk-mooc/kpas-api) project.

## Project Structure

The project uses multiple package managers:
- **Composer** (PHP) - Backend dependencies in `composer.json`
- **npm** (Node.js) - Frontend assets in `package.json`

## Configuration Features

### Minimal Grouping

Only packages with actual version coupling are grouped:

**npm:**
- `vite` - vite + laravel-vite-plugin + @vitejs/plugin-vue (plugins depend on vite)

**Composer:** No groups needed - all packages are independent.

Most packages update individually. System tests catch any incompatibilities.

### System Tests

Runs after each update to verify builds don't break:
1. `composer install` - PHP dependency installation
2. `npm install && npm run build` - Frontend asset compilation

## Usage

```bash
# Check for outdated dependencies
goupdate outdated

# Update Vite ecosystem together
goupdate update --group vite

# Update all minor/patch versions
goupdate update --minor

# Dry run to preview changes
goupdate update --dry-run
```

## GitHub Actions

To use with GitHub Actions auto-update workflow, the workflow will:
1. Detect both `composer` and `npm` rules
2. Setup PHP (with composer) and Node.js (using `.nvmrc`)
3. Run goupdate to check/apply updates
4. Create PR with changes

The `.nvmrc` file specifies Node.js version (v20.17.0) which is automatically detected.
