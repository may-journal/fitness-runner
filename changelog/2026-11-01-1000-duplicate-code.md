### 2026.11.01.1000

- Feat: the `jscpd` check is now `duplicate-code`, named for what its native Go engine finds; no jscpd tool ever ran.
- Feat: a failure reads `ERROR: duplicate code is X% of scanned lines, over the Y% threshold`.
- Feat: `duplicate-code:ignore-start` and `duplicate-code:ignore-end` mark a block to skip; the `jscpd:` markers stay as deprecated aliases.
- Test: requirement 0032.15 proves both marker spellings leave a copied block out.
- Test: requirement 0032.14 folds 0032.6 and 0032.7, binary and untracked files, into one run.
