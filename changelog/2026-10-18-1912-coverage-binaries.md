---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.18.1912

- Perf: `go-test-coverage` builds each package's test binary once and runs every audit measurement from it.
- Perf: packages share a pool sized like the runner's own check pool, and each package's runs stay in sequence.
- Perf: on this repo the check takes 12 s instead of 257 s, with the same findings.
- Test: requirement 0028 now proves unstable coverage is reported and twelve one-second tests finish within fifteen seconds.
