---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.17.2300

- Fix: plan-check skips a Plan moved to another repo before it is judged, and exits 0 instead of failing the job.
- Fix: the skipped Plan shows as a notice and a job summary line, while any other gh failure still fails.
- Fix: close-check skips an issue moved away in the same way, instead of failing.
