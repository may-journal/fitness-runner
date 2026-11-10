### 2026.11.09.1830

- Fix: a slow test runner in a pull request's first scan no longer fails every fast check in the second.
- Fix: a stage's total now runs only while its own checks run.
- Fix: a check its stage's total never let start shows as not run, not as failed, and the Claude hook names it.
- Test: requirements 0146.7 and 0146.8 run a slow test runner beside fast checks, and more stopped checks than run at once.
- Test: requirement 0001.14 runs only the checks it judges, so each starts on a runner with few cores.
