### 2026.10.26.2005
- Feat: the new `requirements-changed` check fails a commit or pull request that changes product code without a changed test and requirement doc.
- Feat: docs, config, changelog, renames, whitespace edits, replays, and pure deletions still pass.
- Feat: a deletion fails when its path is still named in a requirement doc, with or without a Source line.
- Feat: no commit trailer, flag or setting exempts a change, and the commit-msg hook judges requirements-changed.
- Fix: the entry drops its relatedConfigurations front matter naming .fitnessrc.json.
