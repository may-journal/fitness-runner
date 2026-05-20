---
relatedConfigurations: ['../../package.json']
---

# list-publishable-packages

Discovers npm publishable workspaces in this monorepo: non-private `package.json` files under `packages/` whose `name` starts with `@mayjournal/`.

Used by [provision-npm-packages](../provision-npm-packages/README.md). Order is shared first, then check packages, bundle, then runner.

## Usage

```bash
node scripts/list-publishable-packages/index.mjs          # JSON
node scripts/list-publishable-packages/index.mjs names  # one name per line
node scripts/list-publishable-packages/index.mjs dirs   # one path per line
```

## API

```js
import { listPublishablePackages } from './index.mjs';
// → [{ name: '@mayjournal/fitness-shared', dir: 'packages/shared' }, ...]
```
