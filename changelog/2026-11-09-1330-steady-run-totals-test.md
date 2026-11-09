### 2026.11.09.1330

- Fix: a check a stage's total stopped now reads at least that total.
- Fix: a check that started late on a loaded machine no longer reads under the total it ran past.
- Test: this steadies requirement 0146.4 on slow CI runners.
