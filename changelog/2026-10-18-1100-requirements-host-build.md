---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.18.1100

- Perf: `make requirements` builds the candidate for this host's platform only, so it uses about a quarter of the CPU and disk.
- Feat: `fitness-release bundle-hashes` and `assemble` take `--host` to expect this platform's outputs only.
- Feat: a `fitness-release` command warns that a flag it does not read is not handled, instead of ignoring it silently.
- Fix: `assemble` removes the archives, installers and verifiers an earlier build left in `out`.
- Docs: the development guide and the `fitness-release` README describe the host-only build.
