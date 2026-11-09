### 2026.11.09.1300

- Fix: the shared Release Please workflow points a release PR's compare link at the PR's head commit.
- Fix: the compare link no longer answers 404 while the new version's tag does not exist yet.
- Fix: callers that pin `@v1` get the head commit link once the next release moves that tag.
