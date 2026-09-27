---
fitnessFunctions: ['adr-structure']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# adr-structure

Validates that every numbered ADR under `docs/architecture/adr` follows the house template, so the record format holds as a check rather than a habit.

The template lives at [`docs/architecture/adr/template.md`](../../../docs/architecture/adr/template.md) and is shared across may-journal repos.

## Rules

- H1 reads `# NNNN — Title`, with a four-digit id and an em-dash separator.
- The H1 id matches the filename id (`0003-...md` carries `# 0003 — ...`).
- The sections `Context`, `Decision`, `Consequences` are all present, and no other `##` section appears.
- A superseding decision states so in `Context`, not a `Status` section.

The section rules reuse the shared `mdtemplate` mechanism that also backs `plan-structure` and `pr-structure`; only the H1/id rule is ADR-specific.

## Behavior

- Pass: every numbered ADR matches the rules; a repo with no ADRs passes with zero files.
- Fail: an ADR breaks a rule → one error per file naming the violation.
- Skipped files: `template.md` and any unnumbered file are not scanned, so the template never trips the id rule it documents.

## Contributing

This README is the canonical description for this check. This check is a self-contained binary (`fitness-check-adr-structure`). To change the section set or heading rule, extend the check and tests here, update the template, and keep the README in sync.
