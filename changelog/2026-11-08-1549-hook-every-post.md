### 2026.11.08.1549

- Feat: the Claude Code hook judges a post by the target repo's rules, or the org defaults outside it.
- Feat: it denies a body it can't read, such as a variable, a pipe or a file the same line writes.
- Feat: it judges every `gh` call in a command, including loops, `bash -c`, `xargs` and `timeout`.
- Feat: it judges GitHub MCP tools and review replies, and gives PR, Plan and Epic bodies their structure checks.
- Fix: cspell skips the hidden debug block in Renovate PR bodies. Requirements 0155 and 0156 prove each route.
