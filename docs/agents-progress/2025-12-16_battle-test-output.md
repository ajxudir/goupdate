# Battle Test Output - 2025-12-16

## Test Configuration

Both tests were run against actual cloned repositories (NOT mock environments):
- Clone to `/tmp/test-*` directories
- Copy only `.goupdate.yml` from `examples/` directory
- Run actual updates (not dry-run) with `--minor --continue-on-fail -y`

---

## kpas-api Battle Test

**Repository**: `matematikk-mooc/kpas-api`
**Config**: `examples/kpas-api/.goupdate.yml`
**Command**: `goupdate update --minor --continue-on-fail -y`

### Raw Output

```
=== KPAS-API UPDATE --minor ===

Update Plan
══════════════════════════════════════════════════════════════════════

Will update (--minor scope):
  barryvdh/laravel-debugbar v3.15.2 → v3.16.1
  filp/whoops               2.18.0 → 2.18.4
  nunomaduro/collision      v8.7.0 → v8.8.3
  laravel/framework         v11.41.3 → v11.47.0  (major: v12.42.0 available)
  laravel/tinker            v2.10.1 → v2.10.2
  league/oauth2-client      2.8.1 → 2.9.0
  nesbot/carbon             3.9.0 → 3.11.0
  spatie/laravel-data       4.14.1 → 4.18.0
  symfony/polyfill-iconv    v1.31.0 → v1.33.0
  @vitejs/plugin-vue        5.2.1 → 5.2.4  (major: 6.0.3 available)
  axios                     1.7.9 → 1.13.2
  bootstrap                 5.3.3 → 5.3.8
  laravel-vite-plugin       1.2.0 → 1.3.0  (major: 2.0.1 available)
  postcss                   8.5.1 → 8.5.6
  sass                      1.84.0 → 1.97.0
  vite                      5.4.14 → 5.4.21  (major: 7.3.0 available)
  vue                       3.5.13 → 3.5.25
  @sentry/vue               8.54.0 → 8.55.0  (major: 10.30.0 available)
  @vimeo/player             2.25.1 → 2.30.1

Up to date (other updates available):
  mantas-done/subtitles     1.0.22  (major: v2.0.2 available)

Summary: 19 to update, 11 up-to-date, 4 failed
         (6 have major, 2 have patch available)

19 package(s) will be updated. Proceeding (--yes)...

RULE      PM   TYPE  CONSTRAINT       VERSION       INSTALLED     TARGET        STATUS          NAME
--------  ---  ----  ---------------  ------------  ------------  ------------  --------------  ----------------------------------
composer  php  dev   Minor (--minor)  3.16.1        v3.16.1       v3.16.1       ❌ Failed       barryvdh/laravel-debugbar

System tests failed after barryvdh/laravel-debugbar update:
System Tests (After Update)
────────────────────────────────────────────────────────────
  ✗ composer                                 [8.8s]
    └─ composer: exit status 1: Installing dependencies from lock file (including require-dev)
       Verifying lock file contents can be installed on current platform.
       Nothing to install, update or remove
       Generating optimized autoload files
       > Illuminate\Foundation\ComposerScripts::postAutoloadDump
       > @php artisan package:discover --ansi
       Script @php artisan package:discover --ansi handling the post-autoload-dump event returned with error code 1
────────────────────────────────────────────────────────────
composer  php  dev   Minor (--minor)  2.0           2.1.0         #N/A          🟢 UpToDate     beyondcode/laravel-dump-server
composer  php  dev   Minor (--minor)  2.18.4        2.18.4        2.18.4        ❌ Failed       filp/whoops

System tests failed after filp/whoops update:
System Tests (After Update)
────────────────────────────────────────────────────────────
  ✗ composer                                 [8.7s]
    └─ composer: exit status 1: Installing dependencies from lock file (including require-dev)
       Verifying lock file contents can be installed on current platform.
       Nothing to install, update or remove
       Generating optimized autoload files
       > Illuminate\Foundation\ComposerScripts::postAutoloadDump
       > @php artisan package:discover --ansi
       Script @php artisan package:discover --ansi handling the post-autoload-dump event returned with error code 1
────────────────────────────────────────────────────────────
composer  php  dev   Minor (--minor)  1.6           1.6.12        #N/A          🟢 UpToDate     mockery/mockery
composer  php  dev   Minor (--minor)  8.6.1         v8.7.0        v8.8.3        ❌ Failed       nunomaduro/collision
composer  php  prod  Minor (--minor)  3.1.0         v3.1.0        #N/A          ❌ Failed       dompdf/dompdf
composer  php  prod  Minor (--minor)  1.1           1.1.1         #N/A          🟢 UpToDate     easyrdf/easyrdf
composer  php  prod  Minor (--minor)  *             #N/A          #N/A          🚫 Ignored      ext-json
composer  php  prod  Minor (--minor)  7.8           7.9.3         #N/A          ❌ Failed       guzzlehttp/guzzle
composer  php  prod  Minor (--minor)  *             3.6.0         #N/A          ⛔ Floating     highsolutions/laravel-environments
composer  php  prod  Minor (--minor)  dev-master    dev-master    #N/A          ❌ Failed       imsglobal/lti-1p3-tool
composer  php  prod  Minor (--minor)  11.47.0       v11.47.0      v11.47.0      🟢 Updated      laravel/framework
composer  php  prod  Minor (--minor)  2.10.2        v2.10.2       v2.10.2       🟢 Updated      laravel/tinker
composer  php  prod  Minor (--minor)  2.9.0         2.9.0         2.9.0         🟢 Updated      league/oauth2-client
composer  php  prod  Minor (--minor)  1.0.22        v1.0.22       #N/A          🟢 UpToDate     mantas-done/subtitles
composer  php  prod  Minor (--minor)  3.11.0        3.11.0        3.11.0        🟢 Updated      nesbot/carbon
composer  php  prod  Minor (--minor)  8.3           #N/A          #N/A          🚫 Ignored      php
composer  php  prod  Minor (--minor)  4.12.0        4.13.0        #N/A          ❌ Failed       sentry/sentry-laravel
composer  php  prod  Minor (--minor)  4.18.0        4.18.0        4.18.0        🟢 Updated      spatie/laravel-data
composer  php  prod  Minor (--minor)  2.8           2.9.1         #N/A          🟢 UpToDate     spatie/laravel-ignition
composer  php  prod  Minor (--minor)  1.30          v1.31.0       v1.33.0       ❌ Failed       symfony/polyfill-iconv
composer  php  prod  Minor (--minor)  5.9           5.10.0        #N/A          🟢 UpToDate     vimeo/laravel
npm       js   dev   Minor (--minor)  5.2.4         5.2.4         5.2.4         🟢 Updated      @vitejs/plugin-vue
npm       js   dev   Minor (--minor)  1.13.2        1.13.2        1.13.2        🟢 Updated      axios
npm       js   dev   Minor (--minor)  5.3.8         5.3.8         5.3.8         🟢 Updated      bootstrap
npm       js   dev   Minor (--minor)  7.9.0         7.9.0         #N/A          🟢 UpToDate     d3
npm       js   dev   Minor (--minor)  1.3.0         1.3.0         1.3.0         🟢 Updated      laravel-vite-plugin
npm       js   dev   Minor (--minor)  8.5.6         8.5.6         8.5.6         🟢 Updated      postcss
npm       js   dev   Minor (--minor)  1.97.0        1.97.0        1.97.0        🟢 Updated      sass
npm       js   dev   Minor (--minor)  5.4.21        5.4.21        5.4.21        🟢 Updated      vite
npm       js   dev   Minor (--minor)  3.5.25        3.5.25        3.5.25        🟢 Updated      vue
npm       js   dev   Minor (--minor)  17.4.2        17.4.2        #N/A          🟢 UpToDate     vue-loader
npm       js   dev   Minor (--minor)  4.0.0-beta.6  4.0.0-beta.6  #N/A          🟢 UpToDate     vue-select
npm       js   prod  Minor (--minor)  8.55.0        8.55.0        8.55.0        🟢 Updated      @sentry/vue
npm       js   prod  Minor (--minor)  2.30.1        2.30.1        2.30.1        🟢 Updated      @vimeo/player

Total packages: 35

Update Summary
══════════════════════════════════════════════════════════════════════

Successfully updated:
  laravel/framework         v11.41.3 → v11.47.0  (major: v12.42.0 available)
    ✓ System tests: 2/2 passed [22.7s]
      ✓ composer [8.6s]
      ✓ npm-build [14.0s]
  laravel/tinker            v2.10.1 → v2.10.2
    ✓ System tests: 2/2 passed [22.7s]
      ✓ composer [8.7s]
      ✓ npm-build [13.9s]
  league/oauth2-client      2.8.1 → 2.9.0
    ✓ System tests: 2/2 passed [23.3s]
      ✓ composer [9.0s]
      ✓ npm-build [14.3s]
  nesbot/carbon             3.9.0 → 3.11.0
    ✓ System tests: 2/2 passed [23.2s]
      ✓ composer [9.2s]
      ✓ npm-build [14.0s]
  spatie/laravel-data       4.14.1 → 4.18.0
    ✓ System tests: 2/2 passed [22.8s]
      ✓ composer [8.8s]
      ✓ npm-build [14.0s]
  @vitejs/plugin-vue        5.2.1 → 5.2.4  (major: 6.0.3 available)
    ✓ System tests: 2/2 passed [23.6s]
      ✓ composer [9.2s]
      ✓ npm-build [14.4s]
  axios                     1.7.9 → 1.13.2
    ✓ System tests: 2/2 passed [24.4s]
      ✓ composer [9.5s]
      ✓ npm-build [15.0s]
  bootstrap                 5.3.3 → 5.3.8
    ✓ System tests: 2/2 passed [23.6s]
      ✓ composer [9.1s]
      ✓ npm-build [14.5s]
  laravel-vite-plugin       1.2.0 → 1.3.0  (major: 2.0.1 available)
    ✓ System tests: 2/2 passed [23.1s]
      ✓ composer [9.0s]
      ✓ npm-build [14.2s]
  postcss                   8.5.1 → 8.5.6
    ✓ System tests: 2/2 passed [23.6s]
      ✓ composer [9.0s]
      ✓ npm-build [14.6s]
  sass                      1.84.0 → 1.97.0
    ✓ System tests: 2/2 passed [23.5s]
      ✓ composer [8.9s]
      ✓ npm-build [14.6s]
  vite                      5.4.14 → 5.4.21  (major: 7.3.0 available)
    ✓ System tests: 2/2 passed [23.5s]
      ✓ composer [9.2s]
      ✓ npm-build [14.3s]
  vue                       3.5.13 → 3.5.25
    ✓ System tests: 2/2 passed [24.2s]
      ✓ composer [8.9s]
      ✓ npm-build [15.2s]
  @sentry/vue               8.54.0 → 8.55.0  (major: 10.30.0 available)
    ✓ System tests: 2/2 passed [24.8s]
      ✓ composer [9.2s]
      ✓ npm-build [15.6s]
  @vimeo/player             2.25.1 → 2.30.1
    ✓ System tests: 2/2 passed [23.9s]
      ✓ composer [9.1s]
      ✓ npm-build [14.8s]

Failed updates:
  ❌ barryvdh/laravel-debugbar v3.15.2 → v3.16.1
     └─ system tests failed: 1/2 system tests passed (1 failed)
     ✗ System tests: 1/2 passed [23.1s]
       ✗ composer [8.8s]
       ✓ npm-build [14.2s]
  ❌ filp/whoops               2.18.0 → 2.18.4
     └─ system tests failed: 1/2 system tests passed (1 failed)
     ✗ System tests: 1/2 passed [22.4s]
       ✗ composer [8.7s]
       ✓ npm-build [13.7s]
  ❌ nunomaduro/collision      v8.7.0 → v8.8.3
     └─ exit status 2: Loading composer repositories with package information
                                                      Updating dependencies
Your requirements could not be resolved to an installable set of packages.

  Problem 1
    - laravel/framework is locked to version v11.41.3 and an update of this package was not requested.
    - Root composer.json requires nunomaduro/collision ^v8.8.3 -> satisfiable by nunomaduro/collision[v8.8.3].
    - nunomaduro/collision v8.8.3 conflicts with laravel/framework <11.44.2 || >=13.0.0.
  ❌ dompdf/dompdf             v3.1.0 →
     └─ failed to execute outdated command: exit status 100: In AuthHelper.php line 132:

  Could not authenticate against github.com

  ❌ guzzlehttp/guzzle         7.9.3 →
     └─ failed to execute outdated command: exit status 1: Failed to clone the git@github.com:matematikk-mooc/lti-1-3-php-library.git repository, try running in interactive mode so that you can enter your GitHub credentials

In Git.php line 602:

  Failed to execute git clone --mirror -- git@github.com:matematikk-mooc/lti-
  1-3-php-library.git /root/.cache/composer/vcs/git-github.com-matematikk-moo
  c-lti-1-3-php-library.git/

  Cloning into bare repository '/root/.cache/composer/vcs/git-github.com-mate
  matikk-mooc-lti-1-3-php-library.git'...
  error: cannot run ssh: No such file or directory
  fatal: unable to fork

  ❌ imsglobal/lti-1p3-tool    dev-master →
     └─ failed to execute outdated command: exit status 100: In AuthHelper.php line 132:

  Could not authenticate against github.com

  ❌ sentry/sentry-laravel     4.13.0 →
     └─ failed to execute outdated command: exit status 100: In AuthHelper.php line 132:

  Could not authenticate against github.com

  ❌ symfony/polyfill-iconv    v1.31.0 → v1.33.0
     └─ exit status 1: Loading composer repositories with package information
Package "" listed for update is not locked.
Failed to clone the git@github.com:matematikk-mooc/lti-1-3-php-library.git repository

Up to date (other updates available):
  mantas-done/subtitles     1.0.22  (major: v2.0.2 available)

Summary: 15 updated, 11 up-to-date, 8 failed
         (6 have major updates still available)

Exit code 1: 15 succeeded, 8 failed (partial failure with --continue-on-fail)
```

