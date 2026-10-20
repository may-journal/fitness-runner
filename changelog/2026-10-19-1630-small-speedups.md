---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.19.1630

- Perf: `architecture-components` indexes owners by folder once, so it takes 0.05 s on fitness-runner, down from 0.10 s.
- Perf: `jscpd` pre-sizes its window counts and fills one map, so it takes 0.05 s on fitness-runner, down from 0.08 s.
- Perf: a check started beside others caps its Go threads at its share of the cores and any CPU quota.
- Perf: the commit-msg hook runs its checks in one pass and names every failure, as 0009.10 shows; overrides still get `--message`.
- Perf: CI splits the requirement tests over two jobs per platform, about 20 s each on a Mac instead of 36 s.
