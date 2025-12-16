# kpas-frontend Battle Test Output

**Date:** 2025-12-16 05:40:17 UTC
**Config:** examples/kpas-frontend/.goupdate.yml

## Test Environment

- pnpm packages with groups:
  - vue group: vue, vuex
  - react group: react, react-dom
  - webpack group: webpack, webpack-cli, webpack-dev-server
  - babel group: @babel/core, @babel/preset-env, babel-loader
  - playwright group: playwright, @playwright/test

---

## 1. Scan Output

```
⚠️  Development build: this is an unreleased version without a version tag.
   For production use, please install a released version.

Scanned package files in .

RULE  PM  FORMAT  FILE          STATUS  
----  --  ------  ------------  --------
pnpm  js  json    package.json  🟢 valid

Total entries: 1
Unique files: 1
Rules matched: 1
Valid files: 1
Invalid files: 0
```

---

## 2. List Output

```
⚠️  Development build: this is an unreleased version without a version tag.
   For production use, please install a released version.

RULE  PM  TYPE  CONSTRAINT      VERSION  INSTALLED  STATUS        GROUP       NAME              
----  --  ----  --------------  -------  ---------  ------------  ----------  ------------------
pnpm  js  dev   Compatible (^)  7.20.0   7.28.5     🟢 LockFound  babel       @babel/core       
pnpm  js  dev   Compatible (^)  7.20.0   7.28.5     🟢 LockFound  babel       @babel/preset-env 
pnpm  js  dev   Compatible (^)  9.0.0    9.2.1      🟢 LockFound  babel       babel-loader      
pnpm  js  dev   Compatible (^)  1.35.0   1.57.0     🟢 LockFound  playwright  @playwright/test  
pnpm  js  dev   Compatible (^)  1.35.0   1.57.0     🟢 LockFound  playwright  playwright        
pnpm  js  prod  Compatible (^)  18.0.0   18.3.1     🟢 LockFound  react       react             
pnpm  js  prod  Compatible (^)  18.0.0   18.3.1     🟢 LockFound  react       react-dom         
pnpm  js  prod  Compatible (^)  3.3.0    3.5.25     🟢 LockFound  vue         vue               
pnpm  js  prod  Compatible (^)  4.0.0    4.1.0      🟢 LockFound  vue         vuex              
pnpm  js  dev   Compatible (^)  5.80.0   5.103.0    🟢 LockFound  webpack     webpack           
pnpm  js  dev   Compatible (^)  5.0.0    5.1.4      🟢 LockFound  webpack     webpack-cli       
pnpm  js  dev   Compatible (^)  4.10.0   4.15.2     🟢 LockFound  webpack     webpack-dev-server

Total packages: 12
```

---

## 3. Outdated Output

```
⚠️  Development build: this is an unreleased version without a version tag.
   For production use, please install a released version.

RULE  PM  TYPE  CONSTRAINT      VERSION  INSTALLED  MAJOR         MINOR         PATCH         STATUS          GROUP       NAME              
----  --  ----  --------------  -------  ---------  ------------  ------------  ------------  --------------  ----------  ------------------
pnpm  js  dev   Compatible (^)  7.20.0   7.28.5     #N/A          #N/A          #N/A          🟢 UpToDate     babel       @babel/core       
pnpm  js  dev   Compatible (^)  7.20.0   7.28.5     #N/A          #N/A          #N/A          🟢 UpToDate     babel       @babel/preset-env 
pnpm  js  dev   Compatible (^)  9.0.0    9.2.1      10.0.0        #N/A          #N/A          🟠 Outdated     babel       babel-loader      
pnpm  js  dev   Compatible (^)  1.35.0   1.57.0     #N/A          #N/A          #N/A          🟢 UpToDate     playwright  @playwright/test  
pnpm  js  dev   Compatible (^)  1.35.0   1.57.0     #N/A          #N/A          #N/A          🟢 UpToDate     playwright  playwright        
pnpm  js  prod  Compatible (^)  18.0.0   18.3.1     19.2.3        #N/A          #N/A          🟠 Outdated     react       react             
pnpm  js  prod  Compatible (^)  18.0.0   18.3.1     19.2.3        #N/A          #N/A          🟠 Outdated     react       react-dom         
pnpm  js  prod  Compatible (^)  3.3.0    3.5.25     #N/A          #N/A          #N/A          🟢 UpToDate     vue         vue               
pnpm  js  prod  Compatible (^)  4.0.0    4.1.0      #N/A          #N/A          #N/A          🟢 UpToDate     vue         vuex              
pnpm  js  dev   Compatible (^)  5.80.0   5.103.0    #N/A          #N/A          #N/A          🟢 UpToDate     webpack     webpack           
pnpm  js  dev   Compatible (^)  5.0.0    5.1.4      6.0.1         #N/A          #N/A          🟠 Outdated     webpack     webpack-cli       
pnpm  js  dev   Compatible (^)  4.10.0   4.15.2     5.2.2         #N/A          #N/A          🟠 Outdated     webpack     webpack-dev-server

Total packages: 12
```

---

## 4. Update --minor Output

**Exit Code:** 0

