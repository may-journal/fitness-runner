---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.18.1912

- Perf: `go-test-coverage` builds each package's test binary once and runs every audit measurement from it.
- Perf: the audit runs those measurements on a pool sized like the runner's own check pool.
- Perf: on this repo the check takes 5 s instead of 257 s, with the same findings.
