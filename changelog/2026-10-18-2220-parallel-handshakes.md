---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.18.2220

- Perf: the runner runs every check's `--describe` handshake side by side, in a pool of one slot per core.
- Perf: checks start longest timeout budget first, while the results table keeps catalog order.
- Perf: a full run on this repo takes 1.8 s instead of 2.1 s warm, and 1.8 s instead of 2.4 s after an install.