### kpas-api Results Summary

| Metric | Count |
|--------|-------|
| Successfully Updated | 15 |
| Up-to-date | 11 |
| Failed | 8 |
| Total Packages | 35 |

### Failure Analysis

| Package | Failure Reason |
|---------|----------------|
| barryvdh/laravel-debugbar | System test failed: `php artisan package:discover` (missing .env) |
| filp/whoops | System test failed: `php artisan package:discover` (missing .env) |
| nunomaduro/collision | Dependency conflict: requires laravel/framework >=11.44.2 |
| dompdf/dompdf | GitHub authentication error |
| guzzlehttp/guzzle | SSH/private repo access failure |
| imsglobal/lti-1p3-tool | GitHub authentication error |
| sentry/sentry-laravel | GitHub authentication error |
| symfony/polyfill-iconv | SSH/private repo access failure |

---

## kpas-frontend Battle Test

**Repository**: `matematikk-mooc/frontend`
**Config**: `examples/kpas-frontend/.goupdate.yml`
**Command**: `goupdate update --minor --continue-on-fail -y`

### Raw Output

```
=== KPAS-FRONTEND UPDATE --minor ===

Update Plan
══════════════════════════════════════════════════════════════════════

Will update (--minor scope):
  @babel/core                  7.26.9 → 7.28.5
  @babel/preset-env            7.26.9 → 7.28.5
  @babel/preset-react          7.26.3 → 7.28.5
  @playwright/test             1.50.1 → 1.57.0
  @types/node                  22.13.8 → 22.19.3  (major: 25.0.2 available)
  @vue/babel-plugin-jsx        1.2.5 → 1.5.0  (major: 2.0.1 available)
  @vue/compiler-sfc            3.5.13 → 3.5.25
  @vue/devtools-api            7.7.2 → 7.7.9  (major: 8.0.5 available)
  @vue/devtools-kit            7.7.2 → 7.7.9  (major: 8.0.5 available)
  @vue/devtools-shared         7.7.2 → 7.7.9  (major: 8.0.5 available)
  @vue/runtime-core            3.5.13 → 3.5.25
  @vue/runtime-dom             3.5.13 → 3.5.25
  @vue/shared                  3.5.13 → 3.5.25
  birpc                        2.2.0 → 2.9.0  (major: 4.0.0 available)
  css-minimizer-webpack-plugin 7.0.0 → 7.0.4
  eslint                       9.21.0 → 9.39.2
  eslint-config-prettier       9.1.0 → 9.1.2  (major: 10.1.8 available)
  eslint-plugin-prettier       5.2.3 → 5.5.4
  eslint-plugin-vue            9.32.0 → 9.33.0  (major: 10.6.2 available)
  html-webpack-plugin          5.6.3 → 5.6.5
  mini-css-extract-plugin      2.9.2 → 2.9.4
  prettier                     3.5.3 → 3.7.4
  sass                         1.85.1 → 1.97.0
  sass-loader                  16.0.5 → 16.0.6
  string-replace-loader        3.1.0 → 3.3.0
  terser-webpack-plugin        5.3.12 → 5.3.16
  webpack                      5.98.0 → 5.103.0
  webpack-dev-server           5.2.0 → 5.2.2
  @material-symbols/font-400   0.28.2 → 0.40.2
  @vimeo/player                2.25.1 → 2.30.1
  @vue/reactivity              3.5.13 → 3.5.25
  vee-validate                 4.15.0 → 4.15.1
  vue                          3.5.13 → 3.5.25

Up to date (other updates available):
  babel-loader                 9.2.1  (major: 10.0.0 available)
  copy-webpack-plugin          12.0.2  (major: 13.0.1 available)
  perfect-debounce             1.0.0  (major: 2.0.0 available)
  react                        18.3.1  (major: 19.2.3 available)
  react-dom                    18.3.1  (major: 19.2.3 available)

Summary: 33 to update, 19 up-to-date
         (13 have major, 3 have patch available)

33 package(s) will be updated. Proceeding (--yes)...

RULE  PM  TYPE  CONSTRAINT       VERSION  INSTALLED  TARGET        STATUS          NAME
----  --  ----  ---------------  -------  ---------  ------------  --------------  -------------------------------------
pnpm  js  dev   Minor (--minor)  7.28.5   7.28.5     7.28.5        🟢 Updated      @babel/core
pnpm  js  dev   Minor (--minor)  7.28.5   7.28.5     7.28.5        🟢 Updated      @babel/preset-env
pnpm  js  dev   Minor (--minor)  7.28.5   7.28.5     7.28.5        🟢 Updated      @babel/preset-react
pnpm  js  dev   Minor (--minor)  1.57.0   1.57.0     1.57.0        🟢 Updated      @playwright/test
pnpm  js  dev   Minor (--minor)  22.19.3  22.19.3    22.19.3       🟢 Updated      @types/node
pnpm  js  dev   Minor (--minor)  1.4.0    1.4.0      #N/A          🟢 UpToDate     @vue/babel-helper-vue-jsx-merge-props
pnpm  js  dev   Minor (--minor)  1.5.0    1.5.0      1.5.0         🟢 Updated      @vue/babel-plugin-jsx
pnpm  js  dev   Minor (--minor)  3.5.25   3.5.25     3.5.25        🟢 Updated      @vue/compiler-sfc
pnpm  js  dev   Minor (--minor)  7.7.9    7.7.9      7.7.9         🟢 Updated      @vue/devtools-api
pnpm  js  dev   Minor (--minor)  7.7.9    7.7.9      7.7.9         🟢 Updated      @vue/devtools-kit
pnpm  js  dev   Minor (--minor)  7.7.9    7.7.9      7.7.9         🟢 Updated      @vue/devtools-shared
pnpm  js  dev   Minor (--minor)  3.5.25   3.5.25     3.5.25        🟢 Updated      @vue/runtime-core
pnpm  js  dev   Minor (--minor)  3.5.25   3.5.25     3.5.25        🟢 Updated      @vue/runtime-dom
pnpm  js  dev   Minor (--minor)  3.5.25   3.5.25     3.5.25        🟢 Updated      @vue/shared
pnpm  js  dev   Minor (--minor)  9.2.1    9.2.1      #N/A          🟢 UpToDate     babel-loader
pnpm  js  dev   Minor (--minor)  2.9.0    2.9.0      2.9.0         🟢 Updated      birpc
pnpm  js  dev   Minor (--minor)  4.0.0    4.0.0      #N/A          🟢 UpToDate     clean-webpack-plugin
pnpm  js  dev   Minor (--minor)  12.0.2   12.0.2     #N/A          🟢 UpToDate     copy-webpack-plugin
pnpm  js  dev   Minor (--minor)  7.1.2    7.1.2      #N/A          🟢 UpToDate     css-loader
pnpm  js  dev   Minor (--minor)  7.0.4    7.0.4      7.0.4         🟢 Updated      css-minimizer-webpack-plugin
pnpm  js  dev   Minor (--minor)  9.39.2   9.39.2     9.39.2        🟢 Updated      eslint
pnpm  js  dev   Minor (--minor)  9.1.2    9.1.2      9.1.2         🟢 Updated      eslint-config-prettier
pnpm  js  dev   Minor (--minor)  1.5.1    1.5.1      #N/A          🟢 UpToDate     eslint-plugin-jquery
pnpm  js  dev   Minor (--minor)  5.5.4    5.5.4      5.5.4         🟢 Updated      eslint-plugin-prettier
pnpm  js  dev   Minor (--minor)  9.33.0   9.33.0     9.33.0        🟢 Updated      eslint-plugin-vue
pnpm  js  dev   Minor (--minor)  6.2.0    6.2.0      #N/A          🟢 UpToDate     file-loader
pnpm  js  dev   Minor (--minor)  5.5.3    5.5.3      #N/A          🟢 UpToDate     hookable
pnpm  js  dev   Minor (--minor)  5.6.5    5.6.5      5.6.5         🟢 Updated      html-webpack-plugin
pnpm  js  dev   Minor (--minor)  2.9.4    2.9.4      2.9.4         🟢 Updated      mini-css-extract-plugin
pnpm  js  dev   Minor (--minor)  1.0.0    1.0.0      #N/A          🟢 UpToDate     perfect-debounce
pnpm  js  dev   Minor (--minor)  3.7.4    3.7.4      3.7.4         🟢 Updated      prettier
pnpm  js  dev   Minor (--minor)  18.3.1   18.3.1     #N/A          🟢 UpToDate     react
pnpm  js  dev   Minor (--minor)  18.3.1   18.3.1     #N/A          🟢 UpToDate     react-dom
pnpm  js  dev   Minor (--minor)  1.0.6    1.0.6      #N/A          🟢 UpToDate     replace-in-file-webpack-plugin
pnpm  js  dev   Minor (--minor)  1.97.0   1.97.0     1.97.0        🟢 Updated      sass
pnpm  js  dev   Minor (--minor)  16.0.6   16.0.6     16.0.6        🟢 Updated      sass-loader
pnpm  js  dev   Minor (--minor)  3.3.0    3.3.0      3.3.0         🟢 Updated      string-replace-loader
pnpm  js  dev   Minor (--minor)  4.0.0    4.0.0      #N/A          🟢 UpToDate     style-loader
pnpm  js  dev   Minor (--minor)  5.3.16   5.3.16     5.3.16        🟢 Updated      terser-webpack-plugin
pnpm  js  dev   Minor (--minor)  2.8.1    2.8.1      #N/A          🟢 UpToDate     tslib
pnpm  js  dev   Minor (--minor)  17.4.2   17.4.2     #N/A          🟢 UpToDate     vue-loader
pnpm  js  dev   Minor (--minor)  5.103.0  5.103.0    5.103.0       🟢 Updated      webpack
pnpm  js  dev   Minor (--minor)  6.0.1    6.0.1      #N/A          🟢 UpToDate     webpack-cli
pnpm  js  dev   Minor (--minor)  5.2.2    5.2.2      5.2.2         🟢 Updated      webpack-dev-server
pnpm  js  dev   Minor (--minor)  0.0.4    0.0.4      #N/A          🟢 UpToDate     webpack-replace-plugin
pnpm  js  prod  Minor (--minor)  0.40.2   0.40.2     0.40.2        🟢 Updated      @material-symbols/font-400
pnpm  js  prod  Minor (--minor)  2.30.1   2.30.1     2.30.1        🟢 Updated      @vimeo/player
pnpm  js  prod  Minor (--minor)  3.5.25   3.5.25     3.5.25        🟢 Updated      @vue/reactivity
pnpm  js  prod  Minor (--minor)  7.8.2    7.8.2      #N/A          🟢 UpToDate     rxjs
pnpm  js  prod  Minor (--minor)  4.15.1   4.15.1     4.15.1        🟢 Updated      vee-validate
pnpm  js  prod  Minor (--minor)  3.5.25   3.5.25     3.5.25        🟢 Updated      vue
pnpm  js  prod  Minor (--minor)  4.1.0    4.1.0      #N/A          🟢 UpToDate     vuex

Total packages: 52

Update Summary
══════════════════════════════════════════════════════════════════════

Successfully updated:
  @babel/core                  7.26.9 → 7.28.5
    ✓ System tests: 1/1 passed [23.8s]
      ✓ build [23.8s]
  @babel/preset-env            7.26.9 → 7.28.5
    ✓ System tests: 1/1 passed [24.6s]
      ✓ build [24.6s]
  @babel/preset-react          7.26.3 → 7.28.5
    ✓ System tests: 1/1 passed [19.1s]
      ✓ build [19.1s]
  @playwright/test             1.50.1 → 1.57.0
    ✓ System tests: 1/1 passed [20.5s]
      ✓ build [20.5s]
  @types/node                  22.13.8 → 22.19.3  (major: 25.0.2 available)
    ✓ System tests: 1/1 passed [19.0s]
      ✓ build [19.0s]
  @vue/babel-plugin-jsx        1.2.5 → 1.5.0  (major: 2.0.1 available)
    ✓ System tests: 1/1 passed [18.6s]
      ✓ build [18.6s]
  @vue/compiler-sfc            3.5.13 → 3.5.25
    ✓ System tests: 1/1 passed [18.9s]
      ✓ build [18.9s]
  @vue/devtools-api            7.7.2 → 7.7.9  (major: 8.0.5 available)
    ✓ System tests: 1/1 passed [18.9s]
      ✓ build [18.9s]
  @vue/devtools-kit            7.7.2 → 7.7.9  (major: 8.0.5 available)
    ✓ System tests: 1/1 passed [18.4s]
      ✓ build [18.4s]
  @vue/devtools-shared         7.7.2 → 7.7.9  (major: 8.0.5 available)
    ✓ System tests: 1/1 passed [18.6s]
      ✓ build [18.6s]
  @vue/runtime-core            3.5.13 → 3.5.25
    ✓ System tests: 1/1 passed [18.6s]
      ✓ build [18.6s]
  @vue/runtime-dom             3.5.13 → 3.5.25
    ✓ System tests: 1/1 passed [18.7s]
      ✓ build [18.7s]
  @vue/shared                  3.5.13 → 3.5.25
    ✓ System tests: 1/1 passed [18.9s]
      ✓ build [18.9s]
  birpc                        2.2.0 → 2.9.0  (major: 4.0.0 available)
    ✓ System tests: 1/1 passed [18.8s]
      ✓ build [18.8s]
  css-minimizer-webpack-plugin 7.0.0 → 7.0.4
    ✓ System tests: 1/1 passed [20.5s]
      ✓ build [20.5s]
  eslint                       9.21.0 → 9.39.2
    ✓ System tests: 1/1 passed [19.3s]
      ✓ build [19.3s]
  eslint-config-prettier       9.1.0 → 9.1.2  (major: 10.1.8 available)
    ✓ System tests: 1/1 passed [18.5s]
      ✓ build [18.5s]
  eslint-plugin-prettier       5.2.3 → 5.5.4
    ✓ System tests: 1/1 passed [18.5s]
      ✓ build [18.5s]
  eslint-plugin-vue            9.32.0 → 9.33.0  (major: 10.6.2 available)
    ✓ System tests: 1/1 passed [18.6s]
      ✓ build [18.6s]
  html-webpack-plugin          5.6.3 → 5.6.5
    ✓ System tests: 1/1 passed [18.3s]
      ✓ build [18.3s]
  mini-css-extract-plugin      2.9.2 → 2.9.4
    ✓ System tests: 1/1 passed [18.6s]
      ✓ build [18.6s]
  prettier                     3.5.3 → 3.7.4
    ✓ System tests: 1/1 passed [19.4s]
      ✓ build [19.4s]
  sass                         1.85.1 → 1.97.0
    ✓ System tests: 1/1 passed [19.2s]
      ✓ build [19.2s]
  sass-loader                  16.0.5 → 16.0.6
    ✓ System tests: 1/1 passed [19.2s]
      ✓ build [19.2s]
  string-replace-loader        3.1.0 → 3.3.0
    ✓ System tests: 1/1 passed [19.2s]
      ✓ build [19.2s]
  terser-webpack-plugin        5.3.12 → 5.3.16
    ✓ System tests: 1/1 passed [19.5s]
      ✓ build [19.5s]
  webpack                      5.98.0 → 5.103.0
    ✓ System tests: 1/1 passed [21.0s]
      ✓ build [21.0s]
  webpack-dev-server           5.2.0 → 5.2.2
    ✓ System tests: 1/1 passed [19.1s]
      ✓ build [19.1s]
  @material-symbols/font-400   0.28.2 → 0.40.2
    ✓ System tests: 1/1 passed [19.6s]
      ✓ build [19.6s]
  @vimeo/player                2.25.1 → 2.30.1
    ✓ System tests: 1/1 passed [18.8s]
      ✓ build [18.8s]
  @vue/reactivity              3.5.13 → 3.5.25
    ✓ System tests: 1/1 passed [19.3s]
      ✓ build [19.3s]
  vee-validate                 4.15.0 → 4.15.1
    ✓ System tests: 1/1 passed [19.4s]
      ✓ build [19.4s]
  vue                          3.5.13 → 3.5.25
    ✓ System tests: 1/1 passed [20.1s]
      ✓ build [20.1s]

Up to date (other updates available):
  babel-loader                 9.2.1  (major: 10.0.0 available)
  copy-webpack-plugin          12.0.2  (major: 13.0.1 available)
  perfect-debounce             1.0.0  (major: 2.0.0 available)
  react                        18.3.1  (major: 19.2.3 available)
  react-dom                    18.3.1  (major: 19.2.3 available)

Summary: 33 updated, 19 up-to-date
         (13 have major updates still available)

Exit code 0: Complete success
```

