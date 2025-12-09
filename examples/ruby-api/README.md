# Ruby API Example

This example demonstrates how to add support for a package manager **not included in goupdate defaults**.

## Key Concepts

Ruby/Bundler is not built into goupdate, but you can add full support with a custom `.goupdate.yml`:

1. **Define the rule** with `format`, `include`, and `fields`
2. **Add outdated commands** to fetch versions from RubyGems API
3. **Add update commands** to regenerate lock files

## How It Works

```yaml
rules:
  bundler:
    manager: ruby
    format: gemfile
    include: ["**/Gemfile"]

    # Fetch versions from RubyGems API (using grep for portability)
    outdated:
      commands: |
        curl -s "https://rubygems.org/api/v1/versions/{{package}}.json" |
        grep -oE '"number":"[0-9]+\.[0-9]+(\.[0-9]+)*"' |
        grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)*'
      format: raw
      extraction:
        pattern: "(?P<version>[\\d.]+)"

    # Update lock file
    update:
      commands: |
        bundle update {{package}} --conservative
```

## Try It

```bash
cd examples/ruby-api
goupdate scan      # Discovers Gemfile
goupdate list      # Shows declared versions
goupdate outdated  # Fetches latest from RubyGems
```

## Extending to Other Package Managers

Use this pattern for any ecosystem:
- Maven/Gradle (Java)
- Cargo (Rust)
- Mix (Elixir)
- Pub (Dart/Flutter)

Just define the API call to fetch versions and the command to update lock files.
