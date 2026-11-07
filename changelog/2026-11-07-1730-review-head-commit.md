### 2026.11.07.1730

- Fix: a review's comment check in this repo builds the pull request's head commit, since go install cannot fetch its merge commit.
- Ci: issue and close checks keep building the main commit their event names.
- Docs: the comment check caller says why a pull request builds its head commit.
