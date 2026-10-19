---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.19.0031

- Perf: a run of several checks lists tracked files once and shares them through `FITNESS_TRACKED_FILES`, even removing that file on interrupt.
- Perf: a check uses the shared list only while the git index is unchanged, and otherwise lists its own files.
- Perf: each check process lists a root once and asks `check-attr` about each file once, never holding a lock across a git call.
- Perf: a file walk tests each name against the wanted extensions before reading the file's details from disk.
- Perf: the runner reads the staged files once per run.
