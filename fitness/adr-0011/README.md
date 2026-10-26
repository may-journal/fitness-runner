# ADR 0011 check

Keeps [ADR 0011](../../docs/architecture/adr/0011-skills-ship-in-the-bundle.md): skills ship in every release bundle.

- Fails a folder in `skills/` whose name is not kebab case, or whose `SKILL.md` is missing or named otherwise.
- Fails `.goreleaser-bundles.yml` when its bundle stops carrying `skills/`.
- Fix: name the folder and its skill alike, and keep `skills/**/*` in the bundle's files.
