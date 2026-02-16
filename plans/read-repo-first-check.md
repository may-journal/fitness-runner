---
relatedConfigurations: ["../package.json"]
---

# Plan: Read Repo First Check

## Context

Rule text to enforce:

> Before making changes, read the relevant areas of this repo. Conventions and context live in the numbered folders and root docs; use them instead of re-deriving or guessing.
>
> Specifically looking for pre-existing decisions that are relevant to the change you're making. If none exist, strongly consider creating a new decision.

## Problem

This is a behavioral rule for contributors (human or GenAI). Fitness checks validate files and content—they cannot verify that someone "read first." We need machine-checkable proxies.

## Open questions

1. What exactly are "numbered folders"? Repos vary. Common patterns: `1-decisions/`, `2-architecture/`, `3-docs/`; or `docs/1-decisions/`; or no numbering at all.
2. What counts as a "decision"? ADR-style docs, any markdown in a decisions folder, front-matter tagged content?
3. Scope of the check – Should it apply to every repo that uses fitness, or only those that opt in (e.g. via config)?
4. False positives – Trivial changes (typos, deps) should not require a decision. How do we distinguish?

## Design directions

### Direction A: Structure-only validation

- Check that the repo has the expected structure (numbered folders, decisions dir, root docs).
- Pass/fail based on presence. No staged correlation.
- Pro: Simple, always applicable when structure exists.
- Con: Does not enforce "read before change" – only that the convention exists.

### Direction B: Staged correlation (like changelog-updated)

- When staged files include "implementation" paths (e.g. `src/`, `lib/`), require either:
  - A decision doc was also modified, or
  - Commit message references a decision (e.g. `ref: 1-decisions/xyz.md`).
- Pro: Directly ties changes to decision awareness.
- Con: Needs clear definitions of implementation vs. decision paths; commit-msg hook coupling; may annoy on trivial changes.

### Direction C: Advisory / no-op by default

- Check exists but passes unless explicitly configured. Repos opt in via `.fitnessrc` or a marker file.
- Pro: No breaking behavior for repos without numbered folders.
- Con: Easy to ignore; doesn't help discover the convention.

### Direction D: Configurable hybrid

- Phase 1: Structure validation (configurable paths, patterns).
- Phase 2: Staged correlation when structure exists and config enables it.
- Pro: Flexible, can grow over time.
- Con: More complexity; config surface to design.

## Recommendation (draft)

Start with Direction A (structure-only) as a minimal viable check:

- Detect numbered folders at repo root: `readdirSync` + regex `^\d+-`.
- Optionally detect a decisions path (e.g. `1-decisions/`, `decisions/`).
- If none found: pass (do not fail repos that do not use this convention).
- If found: validate that expected root docs exist (e.g. `README.md`). Fail only if structure is broken (e.g. numbered folder empty, README missing when it's the only root doc).

This keeps the check non-invasive while establishing the pattern. Staged correlation can be a follow-up if demand exists.

## Alternatives considered

- Commit message parsing – Would require semantic-commit or a new check to validate decision references. Adds coupling.
- PR body or description – Fitness runs locally and in CI; PR context is not available to checks.
- Warn vs. fail – Fitness checks traditionally fail (exit 1). A "warn" mode would need runner support.

## Next steps

1. Confirm or reject Direction A as the starting point.
2. Define config shape (if any) for paths and patterns.
3. Decide: pass when no numbered folders, or require opt-in (config/marker) to run the check at all?
4. Implement structure validation and tests.
5. Revisit staged correlation in a later iteration.
