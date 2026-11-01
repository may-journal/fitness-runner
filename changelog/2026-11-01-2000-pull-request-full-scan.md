### 2026.11.01.2000

- Feat: a pull request run checks its changed files first, and a failure there ends the run.
- Feat: once the changes pass, the same run scans every file and fails while the repo does.
- Feat: a failed full scan lists the files the pull request does not change apart from the changed ones.
- Feat: `scopedPullRequests` in `.fitnessrc.json` skips that full scan for a huge repo, and a push to main still scans every file.
- Feat: the git hooks keep their scope inside a pull request job too, so a red main never blocks a commit.
