# Tool Comparison: goupdate vs Dependabot vs Renovate

This document provides a comprehensive comparison of goupdate with the two most popular dependency management tools: GitHub Dependabot and Mend Renovate. Use this to make an informed decision about which tool best fits your needs.

## Table of Contents

- [Quick Summary](#quick-summary)
- [Feature Comparison](#feature-comparison)
- [What goupdate Does Well](#what-goupdate-does-well)
- [Where Dependabot and Renovate Fall Short](#where-dependabot-and-renovate-fall-short)
- [What goupdate Is Missing](#what-goupdate-is-missing)
- [Sources](#sources)

---

## Quick Summary

| Aspect | goupdate | Renovate | Dependabot |
|--------|----------|----------|------------|
| **Primary Use** | CLI tool for auditing/updating | Automated PR creation | Automated PR creation |
| **Hosting** | Local/CI | Self-hosted or SaaS | GitHub-native |
| **Platform** | Any (CLI) | GitHub, GitLab, Bitbucket, Azure, Gitea | GitHub, Azure DevOps |
| **License** | MIT | AGPL | MIT |
| **Language** | Go | TypeScript | Ruby |
| **Runs Locally** | Yes | Limited (dry-run only) | No |
| **Cloud Required** | No | Yes (SaaS) or self-hosted | Yes (GitHub) |

## Feature Comparison

### Package Manager Support

| Package Manager | goupdate | Renovate | Dependabot |
|-----------------|:--------:|:--------:|:----------:|
| npm | ✅ | ✅ | ✅ |
| pnpm | ✅ | ✅ | ✅ |
| Yarn | ✅ | ✅ | ✅ |
| Go modules | ✅ | ✅ | ✅ |
| Composer (PHP) | ✅ | ✅ | ✅ |
| pip (requirements.txt) | ✅ | ✅ | ✅ |
| Pipenv | ✅ | ✅ | ✅ |
| NuGet | ✅ | ✅ | ✅ |
| Maven | ⚙️* | ✅ | ✅ |
| Gradle | ⚙️* | ✅ | ✅ |
| Bundler (Ruby) | ⚙️* | ✅ | ✅ |
| Cargo (Rust) | ⚙️* | ✅ | ✅ |
| Hex (Elixir) | ⚙️* | ✅ | ✅ |
| Docker | ⚙️* | ✅ | ✅ |
| Terraform | ⚙️* | ✅ | ✅ |
| Helm | ⚙️* | ✅ | ✅ |
| **Built-in Managers** | **9** | **90+** | **~20** |

*⚙️ = Can be added via configuration using native CLI tools or custom commands. Not officially supported out-of-box, but the config-based architecture allows extending to any package manager. See [examples/ruby-api/](../examples/ruby-api/) for a custom Bundler example.

### Core Features

| Feature | goupdate | Renovate | Dependabot |
|---------|:--------:|:--------:|:----------:|
| **Dependency Discovery** |
| Auto-detect manifests | ✅ | ✅ | ✅ |
| Monorepo support | ✅ | ✅ | ⚠️ Limited |
| Custom file patterns | ✅ | ✅ | ❌ |
| Private registries | ✅† | ✅ | ✅ |
| **Version Management** |
| Lock file parsing | ✅ | ✅ | ✅ |
| Version constraint detection | ✅ | ✅ | ✅ |
| Semantic versioning | ✅ | ✅ | ✅ |
| Pre-release filtering | ✅ | ✅ | ✅ |
| **Update Capabilities** |
| Check for updates | ✅ | ✅ | ✅ |
| Apply updates | ✅ | ✅ | ✅ |
| Automatic PRs | ✅‡ | ✅ | ✅ |
| PR grouping | ✅‡ | ✅ | ✅ |
| Scheduled updates | ✅‡ | ✅ | ✅ |
| **Reporting** |
| CLI output | ✅ | Limited | ❌ |
| JSON/CSV/XML export | ✅ | ❌ | ❌ |
| Dependency dashboard | 🔜 Planned° | ✅ | ❌ |
| Security advisories | ❌ | ✅ | ✅ |
| Merge confidence scores | ❌ | ✅ | ✅ |

†Private registries: Configure via native package manager tools (`.npmrc`, `composer config`, `GOPRIVATE`, etc.). No credentials stored in goupdate config.

°Dependency dashboard: OpenTelemetry integration planned, enabling custom dashboards via Grafana or similar tools.

‡Automation via CI: Schedule goupdate in CI (GitHub Actions, GitLab CI) to run on a cron schedule, create PRs per scope (major/minor/patch), and auto-merge to staging if tests pass. See examples below.

### Configuration & Customization

| Feature | goupdate | Renovate | Dependabot |
|---------|:--------:|:--------:|:----------:|
| YAML configuration | ✅ | ✅ (JSON5) | ✅ |
| Extends/inheritance | ✅ | ✅ | ❌ |
| Package grouping | ✅ | ✅ | ✅ |
| Incremental updates | ✅ | ✅ | ❌ |
| Version exclusion patterns | ✅ | ✅ | ✅ |
| Per-package overrides | ✅ | ✅ | ✅ |
| Custom update commands | ✅ | ✅ | ❌ |
| Regex-based versioning | ✅ | ✅ | ❌ |
| Timeout configuration | ✅ | ✅ | ❌ |

### Platform & Integration

| Feature | goupdate | Renovate | Dependabot |
|---------|:--------:|:--------:|:----------:|
| **Platforms** |
| GitHub | ✅ (CI) | ✅ | ✅ Native |
| GitLab | ✅ (CI) | ✅ | ❌ |
| Bitbucket | ✅ (CI) | ✅ | ❌ |
| Azure DevOps | ✅ (CI) | ✅ | ✅ |
| Self-hosted Git | ✅ | ✅ | ❌ |
| **Deployment** |
| CLI binary | ✅ | Limited§ | ❌ |
| Docker image | ✅ | ✅ | ❌ |
| Self-hosted | ✅ | ✅ | ⚠️ Unofficial |
| **CI/CD Automation** |
| Reusable workflow examples | ✅ Complete | ⚠️ Limited | ❌ GitHub only |
| Platform-portable scripts | ✅ Yes | ❌ Node.js required | ❌ No |
| Release automation | ✅ GoReleaser + Docker | ⚠️ PR-based | ⚠️ PR-based |

§Renovate CLI only supports dry-run mode locally; full functionality requires a git repository and server/CI environment.

### Security Features

| Feature | goupdate | Renovate | Dependabot |
|---------|:--------:|:--------:|:----------:|
| Vulnerability alerts | ❌^ | ✅ | ✅ |
| Security-only updates | ❌ | ✅ | ✅ |
| CVE database integration | ❌ | ✅ | ✅ |

^Vulnerability alerts: OpenTelemetry support planned, enabling custom alert integrations with Slack, Teams, or other notification systems for major updates.

## What goupdate Does Well

| Strength | Description |
|----------|-------------|
| **CLI-first approach** | Fast local auditing without cloud dependencies |
| **Unified view** | Single report across all ecosystems in one command |
| **Enterprise config** | YAML inheritance for organizational standards |
| **Incremental updates** | Step-by-step version upgrades (nearest major/minor/patch) |
| **Lock file awareness** | Explicit status for missing/incomplete locks |
| **Pre-flight validation** | Validates package manager availability before running |
| **Deterministic output** | Consistent output for CI diffing and auditing |
| **Lightweight** | Single Go binary, no runtime dependencies |
| **Reusable CI/CD workflows** | Complete GitHub Actions with GitLab/Bitbucket/Azure examples |
| **Release automation** | GoReleaser + Docker builds with prerelease/stable flow |

## Where Dependabot and Renovate Fall Short

While Dependabot and Renovate are popular choices, both have significant limitations that goupdate addresses. This section provides a balanced look at what each tool lacks.

### Cloud & Third-Party Service Reliance

| Requirement | goupdate | Renovate | Dependabot |
|-------------|:--------:|:--------:|:----------:|
| Requires cloud service | ❌ No | ✅ Mend SaaS or self-hosted | ✅ GitHub |
| Internet for registry queries | ✅ Yes | ✅ Yes | ✅ Yes |
| Internet for Git operations | ❌ No | ✅ Yes | ✅ Yes |
| Third-party account needed | ❌ No | ⚠️ Optional (Mend) | ✅ GitHub |
| Works air-gapped* | ✅ Yes | ❌ No | ❌ No |

*With local/cached package registry mirrors.

**goupdate** operates entirely locally—no cloud services, no third-party accounts, no vendor lock-in. Run it on your laptop, in CI, or air-gapped environments.

**Renovate** requires either Mend's SaaS platform or significant self-hosting infrastructure. Even self-hosted instances need continuous Git server connectivity.

**Dependabot** is inseparable from GitHub—there's no way to use it outside of GitHub's ecosystem.

---

### Silent Failures & Visibility

| Issue | goupdate | Renovate | Dependabot |
|-------|:--------:|:--------:|:----------:|
| Can fail silently | ❌ No | ⚠️ Logs hidden | ✅ Yes |
| Clear error reporting | ✅ CLI output | ⚠️ Debug logs | ❌ Hidden logs |
| Dashboard visibility | 🔜 Planned | ✅ Yes | ❌ None |

> "Dependabot can fail silently. That happened to us multiple times a year when Dependabot would just stop working... There's nothing warning you when Dependabot is broken, and the logs are hidden in an unintuitive location." — [Infield AI](https://www.infield.ai/post/the-limitations-of-dependabot)

**goupdate**: Immediate feedback in the terminal. Errors are visible, not buried in logs.

---

### PR Spam & Noise

Both Dependabot and Renovate are notorious for creating overwhelming numbers of pull requests:

| Issue | goupdate | Renovate | Dependabot |
|-------|:--------:|:--------:|:----------:|
| Creates PR per dependency | ❌ You decide | ⚠️ Configurable | ✅ Default |
| Flexible grouping | ✅ Yes | ⚠️ Complex config | ⚠️ Limited |
| Alert fatigue | ❌ No | ⚠️ Common | ✅ Common |
| Treats all vulnerabilities equally | ❌ No | ⚠️ Partially | ✅ Yes |

> "Dependabot generates tons of pull requests and security alerts without proper prioritization, treats all vulnerabilities equally regardless of actual exploitability." — [Why Every Developer Thinks Dependabot Sucks](https://blog.shivamsaraswat.com/dependabot-sucks/)

**goupdate**: You control when and how updates happen. Group updates by:
- **Scope**: `--major`, `--minor`, `--patch` to update all packages of a given scope together
- **Package manager**: `--package-manager npm` to update only npm packages
- **Rule**: `--rule npm,pnpm` to update specific rules together
- **Group**: Define custom groups in config for related packages (e.g., all React packages)

Run `goupdate outdated` to see what needs updating, then apply grouped updates to a single branch. No surprise PRs flooding your repository.

---

### Scaling & Timeout Issues

Both tools struggle with large repositories:

| Issue | goupdate | Renovate | Dependabot |
|-------|:--------:|:--------:|:----------:|
| Times out on large repos | ❌ No limit | ⚠️ Configurable | ✅ 55 min limit |
| Poor monorepo support | ❌ No | ❌ No | ✅ Yes |
| 100 deps = 100 PRs problem | ❌ You group | ⚠️ Groupable | ✅ Yes |

> "Dependabot has a job timeout of around 45-60 minutes, which can be insufficient for large monorepos with many dependencies."

> "Self-hosting Renovate: If you attempt to run Renovate on a large repository, you may encounter a SIGTERM signal due to timeout." — [Renovate Docs](https://docs.renovatebot.com/self-hosted-configuration/)

**goupdate**: Runs as fast as your package registries respond. No arbitrary timeouts. Handle monorepos with a single command.

---

### Configuration Complexity

| Feature | goupdate | Renovate | Dependabot |
|---------|:--------:|:--------:|:----------:|
| Config inheritance | ✅ Yes | ✅ Yes | ❌ No |
| Per-dependency scheduling | ✅ Yes (CI) | ✅ Yes | ❌ No |
| Custom update commands | ✅ Yes | ✅ Yes | ❌ No |
| Add new package managers | ✅ YAML only | ⚠️ Code required | ❌ Not accepted |

**Dependabot** offers only basic configuration options at the language level—no inheritance, no custom commands, and critically, **GitHub doesn't accept contributions to add new ecosystems**.

**Renovate** requires TypeScript code to add full package manager support. Lock file support needs code changes plus Containerbase integration.

> "Code for package managers goes in the `lib/modules/manager/*` directory. The package manager code is often tightly coupled to the datasource code." — [Renovate Adding a Package Manager](https://github.com/renovatebot/renovate/blob/main/docs/development/adding-a-package-manager.md)

**goupdate**: Add any package manager via pure YAML configuration. See [examples/ruby-api/](../examples/ruby-api/) for adding Bundler without code.

---

### Output & Reporting

| Format | goupdate | Renovate | Dependabot |
|--------|:--------:|:--------:|:----------:|
| CLI table output | ✅ Rich tables | ❌ Logs only | ❌ None |
| JSON export | ✅ Built-in | ⚠️ Experimental | ❌ No |
| CSV export | ✅ Built-in | ❌ No | ❌ No |
| XML export | ✅ Built-in | ❌ No | ❌ No |

Neither Dependabot nor Renovate provides clean, structured output for auditing:
- Dependabot has no CLI output at all
- Renovate's `reportType` is experimental; debug logs can grow to hundreds of MB

**goupdate**: Built-in `--output json|csv|xml` for any command. Clean, structured, CI-friendly output for compliance and auditing.

---

### Atomic Rollback

Neither Dependabot nor Renovate provides automatic rollback when grouped updates fail:

| Capability | goupdate | Renovate | Dependabot |
|------------|:--------:|:--------:|:----------:|
| Group updates | ✅ Built-in | ✅ Built-in | ✅ Manual config |
| Atomic rollback on failure | ✅ Yes | ❌ No | ❌ No |
| Identify failed package | ✅ Automatic | ❌ Manual | ❌ Manual |

When Dependabot or Renovate groups packages into a single PR and tests fail:
- No automatic rollback
- Manual intervention required to identify the culprit
- All-or-nothing without granular control

**goupdate**: Automatic atomic rollback—if any package in a group fails, the entire group reverts to the original state. Manifest and lock files are automatically restored.

---

### Automerge & Scheduling Limitations

**Renovate automerge constraints:**

| Limitation | Impact |
|------------|--------|
| One merge per run | Can only automerge 1 branch per execution cycle |
| Single restart | Repository run restarts at most once after automerge |
| Up-to-date requirement | Branch must be current with target branch |

> "Renovate automerges at most one branch/PR per Renovate run." — [Renovate Known Limitations](https://docs.renovatebot.com/known-limitations/)

**Mend Renovate App timing:**
- Checks repositories only every **3 hours**
- Schedule windows must be at least 3-4 hours
- No guarantee of running during your configured window

**Dependabot limits:**
- 5 PRs initially, 10 for security updates
- No fine-grained scheduling control

**goupdate**: Run on-demand, any time, via CLI or CI cron. Updates applied directly—no PR bottleneck, instant rollback.

---

### Incremental Updates

For step-by-step major version upgrades (v1 → v2 → v3 → v4):

| Tool | Approach |
|------|----------|
| goupdate | ✅ Built-in `incremental: true` |
| Renovate | ⚠️ Requires `:separateMultipleMajorReleases` preset |
| Dependabot | ❌ Not supported—jumps to latest |

With goupdate's incremental mode, if you're on v1 and v4 is latest, it suggests v2 (not v4). No configuration gymnastics required.

---

### License & Installation

| Aspect | goupdate | Renovate | Dependabot |
|--------|:--------:|:--------:|:----------:|
| License | MIT | AGPL-3.0 | MIT |
| Install size | ~8MB binary | ~300MB+ (Node.js ecosystem) | N/A (GitHub-hosted) |
| Runtime dependencies | None | Node.js 18+ | GitHub Actions |
| Self-hosting | ✅ Download and run | ⚠️ Requires setup | ❌ Not available |

**goupdate**: Download a single ~8MB binary. No runtime dependencies—just ensure your package managers (npm, go, etc.) are installed.

**Renovate**: Requires Node.js and npm ecosystem. Self-hosted instances need Docker or a Node.js environment with 300MB+ of dependencies.

**Dependabot**: Runs only on GitHub infrastructure. Cannot be self-hosted or used outside GitHub.

**Renovate's AGPL license** has copyleft implications—modifications must be open-sourced, and SaaS offerings may trigger disclosure requirements.

---

### Git Repository Access

Both Dependabot and Renovate require **write access** to your repositories:

| Requirement | goupdate | Renovate | Dependabot |
|-------------|:--------:|:--------:|:----------:|
| Needs repo write access | ❌ No | ✅ Yes | ✅ Yes |
| Creates branches | ❌ No* | ✅ Yes | ✅ Yes |
| Trust relationship required | ❌ No | ✅ Self-hosted | ✅ GitHub |

> "All self-hosted Renovate instances must operate under a trust relationship with the developers of the monitored repositories." — [Renovate Security Docs](https://docs.renovatebot.com/security-and-permissions/)

*goupdate modifies local files only. Use standard git workflows for branching/PRs.

**goupdate**: Only queries public package registries. Never needs access to your Git server or repository contents.

---

## What goupdate Is Missing

These features exist in Dependabot and/or Renovate but are not yet built into goupdate:

### Built-in Features Not Yet Available

| Feature | Available In | Notes |
|---------|--------------|-------|
| **Security vulnerability alerts** | Both | CVE database integration for automated security updates |
| **Merge confidence scores** | Both | Risk assessment based on community adoption |
| **Dependency dashboard** | Renovate | 🔜 Planned via OpenTelemetry integration for Grafana/custom dashboards |
| **Changelog extraction** | Both | Automatic release notes in PR descriptions |

### Achievable via CI Automation

The following features are not built-in but can be achieved by running goupdate in CI pipelines:

| Feature | How to Achieve |
|---------|----------------|
| **Automatic PR creation** | Schedule goupdate in GitHub Actions/GitLab CI, use `gh pr create` or equivalent |
| **Scheduled updates** | Use cron triggers in CI (e.g., `schedule: - cron: '0 6 * * 1'` for weekly) |
| **PR grouping by scope** | Create separate branches/PRs for `--major`, `--minor`, `--patch` |
| **Auto-merge to staging** | Merge PRs automatically if CI tests pass |
| **Release automation** | Use reusable actions for GoReleaser + Docker builds |

goupdate includes **complete reusable GitHub Actions** and **platform-portable examples** for GitLab CI, Bitbucket Pipelines, and Azure Pipelines.

**See [docs/releasing.md](releasing.md)** for:
- Step-by-step CI/CD setup guide
- Reusable GitHub Actions (`_goreleaser`, `_gh-release`, `_dockerhub`, etc.)
- GitLab CI, Bitbucket Pipelines, and Azure Pipelines examples
- Key environment variables for cross-platform compatibility

Example GitHub Actions workflow for scheduled updates with auto-PR:

```yaml
name: Weekly Dependency Updates
on:
  schedule:
    - cron: '0 6 * * 1'  # Every Monday at 6 AM
jobs:
  update-patch:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Apply patch updates
        run: goupdate update --patch --yes
      - name: Create PR
        run: |
          git checkout -b deps/patch-updates
          git add -A && git commit -m "chore(deps): patch updates"
          gh pr create --title "Patch dependency updates" --body "Auto-generated"
```

Example release workflow (works with any CI platform):

```bash
# Stable release with GoReleaser
export GORELEASER_MAKE_LATEST=true
goreleaser release --clean

# Prerelease with auto-generated tag
TODAY=$(date -u +%Y%m%d)
TAG="_stage-${TODAY}-rc1"
git tag -a "$TAG" -m "Release Candidate $TAG"
git push origin "$TAG"
goreleaser release --clean
```

---

## Sources

### Official Documentation
- [Dependabot Supported Ecosystems](https://docs.github.com/en/code-security/dependabot/ecosystems-supported-by-dependabot/supported-ecosystems-and-repositories)
- [Dependabot Options Reference](https://docs.github.com/en/code-security/dependabot/working-with-dependabot/dependabot-options-reference)
- [Renovate Bot Comparison](https://docs.renovatebot.com/bot-comparison/)
- [Renovate Package Managers](https://docs.renovatebot.com/modules/manager/)
- [Renovate Known Limitations](https://docs.renovatebot.com/known-limitations/)
- [Renovate Local Platform](https://docs.renovatebot.com/modules/platform/local/)
- [Renovate Security and Permissions](https://docs.renovatebot.com/security-and-permissions/)
- [Renovate Self-Hosted Configuration](https://docs.renovatebot.com/self-hosted-configuration/)

### Community & Analysis
- [The Limitations of Dependabot](https://www.infield.ai/post/the-limitations-of-dependabot) — Infield AI
- [Why Every Developer Thinks Dependabot Sucks](https://blog.shivamsaraswat.com/dependabot-sucks/) — Shivam Saraswat
- [12 Tips to Self-host Renovate Bot](https://jerrynsh.com/12-tips-to-self-host-renovate-bot/) — Jerry Ng
- [Adding a Package Manager to Renovate](https://github.com/renovatebot/renovate/blob/main/docs/development/adding-a-package-manager.md) — Renovate GitHub
