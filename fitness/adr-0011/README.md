# 0011 — Skills ship in the bundle

Skills ship in every release bundle.

https://github.com/may-journal/fitness-runner/issues/863

## Check

- Fails a folder in `skills/` whose name is not kebab case, or whose `SKILL.md` is missing or named otherwise.
- Fails `.goreleaser-bundles.yml` when its bundle stops carrying `skills/`.
- Fix: name the folder and its skill alike, and keep `skills/**/*` in the bundle's files.
