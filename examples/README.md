# Examples

This folder contains example configurations for popular frameworks and use cases.

| Example | Language | Package Manager | Demonstrates |
|---------|----------|-----------------|--------------|
| [react-app](react-app/) | JavaScript | npm | Groups, incremental updates, type ignoring |
| [django-app](django-app/) | Python | pip | Multiple manifest files, groups |
| [go-cli](go-cli/) | Go | go mod | Module groups, indirect deps |
| [laravel-app](laravel-app/) | PHP | Composer | Plugin groups, stability preferences |
| [ruby-api](ruby-api/) | Ruby | Bundler | **Custom config for unsupported PM** |

## Quick Start

```bash
# Try any example
cd examples/react-app
goupdate scan
goupdate list
goupdate outdated
```

## Adding Support for Unsupported Package Managers

See [ruby-api](ruby-api/) for a complete example of adding support for a package manager not included in defaults. The key is defining custom `outdated.commands` and `update.commands` in your `.goupdate.yml`.
