---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.19.0031

- Perf: a run of several checks lists tracked files once and shares them through `FITNESS_TRACKED_FILES`, even removing that file on interrupt.
- Perf: a check uses the shared list only while the git index is unchanged, and otherwise lists its own files.
- Perf: each check process lists a root once and asks `check-attr` about each file once, never holding a lock across a git call.
- Perf: a file walk tests names against the wanted extensions before reading from disk, and the runner reads staged files once.
- Test: requirements 0001.15 and 0001.16 prove the single listing and the fallback; 0001.4 folds into 0001.14 with 0001.8, and 0001.3 into 0001.17 with 0001.11.
