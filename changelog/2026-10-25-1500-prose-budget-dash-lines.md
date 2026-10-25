---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.25.1500
- Fix: `prose-budget` and `text-readability` skip a dash-only line, such as a Measurement's `-`, as holding no prose.
- Fix: `requirements` needs no test change for a reformatted doc whose wording is unchanged, and still pairs a test changed beside it.
- Docs: the requirements template and the add-requirements skill show the Measurement and nesting as Biome 2.6 formats them.
- Test: the requirement suite writes its sample docs in that shape, so every requirements test proves the shape passes.
