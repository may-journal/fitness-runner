### 2026.10.18.2220

- Perf: the runner runs every check's `--describe` handshake side by side, in a pool of one slot per core.
- Perf: checks start by the timeout budget they declare, largest first, with no budget counting as the 5 s default.
- Perf: a `.fitnessrc.json` budget never moves a check forward, and an overrunning compiled check gets one more handshake.
- Test: requirement 0001.13 proves slow checks overlap, the longest budget starts first, and the table keeps catalog order.
- Test: requirement 0001.12 folds into 0001.3, which already covered a killed check showing info.
