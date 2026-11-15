### 2026.11.15.0033

- Fix: the shared hooks set `FITNESS_WITH_CHECKS=pinned`: checks install where `checkVersions` is set, or everywhere once built-ins are gone.
- Fix: a hook in a repo that pins no check reads no check index, so a silent proxy delays its commit no more.
- Test: requirement 0164.6 runs the installer as the hooks do, in a repo with pins and one without.
