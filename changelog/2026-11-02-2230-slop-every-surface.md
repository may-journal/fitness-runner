### 2026.11.02.2230

- Feat: `prose-budget` judges a GitHub body and a commit message by the slop rules a code comment meets.
- Feat: `no-contrastive-reframing` judges a GitHub body and a commit message, and joins the checks every issue, PR and comment body gets.
- Feat: the commit-msg hook runs both, naming each finding as `(commit message)`.
- Test: requirement 0142 proves a PR body draft, a review reply draft and a commit message by real runs.
- Fix: a body or message meets hedging and tutorial tone alone, since it may tell what changed and its footer is no decoration.
