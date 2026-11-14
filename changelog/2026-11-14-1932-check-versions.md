### 2026.11.14.1932

- Feat: `.fitnessrc.json` takes `checkVersions`, which pins each check a repo installs to a major such as `v1` or a release.
- Feat: `fitness check-versions` prints each check with its pin, every released check at `v1` when the key is unset.
- Fix: `fitness check-versions` fails on a malformed pin or name, or a pin on `local-branch`, saying how to fix it.
- Test: requirement 0163 runs `fitness check-versions` on each kind of config, and 0102 sees the map schema of `checkVersions`.
