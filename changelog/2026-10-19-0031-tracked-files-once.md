---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.19.0031

- Perf: the runner lists tracked files once and hands the list to every check through `FITNESS_TRACKED_FILES`.
- Perf: each check process asks git once for its tracked files and once per file for `linguist-generated` marks.
- Perf: a check run alone still lists its own tracked files.
- Perf: a file walk tests each name against the wanted extensions before reading the file's details from disk.
- Perf: the runner reads the staged files once per run.
