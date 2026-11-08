# 0004 — One requirement doc per user-facing area

Each user-facing area has exactly one requirement doc.

https://github.com/may-journal/fitness-runner/issues/856

## Check

- Reads the name of each doc in `docs/requirements`.
- Fails a doc whose area does not start with `checks-`, `engine-`, `hooks-` or `github-`.
- Fails two docs that name the same area.
- Fix: extend the area's existing doc with new acceptance IDs instead of adding a doc.
