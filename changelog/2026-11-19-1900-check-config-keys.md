### 2026.11.19.1900

> A check names the `.fitnessrc.json` keys it reads in its `--describe`, so the engine's schema and settings follow the check.

- Feat: `--describe` gains a `config` field, and `conf.KeysOf` builds it from a check's options struct without registering it.
- Feat: `fitness config schema`, `fitness config` and `fitness explain` show the keys installed and local checks declare.
- Feat: two checks that declare one key fail `fitness config`, naming both.
