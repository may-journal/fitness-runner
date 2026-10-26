# ADR 0004 check

Keeps [ADR 0004](../../docs/architecture/adr/0004-one-requirement-doc-per-area.md): each user-facing area has exactly one requirement doc.

- Reads the name of each doc in `docs/requirements`.
- Fails a doc whose area does not start with `checks-`, `engine-`, `hooks-` or `github-`.
- Fails two docs that name the same area.
- Fix: extend the area's existing doc with new acceptance IDs instead of adding a doc.
