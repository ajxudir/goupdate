# kpas-api Battle Test Output

**Date:** 2025-12-16 05:30:33 UTC
**Config:** examples/kpas-api/.goupdate.yml

## Test Environment

- Composer packages: laravel/framework, sentry/sentry-laravel, intervention/image, maatwebsite/excel, spatie/laravel-medialibrary
- npm packages: vite, laravel-vite-plugin, @vitejs/plugin-vue, vue

---

## 1. Scan Output

```
⚠️  Development build: this is an unreleased version without a version tag.
   For production use, please install a released version.

Scanned package files in .

RULE      PM   FORMAT  FILE           STATUS  
--------  ---  ------  -------------  --------
composer  php  json    composer.json  🟢 valid
npm       js   json    package.json   🟢 valid

Total entries: 2
Unique files: 2
Rules matched: 2
Valid files: 2
Invalid files: 0
```

---

## 2. List Output

```
⚠️  Development build: this is an unreleased version without a version tag.
   For production use, please install a released version.

RULE      PM   TYPE  CONSTRAINT      VERSION  INSTALLED  STATUS        GROUP  NAME                       
--------  ---  ----  --------------  -------  ---------  ------------  -----  ---------------------------
composer  php  dev   Compatible (^)  10.0     10.5.60    🚫 Ignored           phpunit/phpunit            
composer  php  prod  Compatible (^)  3.0      3.11.5     🟢 LockFound         intervention/image         
composer  php  prod  Compatible (^)  10.0     v10.50.0   🟢 LockFound         laravel/framework          
composer  php  prod  Compatible (^)  3.1      3.1.67     🟢 LockFound         maatwebsite/excel          
composer  php  prod  Compatible (^)  8.2      #N/A       🚫 Ignored           php                        
composer  php  prod  Compatible (^)  4.0      4.20.0     🟢 LockFound         sentry/sentry-laravel      
composer  php  prod  Compatible (^)  11.0     11.17.7    🟢 LockFound         spatie/laravel-medialibrary
npm       js   dev   Compatible (^)  5.0.0    5.2.4      🟢 LockFound  vite   @vitejs/plugin-vue         
npm       js   dev   Compatible (^)  1.0.0    1.3.0      🟢 LockFound  vite   laravel-vite-plugin        
npm       js   dev   Compatible (^)  5.0.0    5.4.21     🟢 LockFound  vite   vite                       
npm       js   prod  Compatible (^)  3.4.0    3.5.25     🟢 LockFound         vue                        

Total packages: 11
```

---

## 3. Outdated Output

```
⚠️  Development build: this is an unreleased version without a version tag.
   For production use, please install a released version.

RULE      PM   TYPE  CONSTRAINT      VERSION  INSTALLED  MAJOR         MINOR         PATCH         STATUS          GROUP  NAME                       
--------  ---  ----  --------------  -------  ---------  ------------  ------------  ------------  --------------  -----  ---------------------------
composer  php  dev   Compatible (^)  10.0     10.5.60    #N/A          #N/A          #N/A          🚫 Ignored             phpunit/phpunit            
composer  php  prod  Compatible (^)  3.0      3.11.5     #N/A          #N/A          #N/A          🟢 UpToDate            intervention/image         
composer  php  prod  Compatible (^)  10.0     v10.50.0   v12.42.0      #N/A          #N/A          🟠 Outdated            laravel/framework          
composer  php  prod  Compatible (^)  3.1      3.1.67     #N/A          #N/A          #N/A          🟢 UpToDate            maatwebsite/excel          
composer  php  prod  Compatible (^)  8.2      #N/A       #N/A          #N/A          #N/A          🚫 Ignored             php                        
composer  php  prod  Compatible (^)  4.0      4.20.0     #N/A          #N/A          #N/A          🟢 UpToDate            sentry/sentry-laravel      
composer  php  prod  Compatible (^)  11.0     11.17.7    #N/A          #N/A          #N/A          🟢 UpToDate            spatie/laravel-medialibrary
npm       js   dev   Compatible (^)  5.0.0    5.2.4      6.0.3         #N/A          #N/A          🟠 Outdated     vite   @vitejs/plugin-vue         
npm       js   dev   Compatible (^)  1.0.0    1.3.0      2.0.1         #N/A          #N/A          🟠 Outdated     vite   laravel-vite-plugin        
npm       js   dev   Compatible (^)  5.0.0    5.4.21     7.3.0         #N/A          #N/A          🟠 Outdated     vite   vite                       
npm       js   prod  Compatible (^)  3.4.0    3.5.25     #N/A          #N/A          #N/A          🟢 UpToDate            vue                        

Total packages: 11
```

