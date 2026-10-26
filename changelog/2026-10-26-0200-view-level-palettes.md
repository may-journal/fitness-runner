---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.26.0200
- Feat: fitness ships the five architecture view templates, so `doc-template` judges every repo's views against them even without a copy.
- Feat: `doc-template` flags a diagram or legend whose Mermaid type or direction differs from its template's, such as a flowchart in 03.
- Feat: each view template gives its level one palette; `doc-template` flags any other style, color or kind in views 01 to 04.
- Feat: the architecture checks read only C4 people, so a flowchart node classed person is no longer an actor.
- Feat: these four rules warn by default and fail once `.fitnessrc.json` sets `"architectureViews": "error"`; a later release makes that the default.
