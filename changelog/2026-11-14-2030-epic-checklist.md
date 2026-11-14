### 2026.11.14.2030

> An Epic body may be one checklist, so the owner sees what is done, with each Plan and merged pull request linked.

- Feat: the Epic template and `epic-structure` accept `- [ ]` and `- [x]` items that hold their Plan's full issue URL and optional pull request URLs.
- Feat: a Plan may live in any repository, `.github` included, and an Epic needs no sub-issue links.
- Fix: issue-check no longer compares an Epic's list with its sub-issues, so a `.github` Plan the token cannot see no longer fails.
- Test: requirements 0082.3 and 0164.1 prove a list needs no sub-issues and a checklist Epic passes.
