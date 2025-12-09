# Go CLI Example

This example demonstrates goupdate configuration for a Go CLI application using cobra and viper.

## Key Concepts

1. **Module groups** - Keep related modules in sync
2. **Default Go support** - Uses built-in `mod` rule from defaults

## Configuration Highlights

```yaml
extends: [default]

rules:
  mod:
    groups:
      cli: [github.com/spf13/cobra, github.com/spf13/viper]
      logging: [go.uber.org/zap]
```

## Try It

```bash
cd examples/go-cli
go mod download          # Download dependencies
goupdate scan            # Discover go.mod
goupdate list            # Show declared versions
goupdate outdated        # Check for updates
```

## Why Groups?

Cobra and Viper are commonly used together. Grouping ensures version compatibility when both have updates available.

## How Go Modules Work

goupdate uses `go list -m -json -versions` to fetch available versions and `go mod tidy` to update the lock file (go.sum).
