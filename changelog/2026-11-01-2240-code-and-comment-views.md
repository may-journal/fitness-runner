### 2026.11.01.2240

- Feat: a run of several checks reads each file once and packs every file's code view and comment view for its checks.
- Fix: a commented-out import, Vitest threshold or JDK pin no longer counts in build-output-untracked, vitest-coverage-full and java-version.
- Fix: commented-out code no longer counts in the test scanners, the architecture views, or the workflow readers.
- Feat: the comment reader now covers Groovy build scripts, `'''` strings included, XML such as a pom, and settings files such as `.npmrc`.
