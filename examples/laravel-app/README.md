# Laravel App Example

This example demonstrates goupdate configuration for a PHP/Laravel project using Composer.

## Key Concepts

1. **Package groups** - Group framework, plugins, and dev dependencies
2. **Version exclusions** - Skip alpha, beta, and RC releases

## Configuration Highlights

```yaml
extends: [default]

rules:
  composer:
    groups:
      framework: [laravel/framework, laravel/sanctum]
      plugins: [spatie/laravel-permission, predis/predis]
      dev: [phpunit/phpunit, laravel/pint, nunomaduro/larastan]

    exclude_versions:
      - "(?i)alpha"
      - "(?i)beta"
      - "(?i)RC"
```

## Try It

```bash
cd examples/laravel-app
composer install           # Install dependencies
goupdate scan              # Discover composer.json
goupdate list              # Show declared versions
goupdate outdated          # Check for updates
goupdate update --dry-run  # Preview changes
```

## Why Groups?

Laravel framework packages should update together to maintain compatibility. The `spatie/laravel-permission` and `predis/predis` plugins work together for caching permissions.

## How Composer Works

goupdate uses `composer show --all --format=json` to fetch available versions and `composer update --lock` to regenerate the lock file.
