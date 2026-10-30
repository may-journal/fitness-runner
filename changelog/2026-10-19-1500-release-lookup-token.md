### 2026.10.19.1500

- Fix: `fitness-install --version latest` signs its release lookup with `GH_TOKEN`, `GITHUB_TOKEN`, or the `gh` login, so hooks stop sharing the anonymous rate limit.
- Fix: the installer records the latest release for 10 minutes, so hooks in that window send no lookup.
- Fix: when the lookup fails, such as on a rate limit, the installer warns once and uses the recorded or newest installed version.
