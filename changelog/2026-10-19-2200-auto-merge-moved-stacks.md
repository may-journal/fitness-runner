### 2026.10.19.2200

- Fix: Auto-merge passes with a notice when GitHub still counts a pull request on the default branch as stacked and refuses auto-merge.
- Test: requirement 0091.3 proves it, and 0091.4 proves any other auto-merge failure still fails the job.
- Docs: the workflows README explains stacked pull requests GitHub moved onto the default branch.
