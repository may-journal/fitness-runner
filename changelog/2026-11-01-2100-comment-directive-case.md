### 2026.11.01.2100

- Fix: the comment reader skips a tool directive only in the form the tool reads, such as `# shellcheck disable=SC2086`.
- Fix: prose that only starts with a tool name, such as "shellCheck is a repo", is no longer skipped.
- Fix: the prose checks now judge the whole block, including its first line.
- Fix: cSpell directives in any case and ShellCheck directives with any spacing are still skipped.
