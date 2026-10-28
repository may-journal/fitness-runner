### 2026.10.18.1314

- Fix: the `requirements` check accepts a requirement test changed without its doc when it failed the check at its base.
- Fix: that base is HEAD for a commit, the parent for an amend, and the merge base for a pull request.
- Fix: a new test file, or one that passed at its base, still fails without its doc.
- Fix: the base is judged as a full scan, never scoped to a pull request or commit.
