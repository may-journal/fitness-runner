---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.18.1912

- Perf: `go-test-coverage` builds each package's test binary once and runs every audit measurement from it.
- Perf: packages and tests that call `t.Parallel()` share a pool sized like the runner's own, while serial tests stay in sequence.
- Perf: on this repo the check takes 13 s instead of 257 s, with the same findings.
- Test: requirement 0028 now proves unstable coverage is reported and the audit overlaps sixteen slow tests' runs exactly where go test would.
