---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.25.1500
- Fix: `prose-budget` no longer counts a dash-only line, such as a Measurement's `-` set apart by blank lines, as a paragraph.
- Fix: `requirements` pairs a doc changed only in formatting, such as by `biome format`, with no test change.
- Docs: the requirements template and the add-requirements skill show the Measurement and nesting as Biome 2.6 formats them.
- Test: the requirement suite writes its sample docs in that shape, so every requirements test proves the shape passes.
