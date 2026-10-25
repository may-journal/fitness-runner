---
name: changelog-bullets
description: Fix a failing changelog-bullets check on a changelog entry file. Use when an entry has too few or too many bullets, a long bullet, or a bullet with no type.
---

# Tighten changelog bullets

Each entry file's `###` section holds 3 to 5 bullets, each under 365 characters and each starting with a semantic type. [The check README](../../README.md) lists the accepted types.

## Read the failure

- `changelog/2026-05-01-0000-readme.md section "2026.05.01.0000" has 1 bullets; keep entries to 3-5 bullets`: write 3 to 5 bullets covering the whole PR. Merge small changes into one bullet.
- `changelog/2026-05-01-0000-readme.md:7: bullet is 1230 characters; keep each under 365`: shorten the bullet to the user-visible change. The diff holds the detail.
- `changelog/2026-05-01-0000-readme.md:8: bullet must start with a semantic type (Feat:, Fix:, ... or feat(scope):, fix(scope):, ...)`: start the bullet with a type such as `Fix:` or `feat(release):`.

## Check it first

Run `fitness changelog-bullets` from the repo root. Exit 0 means every entry passes.
