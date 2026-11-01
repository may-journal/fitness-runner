### 2026.11.01.2240

- Feat: a run of several checks reads each file once and hands every check its code view and comment view.
- Fix: a commented-out import, Vitest threshold or JDK pin no longer counts in build-output-untracked, vitest-coverage-full and java-version.
- Fix: commented-out code no longer counts in the test scanners, the architecture views, or the workflow readers.
- Feat: the comment reader now covers Groovy build scripts, XML such as a pom, and settings files such as `.npmrc`.
- Perf: the runner packs every view into one file with an index, so a check reads only the views it needs.
