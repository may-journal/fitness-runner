### 2026.10.26.2137

- Fix: a commit message that links an Issue makes the runner probe GitHub's API host first.
- Fix: when the host does not answer, `issue-link-once` skips the open PR read and notes the host and proxy.
- Fix: offline, `issue-link-once` still judges the branch commits, so a repeat link is still refused.
- Docs: the check's README and `docs/ci.md` name the open PR read the probe skips.
- Test: requirement 0011.4 commits behind a silent proxy and ends inside ten seconds.
