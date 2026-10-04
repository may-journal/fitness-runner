---
relatedConfigurations: ['../../../.fitnessrc.json', '../../../release-please-config.json']
---

# release-changelog

Correlates a Release Please version with the repository content a reviewer can inspect. The check applies when `.release-please-manifest.json` exists and contains a root release version.

## Behavior

- Use an explicit availability gate and pass with zero files when the repository has no Release Please manifest.
- Pass during bootstrap when the manifest is an empty JSON object.
- Require the root manifest value and `version.txt` to contain the same numeric `major.minor.patch` version.
- Require one `## VERSION` section in `CHANGELOG.md`. A date may follow the version, and the version may use brackets.
- Require that release section to contain at least one `-` or `*` list item describing what ships.
- Fail a scoped manifest change when `CHANGELOG.md` is absent from the same change.
- Reject malformed JSON, unsupported manifests without the root `.` package, mismatched versions, missing sections, empty sections, and duplicate sections.

Timestamped `### yyyy.mm.dd.HHMM` development entries remain under the existing changelog checks.
