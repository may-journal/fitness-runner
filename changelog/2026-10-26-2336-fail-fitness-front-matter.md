### 2026.10.26.2336
- Feat: the new `no-fitness-front-matter` check fails each Markdown file whose front matter holds `fitnessFunctions` or `relatedConfigurations`, naming `fitness strip-front-matter`.
- Refactor: the check and `fitness strip-front-matter` share one front matter reader, so both read the same blocks.
- Docs: `docs/checks.md` and the strip-front-matter skill say the check fails leftovers; test fixtures stay out of its scope.
- Test: requirement 0123 proves the check; 0096.5 proves the fixer's output passes it; happyRepo keeps only a `title` block.
