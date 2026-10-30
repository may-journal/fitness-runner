# 0052 Checks Prettier

## Why

I want every tracked file in Prettier style, so diffs show real changes and never formatting noise.

## Measurement

tracked files Prettier reports as formatted
-
tracked files checked

## Requirements

- 0052.9
    - Given Developer has the repo's own Prettier config, in a file or in `package.json`, in a CI job that colors output
        - When prettier checks the repo
            - Then it passes formatted files and fails naming each file that config would reformat
- 0052.3
    - Given Developer has a repo with no `package.json`
        - When prettier checks the repo
            - Then it passes without needing Prettier
- 0052.12
    - Given Developer has a project with no config in single quotes, plus the org's synced templates in Prettier defaults
        - When prettier checks the repo
            - Then it passes under the shared config, which keeps only the synced templates at Prettier defaults, from any checkout path
- 0052.5
    - Given Developer has a formatted file and a symbolic link to it staged in a hooked repo, beside an unstaged messy file
        - When Developer commits the staged change
            - Then prettier passes, judging only the staged file and leaving the link out
- 0052.6
    - Given Developer has a messy file and the `--write` argument, with Prettier or Biome
        - When prettier runs with that argument
            - Then the tool gets its own `--write` and the paths, in place of `--check`, and passes; a flag Biome lacks fails by name
- 0052.7
    - Given Developer has a `.prettierignore` file, even without `package.json`
        - When prettier checks the repo
            - Then it fails telling me to delete it
- 0052.8
    - Given Developer has a project without Prettier installed
        - When prettier checks the repo
            - Then it fails with an install hint
- 0052.11
    - Given Developer has a `biome.jsonc`, `@biomejs/biome` at any version under any spec, no Prettier, and more than 20 files Biome would reformat
        - When prettier checks the repo
            - Then Biome 1.8 or later checks formatting, failing naming each of those files; older Biome fails naming its version
