---
relatedConfigurations: ['../../package.json']
---

# provision-npm-packages

Seeds new `@mayjournal/*` packages on the npm registry and configures GitHub Actions trusted publishing for `.github/workflows/publish.yml`.

Trusted publishing is optional locally via `npm run provision:npm -- trust` (interactive npm 2FA). CI does not run this script.

When a contributor adds a new check under `packages/checks/`, this script picks it up automatically via [list-publishable-packages](../list-publishable-packages/README.md).

## When it runs

- Locally only (optional): `npm run provision:npm`
- CI [publish.yml](../../.github/workflows/publish.yml) only runs `npm publish -ws` via OIDC. New packages and trusted publishing are configured on npmjs.com by a maintainer.

## Auth

| Environment | Auth                                                                                             |
| ----------- | ------------------------------------------------------------------------------------------------ |
| CI          | Repo secret `NPM_PROVISION_TOKEN` (granular write token for `@mayjournal`) via `NODE_AUTH_TOKEN` |
| Local       | `npm login` (2FA for trust); repo `.npmrc` is temporarily moved aside so `~/.npmrc` applies      |

Routine publishes use OIDC in the Publish workflow, not this token.

## Usage

```bash
npm run provision:npm              # build, seed, trust (default)
npm run provision:npm -- check       # verify registry + trust only
npm run provision:npm -- seed        # first publish missing packages
npm run provision:npm -- trust       # trusted publishers only
DRY_RUN=1 npm run provision:npm      # print actions without running
```

## Environment

| Variable             | Default                      | Purpose                           |
| -------------------- | ---------------------------- | --------------------------------- |
| `NPM_TRUST_REPO`     | `may-journal/fitness-runner` | GitHub repo for trusted publisher |
| `NPM_TRUST_WORKFLOW` | `publish.yml`                | Workflow filename on npm          |
| `DRY_RUN`            | (unset)                      | Skip mutating commands            |
| `NODE_AUTH_TOKEN`    | (unset)                      | npm token (CI secret)             |

Requires npm 11.14+ (npx installs automatically). The registry trust API requires a `permissions` array since 2026-05-20; this script sends `createPackage` because `npm trust github` still omits it and returns HTTP 400.

When running `trust` locally, each package first runs `npm trust github` (npm’s own CLI + browser/passkey). If that fails (often HTTP 400 without `permissions`), the script retries via the registry API with `createPackage`.

Complete browser sign-in when prompted and wait until npm confirms before returning to the terminal. Enable skip 2FA for 5 minutes on npmjs.com on the first package to bulk-configure the rest.

If the repo `.npmrc` was moved aside, restore it when finished: `mv .npmrc.setup-trust.bak .npmrc` (if that file exists and `.npmrc` is missing).

Website fallback (always works with passkey): package on npmjs.com → Settings → Trusted publishing → GitHub Actions → workflow `publish.yml`, repository `may-journal/fitness-runner`, allow npm publish.
