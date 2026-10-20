---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.20.1900

- Feat: release bundles carry `skills/`, and `fitness-install --link-skills <folder>` links each skill into an agent's skills folder.
- Fix: an odd skill file name is skipped, not fatal, and bundle validation walks `skills/` at any depth.
- Fix: skill files are readable by others, and links to dropped skills are removed.
- Fix: only links into this tool's own bundles are replaced, through unique temporary names.
- Test: requirements 0098.1 to 0098.5 run the real installer. ADR 0011 describes the layout.
