### 2026.11.14.1808

- Feat: fitness finds installed check programs in the `FITNESS_CHECKS_DIR` folder, then beside `fitness`, then on `PATH`.
- Feat: `--describe` reports a `protocol` number, fitness sets `FITNESS_PROTOCOL`, and a mismatch stops the run naming the reinstall command.
- Feat: `fitness --no-builtins` turns the built-in checks off, and a selected built-in that is not installed fails naming its install command.
- Test: requirement 0162 runs the real binary with installed checks in each place.
