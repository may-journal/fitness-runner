---
relatedConfigurations: ['../package.json']
---

# Scripts

Repo maintenance scripts. Each folder has its own README.

| Folder                                                          | Purpose                                                                      |
| --------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| [ensure-changelog-timestamp](./ensure-changelog-timestamp/)     | Pre-commit version and changelog timestamp bump                              |
| [list-publishable-packages](./list-publishable-packages/)       | Discover `@mayjournal/*` workspaces to publish                               |
| [nvm-use](./nvm-use/)                                           | Load nvm and `nvm use` for `.nvmrc`                                          |
| [provision-npm-packages](./provision-npm-packages/)             | Optional local helper to seed npm or set trusted publishing (not used in CI) |
| [setup-npm-trusted-publishers](./setup-npm-trusted-publishers/) | Deprecated alias for provision                                               |