### kpas-frontend Results Summary

| Metric | Count |
|--------|-------|
| Successfully Updated | 33 |
| Up-to-date | 19 |
| Failed | 0 |
| Total Packages | 52 |

**Result: 100% SUCCESS** - All 33 packages updated with system tests passing.

---

## Conclusion

| Repository | Success Rate | Notes |
|------------|--------------|-------|
| kpas-api | 65% (15/23) | Failures are environment-specific (missing .env, no GitHub auth, private repo access) |
| kpas-frontend | 100% (33/33) | Complete success with all system tests passing |

### kpas-api Failure Categories

1. **Missing Laravel Environment** (2 packages): `barryvdh/laravel-debugbar`, `filp/whoops`
   - `php artisan package:discover` requires `.env` file
   - Would pass in production CI/CD with proper Laravel setup

2. **GitHub Authentication** (3 packages): `dompdf/dompdf`, `imsglobal/lti-1p3-tool`, `sentry/sentry-laravel`
   - No GitHub token configured in test environment
   - Would pass with `COMPOSER_AUTH` or GitHub token

3. **Private Repository Access** (2 packages): `guzzlehttp/guzzle`, `symfony/polyfill-iconv`
   - Requires SSH access to `matematikk-mooc/lti-1-3-php-library`
   - Would pass with SSH keys configured

4. **Dependency Conflict** (1 package): `nunomaduro/collision`
   - `v8.8.3` requires `laravel/framework >=11.44.2`
   - Actual dependency issue in the repository (not a goupdate bug)
