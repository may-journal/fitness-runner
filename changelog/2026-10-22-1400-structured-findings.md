### 2026.10.22.1400

- Feat: `fitness --format=json` prints every check's result and findings to stdout, and moves the table to stderr.
- Feat: a check result may carry findings: file, line, rule, message, fix, and the `fitness explain` command.
- Feat: cspell, gofmt, markdown-links, and markdown-no-bold-italic report findings, and error lines stay.
- Feat: a check without findings gets one per error line in the JSON output.
- Feat: in GitHub Actions, annotations and the job summary show the same findings, with the exact file, line, rule, fix and explain command.
