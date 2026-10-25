---
name: no-plans-dir
description: Fix a failing no-plans-dir check on a file under docs/plans. Use when a commit adds a plan file.
---

# Put the plan in an issue

Plans live as `Plan`-labeled GitHub Issues, so no file may sit under `docs/plans/`. [The check README](../../README.md) states the rule.

## Read the failure

- ``docs/plans/foo.md: plans live as `Plan`-labeled GitHub Issues, not files — open a Plan Issue instead of adding to docs/plans/``: open a Plan issue with the content, then delete the file.

Run `git rm docs/plans/foo.md` for each file the check names. Do not move the file to another folder to hide it. Link the new issue from the PR.

## Check it first

Run `fitness no-plans-dir` from the repo root. Exit 0 means `docs/plans/` holds no files.
