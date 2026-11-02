### 2026.11.01.2130
- Fix: the comment checks ask git about generated marks only for files they read, so 100,000 files scan in 46ms, not 293ms.
- Fix: a commit with no comment block skips the changed-lines diff; one with blocks diffs only their files.
- Fix: `requirements` looks for renames only when the base lacks a changed test or doc.
- Fix: the comment checks drop uncovered files before they read the runner's shared views or generated marks.
- Fix: the runner reads built-in checks' metadata itself, so a hook no longer starts every check just for --describe.
