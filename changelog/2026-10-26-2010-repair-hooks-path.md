### 2026.10.26.2010

- Fix: `fitness hook` repairs a stale `core.hooksPath` to the hook folder HEAD ships, and prints one line naming the old and new path.
- Fix: a hooks path outside the repo is left as is, and the hook names it.
- Feat: `fitness init` points git at `.fitness-hooks` when the repo ships it, else installs `.githooks` as before.
- Test: requirement 0122 runs each acceptance from a clone with a stale hooks path.
