---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.20.1900

- Feat: release bundles carry `skills/`, and `fitness-install --link-skills <folder>` links each skill into an agent's skills folder.
- Feat: one `fitness` skill teaches how to read any failing check from its own docs. No check has a skill of its own.
- Fix: an odd skill file name is skipped, and bundle validation walks `skills/` at any depth.
- Fix: skill files are readable by others, and only this tool's links are replaced or removed.
- Test: requirements 0098.1 to 0098.7 run the real installer. ADR 0011 describes the layout.
