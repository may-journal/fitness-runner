---
fitnessFunctions: ['markdown-front-matter']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# markdown-front-matter

Validates that `.md` files with front matter declare `fitnessFunctions` or `relatedConfigurations` and that all paths in those arrays exist within the repo.

## Behavior

- Pass: Every `.md` file either has no front matter, or has front matter with at least one of `fitnessFunctions` or `relatedConfigurations`, and all referenced paths exist.
- Fail: Missing front matter with those keys → `missing front matter with fitnessFunctions or relatedConfigurations`.
- Fail: Empty array → `fitnessFunctions`/`relatedConfigurations` must not be an empty array.
- Fail: Path doesn't exist or escapes repo → `front matter path missing: …` or `front matter path escapes repo: …`.

Paths in `fitnessFunctions`/`relatedConfigurations` may be registered check names (e.g. `eslint`); those are always valid. Paths are resolved relative to the `.md` file’s directory. External URLs and anchors (`http`, `#`, `mailto:`) are skipped.