---

## 4. Update --minor Output

**Exit Code:** 0

```
⚠️  Development build: this is an unreleased version without a version tag.
   For production use, please install a released version.

RULE      PM   TYPE  CONSTRAINT       VERSION  INSTALLED  TARGET        STATUS          NAME                       
--------  ---  ----  ---------------  -------  ---------  ------------  --------------  ---------------------------
composer  php  dev   Minor (--minor)  10.0     10.5.60    #N/A          🚫 Ignored      phpunit/phpunit            
composer  php  prod  Minor (--minor)  3.0      3.11.5     #N/A          🟢 UpToDate     intervention/image         
composer  php  prod  Minor (--minor)  10.0     v10.50.0   #N/A          🟢 UpToDate     laravel/framework          
composer  php  prod  Minor (--minor)  3.1      3.1.67     #N/A          🟢 UpToDate     maatwebsite/excel          
composer  php  prod  Minor (--minor)  8.2      #N/A       #N/A          🚫 Ignored      php                        
composer  php  prod  Minor (--minor)  4.0      4.20.0     #N/A          🟢 UpToDate     sentry/sentry-laravel      
composer  php  prod  Minor (--minor)  11.0     11.17.7    #N/A          🟢 UpToDate     spatie/laravel-medialibrary
npm       js   dev   Minor (--minor)  5.0.0    5.2.4      #N/A          🟢 UpToDate     @vitejs/plugin-vue         
npm       js   dev   Minor (--minor)  1.0.0    1.3.0      #N/A          🟢 UpToDate     laravel-vite-plugin        
npm       js   dev   Minor (--minor)  5.0.0    5.4.21     #N/A          🟢 UpToDate     vite                       
npm       js   prod  Minor (--minor)  3.4.0    3.5.25     #N/A          🟢 UpToDate     vue                        

Total packages: 11
Summary: 11 up-to-date
         (4 have major updates still available)
```

---

## 5. Update --major Output

**Exit Code:** 2

