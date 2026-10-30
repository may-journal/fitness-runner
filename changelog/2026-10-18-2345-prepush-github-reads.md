### 2026.10.18.2345

- Perf: the pre-push gate reads each Plan's labels and comments in one `gh issue view`, once per push.
- Perf: `local-branch` judges the Plans the gate already read, handed over in a private file that only the gate's own `local-branch` run trusts.
- Perf: linked issues, four at a time, and `gh repo view` are read side by side, and one `git log --no-walk` reads pushed subjects.
- Test: requirements 0010.11, 0067.9 and 0068.6 prove a push asks GitHub about each Plan once.
- Test: requirement 0067.4 folds into 0067.9, since that doc holds at most eight acceptances.
