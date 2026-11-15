### 2026.11.14.2100

- Fix: comment-check builds the merge commit of a merged pull request when its head commit is gone, instead of failing the install.
- Fix: an open pull request, another ref, another repo or a failed lookup still fails with an error.
- Test: requirement 0169 runs the install script with stand-ins for `go` and `gh` for both cases.
