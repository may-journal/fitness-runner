---
relatedConfigurations: ['../../../.goreleaser-bundles.yml']
---

# 0011 — Skills ship in the bundle

## Context

Shared skills lived by hand in one agent folder, so a new skill reached no agent until someone linked it. Each check needs a stable place for the skill that teaches it.

## Decision

1. A skill is a folder `skills/<name>/` holding a `SKILL.md`, with the name in kebab case.
2. The folder sits inside the code it teaches, as `go/cmd/<command>/skills/<name>/`. No skill sits in a top-level folder.
3. Every check has a skill that takes the check's name. A skill for a topic sits beside the command closest to it.
4. `fitness-release skills` gathers every skill into the bundle's one `skills/` folder beside the `fitness` binary.
5. `fitness-install --link-skills <folder>` links each bundled skill into the folder, once per agent folder.
6. A rerun repairs a broken link or one to an older bundle, and leaves a real folder or another link alone.

## Consequences

1. A merged skill reaches every agent that reruns the link, with no manual copy.
2. The bundle checks each skill file against its hash, as it does each binary.
3. A skill links existing docs and copies none of their rules.
4. A change to a check and its skill land in one folder, so a reviewer sees both together.
5. Two commands that ship a skill of the same name fail the release build.
