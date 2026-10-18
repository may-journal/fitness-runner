---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.18.1100

- Perf: `make requirements` builds the candidate for this host's platform only, so it uses about a quarter of the CPU and disk.
- Feat: `fitness-release bundle-hashes` and `assemble` take `--host` to expect this platform's outputs only.
- Docs: the development guide and the `fitness-release` README describe the host-only build.
