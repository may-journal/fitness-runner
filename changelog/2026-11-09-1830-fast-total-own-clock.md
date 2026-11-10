### 2026.11.09.1830

- Fix: a slow test runner in a pull request's first scan no longer fails every fast check in the second.
- Fix: a stage's total now runs only while its own checks run.
- Fix: a check its stage's total never let start shows as not run, not as failed, and the run still fails.
- Test: requirement 0146.7 runs a slow test runner beside many fast checks.
- Test: requirement 0146.8 starts more stopped checks than the run runs at once.
