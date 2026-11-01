### 2026.11.01.2130
- Fix: the comment checks ask git about generated marks only for files they read, so 100,000 files scan in 46ms, not 293ms.
- Fix: a commit with no comment block in scope runs no changed-lines diff.
- Fix: a commit with comment blocks diffs only their files when the base holds them all.
- Fix: `requirements` looks for renames only when the base lacks a changed test or doc.
