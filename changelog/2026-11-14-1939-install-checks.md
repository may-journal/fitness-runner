### 2026.11.14.1939

- Feat: `fitness-install --with-checks` installs each pinned check, verified by SHA-256; a failure warns and the built-ins run, unless `--no-builtins`.
- Feat: pins resolve through the fitness-checks release index, or its tags while none is published, and answers are reused for 10 minutes.
- Feat: the action takes `with-checks`, sets `FITNESS_CHECKS_DIR` and outputs `checks-dir`; `--no-builtins` no longer prints a replace note.
- Fix: after an unanswered release lookup no check installs, each check lookup gives up after 2s, and a missing index is remembered.
- Test: requirement 0164 installs pinned checks from their real releases and runs one.
