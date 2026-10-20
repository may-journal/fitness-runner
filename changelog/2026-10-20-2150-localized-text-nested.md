---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.20.2150
- Fix: `localized-text` fails a literal that is a branch of a ternary or `??` expression passed to a view.
- Fix: `localized-text` fails a ternary branch inside an interpolation and passes other literals there, such as dictionary keys.
- Fix: `localized-text` passes a test literal assigned to a variable or passed under an argument label.
- Fix: `localized-text` fails a ternary branch in `Text(verbatim:)` and passes a dictionary literal's value in an interpolation.
