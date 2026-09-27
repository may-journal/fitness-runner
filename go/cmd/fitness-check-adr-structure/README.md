---
fitnessFunctions: ['adr-structure']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# adr-structure

Validates that every numbered ADR under `docs/architecture/adr` follows the house template, so the record format holds as a check rather than a habit.

The template lives at [`docs/architecture/adr/template.md`](../../../docs/architecture/adr/template.md).

## Rules

- H1 reads `# NNNN — Title`, with a four-digit id and an em-dash separator.
- The H1 id matches the filename id (`0003-...md` carries `# 0003 — ...`).
- Sections are `Context`, `Decision`, `Consequences` in that order.
- A `Status` section is optional and, when present, comes first — the house style uses it only to record a supersede link.
- No other `##` sections appear.

## Behavior

- Pass: every numbered ADR matches the rules; a repo with no ADRs passes with zero files.
- Fail: an ADR breaks a rule → one error per file naming the violation.
- Skipped files: `template.md` and any unnumbered file are not scanned, so the template never trips the id rule it documents.

## Contributing

This README is the canonical description for this check. This check is a self-contained binary (`fitness-check-adr-structure`). To change the section set or heading rule, extend the check and tests here, update the template, and keep the README in sync.