```
⚠️  Development build: this is an unreleased version without a version tag.
   For production use, please install a released version.


Update Plan
══════════════════════════════════════════════════════════════════════

Will update (--major scope):
  laravel/framework    v10.50.0 → v12.42.0  
  @vitejs/plugin-vue   5.2.4 → 6.0.3  
  laravel-vite-plugin  1.3.0 → 2.0.1  
  vite                 5.4.21 → 7.3.0  

Summary: 4 to update, 7 up-to-date

4 package(s) will be updated. Proceeding (--yes)...

RULE      PM   TYPE  CONSTRAINT       VERSION  INSTALLED  TARGET        STATUS          NAME                       
--------  ---  ----  ---------------  -------  ---------  ------------  --------------  ---------------------------
composer  php  dev   Major (--major)  10.0     10.5.60    #N/A          🚫 Ignored      phpunit/phpunit            
composer  php  prod  Major (--major)  3.0      3.11.5     #N/A          🟢 UpToDate     intervention/image         
composer  php  prod  Major (--major)  10.0     v10.50.0   v12.42.0      ❌ Failed       laravel/framework          
composer  php  prod  Major (--major)  3.1      3.1.67     #N/A          🟢 UpToDate     maatwebsite/excel          
composer  php  prod  Major (--major)  8.2      #N/A       #N/A          🚫 Ignored      php                        
composer  php  prod  Major (--major)  4.0      4.20.0     #N/A          🟢 UpToDate     sentry/sentry-laravel      
composer  php  prod  Major (--major)  11.0     11.17.7    #N/A          🟢 UpToDate     spatie/laravel-medialibrary
npm       js   dev   Major (--major)  6.0.3    6.0.3      6.0.3         🟢 Updated      @vitejs/plugin-vue         
npm       js   dev   Major (--major)  1.0.0    1.3.0      2.0.1         ❌ Failed       laravel-vite-plugin        
npm       js   dev   Major (--major)  7.3.0    7.3.0      7.3.0         🟢 Updated      vite                       
npm       js   prod  Major (--major)  3.4.0    3.5.25     #N/A          🟢 UpToDate     vue                        

Total packages: 11

Update Summary
══════════════════════════════════════════════════════════════════════

Successfully updated:
  @vitejs/plugin-vue   5.2.4 → 6.0.3  
  vite                 5.4.21 → 7.3.0  

Failed updates:
  ❌ laravel/framework    v10.50.0 → v12.42.0
     └─ exit status 2: Loading composer repositories with package information
Package "" listed for update is not locked.
Updating dependencies
Your requirements could not be resolved to an installable set of packages.

  Problem 1
    - Root composer.json requires laravel/framework ^v12.42.0 -> satisfiable by laravel/framework[v12.42.0].
    - laravel/framework v12.42.0 requires laravel/prompts ^0.3.0 -> found laravel/prompts[v0.3.0, ..., v0.3.8] but the package is fixed to v0.1.25 (lock file version) by a partial update and that version does not match. Make sure you list it as an argument for the update command.

Use the option --with-all-dependencies (-W) to allow upgrades, downgrades and removals for packages currently locked to specific versions.
  ❌ laravel-vite-plugin  1.3.0 → 2.0.1
     └─ exit status 1: npm error code ERESOLVE
npm error ERESOLVE unable to resolve dependency tree
npm error
npm error While resolving: kpas-api@undefined
npm error Found: vite@5.4.21
npm error node_modules/vite
npm error   dev vite@"^5.0.0" from the root project
npm error
npm error Could not resolve dependency:
npm error peer vite@"^7.0.0" from laravel-vite-plugin@2.0.1
npm error node_modules/laravel-vite-plugin
npm error   dev laravel-vite-plugin@"^2.0.1" from the root project
npm error
npm error Fix the upstream dependency conflict, or retry
npm error this command with --force or --legacy-peer-deps
npm error to accept an incorrect (and potentially broken) dependency resolution.
npm error
npm error
npm error For a full report see:
npm error /root/.npm/_logs/2025-12-16T05_29_58_789Z-eresolve-report.txt
npm error A complete log of this run can be found in: /root/.npm/_logs/2025-12-16T05_29_58_789Z-debug-0.log

Summary: 2 updated, 7 up-to-date, 2 failed

❌ laravel/framework (php/composer): exit status 2: Loading composer repositories with package information
Package "" listed for update is not locked.
Updating dependencies
Your requirements could not be resolved to an installable set of packages.

  Problem 1
    - Root composer.json requires laravel/framework ^v12.42.0 -> satisfiable by laravel/framework[v12.42.0].
    - laravel/framework v12.42.0 requires laravel/prompts ^0.3.0 -> found laravel/prompts[v0.3.0, ..., v0.3.8] but the package is fixed to v0.1.25 (lock file version) by a partial update and that version does not match. Make sure you list it as an argument for the update command.

Use the option --with-all-dependencies (-W) to allow upgrades, downgrades and removals for packages currently locked to specific versions.
❌ laravel-vite-plugin (js/npm): exit status 1: npm error code ERESOLVE
npm error ERESOLVE unable to resolve dependency tree
npm error
npm error While resolving: kpas-api@undefined
npm error Found: vite@5.4.21
npm error node_modules/vite
npm error   dev vite@"^5.0.0" from the root project
npm error
npm error Could not resolve dependency:
npm error peer vite@"^7.0.0" from laravel-vite-plugin@2.0.1
npm error node_modules/laravel-vite-plugin
npm error   dev laravel-vite-plugin@"^2.0.1" from the root project
npm error
npm error Fix the upstream dependency conflict, or retry
npm error this command with --force or --legacy-peer-deps
npm error to accept an incorrect (and potentially broken) dependency resolution.
npm error
npm error
npm error For a full report see:
npm error /root/.npm/_logs/2025-12-16T05_29_58_789Z-eresolve-report.txt
npm error A complete log of this run can be found in: /root/.npm/_logs/2025-12-16T05_29_58_789Z-debug-0.log
Exit code 2: 2 failed
Error: laravel/framework (php/composer): exit status 2: Loading composer repositories with package information
Package "" listed for update is not locked.
Updating dependencies
Your requirements could not be resolved to an installable set of packages.

  Problem 1
    - Root composer.json requires laravel/framework ^v12.42.0 -> satisfiable by laravel/framework[v12.42.0].
    - laravel/framework v12.42.0 requires laravel/prompts ^0.3.0 -> found laravel/prompts[v0.3.0, ..., v0.3.8] but the package is fixed to v0.1.25 (lock file version) by a partial update and that version does not match. Make sure you list it as an argument for the update command.

Use the option --with-all-dependencies (-W) to allow upgrades, downgrades and removals for packages currently locked to specific versions.
laravel-vite-plugin (js/npm): exit status 1: npm error code ERESOLVE
npm error ERESOLVE unable to resolve dependency tree
npm error
npm error While resolving: kpas-api@undefined
npm error Found: vite@5.4.21
npm error node_modules/vite
npm error   dev vite@"^5.0.0" from the root project
npm error
npm error Could not resolve dependency:
npm error peer vite@"^7.0.0" from laravel-vite-plugin@2.0.1
npm error node_modules/laravel-vite-plugin
npm error   dev laravel-vite-plugin@"^2.0.1" from the root project
npm error
npm error Fix the upstream dependency conflict, or retry
npm error this command with --force or --legacy-peer-deps
npm error to accept an incorrect (and potentially broken) dependency resolution.
npm error
npm error
npm error For a full report see:
npm error /root/.npm/_logs/2025-12-16T05_29_58_789Z-eresolve-report.txt
npm error A complete log of this run can be found in: /root/.npm/_logs/2025-12-16T05_29_58_789Z-debug-0.log
Usage:
  goupdate update [file...] [flags]

Flags:
  -c, --config string             Config file path
      --continue-on-fail          Continue processing remaining packages after failures
  -d, --directory string          Directory to scan (default ".")
      --dry-run                   Plan updates without writing files
  -f, --file string               Filter by file path patterns (comma-separated, supports globs)
  -g, --group string              Filter by group (comma-separated)
  -h, --help                      help for update
      --incremental               Force incremental updates (one version step at a time)
      --major                     Force major upgrades (cascade to minor/patch)
      --minor                     Force minor upgrades (cascade to patch)
  -n, --name string               Filter by package name (comma-separated)
      --no-timeout                Disable command timeouts
  -o, --output string             Output format: json, csv, xml (default: table)
  -p, --package-manager string    Filter by package manager (comma-separated) (default "all")
      --patch                     Force patch upgrades
  -r, --rule string               Filter by rule (comma-separated) (default "all")
      --skip-lock                 Skip running lock/install command
      --skip-preflight            Skip pre-flight command validation
      --skip-system-tests         Skip all system tests (preflight and validation)
      --system-test-mode string   Override system test run mode: after_each, after_all, none
  -t, --type string               Filter by type (comma-separated): all,prod,dev (default "all")
  -y, --yes                       Skip confirmation prompt

Global Flags:
      --skip-build-checks   Skip build validation warnings (dev build, arch mismatch)
      --verbose             Enable verbose debug output
```

---

## 6. Updated package.json

```json
{
  "name": "kpas-api",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build"
  },
  "devDependencies": {
    "vite": "^7.3.0",
    "laravel-vite-plugin": "^1.0.0",
    "@vitejs/plugin-vue": "^6.0.3"
  },
  "dependencies": {
    "vue": "^3.4.0"
  }
}
```
