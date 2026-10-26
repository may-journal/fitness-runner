# 0011 — Skills ship in the bundle

## Context

Shared skills lived by hand in one agent folder, so a new skill reached no agent until someone linked it. Each check already has a README, requirement docs and architecture docs. A skill per check would repeat them.

## Decision

1. A skill is a folder `skills/<name>/` holding a `SKILL.md`, with the name in kebab case.
2. One `fitness` skill teaches how to read any failing check from those docs. Other skills are named for a topic, never a check.
3. Every release bundle carries the `skills/` folder beside the `fitness` binary.
4. `fitness-install --link-skills <folder>` links each bundled skill into the folder, once per agent folder.
5. A rerun repairs a broken link or one to an older bundle, and leaves a real folder or another link alone.

## Consequences

1. A merged skill reaches every agent that reruns the link, with no manual copy.
2. The bundle checks each skill file against its hash, as it does each binary.
3. A skill links existing docs and copies none of their rules.
4. A new check needs only its README and requirement docs to be taught, since the `fitness` skill points readers there.
