---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.20.1900

- Feat: release bundles carry `skills/`, and `fitness-install --link-skills <folder>` links each skill into an agent's skills folder.
- Fix: an odd skill file name is skipped, not fatal, and bundle validation walks `skills/` at any depth.
- Fix: skill files are readable by others, links to dropped skills are removed, and only this tool's own links are replaced.
- Feat: every check keeps a skill named for it in `go/cmd/<check>/skills/`, which links its README and shows how to fix each failure.
- Test: requirements 0098.1 to 0098.6 run the real installer. ADR 0011 describes the layout.
