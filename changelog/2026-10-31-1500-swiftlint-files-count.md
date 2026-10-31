### 2026.10.31.1500

- Fix: swiftlint reports the Swift files it lints, not 0, on a pass or a fail.
- Feat: a pull request run counts only its changed Swift files.
- Feat: a repo with no Swift files, or none left after its config excludes them, still reports 0.
