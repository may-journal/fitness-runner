### 2026.10.26.1954

- Feat: `workflow-reporting` fails a job with steps but no `if: always()` step writing the job summary, naming the file, line, and step to add.
- Feat: `workflow-reporting` passes a job that only calls a reusable workflow, and judges a matrix job once, as requirement 0133 states.
- Feat: `workflow-reporting` fails a workflow that is not valid YAML, naming the line where reading stopped.
- Fix: `workflow-reporting` README links ADR 0003 at its real `fitness/adr-0003` path, not a missing docs path.
