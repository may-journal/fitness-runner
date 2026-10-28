### 2026.10.26.2218
- Feat: `fitness strip-front-matter` deletes `fitnessFunctions` and `relatedConfigurations` from any front matter, keeps other keys, and drops an emptied block.
- Feat: an edit that only drops front matter from a shipped entry file passes `changelog-updated`, `pr-check` and `branch-check`.
- Docs: the issue and PR templates, view templates, skills and `docs/checks.md` drop the keys, and say never to add them.
- Docs: 216 of this repo's Markdown files drop their blocks; the two test fixtures keep theirs as test data.
- Test: requirements 0096, 0019 and 0069 prove the fixer on mixed blocks and the shipped entry rule.
