### 2026.10.26.1952
- Feat: `no-plans-dir` judges every path in the repo, failing any `plans`, `mockups`, `validation` or `review-artifacts` folder, in any case.
- Feat: `no-plans-dir` fails files named `*-plan.md`, `*-mockup.*` or `pr-<number>-*`, and Markdown holding a Plan checklist outside `.github/`.
- Feat: each finding names the rule it matched and the issue or PR comment the file belongs in.
- Fix: `no-plans-dir` reads staged additions and rename targets, and fails when it cannot list the repo's files instead of passing.
- Fix: `requirements` passes a renamed doc keeping its number, so 0010 can take a new name; relatedConfigurations .fitnessrc.json goes.
