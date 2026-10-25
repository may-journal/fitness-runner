---
name: no-contrastive-reframing
description: Fix a failing no-contrastive-reframing check on markdown prose. Use when a sentence is refused for the "not X, it's Y" pattern.
---

# State the point directly

Prose may not reject a claim and then restate the real point. [The check README](../../README.md) shows both forms it flags and what it leaves alone.

## Read the failure

- `docs/a.md: rewrite as a direct statement, not the "not X, it's Y" pattern: "It's not a workout. It's a lifestyle."`: keep only the claim you mean, as in `Training is a lifestyle.`

A plain negation such as `The check does not read lockfiles.` is fine. `not only X but also Y` is fine. Quoted text and code are never judged.

## Check it first

Run `fitness no-contrastive-reframing` from the repo root. Exit 0 means no sentence matches.
