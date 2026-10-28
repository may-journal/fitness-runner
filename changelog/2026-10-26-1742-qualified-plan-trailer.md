### 2026.10.26.1742
- Feat: a commit can name a Plan in another repo of its owner as `Plan <owner>/<repo>#NN` or by the Plan's issue URL.
- Feat: the pre-push gate reads a qualified Plan once from the repo it names, and only from the pushing repo's owner.
- Feat: local-branch judges a qualified Plan from its own repo, and a refusal for no trailer names the qualified spelling.
- Fix: plan-trailer fails a misspelled qualified trailer, such as `Plan: owner/repo#NN`, with a hint naming all three spellings.
