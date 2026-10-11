---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# 0004 — One requirement doc per user-facing area

## Context

PR #263 added doc 0066 for a doc-template change, though 0023 already stated that check's promise. Two docs for one area drift apart, and readers miss half the rules.

## Decision

1. Each user-facing area has exactly one requirement doc.
2. A change to an area's behavior extends that area's doc with new acceptance IDs. It never adds a new NNNN doc.
3. Two docs cover one area when their acceptances share a When and have similar Givens.
4. The requirements check fails on such a pair and names both docs.
5. Fold the newer doc into the older one with new IDs and renamed tests, then delete it.

## Consequences

1. Readers find every rule for an area in one place.
2. A doc still holds at most eight acceptances, so extending a full one means merging close acceptances under a new ID.
3. The match is loose, so it may flag docs a person sees as separate areas.
4. Rewording an acceptance to dodge the match breaks [ADR 0001](0001-fix-the-work-not-the-limit.md).
5. Folding retires IDs, and git history keeps them from being reused.
