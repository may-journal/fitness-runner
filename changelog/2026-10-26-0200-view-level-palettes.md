---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.26.0200
- Feat: fitness ships the five architecture view templates, so `doc-template` judges every repo's views against them even without a copy.
- Feat: `doc-template` fails a diagram or legend whose Mermaid type differs from its template's, such as a flowchart in the components view.
- Feat: each view template gives its level one palette; `doc-template` fails any other style, color or kind in views 01 to 04.
- Fix: `architecture-actors` reads only C4 people, so a flowchart node classed person is no longer an actor.
