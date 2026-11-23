### 2026.11.23.1430

> A check named required now fails the run when it cannot judge, so CI can prove it really ran.

- Feat: `requiredChecks` in `.fitnessrc.json` and `--require=name,name` name the required checks, merged as one set.
- Feat: A required check's internal error shows as `failed` with exit 1, and other checks keep reporting info.
- Feat: An unknown required name fails setup with the known names, as does one the run leaves out or turns off.
- Docs: `docs/check-protocol.md` gains a "Required checks" section, and requirement 0004.7 states the behavior.