```
⚠️  Development build: this is an unreleased version without a version tag.
   For production use, please install a released version.

RULE  PM  TYPE  CONSTRAINT       VERSION  INSTALLED  TARGET        STATUS          NAME              
----  --  ----  ---------------  -------  ---------  ------------  --------------  ------------------
pnpm  js  dev   Minor (--minor)  7.20.0   7.28.5     #N/A          🟢 UpToDate     @babel/core       
pnpm  js  dev   Minor (--minor)  7.20.0   7.28.5     #N/A          🟢 UpToDate     @babel/preset-env 
pnpm  js  dev   Minor (--minor)  1.35.0   1.57.0     #N/A          🟢 UpToDate     @playwright/test  
pnpm  js  dev   Minor (--minor)  9.0.0    9.2.1      #N/A          🟢 UpToDate     babel-loader      
pnpm  js  dev   Minor (--minor)  1.35.0   1.57.0     #N/A          🟢 UpToDate     playwright        
pnpm  js  dev   Minor (--minor)  5.80.0   5.103.0    #N/A          🟢 UpToDate     webpack           
pnpm  js  dev   Minor (--minor)  5.0.0    5.1.4      #N/A          🟢 UpToDate     webpack-cli       
pnpm  js  dev   Minor (--minor)  4.10.0   4.15.2     #N/A          🟢 UpToDate     webpack-dev-server
pnpm  js  prod  Minor (--minor)  18.0.0   18.3.1     #N/A          🟢 UpToDate     react             
pnpm  js  prod  Minor (--minor)  18.0.0   18.3.1     #N/A          🟢 UpToDate     react-dom         
pnpm  js  prod  Minor (--minor)  3.3.0    3.5.25     #N/A          🟢 UpToDate     vue               
pnpm  js  prod  Minor (--minor)  4.0.0    4.1.0      #N/A          🟢 UpToDate     vuex              

Total packages: 12
Summary: 12 up-to-date
         (5 have major updates still available)
```

---

## 5. Update --major Output

**Exit Code:** 0

```
⚠️  Development build: this is an unreleased version without a version tag.
   For production use, please install a released version.


Update Plan
══════════════════════════════════════════════════════════════════════

Will update (--major scope):
  babel-loader         9.2.1 → 10.0.0  
  webpack-cli          5.1.4 → 6.0.1  
  webpack-dev-server   4.15.2 → 5.2.2  
  react                18.3.1 → 19.2.3  
  react-dom            18.3.1 → 19.2.3  

Summary: 5 to update, 7 up-to-date

5 package(s) will be updated. Proceeding (--yes)...

RULE  PM  TYPE  CONSTRAINT       VERSION  INSTALLED  TARGET        STATUS          NAME              
----  --  ----  ---------------  -------  ---------  ------------  --------------  ------------------
pnpm  js  dev   Major (--major)  7.20.0   7.28.5     #N/A          🟢 UpToDate     @babel/core       
pnpm  js  dev   Major (--major)  7.20.0   7.28.5     #N/A          🟢 UpToDate     @babel/preset-env 
pnpm  js  dev   Major (--major)  1.35.0   1.57.0     #N/A          🟢 UpToDate     @playwright/test  
pnpm  js  dev   Major (--major)  10.0.0   10.0.0     10.0.0        🟢 Updated      babel-loader      
pnpm  js  dev   Major (--major)  1.35.0   1.57.0     #N/A          🟢 UpToDate     playwright        
pnpm  js  dev   Major (--major)  5.80.0   5.103.0    #N/A          🟢 UpToDate     webpack           
pnpm  js  dev   Major (--major)  6.0.1    6.0.1      6.0.1         🟢 Updated      webpack-cli       
pnpm  js  dev   Major (--major)  5.2.2    5.2.2      5.2.2         🟢 Updated      webpack-dev-server
pnpm  js  prod  Major (--major)  19.2.3   19.2.3     19.2.3        🟢 Updated      react             
pnpm  js  prod  Major (--major)  19.2.3   19.2.3     19.2.3        🟢 Updated      react-dom         
pnpm  js  prod  Major (--major)  3.3.0    3.5.25     #N/A          🟢 UpToDate     vue               
pnpm  js  prod  Major (--major)  4.0.0    4.1.0      #N/A          🟢 UpToDate     vuex              

Total packages: 12

Update Summary
══════════════════════════════════════════════════════════════════════

Successfully updated:
  babel-loader         9.2.1 → 10.0.0  
  webpack-cli          5.1.4 → 6.0.1  
  webpack-dev-server   4.15.2 → 5.2.2  
  react                18.3.1 → 19.2.3  
  react-dom            18.3.1 → 19.2.3  

Summary: 5 updated, 7 up-to-date
```

---

## 6. Updated package.json

```json
{
  "name": "kpas-frontend",
  "private": true,
  "packageManager": "pnpm@9.0.0",
  "scripts": {
    "build": "webpack build",
    "dev": "webpack serve",
    "test": "playwright test"
  },
  "dependencies": {
    "vue": "^3.3.0",
    "vuex": "^4.0.0",
    "react": "^19.2.3",
    "react-dom": "^19.2.3"
  },
  "devDependencies": {
    "webpack": "^5.80.0",
    "webpack-cli": "^6.0.1",
    "webpack-dev-server": "^5.2.2",
    "babel-loader": "^10.0.0",
    "@babel/core": "^7.20.0",
    "@babel/preset-env": "^7.20.0",
    "playwright": "^1.35.0",
    "@playwright/test": "^1.35.0"
  }
}
```
