### 2026.10.20.1747

- Feat: Auto-merge merges a native stack whole, through an asynchronous merge of its top pull request, once every pull request is ready.
- Feat: each pull request in a stack gets a comment saying what merged, or which pull request waits and why.
- Feat: the Auto-merge on ready caller also runs on each approval of a branch in the repo, never a fork.
- Fix: a branch chain's comment says the bot cannot merge it, since merging a child dismisses its parent's approval.
- Test: requirement 0095 covers merged, waiting, refused and failed stacks and the review trigger, with the README and adoption guide.
