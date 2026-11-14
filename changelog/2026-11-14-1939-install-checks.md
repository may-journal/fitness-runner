### 2026.11.14.1939

- Feat: `fitness-install --with-checks` installs each check `checkVersions` pins, verified against its release SHA-256.
- Feat: pins resolve through the fitness-checks release index, or its tags while none is published, and answers are reused for 10 minutes.
- Feat: the action takes `with-checks`, sets `FITNESS_CHECKS_DIR` and outputs `checks-dir`; `--no-builtins` no longer prints a replace note.
- Test: requirement 0164 installs pinned checks from their real releases and runs one.
