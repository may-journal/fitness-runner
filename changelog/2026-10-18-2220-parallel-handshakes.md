---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.18.2220

- Perf: the runner runs every check's `--describe` handshake side by side, in a pool of one slot per core.
- Perf: checks start by the timeout budget they declare, largest first, while the results table keeps catalog order.
- Perf: a budget from `.fitnessrc.json` does not move a check forward, since it says nothing about the check's cost.
- Perf: a handshake that runs past its 2 s budget among the others gets one more try alone.
