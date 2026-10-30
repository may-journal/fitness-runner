### 2026.10.19.0031

- Perf: a run of several checks lists tracked files once through `FITNESS_TRACKED_FILES`; an interrupt stops running checks and removes that file.
- Perf: a check uses the list only while the index inode, mtime, and size match, and asks `check-attr` about each file once.
- Perf: a file walk tests names against the wanted extensions before reading from disk, and the runner reads staged files once.
- Fix: prose-budget reads list items through `mdx.ListItem`, so a ticked box is no word, as 0007.10 proves; 0007.1 and 0007.6 fold into 0007.9.
- Test: requirements 0001.15 and 0001.16 prove the single listing and the fallback; 0001.4 folds into 0001.14 with 0001.8, and 0001.3 into 0001.17 with 0001.11.
