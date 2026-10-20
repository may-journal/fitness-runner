---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.20.1900

- Feat: release bundles carry the `skills/` folder, one folder per skill.
- Feat: `fitness-install --link-skills <folder>` links each skill into an agent's skills folder and repairs broken links.
- Test: requirements 0098.1 and 0098.2 run the real installer to link skills and repair a broken link.
- Docs: ADR 0011 and the development guide describe the skill layout.
