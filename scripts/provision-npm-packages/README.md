---
relatedConfigurations: ['../../package.json']
---

# provision-npm-packages

Seeds new `@mayjournal/*` packages on the npm registry and configures GitHub Actions trusted publishing for `.github/workflows/publish.yml`.

When a contributor adds a new check under `packages/checks/`, this script picks it up automatically via [list-publishable-packages](../list-publishable-packages/README.md).

## When it runs

- GitHub Actions: [provision-npm-packages.yml](../../.github/workflows/provision-npm-packages.yml) on `main` when package manifests change; also at the start of [publish.yml](../../.github/workflows/publish.yml).
- Locally: `npm run provision:npm`

## Auth

| Environment | Auth |
|-------------|------|
| CI | Repo secret `NPM_PROVISION_TOKEN` (granular write token for `@mayjournal`) via `NODE_AUTH_TOKEN` |
| Local | `npm login` (2FA for trust); repo `.npmrc` is temporarily moved aside so `~/.npmrc` applies |

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

| Variable | Default | Purpose |
|----------|---------|---------|
| `NPM_TRUST_REPO` | `may-journal/fitness-runner` | GitHub repo for trusted publisher |
| `NPM_TRUST_WORKFLOW` | `publish.yml` | Workflow filename on npm |
| `DRY_RUN` | (unset) | Skip mutating commands |
| `NODE_AUTH_TOKEN` | (unset) | npm token (CI secret) |

Requires npm 11.10+ for `npm trust` (CI installs 11.14).
