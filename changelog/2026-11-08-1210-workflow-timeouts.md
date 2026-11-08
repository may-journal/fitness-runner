### 2026.11.08.1210

- Fix: each workflow job timeout is its slowest observed run plus 25%, rounded up to the next minute.
- Fix: the release approval job keeps a limit above its 9 minute wait for checks.
- Fix: each limit's comment cites the observed maximum, the date and how it was timed.
