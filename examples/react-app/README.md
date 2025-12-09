# React App Example

This example demonstrates advanced goupdate features for a typical React/TypeScript application.

## Key Concepts

1. **Package groups** - Update related packages together
2. **Incremental updates** - Update major versions one at a time for safety
3. **Ignore patterns** - Skip type packages that auto-update with their parents
4. **System tests** - Validate updates with type-check, tests, and build

## Configuration Highlights

```yaml
rules:
  npm:
    groups:
      framework: [react, react-dom, react-router-dom]  # Update together
      state: ["@tanstack/react-query", zustand]
      dev: [typescript, vite, vitest]

    incremental: [react, react-dom]  # One major at a time
    ignore: ["@types/*"]  # Skip type packages

system_tests:
  run_preflight: true
  tests:
    - name: type-check
      commands: npm run type-check
    - name: unit-tests
      commands: npm test -- --run
    - name: build
      commands: npm run build
```

## Try It

```bash
cd examples/react-app
npm install                # Install dependencies
goupdate scan              # Discover package.json
goupdate list              # Show declared versions
goupdate outdated          # Check for updates
goupdate update --dry-run  # Preview changes
```

## Why Groups?

React ecosystem packages often have peer dependency requirements. Updating `react` alone might break `react-dom`. Groups ensure they update atomically.

## Why Incremental?

Jumping from React 17 to 19 might introduce breaking changes. Incremental mode updates to 18 first, letting you validate before the next major bump.
