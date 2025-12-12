# frontend Example

This example demonstrates goupdate configuration for a Vue.js frontend application using pnpm, based on the [matematikk-mooc/frontend](https://github.com/matematikk-mooc/frontend) project.

## Project Structure

The project uses a single package manager:
- **pnpm** (Node.js) - All dependencies in `package.json`

## Configuration Features

### Minimal Grouping

Only packages with actual version coupling are grouped:

**pnpm:**
- `vue` - vue + vue-router + @vue/test-utils (core Vue ecosystem)
- `vite` - vite + @vitejs/plugin-vue (plugins depend on vite)
- `vitest` - vitest + @vitest/coverage-v8 (test runner ecosystem)
- `eslint` - eslint + eslint-plugin-vue + @eslint/js (linting ecosystem)
- `typescript-eslint` - typescript-eslint + @typescript-eslint/parser (TS lint tooling)

Most packages update individually. System tests catch any incompatibilities.

### System Tests

Runs after each update to verify builds don't break:
1. `pnpm install` - Dependency installation
2. `pnpm run build` - Production build
3. `pnpm test` - Unit tests (if configured)

## Usage

```bash
# Check for outdated dependencies
goupdate outdated

# Update Vue ecosystem together
goupdate update --group vue

# Update Vite and plugins together
goupdate update --group vite

# Update all minor/patch versions
goupdate update --minor

# Dry run to preview changes
goupdate update --dry-run
```

## GitHub Actions

To use with GitHub Actions auto-update workflow, the workflow will:
1. Detect `pnpm` rules from `.goupdate.yml`
2. Setup Node.js (version 20)
3. Run goupdate to check/apply updates
4. Create PR with changes
