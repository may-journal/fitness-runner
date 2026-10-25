---
name: prettier
description: Fix a failing prettier check on tracked files. Use when files are reported as unformatted or Prettier is missing.
---

# Format files with Prettier

Every tracked file matches Prettier output under the repo's config, or the shared one. [The check README](../../go/cmd/fitness-check-prettier/README.md) covers config resolution, the Biome variant and the ignore rules.

## Read the failure

- A file path from `prettier --check`: run `fitness prettier --write .` or name one folder, then review and commit the result.
- `Prettier not installed: npm install --save-dev prettier`: install it, plus the plugins the shared config loads.
- `Prettier reported issues. Run: npx prettier . --write`: the output named no file. Read Prettier's own `[error]` lines, such as a plugin it cannot load.
- `.prettierignore exists; Prettier checks every tracked file, so delete it and fix the findings instead`: delete the file and format what it hid.

## Check it first

Run `fitness prettier` from the repo root. Exit 0 means every file is formatted.
