### 2026.11.14.1900

> A monorepo can call the reusable Release Please workflow, and each component moves its own major tag.

- Feat: `release-please-reusable.yml` moves `<component>-vX` for every component Release Please released, and `vX` for the root as before.
- Feat: the workflow outputs `releases`, a JSON array of each released component's path, tag, major tag and commit, for a caller's build job.
- Test: requirement 0092.6 proves two components release together and move only their own major tags.
- Docs: `.github/workflows/README.md` says the major tag step is per component.
- Fix: the step fails with an error on a component tag without its name, instead of moving a wrong tag.
